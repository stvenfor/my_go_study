// profile.go profiles 表领域模型（id = 用户 UUID）。
package entity

import "time"

const ProfilesTable = "profiles"

// Profile 用户资料（与 Flutter BackendProfile 字段对齐，snake_case JSON）。
type Profile struct {
	ID          string     `json:"id" gorm:"type:uuid;primaryKey"`
	DisplayName *string    `json:"display_name"`
	AvatarURL   *string    `json:"avatar_url"`
	Phone       *string    `json:"phone"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

func (Profile) TableName() string { return ProfilesTable }

// UpdateProfileInput 更新资料请求。
type UpdateProfileInput struct {
	DisplayName *string `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}
