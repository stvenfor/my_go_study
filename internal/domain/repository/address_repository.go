package repository

import (
	"context"
	"errors"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

var (
	ErrAddressNotFound   = errors.New("地址不存在")
	ErrAddressInvalid    = errors.New("地址参数无效")
	ErrAddressLimit      = errors.New("地址数量已达上限")
)

// AddressRepository 用户收货地址簿。
type AddressRepository interface {
	ListByUser(ctx context.Context, userID string) ([]entity.WysUserAddress, error)
	GetByID(ctx context.Context, userID string, addressID int64) (*entity.WysUserAddress, error)
	CountByUser(ctx context.Context, userID string) (int64, error)
	Create(ctx context.Context, a *entity.WysUserAddress) error
	Update(ctx context.Context, a *entity.WysUserAddress) error
	SoftDelete(ctx context.Context, userID string, addressID int64) error
	ClearDefault(ctx context.Context, userID string) error
	SetDefault(ctx context.Context, userID string, addressID int64) error
	PromoteNewestDefault(ctx context.Context, userID string) error
}
