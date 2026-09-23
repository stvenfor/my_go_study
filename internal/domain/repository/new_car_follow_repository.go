// new_car_follow_repository.go 新车跟进档案仓储。
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

var (
	ErrNewCarFollowNotFound    = errors.New("跟进档案不存在")
	ErrNewCarFollowNoStore     = errors.New("未选择当前门店")
	ErrNewCarFollowBadCustomer = errors.New("客户无效")
	ErrNewCarFollowBadLevel    = errors.New("跟进级别无效")
	ErrNewCarFollowBadFilter   = errors.New("筛选参数无效")
	ErrNewCarFollowBadStage    = errors.New("档案阶段无效")
	ErrNewCarFollowDuplicate   = errors.New("该客户已有未关闭跟进档案")
)

// NewCarFollowRepository 跟进档案 + 客户读写。
type NewCarFollowRepository interface {
	CountStats(ctx context.Context, storeID int, ownerUserID string, now time.Time) (entity.NewCarFollowStats, error)
	ListFiles(ctx context.Context, storeID int, ownerUserID string, f entity.NewCarFollowListFilter, now time.Time, offset, limit int) ([]entity.WysNewCarFollowFile, int64, error)
	GetFile(ctx context.Context, fileID int64) (*entity.WysNewCarFollowFile, error)
	CreateFile(ctx context.Context, row *entity.WysNewCarFollowFile) error
	// UpdateFileAndCustomerFollow 事务更新档案，并可选回写客户 next_follow_up_at。
	UpdateFileAndCustomerFollow(ctx context.Context, row *entity.WysNewCarFollowFile, syncCustomerFollow bool) error
	FindOpenFileByCustomer(ctx context.Context, storeID int, customerID int64) (*entity.WysNewCarFollowFile, error)

	GetCustomer(ctx context.Context, customerID int64) (*entity.WysStoreCustomer, error)
	CreateCustomer(ctx context.Context, row *entity.WysStoreCustomer) error
	ListCustomers(ctx context.Context, storeID int, q string, offset, limit int) ([]entity.WysStoreCustomer, int64, error)

	GetUserBrief(ctx context.Context, userID string) (userName, avatarURL string, err error)
}
