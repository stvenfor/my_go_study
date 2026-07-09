// transaction_record.go 本地 PostgreSQL transactions 表实体。
package entity

import (
	"fmt"
	"time"
)

// TransactionRecord 映射 transactions 表（自建用户体系，不依赖 Supabase）。
type TransactionRecord struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Type      string    `gorm:"size:32;not null" json:"type"`
	Category  string    `gorm:"size:128;not null" json:"category"`
	Amount    float64   `gorm:"not null" json:"amount"`
	Date      string    `gorm:"size:32;not null" json:"date"`
	Note      *string   `gorm:"size:512" json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定 GORM 表名。
func (TransactionRecord) TableName() string {
	return "transactions"
}

// ToTransaction 转为 API 领域模型。
func (r TransactionRecord) ToTransaction() Transaction {
	userID := formatUintID(r.UserID)
	return Transaction{
		ID:        r.ID,
		UserID:    &userID,
		Type:      r.Type,
		Category:  r.Category,
		Amount:    r.Amount,
		Date:      r.Date,
		Note:      r.Note,
		CreatedAt: &r.CreatedAt,
		UpdatedAt: &r.UpdatedAt,
	}
}

func formatUintID(id uint) string {
	return fmt.Sprintf("%d", id)
}
