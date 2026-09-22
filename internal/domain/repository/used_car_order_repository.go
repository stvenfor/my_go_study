// used_car_order_repository.go 二手车业务单仓储。
package repository

import (
	"context"
	"errors"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

var (
	ErrUsedCarOrderNotFound    = errors.New("二手车业务单不存在")
	ErrUsedCarOrderNoStore     = errors.New("未选择当前门店")
	ErrUsedCarOrderBadCustomer = errors.New("客户无效")
	ErrUsedCarOrderBadFilter   = errors.New("筛选参数无效")
	ErrUsedCarOrderBadInput    = errors.New("业务单参数无效")
)

// UsedCarOrderListFilter 列表筛选。
type UsedCarOrderListFilter struct {
	Statuses   []int16
	Kind       int16
	FilterKind bool
}

// UsedCarOrderRepository 二手车业务单 + 选客户读。
type UsedCarOrderRepository interface {
	CountStats(ctx context.Context, storeID int, uploaderUserID string) (entity.UsedCarOrderStats, error)
	CountPendingByStore(ctx context.Context, storeID int) (int64, error)
	ListOrders(ctx context.Context, storeID int, uploaderUserID string, f UsedCarOrderListFilter, offset, limit int) ([]entity.WysUsedCarOrder, int64, error)
	GetOrder(ctx context.Context, orderID int64) (*entity.WysUsedCarOrder, error)
	CreateOrder(ctx context.Context, row *entity.WysUsedCarOrder) error

	GetCustomer(ctx context.Context, customerID int64) (*entity.WysStoreCustomer, error)
	ListCustomers(ctx context.Context, storeID int, q string, offset, limit int) ([]entity.WysStoreCustomer, int64, error)

	GetUserBrief(ctx context.Context, userID string) (userName, avatarURL string, err error)
}
