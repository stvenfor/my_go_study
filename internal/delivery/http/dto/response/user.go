// user.go 用户相关响应 DTO（JSON 使用 camelCase）。
package response

import (
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

// UserItem 用户基础信息。
type UserItem struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

// UserProfile 用户详情（不含 updatedAt 时可省略）。
type UserProfile struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AuthUserItem 认证接口返回的用户信息（Supabase UUID 为字符串 id）。
type AuthUserItem struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// LoginData 登录成功响应 data。
type LoginData struct {
	Token        string       `json:"token"`
	RefreshToken string       `json:"refresh_token"`
	SessionID    string       `json:"session_id"`
	User         AuthUserItem `json:"user"`
}

// RefreshTokenData refresh 成功响应 data。
type RefreshTokenData struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	SessionID    string `json:"session_id,omitempty"`
}

// FromSupabaseAuthUser 从 Supabase 认证结果转换。
func FromSupabaseAuthUser(userID, username, email string) AuthUserItem {
	return AuthUserItem{
		ID:       userID,
		Username: username,
		Email:    email,
	}
}

// FromUser 从领域实体转换为用户响应项。
func FromUser(user *entity.User) UserItem {
	return UserItem{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

// FromUserProfile 从领域实体转换为用户详情。
func FromUserProfile(user *entity.User) UserProfile {
	return UserProfile{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
