// =============================================================================
// 文件：transaction.go
// 层级：Domain —— 与 Supabase / 本地 transactions 表字段对齐（user_id = UUID）
// =============================================================================
package entity

import "time"

const TransactionsTable = "transactions"

type Transaction struct {
	ID        int64      `json:"id" gorm:"primaryKey"`
	UserID    *string    `json:"user_id,omitempty" gorm:"type:uuid;index;column:user_id"`
	Type      string     `json:"type" gorm:"size:32;not null"`
	Category  string     `json:"category" gorm:"size:64;not null"`
	Amount    float64    `json:"amount" gorm:"not null"`
	Date      string     `json:"date" gorm:"size:32;not null"` // YYYY-MM-DD
	Note      *string    `json:"note,omitempty" gorm:"type:text"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

func (Transaction) TableName() string { return TransactionsTable }

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
