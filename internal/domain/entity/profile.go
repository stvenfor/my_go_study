// profile.go Supabase profiles 表领域模型。
package entity

import "time"

const ProfilesTable = "profiles"

// Profile 用户资料（与 Flutter BackendProfile 字段对齐，snake_case JSON）。
type Profile struct {
	ID          string     `json:"id"`
	DisplayName *string    `json:"display_name"`
	AvatarURL   *string    `json:"avatar_url"`
	Phone       *string    `json:"phone"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

// UpdateProfileInput 更新资料请求。
type UpdateProfileInput struct {
	DisplayName *string `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}
