// supabase.go 通过 Supabase Auth API 校验用户 access token。
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/stvenfor/my_go_study/pkg/config"
)

// SupabaseUser Supabase 认证用户摘要。
type SupabaseUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type userResponse struct {
	User SupabaseUser `json:"user"`
}

// ValidateAccessToken 调用 /auth/v1/user 校验 token 并返回用户信息。
func ValidateAccessToken(ctx context.Context, cfg config.SupabaseConfig, accessToken string) (SupabaseUser, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return SupabaseUser{}, fmt.Errorf("缺少 access token")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.URL+"/auth/v1/user", nil)
	if err != nil {
		return SupabaseUser{}, err
	}
	req.Header.Set("apikey", cfg.AnonKey)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return SupabaseUser{}, fmt.Errorf("验证 token 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return SupabaseUser{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return SupabaseUser{}, fmt.Errorf("token 无效 (%d): %s", resp.StatusCode, body)
	}

	// Supabase 可能返回 {"user":{...}} 或直接返回用户对象 {"id":...}
	var wrapped userResponse
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return SupabaseUser{}, fmt.Errorf("解析用户信息失败: %w", err)
	}
	if wrapped.User.ID != "" {
		return wrapped.User, nil
	}

	var direct SupabaseUser
	if err := json.Unmarshal(body, &direct); err != nil {
		return SupabaseUser{}, fmt.Errorf("解析用户信息失败: %w", err)
	}
	if direct.ID == "" {
		return SupabaseUser{}, fmt.Errorf("token 无效: 未返回用户 ID")
	}
	return direct, nil
}
