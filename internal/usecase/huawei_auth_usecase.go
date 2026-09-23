package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/stvenfor/my_go_study/pkg/config"
)

var (
	// ErrHuaweiNotConfigured 未配置 HUAWEI_CLIENT_ID / HUAWEI_CLIENT_SECRET。
	ErrHuaweiNotConfigured = errors.New("华为登录未配置：请在 .env 填写 HUAWEI_CLIENT_ID 与 HUAWEI_CLIENT_SECRET")
	// ErrHuaweiAuthFailed 华为 OAuth 换票或取用户信息失败。
	ErrHuaweiAuthFailed = errors.New("华为授权失败")
	// ErrHuaweiLocalOnly 当前仅 local Auth 支持华为登录落库。
	ErrHuaweiLocalOnly = errors.New("华为登录仅在 auth.provider=local 时可用")
)

// HuaweiLoginAuth 华为账号 Authorization Code 登录。
type HuaweiLoginAuth interface {
	LoginWithHuaweiCode(ctx context.Context, code string) (*SupabaseAuthOutput, error)
}

// HuaweiUnionAccount local Auth 按 UnionID / 手机号找或建号。
type HuaweiUnionAccount interface {
	LoginOrRegisterHuawei(ctx context.Context, unionID, openID, phone, nickname string) (*SupabaseAuthOutput, error)
}

// HuaweiAuthUsecase 调华为 OAuth + getInfo，再交给 local Auth 发会话。
type HuaweiAuthUsecase struct {
	cfg    config.ThirdPartyConfig
	local  HuaweiUnionAccount
	client *http.Client
}

// NewHuaweiAuthUsecase 创建华为登录用例。
func NewHuaweiAuthUsecase(cfg config.ThirdPartyConfig, local HuaweiUnionAccount) *HuaweiAuthUsecase {
	return &HuaweiAuthUsecase{
		cfg:   cfg,
		local: local,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// LoginWithHuaweiCode 用一键登录 authorization code 换 UnionID/手机号并登录。
func (u *HuaweiAuthUsecase) LoginWithHuaweiCode(ctx context.Context, code string) (*SupabaseAuthOutput, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, ErrInvalidParams
	}
	if u.local == nil {
		return nil, ErrHuaweiLocalOnly
	}
	if !u.cfg.HuaweiLoginConfigured() {
		return nil, ErrHuaweiNotConfigured
	}

	token, err := u.exchangeCode(ctx, code)
	if err != nil {
		return nil, err
	}
	info, err := u.fetchUserInfo(ctx, token.AccessToken)
	if err != nil {
		return nil, err
	}
	unionID := firstNonEmpty(info.UnionID, info.UnionId)
	openID := firstNonEmpty(info.OpenID, info.OpenId)
	phone := firstNonEmpty(info.LoginMobileNumber, info.MobileNumber)
	nickname := firstNonEmpty(info.DisplayName, info.NickName)
	if unionID == "" && openID == "" {
		return nil, fmt.Errorf("%w: 未返回 UnionID/OpenID", ErrHuaweiAuthFailed)
	}
	if unionID == "" {
		unionID = openID
	}
	return u.local.LoginOrRegisterHuawei(ctx, unionID, openID, phone, nickname)
}

type huaweiTokenResp struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
	SubError     int    `json:"sub_error"`
}

func (u *HuaweiAuthUsecase) exchangeCode(ctx context.Context, code string) (*huaweiTokenResp, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", strings.TrimSpace(u.cfg.HuaweiClientID))
	form.Set("client_secret", strings.TrimSpace(u.cfg.HuaweiClientSecret))
	if uri := strings.TrimSpace(u.cfg.HuaweiRedirectURI); uri != "" {
		form.Set("redirect_uri", uri)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://oauth-login.cloud.huawei.com/oauth2/v3/token",
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrHuaweiAuthFailed, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: read body", ErrHuaweiAuthFailed)
	}
	var parsed huaweiTokenResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("%w: bad json", ErrHuaweiAuthFailed)
	}
	if parsed.AccessToken == "" || parsed.Error != "" {
		msg := strings.TrimSpace(parsed.ErrorDesc)
		if msg == "" {
			msg = strings.TrimSpace(parsed.Error)
		}
		if msg == "" {
			msg = "token exchange failed"
		}
		return nil, fmt.Errorf("%w: %s", ErrHuaweiAuthFailed, msg)
	}
	return &parsed, nil
}

type huaweiUserInfoResp struct {
	OpenID            string `json:"openID"`
	OpenId            string `json:"openId"`
	UnionID           string `json:"unionID"`
	UnionId           string `json:"unionId"`
	DisplayName       string `json:"displayName"`
	NickName          string `json:"nickName"`
	LoginMobileNumber string `json:"loginMobileNumber"`
	MobileNumber      string `json:"mobileNumber"`
	Error             string `json:"error"`
	ErrorDesc         string `json:"error_description"`
}

func (u *HuaweiAuthUsecase) fetchUserInfo(ctx context.Context, accessToken string) (*huaweiUserInfoResp, error) {
	form := url.Values{}
	form.Set("access_token", accessToken)
	form.Set("getNickName", "1")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://account.cloud.huawei.com/rest.php?nsp_svc=GOpen.User.getInfo",
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrHuaweiAuthFailed, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: read userinfo", ErrHuaweiAuthFailed)
	}
	var parsed huaweiUserInfoResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("%w: bad userinfo json", ErrHuaweiAuthFailed)
	}
	if parsed.Error != "" {
		msg := strings.TrimSpace(parsed.ErrorDesc)
		if msg == "" {
			msg = parsed.Error
		}
		return nil, fmt.Errorf("%w: %s", ErrHuaweiAuthFailed, msg)
	}
	return &parsed, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
