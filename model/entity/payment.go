// payment.go 渠道支付流水与对账差异实体（P0-2）。
//
// 与 wys_mall_payment / wys_membership_order 的区别：这两张是**业务单**状态，
// 本表记的是**渠道侧**事实（prepay 生成、notify 核销、主动查单结果），
// out_trade_no 全局唯一，是回调幂等的锚点。金额一律整数分。
package entity

import (
	"time"
)

const (
	WysPaymentRecordTable        = "wys_payment_records"
	WysPaymentReconcileDiffTable = "wys_payment_reconcile_diff"
)

// 渠道取值（与 /payments/prepay 入参一致，字符串而非 int16：
// 渠道名要进 URL 路径 notify/{channel}，也要能直接跟微信/支付宝报文对照）。
const (
	PaymentChannelWeChat = "wechat"
	PaymentChannelAlipay = "alipay"
)

// 流水状态：单向流转 pending → paid / failed，不允许回退。
const (
	PaymentStatusPending int16 = 0
	PaymentStatusPaid    int16 = 1
	PaymentStatusFailed  int16 = 2
)

// 业务单引用类型：notify 核销后据此回调对应域做「发货/权益发放」。
const (
	PaymentRefMallOrder       = "mall_order"
	PaymentRefMembershipOrder = "membership_order"
)

// 对账差异类型。
const (
	PaymentDiffLocalMissing   = "local_missing"   // 渠道有、本地无流水
	PaymentDiffChannelMissing = "channel_missing" // 本地 paid、渠道查无
	PaymentDiffAmountMismatch = "amount_mismatch" // 金额不符
	PaymentDiffStatusMismatch = "status_mismatch" // 状态不符
)

// WysPaymentRecord 渠道支付流水。
type WysPaymentRecord struct {
	ID int64 `json:"id" gorm:"primaryKey"`
	// 显式命名唯一索引：与 migrations/20261104160000_payment_ledger 同名，
	// 否则 lab 的 AutoMigrate 会去找 uni_* 名字并在已有迁移表上直接报错。
	OutTradeNo     string  `json:"out_trade_no" gorm:"size:64;uniqueIndex:uk_wys_payment_records_out_trade_no"`
	UserID         string  `json:"user_id" gorm:"size:64"`
	Channel        string  `json:"channel" gorm:"size:16"`
	AmountFen      int64   `json:"amount_fen"`
	Status         int16   `json:"status"`
	RefType        string  `json:"ref_type" gorm:"size:32"`
	RefID          string  `json:"ref_id" gorm:"size:64"`
	ChannelTradeNo *string `json:"channel_trade_no,omitempty" gorm:"size:64"`
	// NotifyRaw 存渠道回调原文：微信是 XML、支付宝是 form-urlencoded，都不是 JSON，
	// 所以列是 text 而不是 jsonb（曾因 jsonb 拒收 XML 直接核销失败）。
	NotifyRaw  string     `json:"notify_raw,omitempty" gorm:"type:text"`
	FailReason string     `json:"fail_reason,omitempty" gorm:"size:128"`
	PaidAt     *time.Time `json:"paid_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func (WysPaymentRecord) TableName() string { return WysPaymentRecordTable }

// IsPaid 流水是否已核销。
func (r *WysPaymentRecord) IsPaid() bool { return r != nil && r.Status == PaymentStatusPaid }

// WysPaymentReconcileDiff 对账差异（T+1 任务写入，供告警与人工处理）。
//
// (recon_date, out_trade_no, kind) 唯一：对账任务至少投一次，重复跑不能刷出翻倍差异。
type WysPaymentReconcileDiff struct {
	ID               int64     `json:"id" gorm:"primaryKey"`
	ReconDate        string    `json:"recon_date" gorm:"size:10;uniqueIndex:uk_wys_payment_reconcile_diff"`
	Channel          string    `json:"channel" gorm:"size:16"`
	OutTradeNo       string    `json:"out_trade_no" gorm:"size:64;uniqueIndex:uk_wys_payment_reconcile_diff"`
	Kind             string    `json:"kind" gorm:"size:32;uniqueIndex:uk_wys_payment_reconcile_diff"`
	LocalStatus      int16     `json:"local_status"`
	LocalAmountFen   int64     `json:"local_amount_fen"`
	ChannelStatus    string    `json:"channel_status" gorm:"size:32"`
	ChannelAmountFen int64     `json:"channel_amount_fen"`
	ChannelTradeNo   string    `json:"channel_trade_no" gorm:"size:64"`
	Resolved         bool      `json:"resolved"`
	ResolveNote      string    `json:"resolve_note" gorm:"size:256"`
	CreatedAt        time.Time `json:"created_at"`
}

func (WysPaymentReconcileDiff) TableName() string { return WysPaymentReconcileDiffTable }
