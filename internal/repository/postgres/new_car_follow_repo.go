package postgres

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"gorm.io/gorm"
)

//go:embed sql/new_car_follow_schema.sql
var newCarFollowSchemaSQL string

// NewCarFollowRepository 跟进档案 Postgres。
type NewCarFollowRepository struct {
	db *gorm.DB
}

func NewNewCarFollowRepository(db *gorm.DB) *NewCarFollowRepository {
	return &NewCarFollowRepository{db: db}
}

func (r *NewCarFollowRepository) CountStats(
	ctx context.Context, storeID int, ownerUserID string, now time.Time,
) (entity.NewCarFollowStats, error) {
	var stats entity.NewCarFollowStats
	base := r.db.WithContext(ctx).Model(&entity.WysNewCarFollowFile{}).
		Where("store_id = ? AND owner_user_id = ?", storeID, ownerUserID)

	if err := base.Where("stage IN ?", openStages()).Count(&stats.Active).Error; err != nil {
		return stats, err
	}
	if err := r.db.WithContext(ctx).Model(&entity.WysNewCarFollowFile{}).
		Where("store_id = ? AND owner_user_id = ? AND stage IN ? AND next_follow_up_at IS NOT NULL AND next_follow_up_at <= ?",
			storeID, ownerUserID, openStages(), now).
		Count(&stats.Overdue).Error; err != nil {
		return stats, err
	}
	if err := r.db.WithContext(ctx).Model(&entity.WysNewCarFollowFile{}).
		Where("store_id = ? AND owner_user_id = ? AND stage IN ? AND follow_level IN ?",
			storeID, ownerUserID, openStages(), []string{entity.FollowLevelH, entity.FollowLevelA}).
		Count(&stats.HighIntent).Error; err != nil {
		return stats, err
	}
	if err := r.db.WithContext(ctx).Model(&entity.WysNewCarFollowFile{}).
		Where("store_id = ? AND owner_user_id = ? AND stage = ?", storeID, ownerUserID, entity.FollowStageLost).
		Count(&stats.Lost).Error; err != nil {
		return stats, err
	}
	return stats, nil
}

func openStages() []int16 {
	return []int16{
		entity.FollowStageNew,
		entity.FollowStageFollowing,
		entity.FollowStageTestDrive,
		entity.FollowStageQuoted,
	}
}

func (r *NewCarFollowRepository) ListFiles(
	ctx context.Context, storeID int, ownerUserID string, f entity.NewCarFollowListFilter, now time.Time, offset, limit int,
) ([]entity.WysNewCarFollowFile, int64, error) {
	q := r.db.WithContext(ctx).Model(&entity.WysNewCarFollowFile{}).
		Where("store_id = ? AND owner_user_id = ?", storeID, ownerUserID)
	if f.FollowLevel != "" {
		q = q.Where("follow_level = ?", f.FollowLevel)
	}
	if levels := entity.FollowLevelsForIntentBand(f.IntentBand); len(levels) > 0 {
		q = q.Where("follow_level IN ?", levels)
	}
	if f.Stage != nil {
		q = q.Where("stage = ?", *f.Stage)
	}
	if f.OpenOnly {
		q = q.Where("stage IN ?", openStages())
	}
	if f.OverdueOnly {
		q = q.Where("stage IN ? AND next_follow_up_at IS NOT NULL AND next_follow_up_at <= ?", openStages(), now)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []entity.WysNewCarFollowFile
	err := q.Order("updated_at DESC, file_id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

func (r *NewCarFollowRepository) GetFile(ctx context.Context, fileID int64) (*entity.WysNewCarFollowFile, error) {
	var row entity.WysNewCarFollowFile
	err := r.db.WithContext(ctx).Where("file_id = ?", fileID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, repository.ErrNewCarFollowNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *NewCarFollowRepository) CreateFile(ctx context.Context, row *entity.WysNewCarFollowFile) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(row).Error; err != nil {
			return err
		}
		if row.NextFollowUpAt != nil {
			return tx.Model(&entity.WysStoreCustomer{}).
				Where("customer_id = ?", row.CustomerID).
				Update("next_follow_up_at", row.NextFollowUpAt).Error
		}
		return nil
	})
}

func (r *NewCarFollowRepository) UpdateFileAndCustomerFollow(
	ctx context.Context, row *entity.WysNewCarFollowFile, syncCustomerFollow bool,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(row).Error; err != nil {
			return err
		}
		if syncCustomerFollow {
			return tx.Model(&entity.WysStoreCustomer{}).
				Where("customer_id = ?", row.CustomerID).
				Update("next_follow_up_at", row.NextFollowUpAt).Error
		}
		return nil
	})
}

