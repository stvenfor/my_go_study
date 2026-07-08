package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/stvenfor/my_go_study/internal/config"
)

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type userResponse struct {
	User User `json:"user"`
}

func ValidateAccessToken(ctx context.Context, cfg config.SupabaseConfig, accessToken string) (User, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return User{}, fmt.Errorf("缺少 access token")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.URL+"/auth/v1/user", nil)
	if err != nil {
		return User{}, err
	}
	req.Header.Set("apikey", cfg.AnonKey)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return User{}, fmt.Errorf("验证 token 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return User{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return User{}, fmt.Errorf("token 无效 (%d): %s", resp.StatusCode, body)
	}

	var payload userResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return User{}, fmt.Errorf("解析用户信息失败: %w", err)
	}
	if payload.User.ID == "" {
		return User{}, fmt.Errorf("token 无效: 未返回用户 ID")
	}
	return payload.User, nil
}
