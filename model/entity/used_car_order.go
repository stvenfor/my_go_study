// used_car_order.go 二手车业务单。
package entity

import (
	"strconv"
	"time"
)

const WysUsedCarOrderTable = "wys_used_car_order"

// 二手车业务单审核态（对齐成交发票）。
const (
	UsedCarOrderPendingReview         int16 = 0
	UsedCarOrderApprovedPendingRating int16 = 1
	UsedCarOrderRated                 int16 = 2
	UsedCarOrderRejected              int16 = 3
)

// 二手车业务类型。
const (
	UsedCarKindTradeIn  int16 = 0 // 置换
	UsedCarKindConsign  int16 = 1 // 专卖
	UsedCarKindPurchase int16 = 2 // 收车
)

const (
	TodoTypeUsedCarPendingReview = "used_car_pending_review"
)

func UsedCarOrderStatusCode(status int16) string {
	switch status {
	case UsedCarOrderPendingReview:
		return "pending_review"
	case UsedCarOrderApprovedPendingRating:
		return "approved_pending_rating"
	case UsedCarOrderRated:
		return "rated"
	case UsedCarOrderRejected:
		return "rejected"
	default:
		return "pending_review"
	}
}

func ParseUsedCarOrderStatusFilter(raw string) []int16 {
	switch raw {
	case "", "all":
		return nil
	case "pending_review":
		return []int16{UsedCarOrderPendingReview}
	case "approved":
		return []int16{UsedCarOrderApprovedPendingRating, UsedCarOrderRated}
	case "rejected":
		return []int16{UsedCarOrderRejected}
	default:
		return nil
	}
}

func UsedCarKindCode(kind int16) string {
	switch kind {
	case UsedCarKindTradeIn:
		return "trade_in"
	case UsedCarKindConsign:
		return "consign"
	case UsedCarKindPurchase:
		return "purchase"
	default:
		return "trade_in"
	}
}

func UsedCarKindLabel(kind int16) string {
	switch kind {
	case UsedCarKindTradeIn:
		return "置换"
	case UsedCarKindConsign:
		return "专卖"
	case UsedCarKindPurchase:
		return "收车"
	default:
		return "置换"
	}
}

func UsedCarAmountLabel(kind int16) string {
	switch kind {
	case UsedCarKindTradeIn:
		return "补差价"
	case UsedCarKindConsign:
		return "成交价"
	case UsedCarKindPurchase:
		return "收车价"
	default:
		return "业务金额"
	}
}

// ParseUsedCarKind 接受 API 字符串或数字。
func ParseUsedCarKind(raw string) (int16, bool) {
	switch raw {
	case "trade_in", "0":
		return UsedCarKindTradeIn, true
	case "consign", "1":
		return UsedCarKindConsign, true
	case "purchase", "2":
		return UsedCarKindPurchase, true
	default:
		return 0, false
	}
}

func ParseUsedCarKindFilter(raw string) (kind int16, filter bool, ok bool) {
	if raw == "" || raw == "all" {
		return 0, false, true
	}
	k, ok := ParseUsedCarKind(raw)
	if !ok {
		return 0, false, false
	}
	return k, true, true
}

// WysUsedCarOrder 二手车业务单行。
type WysUsedCarOrder struct {
	OrderID         int64      `json:"order_id" gorm:"primaryKey"`
	StoreID         int        `json:"store_id"`
	UploaderUserID  string     `json:"uploader_user_id" gorm:"size:64"`
	Kind            int16      `json:"kind"`
	CustomerID      int64      `json:"customer_id"`
	CustomerPhone   string     `json:"customer_phone" gorm:"size:32"`
	CustomerName    string     `json:"customer_name" gorm:"size:128"`
	VehicleModel    string     `json:"vehicle_model" gorm:"size:128"`
	PlateNo         string     `json:"plate_no" gorm:"size:32"`
	VIN             string     `json:"vin" gorm:"column:vin;size:32"`
	MileageKm       int        `json:"mileage_km"`
	ModelYear       int        `json:"model_year"`
	Amount          float64    `json:"amount"`
	Status          int16      `json:"status"`
	ImageURL        *string    `json:"image_url,omitempty"`
	RejectReason    *string    `json:"reject_reason,omitempty"`
	RatingStars     *int16     `json:"rating_stars,omitempty"`
	SubmittedAt     time.Time  `json:"submitted_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (WysUsedCarOrder) TableName() string { return WysUsedCarOrderTable }

// UsedCarOrderStats 本人×当前店四格。
type UsedCarOrderStats struct {
	Submitted     int64 `json:"submitted"`
	PendingReview int64 `json:"pending_review"`
	Approved      int64 `json:"approved"`
	Rejected      int64 `json:"rejected"`
}

// UsedCarOrderSummary 列表顶栏。
type UsedCarOrderSummary struct {
	DisplayName   string             `json:"display_name"`
	AvatarURL     string             `json:"avatar_url"`
	PositionLabel string             `json:"position_label"`
	StoreName     string             `json:"store_name"`
	Stats         UsedCarOrderStats  `json:"stats"`
}

// UsedCarOrderDTO 列表/详情读模型。
type UsedCarOrderDTO struct {
	OrderID      string    `json:"order_id"`
	Kind         string    `json:"kind"`
	KindLabel    string    `json:"kind_label"`
	AmountLabel  string    `json:"amount_label"`
	CustomerName string    `json:"customer_name"`
	Phone        string    `json:"phone"`
	VehicleModel string    `json:"vehicle_model"`
	PlateNo      string    `json:"plate_no"`
	VIN          string    `json:"vin"`
	MileageKm    int       `json:"mileage_km"`
	ModelYear    int       `json:"model_year"`
	Amount       float64   `json:"amount"`
	Status       string    `json:"status"`
	SubmittedAt  time.Time `json:"submitted_at"`
	RejectReason *string   `json:"reject_reason"`
	RatingStars  *int16    `json:"rating_stars"`
	ImageURL     *string   `json:"image_url"`
}

func ToUsedCarOrderDTO(row WysUsedCarOrder) UsedCarOrderDTO {
	return UsedCarOrderDTO{
		OrderID:      strconv.FormatInt(row.OrderID, 10),
		Kind:         UsedCarKindCode(row.Kind),
		KindLabel:    UsedCarKindLabel(row.Kind),
		AmountLabel:  UsedCarAmountLabel(row.Kind),
		CustomerName: row.CustomerName,
		Phone:        row.CustomerPhone,
		VehicleModel: row.VehicleModel,
		PlateNo:      row.PlateNo,
		VIN:          row.VIN,
		MileageKm:    row.MileageKm,
		ModelYear:    row.ModelYear,
		Amount:       row.Amount,
		Status:       UsedCarOrderStatusCode(row.Status),
		SubmittedAt:  row.SubmittedAt,
		RejectReason: row.RejectReason,
		RatingStars:  row.RatingStars,
		ImageURL:     row.ImageURL,
	}
}
