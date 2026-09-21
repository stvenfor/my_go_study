// user.go 用户相关响应 DTO（JSON 使用 camelCase）。
package response

import (
	"time"
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

// AuthUserItem 认证接口返回的用户信息。id 与 user_id 相同。
type AuthUserItem struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	UserName  string `json:"user_name"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Status    int16  `json:"status"`
	Phone     string `json:"phone,omitempty"`
	AvatarURL string `json:"avatar_url,omitempty"`
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
		UserID:   userID,
		UserName: username,
		Username: username,
		Email:    email,
	}
}

// AuthUserFromOutput 用登录结果组装用户对象。id 等于 user_id。
func AuthUserFromOutput(userID, username, email, phone, avatarURL string, status int16) AuthUserItem {
	item := FromSupabaseAuthUser(userID, username, email)
	item.Status = status
	item.Phone = phone
	item.AvatarURL = avatarURL
	return item
}
