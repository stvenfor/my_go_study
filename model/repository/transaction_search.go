package repository

import (
	"context"

	"model/entity"
)

// TransactionSearchStore 交易 ES 索引（可选；nil 表示未启用）。
type TransactionSearchStore interface {
	Index(ctx context.Context, t *entity.Transaction) error
	Delete(ctx context.Context, id int64) error
	Search(ctx context.Context, userID, q string, limit int) ([]entity.Transaction, error)
}
