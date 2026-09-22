package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"gorm.io/gorm"
)

type UsedCarOrderRepository struct {
	db *gorm.DB
}

func NewUsedCarOrderRepository(db *gorm.DB) *UsedCarOrderRepository {
	return &UsedCarOrderRepository{db: db}
}

func (r *UsedCarOrderRepository) CountStats(ctx context.Context, storeID int, uploaderUserID string) (entity.UsedCarOrderStats, error) {
	var stats entity.UsedCarOrderStats
	base := r.db.WithContext(ctx).Model(&entity.WysUsedCarOrder{}).
		Where("store_id = ? AND uploader_user_id = ?", storeID, uploaderUserID)
	if err := base.Count(&stats.Submitted).Error; err != nil {
		return stats, err
	}
	if err := r.db.WithContext(ctx).Model(&entity.WysUsedCarOrder{}).
		Where("store_id = ? AND uploader_user_id = ? AND status = ?", storeID, uploaderUserID, entity.UsedCarOrderPendingReview).
		Count(&stats.PendingReview).Error; err != nil {
		return stats, err
	}
	if err := r.db.WithContext(ctx).Model(&entity.WysUsedCarOrder{}).
		Where("store_id = ? AND uploader_user_id = ? AND status IN ?", storeID, uploaderUserID,
			[]int16{entity.UsedCarOrderApprovedPendingRating, entity.UsedCarOrderRated}).
		Count(&stats.Approved).Error; err != nil {
		return stats, err
	}
	if err := r.db.WithContext(ctx).Model(&entity.WysUsedCarOrder{}).
		Where("store_id = ? AND uploader_user_id = ? AND status = ?", storeID, uploaderUserID, entity.UsedCarOrderRejected).
		Count(&stats.Rejected).Error; err != nil {
		return stats, err
	}
	return stats, nil
}

func (r *UsedCarOrderRepository) CountPendingByStore(ctx context.Context, storeID int) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&entity.WysUsedCarOrder{}).
		Where("store_id = ? AND status = ?", storeID, entity.UsedCarOrderPendingReview).
		Count(&n).Error
	return n, err
}

