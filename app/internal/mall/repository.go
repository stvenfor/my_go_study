package mall

import (
	"context"
	"errors"

	"model/entity"
)

var (
	ErrNotFound           = errors.New("商城资源不存在")
	ErrForbidden          = errors.New("无权操作该门店商城")
	ErrInvalidKind        = errors.New("商品种类无效")
	ErrKindMismatch       = errors.New("SKU 种类与商品不一致")
	ErrInvalidPrice       = errors.New("价格无效")
	ErrInvalidQty         = errors.New("数量无效")
	ErrInvalidChannel     = errors.New("支付渠道无效")
	ErrNeedAddress        = errors.New("实体商品订单需要收货人信息")
	ErrMultiStore         = errors.New("一笔订单只能包含同一门店商品")
	ErrStockInsufficient  = errors.New("库存不足")
	ErrCodeInsufficient   = errors.New("兑换码不足")
	ErrOrderNotPayable    = errors.New("订单不可支付")
	ErrOrderNotCancelable = errors.New("订单不可取消")
	ErrSKUOffShelf        = errors.New("SKU 未上架")
	ErrEmptyCart          = errors.New("购物车为空或无可结算行")
	// 以下 6 条原先由 usecase 层 errors.New 定义，随域一并下沉。
	ErrInvalidIdempotency = errors.New("缺少 idempotency_key")
	ErrInvalidTitle       = errors.New("标题不能为空")
	ErrInvalidStatus      = errors.New("订单状态无效")
	ErrOrderPayExpired    = errors.New("订单支付已超时")
	ErrPaymentModeMixed   = errors.New("一笔订单内商品支付方式必须一致")
	ErrNeedCNYChannel     = errors.New("混合或人民币订单需使用人民币支付渠道")
)

// Repository 商城持久化。
type Repository interface {
	CreateProduct(ctx context.Context, p *entity.WysMallProduct) error
	CreateSKU(ctx context.Context, sku *entity.WysMallSKU) error
	AddVirtualCodes(ctx context.Context, skuID int64, codes []string) error
	GetProduct(ctx context.Context, productID int64) (*entity.WysMallProduct, error)
	ListOnShelfSKUs(ctx context.Context, productID int64) ([]entity.WysMallSKU, error)
	GetSKU(ctx context.Context, skuID int64) (*entity.WysMallSKU, error)
	ListShelfItems(ctx context.Context, storeID, offset, limit int) ([]entity.MallShelfItem, int64, error)

	UpsertCartItem(ctx context.Context, item *entity.WysMallCartItem) error
	ListCart(ctx context.Context, userID string) ([]entity.WysMallCartItem, error)
	ClearCartSKUs(ctx context.Context, userID string, skuIDs []int64) error

	FindOrderByIdempotency(ctx context.Context, buyerUserID, key string) (*entity.WysMallOrder, error)
	CreateOrder(ctx context.Context, order *entity.WysMallOrder, items []entity.WysMallOrderItem) error
	GetOrder(ctx context.Context, orderID int64) (*entity.WysMallOrder, error)
	ListOrdersByBuyer(ctx context.Context, buyerUserID string, statuses []int16, offset, limit int) ([]entity.WysMallOrder, int64, error)
	ListOrderItems(ctx context.Context, orderID int64) ([]entity.WysMallOrderItem, error)
	ListOrderItemsByOrderIDs(ctx context.Context, orderIDs []int64) ([]entity.WysMallOrderItem, error)
	ListPayments(ctx context.Context, orderID int64) ([]entity.WysMallPayment, error)

	// PayOrderLocal 本地模拟支付：任意渠道成功落库并履约。已支付则幂等返回。
	PayOrderLocal(ctx context.Context, orderID int64, buyerUserID string, channel int16) (*entity.MallOrderDetail, error)
	CancelUnpaidOrder(ctx context.Context, orderID int64, buyerUserID string) (*entity.WysMallOrder, error)
	// CancelPaidOrder 取消已支付未履约订单（库存/兑换码不回滚；退款由 usecase 入账钱包）。
	CancelPaidOrder(ctx context.Context, orderID int64, buyerUserID string) (*entity.WysMallOrder, error)
}
