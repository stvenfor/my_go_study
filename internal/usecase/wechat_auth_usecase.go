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
	// ErrWechatNotConfigured 未配置 WECHAT_APP_ID / WECHAT_APP_SECRET。
	ErrWechatNotConfigured = errors.New("微信登录未配置：请在 .env 填写 WECHAT_APP_ID 与 WECHAT_APP_SECRET")
	// ErrWechatAuthFailed 微信 oauth 换票失败。
	ErrWechatAuthFailed = errors.New("微信授权失败")
	// ErrWechatLocalOnly 当前仅 local Auth 支持微信登录落库。
	ErrWechatLocalOnly = errors.New("微信登录仅在 auth.provider=local 时可用")
)

// WeChatLoginAuth 微信授权 code 登录。
type WeChatLoginAuth interface {
	LoginWithWechatCode(ctx context.Context, code string) (*SupabaseAuthOutput, error)
}

// WeChatOpenIDAccount local Auth 按 openid 找或建号。
type WeChatOpenIDAccount interface {
	LoginOrRegisterWechat(ctx context.Context, openID, nickname, avatarURL string) (*SupabaseAuthOutput, error)
}

// WeChatAuthUsecase 调微信 oauth2，再交给 local Auth 发会话。
type WeChatAuthUsecase struct {
	cfg    config.ThirdPartyConfig
	local  WeChatOpenIDAccount
	client *http.Client
}

// NewWeChatAuthUsecase 创建微信登录用例。
func NewWeChatAuthUsecase(cfg config.ThirdPartyConfig, local WeChatOpenIDAccount) *WeChatAuthUsecase {
	return &WeChatAuthUsecase{
		cfg:   cfg,
		local: local,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// LoginWithWechatCode 用移动应用授权 code 换 openid 并登录。
func (u *WeChatAuthUsecase) LoginWithWechatCode(ctx context.Context, code string) (*SupabaseAuthOutput, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, ErrInvalidParams
	}
	if u.local == nil {
		return nil, ErrWechatLocalOnly
	}
	if !u.cfg.WeChatLoginConfigured() {
		return nil, ErrWechatNotConfigured
	}

	token, err := u.exchangeCode(ctx, code)
	if err != nil {
		return nil, err
	}
	nickname, avatar := "", ""
	if token.AccessToken != "" && token.OpenID != "" {
		nickname, avatar = u.fetchProfile(ctx, token.AccessToken, token.OpenID)
	}
	return u.local.LoginOrRegisterWechat(ctx, token.OpenID, nickname, avatar)
}

type wechatTokenResp struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	UnionID      string `json:"unionid"`
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
}

func (u *WeChatAuthUsecase) exchangeCode(ctx context.Context, code string) (*wechatTokenResp, error) {
	q := url.Values{}
	q.Set("appid", strings.TrimSpace(u.cfg.WeChatAppID))
	q.Set("secret", strings.TrimSpace(u.cfg.WeChatAppSecret))
	q.Set("code", code)
	q.Set("grant_type", "authorization_code")
	reqURL := "https://api.weixin.qq.com/sns/oauth2/access_token?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWechatAuthFailed, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: read body", ErrWechatAuthFailed)
	}
	var parsed wechatTokenResp
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("%w: bad json", ErrWechatAuthFailed)
	}
	if parsed.ErrCode != 0 || parsed.OpenID == "" {
		msg := strings.TrimSpace(parsed.ErrMsg)
		if msg == "" {
			msg = "oauth failed"
		}
		return nil, fmt.Errorf("%w: %s", ErrWechatAuthFailed, msg)
	}
	return &parsed, nil
}

type wechatUserInfoResp struct {
	Nickname string `json:"nickname"`
	HeadImg  string `json:"headimgurl"`
}

func (u *WeChatAuthUsecase) fetchProfile(ctx context.Context, accessToken, openID string) (string, string) {
	q := url.Values{}
	q.Set("access_token", accessToken)
	q.Set("openid", openID)
	reqURL := "https://api.weixin.qq.com/sns/userinfo?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", ""
	}
	resp, err := u.client.Do(req)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var info wechatUserInfoResp
	if json.Unmarshal(body, &info) != nil {
		return "", ""
	}
	return strings.TrimSpace(info.Nickname), strings.TrimSpace(info.HeadImg)
}
