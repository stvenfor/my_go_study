// new_car_follow.go 新车跟进档案。
package entity

import (
	"strconv"
	"strings"
	"time"
)

const WysNewCarFollowFileTable = "wys_new_car_follow_file"

// 档案阶段。
const (
	FollowStageNew       int16 = 0
	FollowStageFollowing int16 = 1
	FollowStageTestDrive int16 = 2
	FollowStageQuoted    int16 = 3
	FollowStageWon       int16 = 4
	FollowStageLost      int16 = 5
)

// 跟进级别（仅大写单字符）。
const (
	FollowLevelH = "H"
	FollowLevelA = "A"
	FollowLevelB = "B"
	FollowLevelE = "E"
)

// 购车意向档（派生展示，非库列）。
const (
	IntentBandHigh   = "高"
	IntentBandMedium = "中"
	IntentBandLow    = "低"
)

// NormalizeFollowLevel 校验并规范化跟进级别。
func NormalizeFollowLevel(raw string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case FollowLevelH, FollowLevelA, FollowLevelB, FollowLevelE:
		return strings.ToUpper(strings.TrimSpace(raw)), true
	default:
		return "", false
	}
}

// IntentBandFromFollowLevel H/A→高，B→中，E→低。
func IntentBandFromFollowLevel(level string) string {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case FollowLevelH, FollowLevelA:
		return IntentBandHigh
	case FollowLevelB:
		return IntentBandMedium
	case FollowLevelE:
		return IntentBandLow
	default:
		return ""
	}
}

// FollowLevelsForIntentBand 意向档 → 级别集合。
func FollowLevelsForIntentBand(band string) []string {
	switch strings.TrimSpace(band) {
	case IntentBandHigh, "high":
		return []string{FollowLevelH, FollowLevelA}
	case IntentBandMedium, "medium":
		return []string{FollowLevelB}
	case IntentBandLow, "low":
		return []string{FollowLevelE}
	default:
		return nil
	}
}

// FollowFileStageCode 数字 → API 字符串。
func FollowFileStageCode(stage int16) string {
	switch stage {
	case FollowStageNew:
		return "new"
	case FollowStageFollowing:
		return "following"
	case FollowStageTestDrive:
		return "test_drive"
	case FollowStageQuoted:
		return "quoted"
	case FollowStageWon:
		return "won"
	case FollowStageLost:
		return "lost"
	default:
		return "new"
	}
}

// ParseFollowFileStage 写路径阶段；空则 ok=false 表示未传。
func ParseFollowFileStage(raw string) (int16, bool) {
	switch strings.TrimSpace(raw) {
	case "new":
		return FollowStageNew, true
	case "following":
		return FollowStageFollowing, true
	case "test_drive":
		return FollowStageTestDrive, true
	case "quoted":
		return FollowStageQuoted, true
	case "won":
		return FollowStageWon, true
	case "lost":
		return FollowStageLost, true
	default:
		return 0, false
	}
}

// FollowFileIsOpen 未成交/未战败。
func FollowFileIsOpen(stage int16) bool {
	return stage >= FollowStageNew && stage <= FollowStageQuoted
}

// WysNewCarFollowFile 跟进档案行。
type WysNewCarFollowFile struct {
	FileID           int64      `json:"file_id" gorm:"primaryKey"`
	StoreID          int        `json:"store_id"`
	OwnerUserID      string     `json:"owner_user_id" gorm:"size:64"`
	CustomerID       int64      `json:"customer_id"`
	FollowLevel      string     `json:"follow_level" gorm:"size:1"`
	Stage            int16      `json:"stage"`
	VehicleInterest  string     `json:"vehicle_interest" gorm:"size:256"`
	BudgetNote       string     `json:"budget_note" gorm:"size:256"`
	Source           string     `json:"source" gorm:"size:64"`
	NextFollowUpAt   *time.Time `json:"next_follow_up_at,omitempty"`
	LastFollowAt     *time.Time `json:"last_follow_at,omitempty"`
	ClosedReason     string     `json:"closed_reason" gorm:"size:128"`
	CustomerName     string     `json:"customer_name" gorm:"size:128"`
	CustomerPhone    string     `json:"customer_phone" gorm:"size:32"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (WysNewCarFollowFile) TableName() string { return WysNewCarFollowFileTable }

// NewCarFollowStats 本人×当前店四格。
type NewCarFollowStats struct {
	Active     int64 `json:"active"`      // 跟进中（未关闭）
	Overdue    int64 `json:"overdue"`     // 逾期且未关闭
	HighIntent int64 `json:"high_intent"` // 高意向（H∪A）且未关闭
	Lost       int64 `json:"lost"`        // 战败
}

// NewCarFollowSummary 列表顶栏。
type NewCarFollowSummary struct {
	DisplayName   string            `json:"display_name"`
	AvatarURL     string            `json:"avatar_url"`
	PositionLabel string            `json:"position_label"`
	StoreName     string            `json:"store_name"`
	Stats         NewCarFollowStats `json:"stats"`
}

// NewCarFollowFileDTO 列表/详情读模型。
type NewCarFollowFileDTO struct {
	FileID          string     `json:"file_id"`
	CustomerID      string     `json:"customer_id"`
	CustomerName    string     `json:"customer_name"`
	CustomerPhone   string     `json:"customer_phone"`
	FollowLevel     string     `json:"follow_level"`
	IntentBand      string     `json:"intent_band"`
	Stage           string     `json:"stage"`
	VehicleInterest string     `json:"vehicle_interest"`
	BudgetNote      string     `json:"budget_note"`
	Source          string     `json:"source"`
	NextFollowUpAt  *time.Time `json:"next_follow_up_at,omitempty"`
	LastFollowAt    *time.Time `json:"last_follow_at,omitempty"`
	ClosedReason    string     `json:"closed_reason,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ToNewCarFollowFileDTO 行 → DTO。
func ToNewCarFollowFileDTO(row WysNewCarFollowFile) NewCarFollowFileDTO {
	return NewCarFollowFileDTO{
		FileID:          strconv.FormatInt(row.FileID, 10),
		CustomerID:      strconv.FormatInt(row.CustomerID, 10),
		CustomerName:    row.CustomerName,
		CustomerPhone:   row.CustomerPhone,
		FollowLevel:     row.FollowLevel,
		IntentBand:      IntentBandFromFollowLevel(row.FollowLevel),
		Stage:           FollowFileStageCode(row.Stage),
		VehicleInterest: row.VehicleInterest,
		BudgetNote:      row.BudgetNote,
		Source:          row.Source,
		NextFollowUpAt:  row.NextFollowUpAt,
		LastFollowAt:    row.LastFollowAt,
		ClosedReason:    row.ClosedReason,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

// NewCarFollowListFilter 列表筛选。
type NewCarFollowListFilter struct {
	FollowLevel string
	IntentBand  string
	OverdueOnly bool
	OpenOnly    bool
	Stage       *int16
}
