// repository.go 二手车业务单 —— 域内仓储契约与领域错误。
//
// 由 model/repository/used_car_order_repository.go 演化而来：接口随域下沉，
// 错误哨兵一并搬入（原先 usecase 包里的 ErrUsedCarOrderXxx 别名层同时去掉）。
package used_car

import (
	"context"
	"errors"

	"model/entity"
)

var (
	ErrOrderNotFound = errors.New("二手车业务单不存在")
	ErrNoStore       = errors.New("未选择当前门店")
	ErrBadCustomer   = errors.New("客户无效")
	ErrBadFilter     = errors.New("筛选参数无效")
	ErrBadInput      = errors.New("业务单参数无效")
)

// ListFilter 列表筛选。
type ListFilter struct {
	Statuses   []int16
	Kind       int16
	FilterKind bool
}

// Repository 二手车业务单 + 选客户读。
type Repository interface {
	CountStats(ctx context.Context, storeID int, uploaderUserID string) (entity.UsedCarOrderStats, error)
	CountPendingByStore(ctx context.Context, storeID int) (int64, error)
	ListOrders(ctx context.Context, storeID int, uploaderUserID string, f ListFilter, offset, limit int) ([]entity.WysUsedCarOrder, int64, error)
	GetOrder(ctx context.Context, orderID int64) (*entity.WysUsedCarOrder, error)
	CreateOrder(ctx context.Context, row *entity.WysUsedCarOrder) error

	GetCustomer(ctx context.Context, customerID int64) (*entity.WysStoreCustomer, error)
	ListCustomers(ctx context.Context, storeID int, q string, offset, limit int) ([]entity.WysStoreCustomer, int64, error)

	GetUserBrief(ctx context.Context, userID string) (userName, avatarURL string, err error)
}
