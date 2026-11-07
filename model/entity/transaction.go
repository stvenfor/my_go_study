// =============================================================================
// 文件：transaction.go
// 层级：Domain —— 本地 transactions.user_id 为 users.user_id 字符串。
// =============================================================================
package entity

import (
	"strconv"
	"time"
)

const TransactionsTable = "transactions"

type Transaction struct {
	ID       int64   `json:"id" gorm:"primaryKey"`
	UserID   *string `json:"user_id,omitempty" gorm:"size:64;index;column:user_id"`
	Type     string  `json:"type" gorm:"size:32;not null"`
	Category string  `json:"category" gorm:"size:64;not null"`
	// Amount 对外口径：元。存储口径是 AmountFen（整数分）。
	// 读路径以 AmountFen 为准覆盖本字段（见 repo_postgres.applyAmountFromFen）；
	// 本字段仍映射旧列 amount，用于双写，以及「库尚未回填」时的回落读。
	Amount float64 `json:"amount" gorm:"not null"`
	// AmountFen 存储口径：整数分，不出 JSON（对外契约不变）。
	// nil 表示该库尚未跑迁移/回填 → 读路径保留上面 amount 列的值，不会把 NULL 读成 0 元。
	AmountFen *int64     `json:"-" gorm:"column:amount_fen"`
	Date      string     `json:"date" gorm:"size:32;not null"` // YYYY-MM-DD
	Note      *string    `json:"note,omitempty" gorm:"type:text"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

func (Transaction) TableName() string { return TransactionsTable }

// AmountFenFromYuan 把客户端传来的元（float64）换成整数分；**超过两位小数一律报错，不静默舍入**。
//
// 为什么先转十进制字符串，而不是 big.Rat.SetFloat64：0.1 的二进制真值是
// 0.1000000000000000055511151231257827，按真值判精度会把每一个合法金额都判成脏值。
// FormatFloat 的 -1 精度取「能还原该 float 的最短十进制串」，于是
// 0.1 → "0.1"（换算得 10 分），0.30000000000000004 → "0.30000000000000004"（被拒）。
// 负数同样被拒（ParseYuanToFen 内），与本表 CHECK (amount_fen >= 0) 一致：正负由 Type 表达。
func AmountFenFromYuan(yuan float64) (int64, error) {
	return ParseYuanToFen(strconv.FormatFloat(yuan, 'f', -1, 64))
}

// YuanFromAmountFen 整数分 → 元，仅用于还原对外 JSON 的 amount。
func YuanFromAmountFen(fen int64) float64 { return float64(fen) / 100 }

type CreateTransactionInput struct {
	Type     string  `json:"type"`
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
	Date     string  `json:"date"`
	Note     *string `json:"note,omitempty"`
}

type UpdateTransactionInput struct {
	Type     *string  `json:"type,omitempty"`
	Category *string  `json:"category,omitempty"`
	Amount   *float64 `json:"amount,omitempty"`
	Date     *string  `json:"date,omitempty"`
	Note     *string  `json:"note,omitempty"`
}

type TransactionFilter struct {
	UserID string
	Type   string
	Limit  int
	Offset int
}
