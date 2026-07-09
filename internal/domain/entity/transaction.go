// =============================================================================
// 文件：transaction.go
// 层级：Domain —— 与 Supabase transactions 表字段一一对应
//
// 【初学者】UserID 用 *string 是因为 PostgREST 可能返回 null；
// Flutter 侧 BackendTransaction 用 String? 对齐。
// =============================================================================
package entity

import "time"

const TransactionsTable = "transactions"

type Transaction struct {
	ID        int64      `json:"id"`
	UserID    *string    `json:"user_id,omitempty"` // 所属用户 UUID
	Type      string     `json:"type"`              // income / expense
	Category  string     `json:"category"`
	Amount    float64    `json:"amount"`
	Date      string     `json:"date"`              // YYYY-MM-DD
	Note      *string    `json:"note,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

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
	UserID string // 必填，Repository 强制 Eq
	Type   string // 可选筛选 income/expense
	Limit  int
	Offset int
}
