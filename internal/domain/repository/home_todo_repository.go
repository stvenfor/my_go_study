package repository

import (
	"context"
	"errors"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

var (
	ErrJoinApplicationNotFound     = errors.New("入店申请不存在")
	ErrJoinApplicationInvalid      = errors.New("入店申请无效")
	ErrJoinApplicationNotPending   = errors.New("入店申请不是待审状态")
	ErrJoinApplicationDuplicate    = errors.New("已有待审入店申请")
	ErrJoinAlreadyMember           = errors.New("已是门店成员")
)

// HomeTodoRepository 首页待办四域持久化。
type HomeTodoRepository interface {
	// 入店申请
	CreateJoinApplication(ctx context.Context, app *entity.WysStoreJoinApplication) error
	GetJoinApplication(ctx context.Context, applicationID int64) (*entity.WysStoreJoinApplication, error)
	ListPendingJoinApplications(ctx context.Context, storeID int) ([]entity.WysStoreJoinApplication, error)
	CountPendingJoinApplications(ctx context.Context, storeID int) (int64, error)
	UpdateJoinApplicationStatus(ctx context.Context, applicationID int64, status int16, reviewedBy string) error

	// 跟进客户
	ListOverdueCustomers(ctx context.Context, storeID int, now time.Time) ([]entity.WysStoreCustomer, error)
	CountOverdueCustomers(ctx context.Context, storeID int, now time.Time) (int64, error)

	// 售后预约
	ListPendingAppointments(ctx context.Context, storeID int, today time.Time) ([]entity.WysAfterSalesAppointment, error)
	CountPendingAppointments(ctx context.Context, storeID int, today time.Time) (int64, error)

	// 店务审核单
	ListPendingReviewOrders(ctx context.Context, storeID int) ([]entity.WysStoreReviewOrder, error)
	CountPendingReviewOrders(ctx context.Context, storeID int) (int64, error)

	// 种子
	EnsureSeed(ctx context.Context, storeID int, applicantUserID string) error
}
