package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CashWalletRepository 人民币钱包 Postgres。
type CashWalletRepository struct {
	db *gorm.DB
}

func NewCashWalletRepository(db *gorm.DB) *CashWalletRepository {
	return &CashWalletRepository{db: db}
}

func (r *CashWalletRepository) GetBalanceFen(ctx context.Context, userID string) (int64, error) {
	var w entity.WysCashWallet
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&w).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return w.BalanceFen, nil
}

func (r *CashWalletRepository) Credit(ctx context.Context, userID string, deltaFen int64, reason, refID string) (int64, error) {
	if deltaFen <= 0 {
		return 0, repository.ErrCashInvalidAmount
	}
	var bal int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var w entity.WysCashWallet
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&w).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w = entity.WysCashWallet{UserID: userID, BalanceFen: 0, UpdatedAt: time.Now()}
			if err := tx.Create(&w).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		w.BalanceFen += deltaFen
		w.UpdatedAt = time.Now()
		if err := tx.Model(&w).Updates(map[string]interface{}{
			"balance_fen": w.BalanceFen, "updated_at": w.UpdatedAt,
		}).Error; err != nil {
			return err
		}
		led := entity.WysCashLedger{
			UserID: userID, DeltaFen: deltaFen, BalanceFen: w.BalanceFen,
			Reason: reason, RefID: refID, CreatedAt: time.Now(),
		}
		if err := tx.Create(&led).Error; err != nil {
			return err
		}
		bal = w.BalanceFen
		return nil
	})
	return bal, err
}

func (r *CashWalletRepository) Debit(ctx context.Context, userID string, deltaFen int64, reason, refID string) (int64, error) {
	if deltaFen <= 0 {
		return 0, repository.ErrCashInvalidAmount
	}
	var bal int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var w entity.WysCashWallet
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&w).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return repository.ErrCashInsufficient
		}
		if err != nil {
			return err
		}
		if w.BalanceFen < deltaFen {
			return repository.ErrCashInsufficient
		}
		w.BalanceFen -= deltaFen
		w.UpdatedAt = time.Now()
		if err := tx.Model(&w).Updates(map[string]interface{}{
			"balance_fen": w.BalanceFen, "updated_at": w.UpdatedAt,
		}).Error; err != nil {
			return err
		}
		led := entity.WysCashLedger{
			UserID: userID, DeltaFen: -deltaFen, BalanceFen: w.BalanceFen,
			Reason: reason, RefID: refID, CreatedAt: time.Now(),
		}
		if err := tx.Create(&led).Error; err != nil {
			return err
		}
		bal = w.BalanceFen
		return nil
	})
	return bal, err
}

func (r *CashWalletRepository) ListLedger(ctx context.Context, userID string, limit, offset int) ([]entity.WysCashLedger, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	var total int64
	q := r.db.WithContext(ctx).Model(&entity.WysCashLedger{}).Where("user_id = ?", userID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []entity.WysCashLedger
	err := q.Order("ledger_id DESC").Limit(limit).Offset(offset).Find(&rows).Error
	return rows, total, err
}

func (r *CashWalletRepository) ListCards(ctx context.Context, userID string) ([]entity.WysBankCard, error) {
	var rows []entity.WysBankCard
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).
		Order("is_default DESC, card_id ASC").Find(&rows).Error
	return rows, err
}

func (r *CashWalletRepository) GetCard(ctx context.Context, userID string, cardID int64) (*entity.WysBankCard, error) {
	var row entity.WysBankCard
	err := r.db.WithContext(ctx).Where("user_id = ? AND card_id = ?", userID, cardID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *CashWalletRepository) CreateCard(ctx context.Context, card *entity.WysBankCard) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var n int64
		if err := tx.Model(&entity.WysBankCard{}).Where("user_id = ?", card.UserID).Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			card.IsDefault = true
		}
		now := time.Now()
		card.CreatedAt = now
		card.UpdatedAt = now
		if err := tx.Create(card).Error; err != nil {
			return err
		}
		if card.IsDefault {
			if err := tx.Model(&entity.WysBankCard{}).
				Where("user_id = ? AND card_id <> ?", card.UserID, card.CardID).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *CashWalletRepository) SetDefaultCard(ctx context.Context, userID string, cardID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row entity.WysBankCard
		if err := tx.Where("user_id = ? AND card_id = ?", userID, cardID).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repository.ErrCashCardNotFound
			}
			return err
		}
		if err := tx.Model(&entity.WysBankCard{}).Where("user_id = ?", userID).
			Update("is_default", false).Error; err != nil {
			return err
		}
		return tx.Model(&row).Updates(map[string]interface{}{
			"is_default": true, "updated_at": time.Now(),
		}).Error
	})
}

func (r *CashWalletRepository) DeleteCard(ctx context.Context, userID string, cardID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row entity.WysBankCard
		if err := tx.Where("user_id = ? AND card_id = ?", userID, cardID).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repository.ErrCashCardNotFound
			}
			return err
		}
		if err := tx.Delete(&row).Error; err != nil {
			return err
		}
		if row.IsDefault {
			var next entity.WysBankCard
			err := tx.Where("user_id = ?", userID).Order("card_id ASC").First(&next).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			if err != nil {
				return err
			}
			return tx.Model(&next).Updates(map[string]interface{}{
				"is_default": true, "updated_at": time.Now(),
			}).Error
		}
		return nil
	})
}
