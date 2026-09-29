package membership

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"model/entity"
)

// PostgresRepository 会员 Postgres。
type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// EnsureSchema 幂等建表。
//
// sql/membership_schema.sql 的权威执行点是 postgres.ConsolidateLocalUsers
// （在 cmd/api 的 autoMigrate 里早于各域 EnsureSchema 跑）；域侧只留
// AutoMigrate 兜底，不再重复 embed —— 与批 2 五个域的处置一致。
func EnsureSchema(db *gorm.DB) error {
	return db.AutoMigrate(&entity.WysMembershipEntitlement{}, &entity.WysMembershipOrder{})
}

func (r *PostgresRepository) ListEntitlements(ctx context.Context, userID string) ([]entity.WysMembershipEntitlement, error) {
	var rows []entity.WysMembershipEntitlement
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&rows).Error
	return rows, err
}

func (r *PostgresRepository) GetEntitlement(ctx context.Context, userID, tier string) (*entity.WysMembershipEntitlement, error) {
	var row entity.WysMembershipEntitlement
	err := r.db.WithContext(ctx).Where("user_id = ? AND tier = ?", userID, tier).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *PostgresRepository) ExtendEntitlement(
	ctx context.Context,
	userID, tier, channel string,
	months int,
	now time.Time,
	huaweiToken, huaweiSubID, appleOriginalTxID string,
) (*entity.WysMembershipEntitlement, error) {
	if months <= 0 {
		return nil, fmt.Errorf("months invalid")
	}
	var out entity.WysMembershipEntitlement
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var cur entity.WysMembershipEntitlement
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND tier = ?", userID, tier).
			First(&cur).Error
		base := now
		if err == nil && cur.ExpiresAt.After(now) {
			base = cur.ExpiresAt
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		expires := base.AddDate(0, months, 0)
		row := entity.WysMembershipEntitlement{
			UserID:                     userID,
			Tier:                       tier,
			ExpiresAt:                  expires,
			SourceChannel:              channel,
			HuaweiPurchaseToken:        huaweiToken,
			HuaweiSubscriptionID:       huaweiSubID,
			AppleOriginalTransactionID: appleOriginalTxID,
			UpdatedAt:                  now,
		}
		if cur.HuaweiPurchaseToken != "" && huaweiToken == "" {
			row.HuaweiPurchaseToken = cur.HuaweiPurchaseToken
		}
		if cur.HuaweiSubscriptionID != "" && huaweiSubID == "" {
			row.HuaweiSubscriptionID = cur.HuaweiSubscriptionID
		}
		if cur.AppleOriginalTransactionID != "" && appleOriginalTxID == "" {
			row.AppleOriginalTransactionID = cur.AppleOriginalTransactionID
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "tier"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"expires_at", "source_channel", "huawei_purchase_token",
				"huawei_subscription_id", "apple_original_transaction_id", "updated_at",
			}),
		}).Create(&row).Error; err != nil {
			return err
		}
		out = row
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *PostgresRepository) CreateOrder(ctx context.Context, order *entity.WysMembershipOrder) error {
	return r.db.WithContext(ctx).Create(order).Error
}

func (r *PostgresRepository) GetOrder(ctx context.Context, orderID int64) (*entity.WysMembershipOrder, error) {
	var row entity.WysMembershipOrder
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *PostgresRepository) MarkOrderPaid(ctx context.Context, orderID int64, paidAt time.Time) error {
	res := r.db.WithContext(ctx).Model(&entity.WysMembershipOrder{}).
		Where("order_id = ? AND status = ?", orderID, entity.MembershipOrderPending).
		Updates(map[string]any{
			"status":  entity.MembershipOrderPaid,
			"paid_at": paidAt,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrOrderPaid
	}
	return nil
}

func (r *PostgresRepository) FindPaidByOutTradeNo(ctx context.Context, outTradeNo string) (*entity.WysMembershipOrder, error) {
	if outTradeNo == "" {
		return nil, nil
	}
	var row entity.WysMembershipOrder
	err := r.db.WithContext(ctx).
		Where("out_trade_no = ? AND status = ?", outTradeNo, entity.MembershipOrderPaid).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}
