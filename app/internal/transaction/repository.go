// repository.go transactions（交易流水）数据访问接口。
//
// 由 model/repository/transaction_repository.go 与 transaction_search.go 合并迁入。
// TransactionRepository → Repository，TransactionSearchStore → SearchStore
// （域内即命名空间，错误哨兵与接口一并去掉域前缀）。
//
// 三套实现同域共存：PostgresRepository（本机 Postgres，user_id=UUID）、
// SupabaseRepository（PostgREST + RLS，Flutter 线上路径）、
// ESStore（Elasticsearch 全文搜索，可选，nil 表示未启用）。
package transaction

import (
	"context"

	"model/entity"
)

// Repository transactions 仓储接口。
type Repository interface {
	List(ctx context.Context, accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, error)
	ListPage(ctx context.Context, accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, int64, error)
	GetByID(ctx context.Context, accessToken, userID string, id int64) (*entity.Transaction, error)
	Create(ctx context.Context, accessToken, userID string, input entity.CreateTransactionInput) (*entity.Transaction, error)
	Update(ctx context.Context, accessToken, userID string, id int64, input entity.UpdateTransactionInput) (*entity.Transaction, error)
	Delete(ctx context.Context, accessToken, userID string, id int64) error
}

// SearchStore 交易 ES 索引（可选；nil 表示未启用）。
type SearchStore interface {
	Index(ctx context.Context, t *entity.Transaction) error
	Delete(ctx context.Context, id int64) error
	Search(ctx context.Context, userID, q string, limit int) ([]entity.Transaction, error)
}
