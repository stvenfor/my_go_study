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

// PointsRepository 积分 Postgres 实现。
type PointsRepository struct {
	db *gorm.DB
}

// NewPointsRepository 创建。
func NewPointsRepository(db *gorm.DB) *PointsRepository {
	return &PointsRepository{db: db}
}

func (r *PointsRepository) GetBalance(ctx context.Context, userID string) (int64, error) {
	var w entity.WysPointsWallet
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&w).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return w.Balance, nil
}

func (r *PointsRepository) Credit(ctx context.Context, userID string, delta int64, reason, refID string) (int64, error) {
	if delta <= 0 {
		return 0, errors.New("credit delta must be positive")
	}
	var bal int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var w entity.WysPointsWallet
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&w).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			w = entity.WysPointsWallet{UserID: userID, Balance: 0, UpdatedAt: time.Now()}
			if err := tx.Create(&w).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		w.Balance += delta
		w.UpdatedAt = time.Now()
		if err := tx.Model(&w).Updates(map[string]interface{}{"balance": w.Balance, "updated_at": w.UpdatedAt}).Error; err != nil {
			return err
		}
		led := entity.WysPointsLedger{
			UserID: userID, Delta: delta, Balance: w.Balance, Reason: reason, RefID: refID, CreatedAt: time.Now(),
		}
		if err := tx.Create(&led).Error; err != nil {
			return err
		}
		bal = w.Balance
		return nil
	})
	return bal, err
}

func (r *PointsRepository) Debit(ctx context.Context, userID string, delta int64, reason, refID string) (int64, error) {
	if delta <= 0 {
		return 0, errors.New("debit delta must be positive")
	}
	var bal int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var w entity.WysPointsWallet
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&w).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return repository.ErrPointsInsufficient
		}
		if err != nil {
			return err
		}
		if w.Balance < delta {
			return repository.ErrPointsInsufficient
		}
		w.Balance -= delta
		w.UpdatedAt = time.Now()
		if err := tx.Model(&w).Updates(map[string]interface{}{"balance": w.Balance, "updated_at": w.UpdatedAt}).Error; err != nil {
			return err
		}
		led := entity.WysPointsLedger{
			UserID: userID, Delta: -delta, Balance: w.Balance, Reason: reason, RefID: refID, CreatedAt: time.Now(),
		}
		if err := tx.Create(&led).Error; err != nil {
			return err
		}
		bal = w.Balance
		return nil
	})
	return bal, err
}

func (r *PointsRepository) GetCheckIn(ctx context.Context, userID, day string) (*entity.WysCheckIn, error) {
	var row entity.WysCheckIn
	err := r.db.WithContext(ctx).Where("user_id = ? AND day = ?", userID, day).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *PointsRepository) ListCheckIns(ctx context.Context, userID, fromDay, toDay string) ([]entity.WysCheckIn, error) {
	var rows []entity.WysCheckIn
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND day >= ? AND day <= ?", userID, fromDay, toDay).
		Order("day ASC").
		Find(&rows).Error
	return rows, err
}

func (r *PointsRepository) InsertCheckIn(ctx context.Context, row *entity.WysCheckIn) error {
	err := r.db.WithContext(ctx).Create(row).Error
	if err != nil && isUniqueViolation(err) {
		return repository.ErrPointsAlreadyCheckedIn
	}
	return err
}

func (r *PointsRepository) GetTaskClaim(ctx context.Context, userID, day, taskCode string) (*entity.WysTaskClaim, error) {
	var row entity.WysTaskClaim
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND day = ? AND task_code = ?", userID, day, taskCode).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *PointsRepository) InsertTaskClaim(ctx context.Context, row *entity.WysTaskClaim) error {
	err := r.db.WithContext(ctx).Create(row).Error
	if err != nil && isUniqueViolation(err) {
		return repository.ErrPointsAlreadyCheckedIn
	}
	return err
}

func (r *PointsRepository) CheckInAtomic(ctx context.Context, row *entity.WysCheckIn) (int64, error) {
	var bal int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(row).Error; err != nil {
			if isUniqueViolation(err) {
				return repository.ErrPointsAlreadyCheckedIn
			}
			return err
		}
		var w entity.WysPointsWallet
		e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", row.UserID).First(&w).Error
		if errors.Is(e, gorm.ErrRecordNotFound) {
			w = entity.WysPointsWallet{UserID: row.UserID, Balance: 0, UpdatedAt: time.Now()}
			if err := tx.Create(&w).Error; err != nil {
				return err
			}
		} else if e != nil {
			return e
		}
		w.Balance += row.Points
		w.UpdatedAt = time.Now()
		if err := tx.Model(&w).Updates(map[string]interface{}{"balance": w.Balance, "updated_at": w.UpdatedAt}).Error; err != nil {
			return err
		}
		led := entity.WysPointsLedger{
			UserID: row.UserID, Delta: row.Points, Balance: w.Balance,
			Reason: entity.PointsReasonCheckIn, RefID: row.Day, CreatedAt: time.Now(),
		}
		if err := tx.Create(&led).Error; err != nil {
			return err
		}
		bal = w.Balance
		return nil
	})
	return bal, err
}

func (r *PointsRepository) ClaimTaskAtomic(ctx context.Context, claim *entity.WysTaskClaim, reason string) (int64, error) {
	var bal int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(claim).Error; err != nil {
			if isUniqueViolation(err) {
				return repository.ErrPointsAlreadyCheckedIn
			}
			return err
		}
		var w entity.WysPointsWallet
		e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", claim.UserID).First(&w).Error
		if errors.Is(e, gorm.ErrRecordNotFound) {
			w = entity.WysPointsWallet{UserID: claim.UserID, Balance: 0, UpdatedAt: time.Now()}
			if err := tx.Create(&w).Error; err != nil {
				return err
			}
		} else if e != nil {
			return e
		}
		w.Balance += claim.Points
		w.UpdatedAt = time.Now()
		if err := tx.Model(&w).Updates(map[string]interface{}{"balance": w.Balance, "updated_at": w.UpdatedAt}).Error; err != nil {
			return err
		}
		led := entity.WysPointsLedger{
			UserID: claim.UserID, Delta: claim.Points, Balance: w.Balance,
			Reason: reason, RefID: claim.Day + ":" + claim.TaskCode, CreatedAt: time.Now(),
		}
		if err := tx.Create(&led).Error; err != nil {
			return err
		}
		bal = w.Balance
		return nil
	})
	return bal, err
}

