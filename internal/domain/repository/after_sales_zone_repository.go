package repository

import (
	"context"
	"errors"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

var (
	ErrAfterSalesRecordNotFound      = errors.New("维修保养记录不存在")
	ErrAfterSalesNoStore             = errors.New("未选择当前门店")
	ErrAfterSalesNotMember           = errors.New("非当前店成员")
	ErrAfterSalesBadKind             = errors.New("服务类型无效")
	ErrAfterSalesBadCustomer         = errors.New("客户信息无效")
	ErrAfterSalesBadAppointment      = errors.New("预约无效")
	ErrAfterSalesAppointmentTaken    = errors.New("该预约已有记录")
	ErrAfterSalesForbidden           = errors.New("无权操作")
)

// AfterSalesZoneRepository 售后专区仓储。
type AfterSalesZoneRepository interface {
	ListByStore(ctx context.Context, storeID int, offset, limit int) ([]entity.WysAfterSalesRecord, int64, error)
	ListByCustomerUser(ctx context.Context, customerUserID string, offset, limit int) ([]entity.WysAfterSalesRecord, int64, error)
	GetByID(ctx context.Context, recordID int64) (*entity.WysAfterSalesRecord, error)
	GetStoreCustomer(ctx context.Context, storeID int, customerID int64) (*entity.WysStoreCustomer, error)
	GetAppointment(ctx context.Context, appointmentID int64) (*entity.WysAfterSalesAppointment, error)
	ListPendingAppointments(ctx context.Context, storeID int, today time.Time) ([]entity.WysAfterSalesAppointment, error)
	// CreateRecord 插入记录；若 appointmentID 非空则同事务将预约标为 done。
	CreateRecord(ctx context.Context, row *entity.WysAfterSalesRecord) error
}
