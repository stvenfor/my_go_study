package domain

import "time"

const TransactionsTable = "transactions"

type Transaction struct {
	ID        int64      `json:"id"`
	Type      string     `json:"type"`
	Category  string     `json:"category"`
	Amount    float64    `json:"amount"`
	Date      string     `json:"date"`
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
	Type   string
	Limit  int
	Offset int
}
