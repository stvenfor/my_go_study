// mall_usecase.go 门店商城：目录、购物车、下单、本地模拟支付。
package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/google/uuid"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

var (
	ErrMallNotFound           = repository.ErrMallNotFound
	ErrMallForbidden          = repository.ErrMallForbidden
	ErrMallInvalidKind        = repository.ErrMallInvalidKind
	ErrMallKindMismatch       = repository.ErrMallKindMismatch
	ErrMallInvalidPrice       = repository.ErrMallInvalidPrice
	ErrMallInvalidQty         = repository.ErrMallInvalidQty
	ErrMallInvalidChannel     = repository.ErrMallInvalidChannel
	ErrMallNeedAddress        = repository.ErrMallNeedAddress
	ErrMallMultiStore         = repository.ErrMallMultiStore
	ErrMallStockInsufficient  = repository.ErrMallStockInsufficient
	ErrMallCodeInsufficient   = repository.ErrMallCodeInsufficient
	ErrMallOrderNotPayable    = repository.ErrMallOrderNotPayable
	ErrMallOrderNotCancelable = repository.ErrMallOrderNotCancelable
	ErrMallSKUOffShelf        = repository.ErrMallSKUOffShelf
	ErrMallEmptyCart          = repository.ErrMallEmptyCart
	ErrMallInvalidIdempotency = errors.New("缺少 idempotency_key")
	ErrMallInvalidTitle       = errors.New("标题不能为空")
)

// MallUsecase 商城用例。
type MallUsecase struct {
	repo   repository.MallRepository
	access *AccessUsecase
}

// NewMallUsecase 创建。
func NewMallUsecase(repo repository.MallRepository, access *AccessUsecase) *MallUsecase {
	return &MallUsecase{repo: repo, access: access}
}

