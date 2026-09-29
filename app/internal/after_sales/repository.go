// repository.go 售后专区 —— 域内仓储契约与领域错误。
//
// 由 model/repository/after_sales_zone_repository.go 演化而来：接口随域下沉，
// 错误哨兵一并搬入并去掉 AfterSales 前缀。
package after_sales

import (
	"context"
	"errors"
	"time"

	"model/entity"
)

var (
	ErrRecordNotFound   = errors.New("维修保养记录不存在")
	ErrNoStore          = errors.New("未选择当前门店")
	ErrNotMember        = errors.New("非当前店成员")
	ErrBadKind          = errors.New("服务类型无效")
	ErrBadCustomer      = errors.New("客户信息无效")
	ErrBadAppointment   = errors.New("预约无效")
	ErrAppointmentTaken = errors.New("该预约已有记录")
	ErrForbidden        = errors.New("无权操作")
	// 以下两条原先直接定义在 usecase 层（非 repository 别名），随域一并下沉。
	ErrBadTitle = errors.New("标题不能为空")
	ErrBadDate  = errors.New("服务日期无效")
)

// Repository 售后专区仓储。
type Repository interface {
	ListByStore(ctx context.Context, storeID int, offset, limit int) ([]entity.WysAfterSalesRecord, int64, error)
	ListByCustomerUser(ctx context.Context, customerUserID string, offset, limit int) ([]entity.WysAfterSalesRecord, int64, error)
	GetByID(ctx context.Context, recordID int64) (*entity.WysAfterSalesRecord, error)
	GetStoreCustomer(ctx context.Context, storeID int, customerID int64) (*entity.WysStoreCustomer, error)
	GetAppointment(ctx context.Context, appointmentID int64) (*entity.WysAfterSalesAppointment, error)
	ListPendingAppointments(ctx context.Context, storeID int, today time.Time) ([]entity.WysAfterSalesAppointment, error)
	// CreateRecord 插入记录；若 appointmentID 非空则同事务将预约标为 done。
	CreateRecord(ctx context.Context, row *entity.WysAfterSalesRecord) error
}
