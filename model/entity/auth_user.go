// auth_user.go 本地 refresh token。user_id 指向 users.user_id。
package entity

import "time"

const AuthRefreshTokensTable = "auth_refresh_tokens"

// AuthRefreshToken 本地 opaque refresh token。
type AuthRefreshToken struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    string    `gorm:"size:64;index;not null;column:user_id" json:"-"`
	TokenHash string    `gorm:"size:64;uniqueIndex;not null" json:"-"`
	ExpiresAt time.Time `gorm:"index;not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

func (AuthRefreshToken) TableName() string { return AuthRefreshTokensTable }
