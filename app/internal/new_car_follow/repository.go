// repository.go 新车跟进档案 —— 域内仓储契约与领域错误。
//
// 由 model/repository/new_car_follow_repository.go 演化而来：接口随域下沉，
// 错误哨兵一并搬入并去掉 NewCarFollow 前缀。
package new_car_follow

import (
	"context"
	"errors"
	"time"

	"model/entity"
)

var (
	ErrNotFound    = errors.New("跟进档案不存在")
	ErrNoStore     = errors.New("未选择当前门店")
	ErrBadCustomer = errors.New("客户无效")
	ErrBadLevel    = errors.New("跟进级别无效")
	ErrBadFilter   = errors.New("筛选参数无效")
	ErrBadStage    = errors.New("档案阶段无效")
	ErrDuplicate   = errors.New("该客户已有未关闭跟进档案")
	ErrBadLogBody  = errors.New("跟进内容不能为空")
)

// Repository 跟进档案 + 客户读写。
// CountStats / ListFiles：ownerUserID 为空表示当前店全员（店管口径）。
type Repository interface {
	CountStats(ctx context.Context, storeID int, ownerUserID string, now time.Time) (entity.NewCarFollowStats, error)
	ListFiles(ctx context.Context, storeID int, ownerUserID string, f entity.NewCarFollowListFilter, now time.Time, offset, limit int) ([]entity.WysNewCarFollowFile, int64, error)
	GetFile(ctx context.Context, fileID int64) (*entity.WysNewCarFollowFile, error)
	CreateFile(ctx context.Context, row *entity.WysNewCarFollowFile) error
	// UpdateFileAndCustomerFollow 事务更新档案，并可选回写客户 next_follow_up_at。
	UpdateFileAndCustomerFollow(ctx context.Context, row *entity.WysNewCarFollowFile, syncCustomerFollow bool) error
	FindOpenFileByCustomer(ctx context.Context, storeID int, customerID int64) (*entity.WysNewCarFollowFile, error)

	ListLogs(ctx context.Context, fileID int64, offset, limit int) ([]entity.WysNewCarFollowLog, int64, error)
	// CreateLogAndTouchFile 插流水并更新档案（last_follow / 可选级别与下次跟进）；syncCustomerFollow 时回写客户。
	CreateLogAndTouchFile(ctx context.Context, log *entity.WysNewCarFollowLog, file *entity.WysNewCarFollowFile, syncCustomerFollow bool) error

	GetCustomer(ctx context.Context, customerID int64) (*entity.WysStoreCustomer, error)
	CreateCustomer(ctx context.Context, row *entity.WysStoreCustomer) error
	ListCustomers(ctx context.Context, storeID int, q string, offset, limit int) ([]entity.WysStoreCustomer, int64, error)

	GetUserBrief(ctx context.Context, userID string) (userName, avatarURL string, err error)
}
