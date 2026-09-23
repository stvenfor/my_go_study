package entity

import "time"

const WysPushDeviceTable = "wys_push_devices"

// WysPushDevice 客户端极光 RegistrationID 登记（按 user+device 唯一）。
type WysPushDevice struct {
	ID             int64     `json:"id" gorm:"primaryKey"`
	UserID         string    `json:"user_id" gorm:"size:64;index"`
	DeviceID       string    `json:"device_id" gorm:"size:128"`
	Platform       string    `json:"platform" gorm:"size:32"` // ios|android|harmony|ohos|unknown
	RegistrationID string    `json:"registration_id" gorm:"size:128;index"`
	Alias          string    `json:"alias" gorm:"size:128;index"`
	Mock           bool      `json:"mock"`
	UpdatedAt      time.Time `json:"updated_at"`
	CreatedAt      time.Time `json:"created_at"`
}

func (WysPushDevice) TableName() string { return WysPushDeviceTable }
