// user.go 本地统一用户表。user_id 是唯一主键。
package entity

import "time"

const UsersTable = "users"

// 账号状态。停用只能由库内维护，本服务不提供把它写成 1 的接口。
const (
	UserStatusActive   int16 = 0
	UserStatusDisabled int16 = 1
	UserStatusLocked   int16 = 2
)

// User 本地账号。凭证、资料和账号状态在同一行。
type User struct {
	UserID            string     `gorm:"column:user_id;primaryKey;size:64" json:"user_id"`
	UserName          string     `gorm:"column:user_name;size:128;not null" json:"user_name"`
	Email             string     `gorm:"size:255;not null" json:"email"`
	Phone             string     `gorm:"size:32" json:"phone,omitempty"`
	PasswordHash      string     `gorm:"size:255;not null" json:"-"`
	Status            int16      `gorm:"not null;default:0" json:"status"`
	EmailVerifiedAt   *time.Time `json:"email_verified_at,omitempty"`
	PhoneVerifiedAt   *time.Time `json:"phone_verified_at,omitempty"`
	FailedLoginCount  int        `gorm:"not null;default:0" json:"-"`
	LockedUntil       *time.Time `json:"-"`
	PasswordChangedAt *time.Time `json:"password_changed_at,omitempty"`
	LastLoginAt       *time.Time `json:"last_login_at,omitempty"`
	AvatarURL         string     `gorm:"type:text" json:"avatar_url,omitempty"`
	CurrentStoreID    *int       `gorm:"column:current_store_id" json:"current_store_id,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	DeletedAt         *time.Time `json:"-"`
}

func (User) TableName() string { return UsersTable }
