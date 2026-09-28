// =============================================================================
// 文件：transaction_usecase.go
// 层级：Usecase —— 薄封装，业务规则可在此扩展（如金额校验、分类白名单）
//
// 【初学者】为什么每个方法都传 accessToken 和 userID？
//
//	accessToken：PostgREST 以用户身份访问（RLS）
//	userID：Repository 显式 .Eq("user_id", userID) 双保险
//
// =============================================================================
package usecase

import (
	"context"
	"errors"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"github.com/stvenfor/my_go_study/pkg/metrics"
	"go.uber.org/zap"
)

// ErrTransactionSearchUnavailable ES 未配置或不可用。
var ErrTransactionSearchUnavailable = errors.New("transaction search unavailable")

type TransactionUsecase struct {
	repo  repository.TransactionRepository
	store repository.TransactionSearchStore // optional
	log   *zap.Logger
}

func NewTransactionUsecase(repo repository.TransactionRepository) *TransactionUsecase {
	return &TransactionUsecase{repo: repo}
}

// SetSearchStore 挂载 ES；store 可为 nil。
// ponytail: 同步索引无 outbox；ES 失败只打日志+指标，写路径不回滚。升级：事务 outbox / 异步补偿。
func (u *TransactionUsecase) SetSearchStore(store repository.TransactionSearchStore, log *zap.Logger) {
	u.store = store
	u.log = log
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
	item, err := u.repo.Create(ctx, accessToken, userID, input)
	if err != nil {
		return nil, err
	}
	u.bestEffortIndex(ctx, item)
	return item, nil
}

func (u *TransactionUsecase) Update(ctx context.Context, accessToken, userID string, id int64, input entity.UpdateTransactionInput) (*entity.Transaction, error) {
	item, err := u.repo.Update(ctx, accessToken, userID, id, input)
	if err != nil {
		return nil, err
	}
	u.bestEffortIndex(ctx, item)
	return item, nil
}

func (u *TransactionUsecase) Delete(ctx context.Context, accessToken, userID string, id int64) error {
	if err := u.repo.Delete(ctx, accessToken, userID, id); err != nil {
		return err
	}
	u.bestEffortDelete(ctx, id)
	return nil
}

// Search 全文搜索；强制仅会话 user_id（由调用方传入 session user）。
func (u *TransactionUsecase) Search(ctx context.Context, userID, q string, limit int) ([]entity.Transaction, error) {
	if u.store == nil {
		return nil, ErrTransactionSearchUnavailable
	}
	return u.store.Search(ctx, userID, q, limit)
}

func (u *TransactionUsecase) bestEffortIndex(ctx context.Context, item *entity.Transaction) {
	if u.store == nil || item == nil {
		return
	}
	if err := u.store.Index(ctx, item); err != nil {
		metrics.ESTransactionIndexFailures.Inc()
		if u.log != nil {
			u.log.Warn("elasticsearch index transaction failed", zap.Int64("id", item.ID), zap.Error(err))
		}
	}
}

func (u *TransactionUsecase) bestEffortDelete(ctx context.Context, id int64) {
	if u.store == nil {
		return
	}
	if err := u.store.Delete(ctx, id); err != nil {
		metrics.ESTransactionIndexFailures.Inc()
		if u.log != nil {
			u.log.Warn("elasticsearch delete transaction failed", zap.Int64("id", id), zap.Error(err))
		}
	}
}
