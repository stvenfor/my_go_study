package repository

import (
	"context"
	"errors"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

var (
	ErrMallNotFound         = errors.New("商城资源不存在")
	ErrMallForbidden        = errors.New("无权操作该门店商城")
	ErrMallInvalidKind      = errors.New("商品种类无效")
	ErrMallKindMismatch     = errors.New("SKU 种类与商品不一致")
	ErrMallInvalidPrice     = errors.New("价格无效")
	ErrMallInvalidQty       = errors.New("数量无效")
	ErrMallInvalidChannel   = errors.New("支付渠道无效")
	ErrMallNeedAddress      = errors.New("实体商品订单需要收货人信息")
	ErrMallMultiStore       = errors.New("一笔订单只能包含同一门店商品")
	ErrMallStockInsufficient = errors.New("库存不足")
	ErrMallCodeInsufficient = errors.New("兑换码不足")
	ErrMallOrderNotPayable  = errors.New("订单不可支付")
	ErrMallOrderNotCancelable = errors.New("订单不可取消")
	ErrMallSKUOffShelf      = errors.New("SKU 未上架")
	ErrMallEmptyCart        = errors.New("购物车为空或无可结算行")
)

// MallRepository 商城持久化。
type MallRepository interface {
	CreateProduct(ctx context.Context, p *entity.WysMallProduct) error
	CreateSKU(ctx context.Context, sku *entity.WysMallSKU) error
	AddVirtualCodes(ctx context.Context, skuID int64, codes []string) error
	GetProduct(ctx context.Context, productID int64) (*entity.WysMallProduct, error)
	ListOnShelfSKUs(ctx context.Context, productID int64) ([]entity.WysMallSKU, error)
	GetSKU(ctx context.Context, skuID int64) (*entity.WysMallSKU, error)
	ListOnShelfProducts(ctx context.Context, storeID int) ([]entity.MallProductWithSKUs, error)
	ListShelfItems(ctx context.Context, storeID, offset, limit int) ([]entity.MallShelfItem, int64, error)

	UpsertCartItem(ctx context.Context, item *entity.WysMallCartItem) error
	ListCart(ctx context.Context, userID string) ([]entity.WysMallCartItem, error)
	ClearCartSKUs(ctx context.Context, userID string, skuIDs []int64) error

	FindOrderByIdempotency(ctx context.Context, buyerUserID, key string) (*entity.WysMallOrder, error)
	CreateOrder(ctx context.Context, order *entity.WysMallOrder, items []entity.WysMallOrderItem) error
	GetOrder(ctx context.Context, orderID int64) (*entity.WysMallOrder, error)
	ListOrderItems(ctx context.Context, orderID int64) ([]entity.WysMallOrderItem, error)
	ListPayments(ctx context.Context, orderID int64) ([]entity.WysMallPayment, error)

	// PayOrderLocal 本地模拟支付：任意渠道成功落库并履约。已支付则幂等返回。
	PayOrderLocal(ctx context.Context, orderID int64, buyerUserID string, channel int16) (*entity.MallOrderDetail, error)
	CancelUnpaidOrder(ctx context.Context, orderID int64, buyerUserID string) (*entity.WysMallOrder, error)
}
