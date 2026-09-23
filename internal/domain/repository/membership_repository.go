package repository

import (
	"context"
	"errors"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

var (
	ErrMembershipOrderNotFound = errors.New("会员订单不存在")
	ErrMembershipOrderPaid     = errors.New("会员订单已支付")
	ErrMembershipOrderOwner    = errors.New("会员订单不属于当前用户")
)

// MembershipRepository 会员权益与订单。
type MembershipRepository interface {
	ListEntitlements(ctx context.Context, userID string) ([]entity.WysMembershipEntitlement, error)
	GetEntitlement(ctx context.Context, userID, tier string) (*entity.WysMembershipEntitlement, error)
	// ExtendEntitlement 将 expires_at 设为 max(now, current)+months；返回更新后行。
	ExtendEntitlement(ctx context.Context, userID, tier, channel string, months int, now time.Time, huaweiToken, huaweiSubID, appleOriginalTxID string) (*entity.WysMembershipEntitlement, error)
	CreateOrder(ctx context.Context, order *entity.WysMembershipOrder) error
	GetOrder(ctx context.Context, orderID int64) (*entity.WysMembershipOrder, error)
	MarkOrderPaid(ctx context.Context, orderID int64, paidAt time.Time) error
	FindPaidByOutTradeNo(ctx context.Context, outTradeNo string) (*entity.WysMembershipOrder, error)
}