func (r *NewCarFollowRepository) FindOpenFileByCustomer(
	ctx context.Context, storeID int, customerID int64,
) (*entity.WysNewCarFollowFile, error) {
	var row entity.WysNewCarFollowFile
	err := r.db.WithContext(ctx).
		Where("store_id = ? AND customer_id = ? AND stage IN ?", storeID, customerID, openStages()).
		Order("file_id DESC").
		First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *NewCarFollowRepository) GetCustomer(ctx context.Context, customerID int64) (*entity.WysStoreCustomer, error) {
	var row entity.WysStoreCustomer
	err := r.db.WithContext(ctx).Where("customer_id = ?", customerID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, repository.ErrNewCarFollowBadCustomer
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *NewCarFollowRepository) CreateCustomer(ctx context.Context, row *entity.WysStoreCustomer) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *NewCarFollowRepository) ListCustomers(
	ctx context.Context, storeID int, q string, offset, limit int,
) ([]entity.WysStoreCustomer, int64, error) {
	db := r.db.WithContext(ctx).Model(&entity.WysStoreCustomer{}).Where("store_id = ?", storeID)
	q = strings.TrimSpace(q)
	if q != "" {
		like := "%" + q + "%"
		db = db.Where("display_name ILIKE ? OR phone ILIKE ?", like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []entity.WysStoreCustomer
	err := db.Order("customer_id ASC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

func (r *NewCarFollowRepository) GetUserBrief(ctx context.Context, userID string) (string, string, error) {
	var user entity.User
	err := r.db.WithContext(ctx).
		Select("user_name", "avatar_url").
		Where("user_id = ? AND deleted_at IS NULL", userID).
		First(&user).Error
	if err == gorm.ErrRecordNotFound {
		return "", "", nil
	}
	if err != nil {
		return "", "", err
	}
	return user.UserName, user.AvatarURL, nil
}

// EnsureNewCarFollowSchema 建表兜底。
func EnsureNewCarFollowSchema(db *gorm.DB) error {
	if err := db.Exec(newCarFollowSchemaSQL).Error; err != nil {
		return fmt.Errorf("new car follow schema: %w", err)
	}
	if err := db.AutoMigrate(&entity.WysNewCarFollowFile{}); err != nil {
		return fmt.Errorf("new car follow migrate: %w", err)
	}
	return nil
}

// EnsureNewCarFollowSeed 为测试号种几条档案（幂等）。
func EnsureNewCarFollowSeed(db *gorm.DB) error {
	const phone = "13400000000"
	return db.Transaction(func(tx *gorm.DB) error {
		var user entity.User
		err := tx.Where("(phone = ? OR email = ?) AND deleted_at IS NULL",
			phone, phone+"@dev.test.local").
			First(&user).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}
			return err
		}
		storeID := 1
		if user.CurrentStoreID != nil && *user.CurrentStoreID > 0 {
			storeID = *user.CurrentStoreID
		}

		var n int64
		if err := tx.Model(&entity.WysNewCarFollowFile{}).
			Where("store_id = ? AND owner_user_id = ?", storeID, user.UserID).
			Count(&n).Error; err != nil {
			return err
		}
		if n > 0 {
			return nil
		}

		var customers []entity.WysStoreCustomer
		if err := tx.Where("store_id = ?", storeID).Order("customer_id ASC").Limit(4).Find(&customers).Error; err != nil {
			return err
		}
		if len(customers) == 0 {
			past := time.Now().Add(-24 * time.Hour)
			for i, pair := range []struct{ name, phone, level string }{
				{"跟进张女士", "13800001111", entity.FollowLevelH},
				{"跟进王先生", "13800002222", entity.FollowLevelB},
				{"跟进李女士", "13800003333", entity.FollowLevelE},
			} {
				c := entity.WysStoreCustomer{
					StoreID: storeID, DisplayName: pair.name, Phone: pair.phone,
				}
				if i == 0 {
					c.NextFollowUpAt = &past
				}
				if err := tx.Create(&c).Error; err != nil {
					return err
				}
				customers = append(customers, c)
			}
		}

		now := time.Now()
		past := now.Add(-48 * time.Hour)
		future := now.Add(72 * time.Hour)
		seeds := []struct {
			cust  entity.WysStoreCustomer
			level string
			stage int16
			next  *time.Time
		}{}
		levels := []string{entity.FollowLevelH, entity.FollowLevelA, entity.FollowLevelB, entity.FollowLevelE}
		for i, c := range customers {
			if i >= len(levels) {
				break
			}
			var next *time.Time
			stage := entity.FollowStageFollowing
			switch i {
			case 0:
				next = &past
			case 1:
				next = &future
			case 2:
				next = &future
			case 3:
				stage = entity.FollowStageLost
			}
			seeds = append(seeds, struct {
				cust  entity.WysStoreCustomer
				level string
				stage int16
				next  *time.Time
			}{c, levels[i], stage, next})
		}
		for _, s := range seeds {
			row := entity.WysNewCarFollowFile{
				StoreID:         storeID,
				OwnerUserID:     user.UserID,
				CustomerID:      s.cust.CustomerID,
				FollowLevel:     s.level,
				Stage:           s.stage,
				VehicleInterest: "意向车型示例",
				CustomerName:    s.cust.DisplayName,
				CustomerPhone:   s.cust.Phone,
				NextFollowUpAt:  s.next,
				CreatedAt:       now,
				UpdatedAt:       now,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			if s.next != nil && entity.FollowFileIsOpen(s.stage) {
				if err := tx.Model(&entity.WysStoreCustomer{}).
					Where("customer_id = ?", s.cust.CustomerID).
					Update("next_follow_up_at", s.next).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}
