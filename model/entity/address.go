// address.go 用户收货地址簿。
package entity

import "time"

const WysUserAddressTable = "wys_user_address"

// WysUserAddress 可复用收货地址。订单只快照字段，不存 address_id。
type WysUserAddress struct {
	AddressID     int64      `json:"address_id" gorm:"primaryKey"`
	UserID        string     `json:"user_id" gorm:"size:64"`
	ReceiverName  string     `json:"receiver_name"`
	ReceiverPhone string     `json:"receiver_phone" gorm:"size:20"`
	Province      string     `json:"province"`
	City          string     `json:"city"`
	District      string     `json:"district"`
	DetailAddress string     `json:"detail_address"`
	PostalCode    string     `json:"postal_code" gorm:"size:16"`
	IsDefault     bool       `json:"is_default"`
	Label         string     `json:"label"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}

func (WysUserAddress) TableName() string { return WysUserAddressTable }

// FullAddress 省市区+详细地址，供展示与订单 receiver_address。
func (a WysUserAddress) FullAddress() string {
	return a.Province + a.City + a.District + a.DetailAddress
}

// AddressView API 出参（含 full_address）。
type AddressView struct {
	AddressID     int64  `json:"address_id"`
	ReceiverName  string `json:"receiver_name"`
	ReceiverPhone string `json:"receiver_phone"`
	Province      string `json:"province"`
	City          string `json:"city"`
	District      string `json:"district"`
	DetailAddress string `json:"detail_address"`
	PostalCode    string `json:"postal_code"`
	IsDefault     bool   `json:"is_default"`
	Label         string `json:"label"`
	FullAddress   string `json:"full_address"`
}

// ToAddressView 转出参。
func (a WysUserAddress) ToAddressView() AddressView {
	return AddressView{
		AddressID:     a.AddressID,
		ReceiverName:  a.ReceiverName,
		ReceiverPhone: a.ReceiverPhone,
		Province:      a.Province,
		City:          a.City,
		District:      a.District,
		DetailAddress: a.DetailAddress,
		PostalCode:    a.PostalCode,
		IsDefault:     a.IsDefault,
		Label:         a.Label,
		FullAddress:   a.FullAddress(),
	}
}
