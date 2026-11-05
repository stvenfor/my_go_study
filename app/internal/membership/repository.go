package membership

import (
	"context"
	"errors"
	"time"

	"model/entity"
)

var (
	ErrOrderNotFound = errors.New("会员订单不存在")
	ErrOrderPaid     = errors.New("会员订单已支付")
	ErrOrderOwner    = errors.New("会员订单不属于当前用户")
)

// Repository 会员权益与订单。
type Repository interface {
	ListEntitlements(ctx context.Context, userID string) ([]entity.WysMembershipEntitlement, error)
	GetEntitlement(ctx context.Context, userID, tier string) (*entity.WysMembershipEntitlement, error)
	// ExtendEntitlement 将 expires_at 设为 max(now, current)+months；返回更新后行。
	ExtendEntitlement(ctx context.Context, userID, tier, channel string, months int, now time.Time, huaweiToken, huaweiSubID, appleOriginalTxID string) (*entity.WysMembershipEntitlement, error)
	CreateOrder(ctx context.Context, order *entity.WysMembershipOrder) error
	GetOrder(ctx context.Context, orderID int64) (*entity.WysMembershipOrder, error)
	MarkOrderPaid(ctx context.Context, orderID int64, paidAt time.Time) error
	FindPaidByOutTradeNo(ctx context.Context, outTradeNo string) (*entity.WysMembershipOrder, error)
	// Transaction 把 fn 里的写落到同一事务（复用 P0-3 的 txdb）。
	// 余额购买必须「扣款 + 改单 + 发权益」同生共死：以前失败时靠 Credit 补偿，
	// 而补偿是 `_, _ =` 写的，补偿失败就等于钱凭空消失。
	Transaction(ctx context.Context, fn func(ctx context.Context) error) error
}
