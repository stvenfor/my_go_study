// =============================================================================
// 文件：supabase.go
// 作用：校验 Supabase access token（业务 API 中间件的核心）
//
// 【初学者】为什么不本地解析 JWT？
//   调 Supabase /auth/v1/user 由官方校验签名与过期，避免自己维护 JWT 密钥逻辑。
// =============================================================================
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

// SupabaseUser 从 token 解析出的用户摘要，注入 Gin Context 供 Controller 使用。
type SupabaseUser struct {
	ID    string `json:"id"`    // UUID，与 transactions.user_id 对应
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type userResponse struct {
	User SupabaseUser `json:"user"`
}

// ValidateAccessToken 用用户 token 问 Supabase「这个 token 还有效吗？」
func ValidateAccessToken(ctx context.Context, cfg config.SupabaseConfig, accessToken string) (SupabaseUser, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return SupabaseUser{}, fmt.Errorf("缺少 access token")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.URL+"/auth/v1/user", nil)
	if err != nil {
		return SupabaseUser{}, err
	}
	// apikey 用 anon；Authorization 用用户的 access_token
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

	// Supabase 响应格式可能包一层 {"user":{...}} 或直接用户对象
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
