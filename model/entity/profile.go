// profile.go 资料读模型。本地落在 users 行上，不是独立表。
// ProfilesTable 仅给 Supabase Cloud 的 profiles 查询使用。
package entity

import "time"

const ProfilesTable = "profiles"

// Profile 从 users 行投影出的资料。本地不建 profiles 表。
type Profile struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	UserName       string     `json:"user_name"`
	Email          string     `json:"email"`
	Status         int16      `json:"status"`
	DisplayName    *string    `json:"display_name"`
	AvatarURL      *string    `json:"avatar_url"`
	Phone          *string    `json:"phone"`
	CurrentStoreID *int       `json:"current_store_id,omitempty"`
	CreatedAt      *time.Time `json:"created_at,omitempty"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}

// UpdateProfileInput 更新资料请求。只允许改 user_name / 头像 / phone。
// status、deleted_at、email、user_id、password 即使传入也不写入。
type UpdateProfileInput struct {
	UserName     *string `json:"user_name"`
	Username     *string `json:"username"`
	DisplayName  *string `json:"display_name"`
	AvatarURL    *string `json:"avatar_url"`
	AvatarBase64 *string `json:"avatar_base64"`
	AvatarMime   *string `json:"avatar_mime"`
	Phone        *string `json:"phone"`
	Status       *int16  `json:"status"`
}

// ResolvedUserName user_name 优先，其次 display_name、username。
func (input UpdateProfileInput) ResolvedUserName() *string {
	switch {
	case input.UserName != nil:
		return input.UserName
	case input.DisplayName != nil:
		return input.DisplayName
	case input.Username != nil:
		return input.Username
	default:
		return nil
	}
}
