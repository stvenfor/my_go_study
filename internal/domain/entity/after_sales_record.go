package entity

import "time"

const (
	WysAfterSalesRecordTable = "wys_after_sales_record"

	ServiceKindRepair       int16 = 0
	ServiceKindMaintenance  int16 = 1
)

// WysAfterSalesRecord 售后专区维修保养记录。
type WysAfterSalesRecord struct {
	RecordID        int64     `json:"record_id" gorm:"primaryKey"`
	StoreID         int       `json:"store_id"`
	AppointmentID   *int64    `json:"appointment_id,omitempty"`
	CustomerID      *int64    `json:"customer_id,omitempty"`
	CustomerUserID  *string   `json:"customer_user_id,omitempty" gorm:"size:64"`
	CustomerName    string    `json:"customer_name" gorm:"size:128"`
	CustomerPhone   string    `json:"customer_phone" gorm:"size:32"`
	PlateNo         string    `json:"plate_no" gorm:"size:32"`
	Mileage         *int      `json:"mileage,omitempty"`
	ServiceKind     int16     `json:"service_kind"`
	Title           string    `json:"title" gorm:"size:256"`
	Content         string    `json:"content"`
	ServiceDate     time.Time `json:"service_date" gorm:"type:date"`
	CreatedBy       string    `json:"created_by" gorm:"size:64"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (WysAfterSalesRecord) TableName() string { return WysAfterSalesRecordTable }

// ServiceKindLabel 展示文案。
func ServiceKindLabel(kind int16) string {
	switch kind {
	case ServiceKindRepair:
		return "维修"
	case ServiceKindMaintenance:
		return "保养"
	default:
		return ""
	}
}

// ParseServiceKind 接受 0/1 或 repair/maintenance。
func ParseServiceKind(raw string) (int16, bool) {
	switch raw {
	case "0", "repair", "维修":
		return ServiceKindRepair, true
	case "1", "maintenance", "保养":
		return ServiceKindMaintenance, true
	default:
		return 0, false
	}
}
