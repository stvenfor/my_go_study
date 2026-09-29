// deal_invoice.go 新车成交发票。
package entity

import (
	"strconv"
	"time"
)

const WysDealInvoiceTable = "wys_deal_invoice"

// 成交发票审核态。
const (
	DealInvoicePendingReview          int16 = 0
	DealInvoiceApprovedPendingRating  int16 = 1
	DealInvoiceRated                  int16 = 2
	DealInvoiceRejected               int16 = 3
)

// DealInvoiceStatusCode 数字 → API 字符串。
func DealInvoiceStatusCode(status int16) string {
	switch status {
	case DealInvoicePendingReview:
		return "pending_review"
	case DealInvoiceApprovedPendingRating:
		return "approved_pending_rating"
	case DealInvoiceRated:
		return "rated"
	case DealInvoiceRejected:
		return "rejected"
	default:
		return "pending_review"
	}
}

// ParseDealInvoiceStatusFilter Tab 查询参数 → 状态集合；all 返回 nil。
func ParseDealInvoiceStatusFilter(raw string) []int16 {
	switch raw {
	case "", "all":
		return nil
	case "pending_review":
		return []int16{DealInvoicePendingReview}
	case "approved":
		return []int16{DealInvoiceApprovedPendingRating, DealInvoiceRated}
	case "rejected":
		return []int16{DealInvoiceRejected}
	default:
		return nil
	}
}

// WysDealInvoice 成交发票行。
type WysDealInvoice struct {
	InvoiceID       int64      `json:"invoice_id" gorm:"primaryKey"`
	StoreID         int        `json:"store_id"`
	UploaderUserID  string     `json:"uploader_user_id" gorm:"size:64"`
	CustomerID      int64      `json:"customer_id"`
	CustomerPhone   string     `json:"customer_phone" gorm:"size:32"`
	CustomerName    string     `json:"customer_name" gorm:"size:128"`
	Status          int16      `json:"status"`
	ImageURL        *string    `json:"image_url,omitempty"`
	RejectReason    *string    `json:"reject_reason,omitempty"`
	RatingStars     *int16     `json:"rating_stars,omitempty"`
	SubmittedAt     time.Time  `json:"submitted_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (WysDealInvoice) TableName() string { return WysDealInvoiceTable }

// DealInvoiceStats 本人×当前店四格统计。
type DealInvoiceStats struct {
	Uploaded       int64 `json:"uploaded"`
	PendingReview  int64 `json:"pending_review"`
	Approved       int64 `json:"approved"`
	Rejected       int64 `json:"rejected"`
}

// DealInvoiceSummary 列表顶栏。
type DealInvoiceSummary struct {
	DisplayName   string           `json:"display_name"`
	AvatarURL     string           `json:"avatar_url"`
	PositionLabel string           `json:"position_label"`
	StoreName     string           `json:"store_name"`
	Stats         DealInvoiceStats `json:"stats"`
}

// DealInvoiceDTO 列表/详情读模型。
type DealInvoiceDTO struct {
	InvoiceID    string    `json:"invoice_id"`
	Phone        string    `json:"phone"`
	CustomerName string    `json:"customer_name"`
	Status       string    `json:"status"`
	SubmittedAt  time.Time `json:"submitted_at"`
	RejectReason *string   `json:"reject_reason"`
	RatingStars  *int16    `json:"rating_stars"`
	ImageURL     *string   `json:"image_url"`
}

// ToDealInvoiceDTO 行 → DTO。
func ToDealInvoiceDTO(row WysDealInvoice) DealInvoiceDTO {
	return DealInvoiceDTO{
		InvoiceID:    strconv.FormatInt(row.InvoiceID, 10),
		Phone:        row.CustomerPhone,
		CustomerName: row.CustomerName,
		Status:       DealInvoiceStatusCode(row.Status),
		SubmittedAt:  row.SubmittedAt,
		RejectReason: row.RejectReason,
		RatingStars:  row.RatingStars,
		ImageURL:     row.ImageURL,
	}
}

