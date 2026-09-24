package rongcloud

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultAPIBase = "https://api.rong-api.com"

// Config 融云服务端凭证（国内数据中心默认 api.rong-api.com）。
type Config struct {
	AppKey     string
	AppSecret  string
	APIBaseURL string
	HTTPClient *http.Client
}

// Configured 是否具备签发 Token 所需密钥（占位 AppKey 不算）。
func (c Config) Configured() bool {
	key := strings.TrimSpace(c.AppKey)
	secret := strings.TrimSpace(c.AppSecret)
	if key == "" || secret == "" {
		return false
	}
	return !strings.Contains(strings.ToUpper(key), "PLACEHOLDER")
}

// Client 融云 Server API 客户端。
type Client struct {
	cfg Config
}

// NewClient 创建客户端；未配置时仍可构造，调用返回 ErrNotConfigured。
func NewClient(cfg Config) *Client {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	if strings.TrimSpace(cfg.APIBaseURL) == "" {
		cfg.APIBaseURL = defaultAPIBase
	}
	return &Client{cfg: cfg}
}

// ErrNotConfigured 未填 App Key / App Secret。
var ErrNotConfigured = fmt.Errorf("rongcloud not configured")

// TokenResult getToken 成功响应。
type TokenResult struct {
	Code   int    `json:"code"`
	Token  string `json:"token"`
	UserID string `json:"userId"`
}

// GetToken 注册/换取用户 Token（userId=业务 UUID）。
func (c *Client) GetToken(ctx context.Context, userID, name string) (TokenResult, error) {
	if c == nil || !c.cfg.Configured() {
		return TokenResult{}, ErrNotConfigured
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return TokenResult{}, fmt.Errorf("rongcloud userId empty")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = userID
	}
	form := url.Values{}
	form.Set("userId", userID)
	form.Set("name", name)
	var out TokenResult
	if err := c.postForm(ctx, "/user/getToken.json", form, &out); err != nil {
		return TokenResult{}, err
	}
	if out.Code != 200 {
		return TokenResult{}, fmt.Errorf("rongcloud getToken code=%d", out.Code)
	}
	if strings.TrimSpace(out.Token) == "" {
		return TokenResult{}, fmt.Errorf("rongcloud getToken empty token")
	}
	if strings.TrimSpace(out.UserID) == "" {
		out.UserID = userID
	}
	return out, nil
}

// CreateGroup 创建群组并把成员加入。
func (c *Client) CreateGroup(ctx context.Context, groupID, groupName string, memberIDs []string) error {
	if c == nil || !c.cfg.Configured() {
		return ErrNotConfigured
	}
	form := url.Values{}
	form.Set("groupId", strings.TrimSpace(groupID))
	form.Set("groupName", strings.TrimSpace(groupName))
	for _, id := range memberIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			form.Add("userId", id)
		}
	}
	var out struct {
		Code int `json:"code"`
	}
	if err := c.postForm(ctx, "/group/create.json", form, &out); err != nil {
		return err
	}
	if out.Code != 200 {
		return fmt.Errorf("rongcloud createGroup code=%d", out.Code)
	}
	return nil
}

// JoinGroup 拉用户入群。
func (c *Client) JoinGroup(ctx context.Context, groupID string, memberIDs []string) error {
	if c == nil || !c.cfg.Configured() {
		return ErrNotConfigured
	}
	form := url.Values{}
	form.Set("groupId", strings.TrimSpace(groupID))
	for _, id := range memberIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			form.Add("userId", id)
		}
	}
	var out struct {
		Code int `json:"code"`
	}
	if err := c.postForm(ctx, "/group/join.json", form, &out); err != nil {
		return err
	}
	if out.Code != 200 {
		return fmt.Errorf("rongcloud joinGroup code=%d", out.Code)
	}
	return nil
}

// QuitGroup 用户退群。
func (c *Client) QuitGroup(ctx context.Context, groupID string, memberIDs []string) error {
	if c == nil || !c.cfg.Configured() {
		return ErrNotConfigured
	}
	form := url.Values{}
	form.Set("groupId", strings.TrimSpace(groupID))
	for _, id := range memberIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			form.Add("userId", id)
		}
	}
	var out struct {
		Code int `json:"code"`
	}
	if err := c.postForm(ctx, "/group/quit.json", form, &out); err != nil {
		return err
	}
	if out.Code != 200 {
		return fmt.Errorf("rongcloud quitGroup code=%d", out.Code)
	}
	return nil
}

// DismissGroup 解散群。
func (c *Client) DismissGroup(ctx context.Context, groupID, operatorUserID string) error {
	if c == nil || !c.cfg.Configured() {
		return ErrNotConfigured
	}
	form := url.Values{}
	form.Set("groupId", strings.TrimSpace(groupID))
	form.Set("userId", strings.TrimSpace(operatorUserID))
	var out struct {
		Code int `json:"code"`
	}
	if err := c.postForm(ctx, "/group/dismiss.json", form, &out); err != nil {
		return err
	}
	if out.Code != 200 {
		return fmt.Errorf("rongcloud dismissGroup code=%d", out.Code)
	}
	return nil
}

func (c *Client) postForm(ctx context.Context, path string, form url.Values, dest any) error {
	nonce := strconv.FormatInt(time.Now().UnixNano()%1e15, 10)
	if len(nonce) > 18 {
		nonce = nonce[:18]
	}
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	sig := sha1Hex(c.cfg.AppSecret + nonce + ts)

	endpoint := strings.TrimRight(c.cfg.APIBaseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("App-Key", strings.TrimSpace(c.cfg.AppKey))
	req.Header.Set("Nonce", nonce)
	req.Header.Set("Timestamp", ts)
	req.Header.Set("Signature", sig)

	resp, err := c.cfg.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("rongcloud http %d: %s", resp.StatusCode, truncate(string(body), 200))
	}
	if dest == nil {
		return nil
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("rongcloud decode: %w", err)
	}
	return nil
}

func sha1Hex(s string) string {
	sum := sha1.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