// CreateProduct 店员创建商品。
func (u *MallUsecase) CreateProduct(ctx context.Context, actorID string, storeID int, kind int16, title string, coverURL *string, status int16) (*entity.WysMallProduct, error) {
	if err := u.requireCatalogWrite(ctx, actorID, storeID); err != nil {
		return nil, err
	}
	if kind != entity.MallKindPhysical && kind != entity.MallKindVirtual {
		return nil, ErrMallInvalidKind
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrMallInvalidTitle
	}
	if status != entity.MallProductDraft && status != entity.MallProductOnShelf && status != entity.MallProductOffShelf {
		status = entity.MallProductDraft
	}
	p := &entity.WysMallProduct{
		StoreID:     storeID,
		Kind:        kind,
		Title:       title,
		CoverURL:    coverURL,
		CoverAspect: 1.0,
		Status:      status,
	}
	if err := u.repo.CreateProduct(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

// CreateSKUInput 创建 SKU 入参。
type CreateSKUInput struct {
	ProductID   int64
	SKUCode     string
	Title       string
	SpecsJSON   json.RawMessage
	Price       string
	StockQty    int
	Status      int16
	DeliverType *int16
	ContentURL  *string
}

// CreateSKU 店员创建 SKU。
func (u *MallUsecase) CreateSKU(ctx context.Context, actorID string, in CreateSKUInput) (*entity.WysMallSKU, error) {
	p, err := u.repo.GetProduct(ctx, in.ProductID)
	if err != nil {
		return nil, err
	}
	if err := u.requireCatalogWrite(ctx, actorID, p.StoreID); err != nil {
		return nil, err
	}
	price, err := normalizeMoney(in.Price)
	if err != nil {
		return nil, ErrMallInvalidPrice
	}
	code := strings.TrimSpace(in.SKUCode)
	if code == "" {
		return nil, errors.New("sku_code 不能为空")
	}
	specs := in.SpecsJSON
	if len(specs) == 0 {
		specs = json.RawMessage(`{}`)
	}
	sku := &entity.WysMallSKU{
		ProductID: p.ProductID,
		SKUCode:   code,
		Title:     strings.TrimSpace(in.Title),
		Specs:     specs,
		Price:     price,
		Status:    in.Status,
	}
	if sku.Status != entity.MallSKUOn && sku.Status != entity.MallSKUOff {
		sku.Status = entity.MallSKUOff
	}
	switch p.Kind {
	case entity.MallKindPhysical:
		if in.StockQty < 0 {
			return nil, ErrMallInvalidQty
		}
		sku.StockQty = in.StockQty
		sku.DeliverType = nil
		sku.ContentURL = nil
	case entity.MallKindVirtual:
		sku.StockQty = 0
		if in.DeliverType == nil {
			return nil, ErrMallInvalidKind
		}
		dt := *in.DeliverType
		if dt != entity.MallDeliverRedeemCode && dt != entity.MallDeliverContentURL {
			return nil, ErrMallInvalidKind
		}
		sku.DeliverType = &dt
		if dt == entity.MallDeliverContentURL {
			if in.ContentURL == nil || strings.TrimSpace(*in.ContentURL) == "" {
				return nil, errors.New("内容地址不能为空")
			}
			url := strings.TrimSpace(*in.ContentURL)
			sku.ContentURL = &url
		}
	default:
		return nil, ErrMallInvalidKind
	}
	if err := u.repo.CreateSKU(ctx, sku); err != nil {
		return nil, err
	}
	return sku, nil
}

// AddVirtualCodes 店员导入兑换码。
func (u *MallUsecase) AddVirtualCodes(ctx context.Context, actorID string, skuID int64, codes []string) error {
	sku, err := u.repo.GetSKU(ctx, skuID)
	if err != nil {
		return err
	}
	p, err := u.repo.GetProduct(ctx, sku.ProductID)
	if err != nil {
		return err
	}
	if p.Kind != entity.MallKindVirtual || sku.DeliverType == nil || *sku.DeliverType != entity.MallDeliverRedeemCode {
		return ErrMallInvalidKind
	}
	if err := u.requireCatalogWrite(ctx, actorID, p.StoreID); err != nil {
		return err
	}
	return u.repo.AddVirtualCodes(ctx, skuID, codes)
}

// ListOnShelfProducts 买家浏览在售（全量，兼容旧调用）。
func (u *MallUsecase) ListOnShelfProducts(ctx context.Context, storeID int) ([]entity.MallProductWithSKUs, error) {
	if storeID <= 0 {
		return nil, repository.ErrAccessInvalidStoreID
	}
	return u.repo.ListOnShelfProducts(ctx, storeID)
}

// GetShelfProduct 买家读一门店的在售商品及上架规格。下架、错店、已删都当不存在。
func (u *MallUsecase) GetShelfProduct(ctx context.Context, storeID int, productID int64) (*entity.MallProductDetail, error) {
	if storeID <= 0 {
		return nil, repository.ErrAccessInvalidStoreID
	}
	if productID <= 0 {
		return nil, ErrMallNotFound
	}
	p, err := u.repo.GetProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	if p.StoreID != storeID || p.Status != entity.MallProductOnShelf {
		return nil, ErrMallNotFound
	}
	skus, err := u.repo.ListOnShelfSKUs(ctx, productID)
	if err != nil {
		return nil, err
	}
	offers := make([]entity.MallSKUOffer, 0, len(skus))
	for _, s := range skus {
		specs := s.Specs
		if len(specs) == 0 {
			specs = json.RawMessage(`{}`)
		}
		offers = append(offers, entity.MallSKUOffer{
			SKUID:       s.SKUID,
			Title:       s.Title,
			Specs:       specs,
			Price:       s.Price,
			StockQty:    s.StockQty,
			DeliverType: s.DeliverType,
		})
	}
	return &entity.MallProductDetail{Product: *p, SKUs: offers}, nil
}

// ListShelfItems 买家分页浏览在售扁平行。page 从 1 起，默认每页 10。
func (u *MallUsecase) ListShelfItems(ctx context.Context, storeID, page, size int) ([]entity.MallShelfItem, int64, error) {
	if storeID <= 0 {
		return nil, 0, repository.ErrAccessInvalidStoreID
	}
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	offset := (page - 1) * size
	return u.repo.ListShelfItems(ctx, storeID, offset, size)
}

// UpsertCart 买家改购物车数量。
func (u *MallUsecase) UpsertCart(ctx context.Context, userID string, skuID int64, qty int) (*entity.WysMallCartItem, error) {
	if qty <= 0 {
		return nil, ErrMallInvalidQty
	}
	sku, err := u.repo.GetSKU(ctx, skuID)
	if err != nil {
		return nil, err
	}
	if sku.Status != entity.MallSKUOn {
		return nil, ErrMallSKUOffShelf
	}
	item := &entity.WysMallCartItem{UserID: userID, SKUID: skuID, Qty: qty}
	if err := u.repo.UpsertCartItem(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

// ListCart 买家购物车。
func (u *MallUsecase) ListCart(ctx context.Context, userID string) ([]entity.WysMallCartItem, error) {
	return u.repo.ListCart(ctx, userID)
}

// CreateOrderInput 下单入参。
type CreateOrderInput struct {
	IdempotencyKey  string
	StoreID         int
	Lines           []CreateOrderLine
	ReceiverName    string
	ReceiverPhone   string
	ReceiverAddress string
	ClearCart       bool
}

// CreateOrderLine 下单行。
type CreateOrderLine struct {
	SKUID int64
	Qty   int
}

// CreateOrder 创建待支付订单（不扣库存）。
func (u *MallUsecase) CreateOrder(ctx context.Context, buyerUserID string, in CreateOrderInput) (*entity.MallOrderDetail, error) {
	key := strings.TrimSpace(in.IdempotencyKey)
	if key == "" {
		return nil, ErrMallInvalidIdempotency
	}
	if existing, err := u.repo.FindOrderByIdempotency(ctx, buyerUserID, key); err != nil {
		return nil, err
	} else if existing != nil {
		return u.GetOrder(ctx, buyerUserID, existing.OrderID)
	}
	if len(in.Lines) == 0 {
		return nil, ErrMallEmptyCart
	}
	if in.StoreID <= 0 {
		return nil, repository.ErrAccessInvalidStoreID
	}

	var items []entity.WysMallOrderItem
	total := big.NewRat(0, 1)
	needAddress := false
	for _, line := range in.Lines {
		if line.Qty <= 0 {
			return nil, ErrMallInvalidQty
		}
		sku, err := u.repo.GetSKU(ctx, line.SKUID)
		if err != nil {
			return nil, err
		}
		if sku.Status != entity.MallSKUOn {
			return nil, ErrMallSKUOffShelf
		}
		p, err := u.repo.GetProduct(ctx, sku.ProductID)
		if err != nil {
			return nil, err
		}
		if p.StoreID != in.StoreID {
			return nil, ErrMallMultiStore
		}
		if p.Status != entity.MallProductOnShelf {
			return nil, ErrMallSKUOffShelf
		}
		if p.Kind == entity.MallKindPhysical {
			needAddress = true
			if sku.StockQty < line.Qty {
				return nil, ErrMallStockInsufficient
			}
		}
		unit, ok := new(big.Rat).SetString(sku.Price)
		if !ok {
			return nil, ErrMallInvalidPrice
		}
		lineAmt := new(big.Rat).Mul(unit, big.NewRat(int64(line.Qty), 1))
		total.Add(total, lineAmt)
		specs := sku.Specs
		if len(specs) == 0 {
			specs = json.RawMessage(`{}`)
		}
		items = append(items, entity.WysMallOrderItem{
			SKUID:        sku.SKUID,
			ProductID:    p.ProductID,
			Kind:         p.Kind,
			ProductTitle: p.Title,
			CoverURL:     p.CoverURL,
			Specs:        specs,
			Price:        sku.Price,
			Qty:          line.Qty,
			LineAmount:   formatMoney(lineAmt),
		})
	}
	if needAddress {
		if strings.TrimSpace(in.ReceiverName) == "" || strings.TrimSpace(in.ReceiverPhone) == "" || strings.TrimSpace(in.ReceiverAddress) == "" {
			return nil, ErrMallNeedAddress
		}
	}

	order := &entity.WysMallOrder{
		OrderNo:        "O" + strings.ReplaceAll(uuid.NewString(), "-", "")[:20],
		StoreID:        in.StoreID,
		BuyerUserID:    buyerUserID,
		IdempotencyKey: key,
		Status:         entity.MallOrderUnpaid,
		Amount:         formatMoney(total),
	}
	if needAddress {
		n := strings.TrimSpace(in.ReceiverName)
		ph := strings.TrimSpace(in.ReceiverPhone)
		addr := strings.TrimSpace(in.ReceiverAddress)
		order.ReceiverName = &n
		order.ReceiverPhone = &ph
		order.ReceiverAddress = &addr
	}
	if err := u.repo.CreateOrder(ctx, order, items); err != nil {
		// 并发幂等：唯一约束冲突时再读一次。
		if existing, e2 := u.repo.FindOrderByIdempotency(ctx, buyerUserID, key); e2 == nil && existing != nil {
			return u.GetOrder(ctx, buyerUserID, existing.OrderID)
		}
		return nil, err
	}
	if in.ClearCart {
		ids := make([]int64, 0, len(in.Lines))
		for _, l := range in.Lines {
			ids = append(ids, l.SKUID)
		}
		_ = u.repo.ClearCartSKUs(ctx, buyerUserID, ids)
	}
	return u.GetOrder(ctx, buyerUserID, order.OrderID)
}

// GetOrder 买家读自己的订单。
func (u *MallUsecase) GetOrder(ctx context.Context, buyerUserID string, orderID int64) (*entity.MallOrderDetail, error) {
	order, err := u.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.BuyerUserID != buyerUserID {
		return nil, ErrMallNotFound
	}
	items, err := u.repo.ListOrderItems(ctx, orderID)
	if err != nil {
		return nil, err
	}
	pays, err := u.repo.ListPayments(ctx, orderID)
	if err != nil {
		return nil, err
	}
	return &entity.MallOrderDetail{Order: *order, Items: items, Payments: pays}, nil
}

// PayOrder 本地模拟支付：渠道 1–4 均直接成功落库。
func (u *MallUsecase) PayOrder(ctx context.Context, buyerUserID string, orderID int64, channel int16) (*entity.MallOrderDetail, error) {
	if !entity.ValidMallPaymentChannel(channel) {
		return nil, ErrMallInvalidChannel
	}
	return u.repo.PayOrderLocal(ctx, orderID, buyerUserID, channel)
}

// CancelOrder 取消未支付订单。
func (u *MallUsecase) CancelOrder(ctx context.Context, buyerUserID string, orderID int64) (*entity.WysMallOrder, error) {
	return u.repo.CancelUnpaidOrder(ctx, orderID, buyerUserID)
}

func (u *MallUsecase) requireCatalogWrite(ctx context.Context, actorID string, storeID int) error {
	sid := storeID
	if err := u.access.RequirePermission(ctx, actorID, entity.PermMallCatalogWrite, &sid); err != nil {
		if errors.Is(err, ErrAccessForbidden) {
			return ErrMallForbidden
		}
		return err
	}
	return nil
}

func normalizeMoney(s string) (string, error) {
	s = strings.TrimSpace(s)
	r, ok := new(big.Rat).SetString(s)
	if !ok || r.Sign() < 0 {
		return "", fmt.Errorf("invalid money")
	}
	return formatMoney(r), nil
}

func formatMoney(r *big.Rat) string {
	return r.FloatString(2)
}