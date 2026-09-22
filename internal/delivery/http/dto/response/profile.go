// profile.go Supabase Profile 统一响应 DTO（camelCase；stats 为 snake_case 键）。
package response

import (
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

// ProfileStatsItem Mine 统计（稳定 snake_case；role 为数字枚举）。
type ProfileStatsItem struct {
	StoreID        int    `json:"store_id"`
	StoreName      string `json:"store_name"`
	DaysJoined     int    `json:"days_joined"`
	EmployeeCount  int    `json:"employee_count"`
	StoreDays      int    `json:"store_days"`
	TotalCustomers int    `json:"total_customers"`
	Role           *int16 `json:"role"`
	RoleLabel      string `json:"role_label"`
}

// FromUserStoreStats 转换统计。
func FromUserStoreStats(s entity.UserStoreStats) ProfileStatsItem {
	label := s.RoleLabel
	if s.Role != nil && label == "" {
		label = entity.StoreRoleLabel(*s.Role)
	}
	return ProfileStatsItem{
		StoreID:        s.StoreID,
		StoreName:      s.StoreName,
		DaysJoined:     s.DaysJoined,
		EmployeeCount:  s.EmployeeCount,
		StoreDays:      s.StoreDays,
		TotalCustomers: s.TotalCustomers,
		Role:           s.Role,
		RoleLabel:      label,
	}
}

// UserStoreListBody GET /me/stores 响应 data。
type UserStoreListBody struct {
	List           []UserStoreListItem `json:"list"`
	CurrentStoreID int                 `json:"current_store_id"`
}

// UserStoreListItem 可切换经销商一项。
type UserStoreListItem struct {
	StoreID   int    `json:"store_id"`
	StoreName string `json:"store_name"`
	Role      *int16 `json:"role"`
	RoleLabel string `json:"role_label"`
	IsCurrent bool   `json:"is_current"`
}

// FromUserStoreList 组装经销商列表。
func FromUserStoreList(items []entity.UserStoreListItem, currentStoreID int) UserStoreListBody {
	out := make([]UserStoreListItem, 0, len(items))
	for _, it := range items {
		label := it.RoleLabel
		if it.Role != nil && label == "" {
			label = entity.StoreRoleLabel(*it.Role)
		}
		out = append(out, UserStoreListItem{
			StoreID:   it.StoreID,
			StoreName: it.StoreName,
			Role:      it.Role,
			RoleLabel: label,
			IsCurrent: it.IsCurrent,
		})
	}
	return UserStoreListBody{List: out, CurrentStoreID: currentStoreID}
}

// ProfileItem 用户资料响应项。id 等于 user_id。
type ProfileItem struct {
	ID          string           `json:"id"`
	UserID      string           `json:"user_id"`
	UserName    string           `json:"user_name"`
	Email       string           `json:"email"`
	Status      int16            `json:"status"`
	DisplayName *string          `json:"displayName,omitempty"`
	AvatarURL   *string          `json:"avatarUrl,omitempty"`
	Phone       *string          `json:"phone,omitempty"`
	CreatedAt   *time.Time       `json:"createdAt,omitempty"`
	UpdatedAt   *time.Time       `json:"updatedAt,omitempty"`
	Stats       ProfileStatsItem `json:"stats"`
}

// FromProfile 从领域实体转换为响应 DTO。
func FromProfile(p *entity.Profile, stats entity.UserStoreStats) ProfileItem {
	if p == nil {
		return ProfileItem{Stats: FromUserStoreStats(stats)}
	}
	return ProfileItem{
		ID:          accountID(p),
		UserID:      accountID(p),
		UserName:    accountName(p),
		Email:       p.Email,
		Status:      p.Status,
		DisplayName: displayPtr(accountName(p)),
		AvatarURL:   p.AvatarURL,
		Phone:       p.Phone,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		Stats:       FromUserStoreStats(stats),
	}
}

// ProfileLegacyBody Flutter `/me/profile` snake_case 直出（含 stats）。
type ProfileLegacyBody struct {
	ID          string           `json:"id"`
	UserID      string           `json:"user_id"`
	UserName    string           `json:"user_name"`
	Email       string           `json:"email"`
	Status      int16            `json:"status"`
	DisplayName *string          `json:"display_name"`
	AvatarURL   *string          `json:"avatar_url"`
	Phone       *string          `json:"phone"`
	CreatedAt   *time.Time       `json:"created_at,omitempty"`
	UpdatedAt   *time.Time       `json:"updated_at,omitempty"`
	Stats       ProfileStatsItem `json:"stats"`
}

// FromProfileLegacy 组装 legacy body。
func FromProfileLegacy(p *entity.Profile, stats entity.UserStoreStats) ProfileLegacyBody {
	if p == nil {
		return ProfileLegacyBody{Stats: FromUserStoreStats(stats)}
	}
	name := accountName(p)
	return ProfileLegacyBody{
		ID:          accountID(p),
		UserID:      accountID(p),
		UserName:    name,
		Email:       p.Email,
		Status:      p.Status,
		DisplayName: displayPtr(name),
		AvatarURL:   p.AvatarURL,
		Phone:       p.Phone,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
		Stats:       FromUserStoreStats(stats),
	}
}

func accountID(p *entity.Profile) string {
	if p.UserID != "" {
		return p.UserID
	}
	return p.ID
}

func accountName(p *entity.Profile) string {
	if p.UserName != "" {
		return p.UserName
	}
	if p.DisplayName != nil {
		return *p.DisplayName
	}
	return ""
}

func displayPtr(name string) *string {
	if name == "" {
		return nil
	}
	return &name
}
