// =============================================================================
// 文件：transaction_usecase.go
// 层级：Usecase —— 薄封装，业务规则可在此扩展（如金额校验、分类白名单）
//
// 【初学者】为什么每个方法都传 accessToken 和 userID？
//   accessToken：PostgREST 以用户身份访问（RLS）
//   userID：Repository 显式 .Eq("user_id", userID) 双保险
// =============================================================================
package usecase

import (
	"context"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

type TransactionUsecase struct {
	repo repository.TransactionRepository
}

func NewTransactionUsecase(repo repository.TransactionRepository) *TransactionUsecase {
	return &TransactionUsecase{repo: repo}
}

func (u *TransactionUsecase) List(ctx context.Context, accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, error) {
	return u.repo.List(ctx, accessToken, userID, filter)
}

func (u *TransactionUsecase) ListPage(ctx context.Context, accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, int64, error) {
	return u.repo.ListPage(ctx, accessToken, userID, filter)
}

func (u *TransactionUsecase) Get(ctx context.Context, accessToken, userID string, id int64) (*entity.Transaction, error) {
	return u.repo.GetByID(ctx, accessToken, userID, id)
}

func (u *TransactionUsecase) Create(ctx context.Context, accessToken, userID string, input entity.CreateTransactionInput) (*entity.Transaction, error) {
	return u.repo.Create(ctx, accessToken, userID, input)
}

func (u *TransactionUsecase) Update(ctx context.Context, accessToken, userID string, id int64, input entity.UpdateTransactionInput) (*entity.Transaction, error) {
	return u.repo.Update(ctx, accessToken, userID, id, input)
}

func (u *TransactionUsecase) Delete(ctx context.Context, accessToken, userID string, id int64) error {
	return u.repo.Delete(ctx, accessToken, userID, id)
}
