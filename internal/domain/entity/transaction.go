// transaction.go Supabase transactions 表领域模型。
package entity

import "time"

const TransactionsTable = "transactions"

// Transaction 收支记录（与 Flutter BackendTransaction 字段对齐）。
type Transaction struct {
	ID        int64      `json:"id"`
	UserID    *string    `json:"user_id,omitempty"`
	Type      string     `json:"type"`
	Category  string     `json:"category"`
	Amount    float64    `json:"amount"`
	Date      string     `json:"date"`
	Note      *string    `json:"note,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

// CreateTransactionInput 创建收支记录。
type CreateTransactionInput struct {
	Type     string  `json:"type"`
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
	Date     string  `json:"date"`
	Note     *string `json:"note,omitempty"`
}

// UpdateTransactionInput 更新收支记录。
type UpdateTransactionInput struct {
	Type     *string  `json:"type,omitempty"`
	Category *string  `json:"category,omitempty"`
	Amount   *float64 `json:"amount,omitempty"`
	Date     *string  `json:"date,omitempty"`
	Note     *string  `json:"note,omitempty"`
}

// TransactionFilter 列表筛选条件。
type TransactionFilter struct {
	UserID string
	Type   string
	Limit  int
	Offset int
}
