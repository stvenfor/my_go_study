// transaction_record.go 本地 PostgreSQL transactions 表实体。
package entity

import (
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

// TableName 遗留 uint 交易表（已弃用；本地 UUID 路径使用 entity.Transaction → transactions）。
func (TransactionRecord) TableName() string {
	return "transaction_records"
}
