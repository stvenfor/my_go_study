// transaction_usecase.go Transaction 业务用例。
package usecase

import (
	"context"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

// TransactionUsecase 处理收支记录 CRUD。
type TransactionUsecase struct {
	repo repository.TransactionRepository
}

// NewTransactionUsecase 创建 Transaction 用例。
func NewTransactionUsecase(repo repository.TransactionRepository) *TransactionUsecase {
	return &TransactionUsecase{repo: repo}
}

// List 分页列表（Flutter limit/offset）。
func (u *TransactionUsecase) List(ctx context.Context, accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, error) {
	return u.repo.List(ctx, accessToken, userID, filter)
}

// ListPage 分页列表（统一响应 page/size，含 total）。
func (u *TransactionUsecase) ListPage(ctx context.Context, accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, int64, error) {
	return u.repo.ListPage(ctx, accessToken, userID, filter)
}

// Get 按 ID 查询。
func (u *TransactionUsecase) Get(ctx context.Context, accessToken, userID string, id int64) (*entity.Transaction, error) {
	return u.repo.GetByID(ctx, accessToken, userID, id)
}

// Create 创建记录。
func (u *TransactionUsecase) Create(ctx context.Context, accessToken, userID string, input entity.CreateTransactionInput) (*entity.Transaction, error) {
	return u.repo.Create(ctx, accessToken, userID, input)
}

// Update 更新记录。
func (u *TransactionUsecase) Update(ctx context.Context, accessToken, userID string, id int64, input entity.UpdateTransactionInput) (*entity.Transaction, error) {
	return u.repo.Update(ctx, accessToken, userID, id, input)
}

// Delete 删除记录。
func (u *TransactionUsecase) Delete(ctx context.Context, accessToken, userID string, id int64) error {
	return u.repo.Delete(ctx, accessToken, userID, id)
}
