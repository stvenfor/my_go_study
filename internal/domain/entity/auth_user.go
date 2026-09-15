// auth_user.go 本地 Auth 用户表（对齐 Supabase auth.users 的 UUID 身份）。
package entity

import "time"

const AuthUsersTable = "auth_users"

// AuthUser 本机认证用户。
type AuthUser struct {
	ID           string    `gorm:"type:uuid;primaryKey" json:"id"`
	Email        string    `gorm:"size:255;uniqueIndex;not null" json:"email"`
	Phone        string    `gorm:"size:32;index" json:"phone,omitempty"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	DisplayName  string    `gorm:"size:128" json:"display_name,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (AuthUser) TableName() string { return AuthUsersTable }

// AuthRefreshToken 本地 opaque refresh token。
type AuthRefreshToken struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    string    `gorm:"type:uuid;index;not null" json:"user_id"`
	TokenHash string    `gorm:"size:64;uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time `gorm:"index;not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

func (AuthRefreshToken) TableName() string { return "auth_refresh_tokens" }
