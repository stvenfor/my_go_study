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

// DealInvoiceRepository 成交发票 Postgres 实现。
type DealInvoiceRepository struct {
	db *gorm.DB
}

func NewDealInvoiceRepository(db *gorm.DB) *DealInvoiceRepository {
	return &DealInvoiceRepository{db: db}
}

func (r *DealInvoiceRepository) CountStats(ctx context.Context, storeID int, uploaderUserID string) (entity.DealInvoiceStats, error) {
	var stats entity.DealInvoiceStats
	base := r.db.WithContext(ctx).Model(&entity.WysDealInvoice{}).
		Where("store_id = ? AND uploader_user_id = ?", storeID, uploaderUserID)
	if err := base.Count(&stats.Uploaded).Error; err != nil {
		return stats, err
	}
	if err := r.db.WithContext(ctx).Model(&entity.WysDealInvoice{}).
		Where("store_id = ? AND uploader_user_id = ? AND status = ?", storeID, uploaderUserID, entity.DealInvoicePendingReview).
		Count(&stats.PendingReview).Error; err != nil {
		return stats, err
	}
	if err := r.db.WithContext(ctx).Model(&entity.WysDealInvoice{}).
		Where("store_id = ? AND uploader_user_id = ? AND status IN ?", storeID, uploaderUserID,
			[]int16{entity.DealInvoiceApprovedPendingRating, entity.DealInvoiceRated}).
		Count(&stats.Approved).Error; err != nil {
		return stats, err
	}
	if err := r.db.WithContext(ctx).Model(&entity.WysDealInvoice{}).
		Where("store_id = ? AND uploader_user_id = ? AND status = ?", storeID, uploaderUserID, entity.DealInvoiceRejected).
		Count(&stats.Rejected).Error; err != nil {
		return stats, err
	}
	return stats, nil
}

func (r *DealInvoiceRepository) ListInvoices(
	ctx context.Context, storeID int, uploaderUserID string, statuses []int16, offset, limit int,
) ([]entity.WysDealInvoice, int64, error) {
	q := r.db.WithContext(ctx).Model(&entity.WysDealInvoice{}).
		Where("store_id = ? AND uploader_user_id = ?", storeID, uploaderUserID)
	if len(statuses) > 0 {
		q = q.Where("status IN ?", statuses)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []entity.WysDealInvoice
	err := q.Order("submitted_at DESC, invoice_id DESC").Offset(offset).Limit(limit).Find(&rows).Error
	return rows, total, err
}

func (r *DealInvoiceRepository) GetInvoice(ctx context.Context, invoiceID int64) (*entity.WysDealInvoice, error) {
	var row entity.WysDealInvoice
	err := r.db.WithContext(ctx).Where("invoice_id = ?", invoiceID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, repository.ErrDealInvoiceNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *DealInvoiceRepository) CreateInvoice(ctx context.Context, row *entity.WysDealInvoice) error {
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *DealInvoiceRepository) UpdateInvoice(ctx context.Context, row *entity.WysDealInvoice) error {
	return r.db.WithContext(ctx).Save(row).Error
}

func (r *DealInvoiceRepository) GetCustomer(ctx context.Context, customerID int64) (*entity.WysStoreCustomer, error) {
	var row entity.WysStoreCustomer
	err := r.db.WithContext(ctx).Where("customer_id = ?", customerID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return nil, repository.ErrDealInvoiceBadCustomer
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *DealInvoiceRepository) ListCustomers(
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

func (r *DealInvoiceRepository) GetUserBrief(ctx context.Context, userID string) (string, string, error) {
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

// EnsureDealInvoiceSchema 确保列/表存在（AutoMigrate 之外的兜底）。
func EnsureDealInvoiceSchema(db *gorm.DB) error {
	if err := db.Exec(`
		ALTER TABLE wys_store_customer
		  ADD COLUMN IF NOT EXISTS phone varchar(32) NOT NULL DEFAULT ''
	`).Error; err != nil {
		return fmt.Errorf("deal invoice phone column: %w", err)
	}
	if err := db.AutoMigrate(&entity.WysDealInvoice{}); err != nil {
		return fmt.Errorf("deal invoice migrate: %w", err)
	}
	return nil
}

// EnsureDealInvoiceSeed 为测试号 13400000000 种四态发票 + 带手机号客户。
func EnsureDealInvoiceSeed(db *gorm.DB) error {
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

		// 确保至少 3 个带手机号客户
		var custN int64
		if err := tx.Model(&entity.WysStoreCustomer{}).Where("store_id = ?", storeID).Count(&custN).Error; err != nil {
			return err
		}
		phones := []string{"13812345678", "13612345678", "13987654321"}
		names := []string{"小张女士", "王先生", "李女士"}
		if custN == 0 {
			for i := range phones {
				row := entity.WysStoreCustomer{
					StoreID:     storeID,
					DisplayName: names[i],
					Phone:       phones[i],
				}
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
			}
		} else {
			var customers []entity.WysStoreCustomer
			if err := tx.Where("store_id = ?", storeID).Order("customer_id ASC").Find(&customers).Error; err != nil {
				return err
			}
			for i, c := range customers {
				if strings.TrimSpace(c.Phone) != "" {
					continue
				}
				p := phones[i%len(phones)]
				n := names[i%len(names)]
				if strings.TrimSpace(c.DisplayName) == "" {
					c.DisplayName = n
				}
				c.Phone = p
				if err := tx.Save(&c).Error; err != nil {
					return err
				}
			}
		}

		var invN int64
		if err := tx.Model(&entity.WysDealInvoice{}).
			Where("store_id = ? AND uploader_user_id = ?", storeID, user.UserID).
			Count(&invN).Error; err != nil {
			return err
		}
		if invN > 0 {
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
		reject := "发票信息不清晰，请重新上传"
		stars := int16(5)
		seeds := []entity.WysDealInvoice{
			{
				StoreID: storeID, UploaderUserID: user.UserID,
				CustomerID: cust.CustomerID, CustomerPhone: cust.Phone, CustomerName: cust.DisplayName,
				Status: entity.DealInvoiceRated, RatingStars: &stars, SubmittedAt: now.Add(-96 * time.Hour),
			},
			{
				StoreID: storeID, UploaderUserID: user.UserID,
				CustomerID: cust.CustomerID, CustomerPhone: cust.Phone, CustomerName: cust.DisplayName,
				Status: entity.DealInvoiceApprovedPendingRating, SubmittedAt: now.Add(-72 * time.Hour),
			},
			{
				StoreID: storeID, UploaderUserID: user.UserID,
				CustomerID: cust.CustomerID, CustomerPhone: cust.Phone, CustomerName: cust.DisplayName,
				Status: entity.DealInvoicePendingReview, SubmittedAt: now.Add(-48 * time.Hour),
			},
			{
				StoreID: storeID, UploaderUserID: user.UserID,
				CustomerID: cust.CustomerID, CustomerPhone: cust.Phone, CustomerName: cust.DisplayName,
				Status: entity.DealInvoicePendingReview, SubmittedAt: now.Add(-24 * time.Hour),
			},
			{
				StoreID: storeID, UploaderUserID: user.UserID,
				CustomerID: cust.CustomerID, CustomerPhone: cust.Phone, CustomerName: cust.DisplayName,
				Status: entity.DealInvoiceRejected, RejectReason: &reject, SubmittedAt: now.Add(-12 * time.Hour),
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
