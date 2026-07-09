// transaction_request.go Transaction 请求 DTO。
package request

// CreateTransactionRequest 创建收支记录。
type CreateTransactionRequest struct {
	Type     string  `json:"type" binding:"required"`
	Category string  `json:"category" binding:"required"`
	Amount   float64 `json:"amount" binding:"required"`
	Date     string  `json:"date" binding:"required"`
	Note     *string `json:"note"`
}

// UpdateTransactionRequest 更新收支记录。
type UpdateTransactionRequest struct {
	Type     *string  `json:"type"`
	Category *string  `json:"category"`
	Amount   *float64 `json:"amount"`
	Date     *string  `json:"date"`
	Note     *string  `json:"note"`
}
