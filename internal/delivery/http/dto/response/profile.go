// profile.go Supabase Profile 统一响应 DTO（camelCase）。
package response

import (
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

// ProfileItem 用户资料响应项。
type ProfileItem struct {
	ID          string     `json:"id"`
	DisplayName *string    `json:"displayName,omitempty"`
	AvatarURL   *string    `json:"avatarUrl,omitempty"`
	Phone       *string    `json:"phone,omitempty"`
	CreatedAt   *time.Time `json:"createdAt,omitempty"`
	UpdatedAt   *time.Time `json:"updatedAt,omitempty"`
}

// FromProfile 从领域实体转换为响应 DTO。
func FromProfile(p *entity.Profile) ProfileItem {
	if p == nil {
		return ProfileItem{}
	}
	return ProfileItem{
		ID:          p.ID,
		DisplayName: p.DisplayName,
		AvatarURL:   p.AvatarURL,
		Phone:       p.Phone,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}
