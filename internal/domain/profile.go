package domain

import "time"

const ProfilesTable = "profiles"

type Profile struct {
	ID          string     `json:"id"`
	DisplayName *string    `json:"display_name"`
	AvatarURL   *string    `json:"avatar_url"`
	Phone       *string    `json:"phone"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

type UpdateProfileInput struct {
	DisplayName *string `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}
