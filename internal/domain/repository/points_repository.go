package repository

import (
	"context"
	"errors"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

var (
	ErrPointsAlreadyCheckedIn = errors.New("今日已签到")
	ErrPointsInsufficient     = errors.New("积分不足")
	ErrPointsTaskNotClaimable = errors.New("任务不可领取")
	ErrPointsTaskUnknown      = errors.New("未知任务")
)

// PointsRepository 积分账本与签到持久化。
type PointsRepository interface {
	GetBalance(ctx context.Context, userID string) (int64, error)
	// Credit 增加积分并写流水；返回新余额。
	Credit(ctx context.Context, userID string, delta int64, reason, refID string) (int64, error)
	// Debit 扣减积分并写流水；余额不足返回 ErrPointsInsufficient。
	Debit(ctx context.Context, userID string, delta int64, reason, refID string) (int64, error)

	GetCheckIn(ctx context.Context, userID, day string) (*entity.WysCheckIn, error)
	ListCheckIns(ctx context.Context, userID string, fromDay, toDay string) ([]entity.WysCheckIn, error)
	InsertCheckIn(ctx context.Context, row *entity.WysCheckIn) error

	GetTaskClaim(ctx context.Context, userID, day, taskCode string) (*entity.WysTaskClaim, error)
	InsertTaskClaim(ctx context.Context, row *entity.WysTaskClaim) error
	// ClaimTaskAtomic 写入领取记录并加积分，同事务。
	ClaimTaskAtomic(ctx context.Context, claim *entity.WysTaskClaim, reason string) (balance int64, err error)
	// CheckInAtomic 写入签到记录并加积分，同事务。
	CheckInAtomic(ctx context.Context, row *entity.WysCheckIn) (balance int64, err error)
}
