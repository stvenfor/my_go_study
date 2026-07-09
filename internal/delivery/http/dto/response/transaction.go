// transaction.go Supabase Transaction 统一响应 DTO（camelCase）。
package response

import (
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

// TransactionItem 收支记录响应项。
type TransactionItem struct {
	ID        int64      `json:"id"`
	UserID    *string    `json:"userId,omitempty"`
	Type      string     `json:"type"`
	Category  string     `json:"category"`
	Amount    float64    `json:"amount"`
	Date      string     `json:"date"`
	Note      *string    `json:"note,omitempty"`
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

// FromTransaction 从领域实体转换为响应 DTO。
func FromTransaction(t *entity.Transaction) TransactionItem {
	if t == nil {
		return TransactionItem{}
	}
	return TransactionItem{
		ID:        t.ID,
		UserID:    t.UserID,
		Type:      t.Type,
		Category:  t.Category,
		Amount:    t.Amount,
		Date:      t.Date,
		Note:      t.Note,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

// FromTransactions 批量转换。
func FromTransactions(items []entity.Transaction) []TransactionItem {
	list := make([]TransactionItem, 0, len(items))
	for i := range items {
		list = append(list, FromTransaction(&items[i]))
	}
	return list
}
