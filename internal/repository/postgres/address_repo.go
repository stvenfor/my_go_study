package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"gorm.io/gorm"
)

// AddressRepository 用户收货地址。
type AddressRepository struct {
	db *gorm.DB
}

// NewAddressRepository 创建。
func NewAddressRepository(db *gorm.DB) *AddressRepository {
	return &AddressRepository{db: db}
}

func (r *AddressRepository) ListByUser(ctx context.Context, userID string) ([]entity.WysUserAddress, error) {
	var rows []entity.WysUserAddress
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("is_default DESC, updated_at DESC, address_id DESC").
		Find(&rows).Error
	return rows, err
}

func (r *AddressRepository) GetByID(ctx context.Context, userID string, addressID int64) (*entity.WysUserAddress, error) {
	var a entity.WysUserAddress
	err := r.db.WithContext(ctx).
		Where("address_id = ? AND user_id = ? AND deleted_at IS NULL", addressID, userID).
		First(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, repository.ErrAddressNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AddressRepository) CountByUser(ctx context.Context, userID string) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).
		Model(&entity.WysUserAddress{}).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Count(&n).Error
	return n, err
}

func (r *AddressRepository) Create(ctx context.Context, a *entity.WysUserAddress) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if a.IsDefault {
			if err := tx.Model(&entity.WysUserAddress{}).
				Where("user_id = ? AND deleted_at IS NULL AND is_default = true", a.UserID).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(a).Error
	})
}

func (r *AddressRepository) Update(ctx context.Context, a *entity.WysUserAddress) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if a.IsDefault {
			if err := tx.Model(&entity.WysUserAddress{}).
				Where("user_id = ? AND deleted_at IS NULL AND address_id <> ? AND is_default = true", a.UserID, a.AddressID).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		res := tx.Model(&entity.WysUserAddress{}).
			Where("address_id = ? AND user_id = ? AND deleted_at IS NULL", a.AddressID, a.UserID).
			Updates(map[string]interface{}{
				"receiver_name":  a.ReceiverName,
				"receiver_phone": a.ReceiverPhone,
				"province":       a.Province,
				"city":           a.City,
				"district":       a.District,
				"detail_address": a.DetailAddress,
				"postal_code":    a.PostalCode,
				"is_default":     a.IsDefault,
				"label":          a.Label,
				"updated_at":     time.Now(),
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return repository.ErrAddressNotFound
		}
		return nil
	})
}

func (r *AddressRepository) SoftDelete(ctx context.Context, userID string, addressID int64) error {
	now := time.Now()
	res := r.db.WithContext(ctx).
		Model(&entity.WysUserAddress{}).
		Where("address_id = ? AND user_id = ? AND deleted_at IS NULL", addressID, userID).
		Updates(map[string]interface{}{
			"deleted_at": now,
			"is_default": false,
			"updated_at": now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return repository.ErrAddressNotFound
	}
	return nil
}

func (r *AddressRepository) ClearDefault(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).
		Model(&entity.WysUserAddress{}).
		Where("user_id = ? AND deleted_at IS NULL AND is_default = true", userID).
		Update("is_default", false).Error
}

func (r *AddressRepository) SetDefault(ctx context.Context, userID string, addressID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var a entity.WysUserAddress
		if err := tx.Where("address_id = ? AND user_id = ? AND deleted_at IS NULL", addressID, userID).
			First(&a).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return repository.ErrAddressNotFound
			}
			return err
		}
		if err := tx.Model(&entity.WysUserAddress{}).
			Where("user_id = ? AND deleted_at IS NULL AND is_default = true", userID).
			Update("is_default", false).Error; err != nil {
			return err
		}
		return tx.Model(&entity.WysUserAddress{}).
			Where("address_id = ?", addressID).
			Updates(map[string]interface{}{
				"is_default": true,
				"updated_at": time.Now(),
			}).Error
	})
}

func (r *AddressRepository) PromoteNewestDefault(ctx context.Context, userID string) error {
	var a entity.WysUserAddress
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Order("updated_at DESC, address_id DESC").
		First(&a).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return r.SetDefault(ctx, userID, a.AddressID)
}