func (r *UsedCarOrderRepository) ListOrders(
	ctx context.Context, storeID int, uploaderUserID string, f repository.UsedCarOrderListFilter, offset, limit int,
) ([]entity.WysUsedCarOrder, int64, error) {
	q := r.db.WithContext(ctx).Model(&entity.WysUsedCarOrder{}).
		Where("store_id = ? AND uploader_user_id = ?", storeID, uploaderUserID)
	if len(f.Statuses) > 0 {
		q = q.Where("status IN ?", f.Statuses)
	}
	if f.FilterKind {
		q = q.Where("kind = ?", f.Kind)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []entity.WysUsedCarOrder
	err := q.Order("submitted_at DESC, order_id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

func (r *UsedCarOrderRepository) GetOrder(ctx context.Context, orderID int64) (*entity.WysUsedCarOrder, error) {
	var row entity.WysUsedCarOrder
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, repository.ErrUsedCarOrderNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *UsedCarOrderRepository) CreateOrder(ctx context.Context, row *entity.WysUsedCarOrder) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *UsedCarOrderRepository) GetCustomer(ctx context.Context, customerID int64) (*entity.WysStoreCustomer, error) {
	var row entity.WysStoreCustomer
	err := r.db.WithContext(ctx).Where("customer_id = ?", customerID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, repository.ErrUsedCarOrderBadCustomer
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *UsedCarOrderRepository) ListCustomers(
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

func (r *UsedCarOrderRepository) GetUserBrief(ctx context.Context, userID string) (string, string, error) {
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

func EnsureUsedCarOrderSchema(db *gorm.DB) error {
	if err := db.AutoMigrate(&entity.WysUsedCarOrder{}); err != nil {
		return fmt.Errorf("used car order migrate: %w", err)
	}
	return nil
}

func EnsureUsedCarOrderSeed(db *gorm.DB) error {
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

		var custN int64
		if err := tx.Model(&entity.WysStoreCustomer{}).Where("store_id = ?", storeID).Count(&custN).Error; err != nil {
			return err
		}
		if custN == 0 {
			for i, p := range []string{"13812345678", "13612345678", "13987654321"} {
				names := []string{"小张女士", "王先生", "李女士"}
				row := entity.WysStoreCustomer{StoreID: storeID, DisplayName: names[i], Phone: p}
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
			}
		}

		var orderN int64
		if err := tx.Model(&entity.WysUsedCarOrder{}).
			Where("store_id = ? AND uploader_user_id = ?", storeID, user.UserID).
			Count(&orderN).Error; err != nil {
			return err
		}
		if orderN > 0 {
			return nil
		}

		var customers []entity.WysStoreCustomer
		if err := tx.Where("store_id = ?", storeID).Order("customer_id ASC").Limit(3).Find(&customers).Error; err != nil {
			return err
		}
		if len(customers) == 0 {
			return nil
		}
		cust := customers[0]
		now := time.Now()
		reject := "车况描述与实车不符"
		stars := int16(4)
		seeds := []entity.WysUsedCarOrder{
			{
				StoreID: storeID, UploaderUserID: user.UserID, Kind: entity.UsedCarKindConsign,
				CustomerID: cust.CustomerID, CustomerPhone: cust.Phone, CustomerName: cust.DisplayName,
				VehicleModel: "宝马 5系 530Li", PlateNo: "京A12345", VIN: "WBAXXXXX001",
				MileageKm: 32000, ModelYear: 2021, Amount: 268000,
				Status: entity.UsedCarOrderRated, RatingStars: &stars, SubmittedAt: now.Add(-96 * time.Hour),
			},
			{
				StoreID: storeID, UploaderUserID: user.UserID, Kind: entity.UsedCarKindTradeIn,
				CustomerID: cust.CustomerID, CustomerPhone: cust.Phone, CustomerName: cust.DisplayName,
				VehicleModel: "丰田 凯美瑞", PlateNo: "京B67890", VIN: "JTNXXXXX002",
				MileageKm: 48000, ModelYear: 2019, Amount: 35000,
				Status: entity.UsedCarOrderApprovedPendingRating, SubmittedAt: now.Add(-72 * time.Hour),
			},
			{
				StoreID: storeID, UploaderUserID: user.UserID, Kind: entity.UsedCarKindPurchase,
				CustomerID: cust.CustomerID, CustomerPhone: cust.Phone, CustomerName: cust.DisplayName,
				VehicleModel: "大众 帕萨特", PlateNo: "京C11111", VIN: "LSVXXXXX003",
				MileageKm: 61000, ModelYear: 2018, Amount: 98000,
				Status: entity.UsedCarOrderPendingReview, SubmittedAt: now.Add(-48 * time.Hour),
			},
			{
				StoreID: storeID, UploaderUserID: user.UserID, Kind: entity.UsedCarKindTradeIn,
				CustomerID: cust.CustomerID, CustomerPhone: cust.Phone, CustomerName: cust.DisplayName,
				VehicleModel: "本田 雅阁", PlateNo: "京D22222", VIN: "LHGXXXXX004",
				MileageKm: 27000, ModelYear: 2020, Amount: 22000,
				Status: entity.UsedCarOrderPendingReview, SubmittedAt: now.Add(-24 * time.Hour),
			},
			{
				StoreID: storeID, UploaderUserID: user.UserID, Kind: entity.UsedCarKindConsign,
				CustomerID: cust.CustomerID, CustomerPhone: cust.Phone, CustomerName: cust.DisplayName,
				VehicleModel: "奥迪 A6L", PlateNo: "京E33333", VIN: "LFVXXXXX005",
				MileageKm: 55000, ModelYear: 2017, Amount: 188000,
				Status: entity.UsedCarOrderRejected, RejectReason: &reject, SubmittedAt: now.Add(-12 * time.Hour),
			},
		}
		for i := range seeds {
			seeds[i].CreatedAt = seeds[i].SubmittedAt
			seeds[i].UpdatedAt = seeds[i].SubmittedAt
			if err := tx.Create(&seeds[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
