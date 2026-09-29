// service.go 交易流水（列表 / 分页 / 增删改 / ES 全文搜索）—— 域内业务逻辑。
//
// 由 app/internal/usecase/transaction_usecase.go 迁入。三点改动：
//  1. TransactionUsecase → Service，NewTransactionUsecase → NewService；
//  2. repository.TransactionRepository → Repository，
//     repository.TransactionSearchStore → SearchStore；
//  3. 搜索不可用哨兵改指 shared.ErrSearchUnavailable（见下方注释）。
//
// 【初学者】为什么每个方法都传 accessToken 和 userID？
//
//	accessToken：PostgREST 以用户身份访问（RLS）
//	userID：Repository 显式 .Eq("user_id", userID) 双保险
package transaction

import (
	"context"

	"app/internal/shared"
	"common/metrics"
	"go.uber.org/zap"
	"model/entity"
)

// ErrSearchUnavailable ES 未配置或不可用（= shared.ErrSearchUnavailable）。
//
// ai/cs-agent 也把这个「搜索不可用」当良性分支判定，跨域共享同一值才能让
// errors.Is 两边一致，故哨兵本体上提 shared，域内保留同名别名。
var ErrSearchUnavailable = shared.ErrSearchUnavailable

type Service struct {
	repo  Repository
	store SearchStore // optional
	log   *zap.Logger
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// SetSearchStore 挂载 ES；store 可为 nil。
// ponytail: 同步索引无 outbox；ES 失败只打日志+指标，写路径不回滚。升级：事务 outbox / 异步补偿。
func (u *Service) SetSearchStore(store SearchStore, log *zap.Logger) {
	u.store = store
	u.log = log
}

func (u *Service) List(ctx context.Context, accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, error) {
	return u.repo.List(ctx, accessToken, userID, filter)
}

func (u *Service) ListPage(ctx context.Context, accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, int64, error) {
	return u.repo.ListPage(ctx, accessToken, userID, filter)
}

func (u *Service) Get(ctx context.Context, accessToken, userID string, id int64) (*entity.Transaction, error) {
	return u.repo.GetByID(ctx, accessToken, userID, id)
}

func (u *Service) Create(ctx context.Context, accessToken, userID string, input entity.CreateTransactionInput) (*entity.Transaction, error) {
	item, err := u.repo.Create(ctx, accessToken, userID, input)
	if err != nil {
		return nil, err
	}
	u.bestEffortIndex(ctx, item)
	return item, nil
}

func (u *Service) Update(ctx context.Context, accessToken, userID string, id int64, input entity.UpdateTransactionInput) (*entity.Transaction, error) {
	item, err := u.repo.Update(ctx, accessToken, userID, id, input)
	if err != nil {
		return nil, err
	}
	u.bestEffortIndex(ctx, item)
	return item, nil
}

func (u *Service) Delete(ctx context.Context, accessToken, userID string, id int64) error {
	if err := u.repo.Delete(ctx, accessToken, userID, id); err != nil {
		return err
	}
	u.bestEffortDelete(ctx, id)
	return nil
}

// Search 全文搜索；强制仅会话 user_id（由调用方传入 session user）。
func (u *Service) Search(ctx context.Context, userID, q string, limit int) ([]entity.Transaction, error) {
	if u.store == nil {
		return nil, ErrSearchUnavailable
	}
	return u.store.Search(ctx, userID, q, limit)
}

func (u *Service) bestEffortIndex(ctx context.Context, item *entity.Transaction) {
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

func (u *Service) bestEffortDelete(ctx context.Context, id int64) {
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
