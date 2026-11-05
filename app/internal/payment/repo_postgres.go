// repo_postgres.go 支付流水仓储；资金动作复用 P0-3 的 txdb 事务传递。
package payment

import (
	"context"
	"errors"
	"time"

	"app/internal/txdb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"model/entity"
)

type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// EnsureSchema 仅 lab 环境的 AutoMigrate 建表；线上走
// migrations/20261104160000_payment_ledger（P0-1 闸门不允许启动期 DDL）。
func EnsureSchema(db *gorm.DB) error {
	return db.AutoMigrate(&entity.WysPaymentRecord{}, &entity.WysPaymentReconcileDiff{})
}

var _ Repository = (*PostgresRepository)(nil)

func (r *PostgresRepository) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return txdb.Begin(ctx, r.db, fn)
}

func (r *PostgresRepository) CreatePending(ctx context.Context, rec *entity.WysPaymentRecord) error {
	now := time.Now()
	rec.Status = entity.PaymentStatusPending
	rec.CreatedAt = now
	rec.UpdatedAt = now
	return txdb.DB(ctx, r.db).WithContext(ctx).Create(rec).Error
}

func (r *PostgresRepository) GetByOutTradeNo(ctx context.Context, outTradeNo string) (*entity.WysPaymentRecord, error) {
	var rec entity.WysPaymentRecord
	err := txdb.DB(ctx, r.db).WithContext(ctx).Where("out_trade_no = ?", outTradeNo).First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRecordNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *PostgresRepository) GetPaidByRef(ctx context.Context, refType, refID string) (*entity.WysPaymentRecord, error) {
	var rec entity.WysPaymentRecord
	err := txdb.DB(ctx, r.db).WithContext(ctx).
		Where("ref_type = ? AND ref_id = ? AND status = ?", refType, refID, entity.PaymentStatusPaid).
		Order("id DESC").First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRecordNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *PostgresRepository) GetLatestByRef(ctx context.Context, refType, refID string) (*entity.WysPaymentRecord, error) {
	var rec entity.WysPaymentRecord
	err := txdb.DB(ctx, r.db).WithContext(ctx).
		Where("ref_type = ? AND ref_id = ?", refType, refID).
		Order("id DESC").First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrRecordNotFound
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (r *PostgresRepository) SettlePaid(ctx context.Context, outTradeNo, channelTradeNo string, amountFen int64, notifyRaw string, paidAt time.Time) (*entity.WysPaymentRecord, bool, error) {
	var out *entity.WysPaymentRecord
	var newly bool
	err := txdb.DB(ctx, r.db).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rec entity.WysPaymentRecord
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("out_trade_no = ?", outTradeNo).First(&rec).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRecordNotFound
			}
			return err
		}
		if rec.Status == entity.PaymentStatusPaid {
			out = &rec
			return nil
		}
		if rec.Status == entity.PaymentStatusFailed {
			return ErrRecordNotPending
		}
		if rec.AmountFen != amountFen {
			return ErrAmountMismatch
		}
		res := tx.Model(&entity.WysPaymentRecord{}).
			Where("id = ? AND status = ?", rec.ID, entity.PaymentStatusPending).
			Updates(map[string]interface{}{
				"status":           entity.PaymentStatusPaid,
				"channel_trade_no": channelTradeNo,
				"notify_raw":       notifyRaw,
				"paid_at":          paidAt,
				"updated_at":       time.Now(),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			// 并发核销：另一请求已抢先置 paid，本次按幂等处理。
			return ErrRecordNotPending
		}
		rec.Status = entity.PaymentStatusPaid
		rec.ChannelTradeNo = &channelTradeNo
		rec.NotifyRaw = notifyRaw
		rec.PaidAt = &paidAt
		out = &rec
		newly = true
		return nil
	})
	return out, newly, err
}

func (r *PostgresRepository) MarkFailed(ctx context.Context, outTradeNo, reason string) error {
	if len(reason) > 128 {
		reason = reason[:128]
	}
	return txdb.DB(ctx, r.db).WithContext(ctx).Model(&entity.WysPaymentRecord{}).
		Where("out_trade_no = ? AND status = ?", outTradeNo, entity.PaymentStatusPending).
		Updates(map[string]interface{}{
			"status":      entity.PaymentStatusFailed,
			"fail_reason": reason,
			"updated_at":  time.Now(),
		}).Error
}

func (r *PostgresRepository) ListStalePending(ctx context.Context, olderThan time.Time, limit int) ([]entity.WysPaymentRecord, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	var rows []entity.WysPaymentRecord
	err := txdb.DB(ctx, r.db).WithContext(ctx).
		Where("status = ? AND created_at <= ?", entity.PaymentStatusPending, olderThan).
		Order("created_at").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *PostgresRepository) ListWindow(ctx context.Context, channel string, from, to time.Time) ([]entity.WysPaymentRecord, error) {
	var rows []entity.WysPaymentRecord
	q := txdb.DB(ctx, r.db).WithContext(ctx).Where("created_at >= ? AND created_at < ?", from, to)
	if channel != "" {
		q = q.Where("channel = ?", channel)
	}
	err := q.Order("id").Find(&rows).Error
	return rows, err
}

func (r *PostgresRepository) CreateDiff(ctx context.Context, diff *entity.WysPaymentReconcileDiff) error {
	diff.CreatedAt = time.Now()
	return txdb.DB(ctx, r.db).WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).Create(diff).Error
}
