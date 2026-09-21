// store.go 门店统计。职务在成员上，不在统计行上。
package entity

import "time"

const WysUserStoreStatsTable = "wys_user_store_stats"

// 门店职务（存库为数字）。不是权限角色。
const (
	StoreRoleAdvisor int16 = 0 // 销售顾问
	StoreRoleManager int16 = 1 // 销售经理
	StoreRoleGeneral int16 = 2 // 总经理
)

// StoreRoleLabel 职务数字 → 展示文案。未知数字不冒充销售顾问。
func StoreRoleLabel(role int16) string {
	switch role {
	case StoreRoleAdvisor:
		return "销售顾问"
	case StoreRoleManager:
		return "销售经理"
	case StoreRoleGeneral:
		return "总经理"
	default:
		return ""
	}
}

// WysUserStoreStats 某人在某店的展示数字。职务不在这行里读。
type WysUserStoreStats struct {
	UserID         string    `json:"user_id" gorm:"primaryKey;size:64"`
	StoreID        int       `json:"store_id" gorm:"primaryKey"`
	StoreName      string    `json:"store_name" gorm:"not null;default:''"`
	Role           int16     `json:"role" gorm:"type:smallint;not null;default:0"`
	DaysJoined     int       `json:"days_joined" gorm:"not null;default:0"`
	EmployeeCount  int       `json:"employee_count" gorm:"not null;default:0"`
	StoreDays      int       `json:"store_days" gorm:"not null;default:0"`
	TotalCustomers int       `json:"total_customers" gorm:"not null;default:0"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (WysUserStoreStats) TableName() string { return WysUserStoreStatsTable }

// UserStoreStats Mine / Profile API 嵌套 stats（snake_case）。
type UserStoreStats struct {
	StoreID        int    `json:"store_id"`
	StoreName      string `json:"store_name"`
	DaysJoined     int    `json:"days_joined"`
	EmployeeCount  int    `json:"employee_count"`
	StoreDays      int    `json:"store_days"`
	TotalCustomers int    `json:"total_customers"`
	Role           *int16 `json:"role,omitempty"`
	RoleLabel      string `json:"role_label,omitempty"`
}

// ZeroUserStoreStats 没有统计、也没有职务。0 不是销售顾问。
func ZeroUserStoreStats() UserStoreStats {
	return UserStoreStats{}
}

// ToAPIStats 行 → 展示数字。不含职务。
func (r *WysUserStoreStats) ToAPIStats() UserStoreStats {
	if r == nil {
		return ZeroUserStoreStats()
	}
	return UserStoreStats{
		StoreID:        r.StoreID,
		StoreName:      r.StoreName,
		DaysJoined:     r.DaysJoined,
		EmployeeCount:  r.EmployeeCount,
		StoreDays:      r.StoreDays,
		TotalCustomers: r.TotalCustomers,
	}
}

// WithPosition 把成员职务贴上。position 为空表示不是该店成员。
func (s UserStoreStats) WithPosition(position *int16) UserStoreStats {
	if position == nil {
		s.Role = nil
		s.RoleLabel = ""
		return s
	}
	p := *position
	s.Role = &p
	s.RoleLabel = StoreRoleLabel(p)
	return s
}
