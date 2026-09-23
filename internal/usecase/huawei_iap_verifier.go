package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/pkg/config"
)

// LiveHuaweiSubscriptionVerifier 调用华为 IAP 订阅查询（JWT 鉴权）。
// 未配置密钥时 Configured()=false，由 MembershipUsecase 走本地 dev 路径。
type LiveHuaweiSubscriptionVerifier struct {
	cfg    config.HuaweiIAPConfig
	client *http.Client
}

func NewLiveHuaweiSubscriptionVerifier(cfg config.HuaweiIAPConfig) *LiveHuaweiSubscriptionVerifier {
	return &LiveHuaweiSubscriptionVerifier{
		cfg: cfg,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (v *LiveHuaweiSubscriptionVerifier) Configured() bool {
	return v != nil && v.cfg.Configured()
}

func (v *LiveHuaweiSubscriptionVerifier) Verify(ctx context.Context, in HuaweiVerifyInput, plan entity.MembershipPlan) (*HuaweiVerifyResult, error) {
	if !v.Configured() {
		return nil, ErrMembershipHuaweiNotConfig
	}
	token, err := v.issueJWT()
	if err != nil {
		return nil, fmt.Errorf("%w: jwt: %v", ErrMembershipHuaweiInvalid, err)
	}
	root := strings.TrimRight(strings.TrimSpace(v.cfg.SubscriptionRootURL), "/")
	if root == "" {
		root = "https://subscr-drcn.iap.cloud.huawei.com.cn"
	}
	body := map[string]string{
		"purchaseToken":  strings.TrimSpace(in.PurchaseToken),
		"subscriptionId": strings.TrimSpace(in.SubscriptionID),
	}
	if body["subscriptionId"] == "" {
		body["subscriptionId"] = strings.TrimSpace(in.PurchaseOrderID)
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, root+"/sub/applications/v2/purchases/get", strings.NewReader(string(raw)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMembershipHuaweiInvalid, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: http %d %s", ErrMembershipHuaweiInvalid, resp.StatusCode, string(b))
	}
	var parsed struct {
		ResponseCode    string `json:"responseCode"`
		ResponseMessage string `json:"responseMessage"`
	}
	_ = json.Unmarshal(b, &parsed)
	if parsed.ResponseCode != "" && parsed.ResponseCode != "0" {
		return nil, fmt.Errorf("%w: %s %s", ErrMembershipHuaweiInvalid, parsed.ResponseCode, parsed.ResponseMessage)
	}
	_ = plan
	return &HuaweiVerifyResult{
		PurchaseToken:  strings.TrimSpace(in.PurchaseToken),
		SubscriptionID: strings.TrimSpace(in.SubscriptionID),
		AckRequired:    true,
	}, nil
}

func (v *LiveHuaweiSubscriptionVerifier) issueJWT() (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss": v.cfg.IssuerID,
		"aud": "https://oauth-login.cloud.huawei.com/oauth2/v3/token",
		"iat": now.Unix(),
		"exp": now.Add(5 * time.Minute).Unix(),
		"aid": v.cfg.AppID,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(normalizeHuaweiPEM(v.cfg.PrivateKeyPEM)))
	if err != nil {
		return "", err
	}
	if kid := strings.TrimSpace(v.cfg.KeyID); kid != "" {
		t.Header["kid"] = kid
	}
	return t.SignedString(key)
}

func normalizeHuaweiPEM(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, `\n`, "\n")
	return s
}
