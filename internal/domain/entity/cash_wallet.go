package entity

import "time"

const (
	WysCashWalletTable     = "wys_cash_wallet"
	WysCashLedgerTable     = "wys_cash_ledger"
	WysBankCardTable       = "wys_bank_card"

	CashReasonRecharge   = "recharge"
	CashReasonMallPay    = "mall_pay"
	CashReasonMallRefund = "mall_refund"

	// 充值模拟渠道
	WalletRechargeAlipay = 1
	WalletRechargeWeChat = 2
	WalletRechargeCard   = 3

	// MallPayBalance 商城人民币余额支付渠道。
	MallPayBalance int16 = 6

	WalletRechargeMinFen int64 = 1      // 0.01 元
	WalletRechargeMaxFen int64 = 5000000 // 50000.00 元
)

// WysCashWallet 账号全局人民币余额（分）。
type WysCashWallet struct {
	UserID    string    `json:"user_id" gorm:"primaryKey"`
	BalanceFen int64    `json:"balance_fen" gorm:"column:balance_fen"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (WysCashWallet) TableName() string { return WysCashWalletTable }

// WysCashLedger 人民币流水（分）。
type WysCashLedger struct {
	LedgerID   int64     `json:"ledger_id" gorm:"primaryKey"`
	UserID     string    `json:"user_id"`
	DeltaFen   int64     `json:"delta_fen" gorm:"column:delta_fen"`
	BalanceFen int64     `json:"balance_fen" gorm:"column:balance_fen"`
	Reason     string    `json:"reason"`
	RefID      string    `json:"ref_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

func (WysCashLedger) TableName() string { return WysCashLedgerTable }

// WysBankCard 本地模拟银行卡（不存完整卡号）。
type WysBankCard struct {
	CardID     int64     `json:"card_id" gorm:"primaryKey"`
	UserID     string    `json:"user_id"`
	BankName   string    `json:"bank_name"`
	CardLast4  string    `json:"card_last4"`
	HolderName string    `json:"holder_name,omitempty"`
	IsDefault  bool      `json:"is_default"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (WysBankCard) TableName() string { return WysBankCardTable }

// WalletSummary 钱包首页读模型。
type WalletSummary struct {
	Balance    string        `json:"balance"` // 元，两位小数
	BalanceFen int64         `json:"balance_fen"`
	Cards      []WysBankCard `json:"cards"`
}

// WalletLedgerPage 流水分页。
type WalletLedgerPage struct {
	Items []WysCashLedger `json:"items"`
	Total int64           `json:"total"`
}

// WalletRechargeResult 充值结果。
type WalletRechargeResult struct {
	Balance    string `json:"balance"`
	BalanceFen int64  `json:"balance_fen"`
	DeltaFen   int64  `json:"delta_fen"`
}
