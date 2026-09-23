// Package jpush 调用极光 REST v3 推送（无官方 Go SDK，用 stdlib HTTP）。
package jpush

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultPushURL = "https://api.jpush.cn/v3/push"

// Config 极光服务端凭证。
type Config struct {
	AppKey       string
	MasterSecret string
	PushURL      string
	// APNsProduction 对应 options.apns_production；dev 通常 false。
	APNsProduction bool
	HTTPClient     *http.Client
}

// Configured 是否具备发推送所需密钥。
func (c Config) Configured() bool {
	return strings.TrimSpace(c.AppKey) != "" &&
		strings.TrimSpace(c.MasterSecret) != "" &&
		!strings.Contains(strings.ToUpper(c.AppKey), "PLACEHOLDER")
}

// Notification 推送内容。
type Notification struct {
	Title    string
	Alert    string
	Deeplink string
	Extras   map[string]any
	// Platform: all | android | ios | hmos；空=all。
	Platform string
	// AudienceAlias 按别名（通常=user_id）。
	AudienceAlias []string
	// AudienceRegistrationIDs 按 RegistrationID。
	AudienceRegistrationIDs []string
}

// Result 极光响应摘要。
type Result struct {
	MsgID  string `json:"msg_id"`
	SendNo string `json:"sendno"`
	Raw    map[string]any
}

// Client 极光推送客户端。
type Client struct {
	cfg Config
}

// NewClient 创建客户端；未配置时仍可构造，Send 返回 ErrNotConfigured。
func NewClient(cfg Config) *Client {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	if strings.TrimSpace(cfg.PushURL) == "" {
		cfg.PushURL = defaultPushURL
	}
	return &Client{cfg: cfg}
}

// ErrNotConfigured 未填 AppKey/MasterSecret。
var ErrNotConfigured = fmt.Errorf("jpush not configured")

// Send 发送一条通知（含 extras.deeplink，三端共用）。
func (c *Client) Send(ctx context.Context, n Notification) (Result, error) {
	if c == nil || !c.cfg.Configured() {
		return Result{}, ErrNotConfigured
	}
	if len(n.AudienceAlias) == 0 && len(n.AudienceRegistrationIDs) == 0 {
		return Result{}, fmt.Errorf("jpush audience empty")
	}
	alert := strings.TrimSpace(n.Alert)
	if alert == "" {
		alert = strings.TrimSpace(n.Title)
	}
	if alert == "" {
		return Result{}, fmt.Errorf("jpush alert empty")
	}

	extras := map[string]any{}
	for k, v := range n.Extras {
		extras[k] = v
	}
	if dl := strings.TrimSpace(n.Deeplink); dl != "" {
		extras["deeplink"] = dl
	}

	platform := strings.TrimSpace(n.Platform)
	if platform == "" {
		platform = "all"
	}

	audience := map[string]any{}
	if len(n.AudienceAlias) > 0 {
		audience["alias"] = n.AudienceAlias
	}
	if len(n.AudienceRegistrationIDs) > 0 {
		audience["registration_id"] = n.AudienceRegistrationIDs
	}

	title := strings.TrimSpace(n.Title)
	android := map[string]any{"alert": alert, "extras": extras}
	if title != "" {
		android["title"] = title
	}
	ios := map[string]any{
		"alert":  map[string]any{"title": title, "body": alert},
		"sound":  "default",
		"extras": extras,
	}
	hmos := map[string]any{"alert": alert, "title": title, "extras": extras}

	body := map[string]any{
		"platform": platform,
		"audience": audience,
		"notification": map[string]any{
			"alert":   alert,
			"android": android,
			"ios":     ios,
			"hmos":    hmos,
		},
		"options": map[string]any{
			"apns_production": c.cfg.APNsProduction,
		},
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return Result{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.PushURL, bytes.NewReader(raw))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+basicAuth(c.cfg.AppKey, c.cfg.MasterSecret))

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	var parsed map[string]any
	_ = json.Unmarshal(respBody, &parsed)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Result{Raw: parsed}, fmt.Errorf("jpush http %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	out := Result{Raw: parsed}
	if v, ok := parsed["msg_id"].(string); ok {
		out.MsgID = v
	} else if v, ok := parsed["msg_id"].(float64); ok {
		out.MsgID = fmt.Sprintf("%.0f", v)
	}
	if v, ok := parsed["sendno"].(string); ok {
		out.SendNo = v
	}
	return out, nil
}

func basicAuth(appKey, masterSecret string) string {
	token := appKey + ":" + masterSecret
	return base64.StdEncoding.EncodeToString([]byte(token))
}
