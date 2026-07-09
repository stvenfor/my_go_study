// transaction_repository.go 定义 Transaction 数据访问接口。
package repository

import (
	"context"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

// TransactionRepository Supabase transactions 仓储接口。
type TransactionRepository interface {
	List(ctx context.Context, accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, error)
	ListPage(ctx context.Context, accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, int64, error)
	GetByID(ctx context.Context, accessToken, userID string, id int64) (*entity.Transaction, error)
	Create(ctx context.Context, accessToken, userID string, input entity.CreateTransactionInput) (*entity.Transaction, error)
	Update(ctx context.Context, accessToken, userID string, id int64, input entity.UpdateTransactionInput) (*entity.Transaction, error)
	Delete(ctx context.Context, accessToken, userID string, id int64) error
}
