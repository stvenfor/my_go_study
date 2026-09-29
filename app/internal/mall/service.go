// service.go 门店商城：目录、购物车、下单、本地模拟支付 —— 域内业务逻辑。
//
// 由 app/internal/usecase/mall_usecase.go 迁入。三处改动：
//  1. MallUsecase → Service（CreateXxx 入参类型本就没有域前缀，原名保留）；
//  2. 错误哨兵下沉 repository.go 并去掉 Mall 前缀（域内即命名空间），
//     原先指向 model/repository 的别名 var 块整块删除；
//  3. 跨域依赖收敛为 access / points / wallet 三个公开 Service。
package mall

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"app/internal/access"
	"app/internal/points"
	"app/internal/wallet"
	"github.com/google/uuid"
	"model/entity"
)

// Service 商城用例。
type Service struct {
	repo      Repository
	accessSvc *access.Service
	pointsSvc *points.Service
	walletSvc *wallet.Service
}

// NewService 创建。points / wallet 可为 nil。
func NewService(repo Repository, accessSvc *access.Service, pointsSvc *points.Service, walletSvc *wallet.Service) *Service {
	return &Service{repo: repo, accessSvc: accessSvc, pointsSvc: pointsSvc, walletSvc: walletSvc}
}

// CreateProduct 店员创建商品。
func (u *Service) CreateProduct(ctx context.Context, actorID string, storeID int, kind int16, title string, coverURL *string, status int16) (*entity.WysMallProduct, error) {
	if err := u.requireCatalogWrite(ctx, actorID, storeID); err != nil {
		return nil, err
	}
	if kind != entity.MallKindPhysical && kind != entity.MallKindVirtual {
		return nil, ErrInvalidKind
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrInvalidTitle
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
	PricePoints int64
	StockQty    int
	Status      int16
	DeliverType *int16
	ContentURL  *string
}

// CreateSKU 店员创建 SKU。
func (u *Service) CreateSKU(ctx context.Context, actorID string, in CreateSKUInput) (*entity.WysMallSKU, error) {
	p, err := u.repo.GetProduct(ctx, in.ProductID)
	if err != nil {
		return nil, err
	}
	if err := u.requireCatalogWrite(ctx, actorID, p.StoreID); err != nil {
		return nil, err
	}
	price, err := normalizeMoney(in.Price)
	if err != nil {
		return nil, ErrInvalidPrice
	}
	code := strings.TrimSpace(in.SKUCode)
	if code == "" {
		return nil, errors.New("sku_code 不能为空")
	}
	specs := in.SpecsJSON
	if len(specs) == 0 {
		specs = json.RawMessage(`{}`)
	}
	if in.PricePoints < 0 {
		return nil, ErrInvalidPrice
	}
	sku := &entity.WysMallSKU{
		ProductID:   p.ProductID,
		SKUCode:     code,
		Title:       strings.TrimSpace(in.Title),
		Specs:       specs,
		Price:       price,
		PricePoints: in.PricePoints,
		Status:      in.Status,
	}
	if sku.Status != entity.MallSKUOn && sku.Status != entity.MallSKUOff {
		sku.Status = entity.MallSKUOff
	}
	switch p.Kind {
	case entity.MallKindPhysical:
		if in.StockQty < 0 {
			return nil, ErrInvalidQty
		}
		sku.StockQty = in.StockQty
		sku.DeliverType = nil
		sku.ContentURL = nil
	case entity.MallKindVirtual:
		sku.StockQty = 0
		if in.DeliverType == nil {
			return nil, ErrInvalidKind
		}
		dt := *in.DeliverType
		if dt != entity.MallDeliverRedeemCode && dt != entity.MallDeliverContentURL {
			return nil, ErrInvalidKind
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
		return nil, ErrInvalidKind
	}
	if err := u.repo.CreateSKU(ctx, sku); err != nil {
		return nil, err
	}
	return sku, nil
}

// AddVirtualCodes 店员导入兑换码。
func (u *Service) AddVirtualCodes(ctx context.Context, actorID string, skuID int64, codes []string) error {
	sku, err := u.repo.GetSKU(ctx, skuID)
	if err != nil {
		return err
	}
	p, err := u.repo.GetProduct(ctx, sku.ProductID)
	if err != nil {
		return err
	}
	if p.Kind != entity.MallKindVirtual || sku.DeliverType == nil || *sku.DeliverType != entity.MallDeliverRedeemCode {
		return ErrInvalidKind
	}
	if err := u.requireCatalogWrite(ctx, actorID, p.StoreID); err != nil {
		return err
	}
	return u.repo.AddVirtualCodes(ctx, skuID, codes)
}

// GetShelfProduct 买家读一门店的在售商品及上架规格。下架、错店、已删都当不存在。
func (u *Service) GetShelfProduct(ctx context.Context, storeID int, productID int64) (*entity.MallProductDetail, error) {
	if storeID <= 0 {
		return nil, access.ErrInvalidStoreID
	}
	if productID <= 0 {
		return nil, ErrNotFound
	}
	p, err := u.repo.GetProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	if p.StoreID != storeID || p.Status != entity.MallProductOnShelf {
		return nil, ErrNotFound
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
			PricePoints: s.PricePoints,
			PaymentMode: entity.MallSKUPaymentMode(s.Price, s.PricePoints),
			StockQty:    s.StockQty,
			DeliverType: s.DeliverType,
		})
	}
	return &entity.MallProductDetail{Product: *p, SKUs: offers}, nil
}

// ListShelfItems 买家分页浏览在售扁平行。page 从 1 起，默认每页 10。
func (u *Service) ListShelfItems(ctx context.Context, storeID, page, size int) ([]entity.MallShelfItem, int64, error) {
	if storeID <= 0 {
		return nil, 0, access.ErrInvalidStoreID
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
func (u *Service) UpsertCart(ctx context.Context, userID string, skuID int64, qty int) (*entity.WysMallCartItem, error) {
	if qty <= 0 {
		return nil, ErrInvalidQty
	}
	sku, err := u.repo.GetSKU(ctx, skuID)
	if err != nil {
		return nil, err
	}
	if sku.Status != entity.MallSKUOn {
		return nil, ErrSKUOffShelf
	}
	item := &entity.WysMallCartItem{UserID: userID, SKUID: skuID, Qty: qty}
	if err := u.repo.UpsertCartItem(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

// ListCart 买家购物车。
func (u *Service) ListCart(ctx context.Context, userID string) ([]entity.WysMallCartItem, error) {
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
func (u *Service) CreateOrder(ctx context.Context, buyerUserID string, in CreateOrderInput) (*entity.MallOrderDetail, error) {
	key := strings.TrimSpace(in.IdempotencyKey)
	if key == "" {
		return nil, ErrInvalidIdempotency
	}
	if existing, err := u.repo.FindOrderByIdempotency(ctx, buyerUserID, key); err != nil {
		return nil, err
	} else if existing != nil {
		return u.GetOrder(ctx, buyerUserID, existing.OrderID)
	}
	if len(in.Lines) == 0 {
		return nil, ErrEmptyCart
	}
	if in.StoreID <= 0 {
		return nil, access.ErrInvalidStoreID
	}

	var items []entity.WysMallOrderItem
	total := big.NewRat(0, 1)
	var totalPoints int64
	var orderMode int16
	modeSet := false
	needAddress := false
	for _, line := range in.Lines {
		if line.Qty <= 0 {
			return nil, ErrInvalidQty
		}
		sku, err := u.repo.GetSKU(ctx, line.SKUID)
		if err != nil {
			return nil, err
		}
		if sku.Status != entity.MallSKUOn {
			return nil, ErrSKUOffShelf
		}
		p, err := u.repo.GetProduct(ctx, sku.ProductID)
		if err != nil {
			return nil, err
		}
		if p.StoreID != in.StoreID {
			return nil, ErrMultiStore
		}
		if p.Status != entity.MallProductOnShelf {
			return nil, ErrSKUOffShelf
		}
		mode := entity.MallSKUPaymentMode(sku.Price, sku.PricePoints)
		if !modeSet {
			orderMode = mode
			modeSet = true
		} else if mode != orderMode {
			return nil, ErrPaymentModeMixed
		}
		if p.Kind == entity.MallKindPhysical {
			needAddress = true
			if sku.StockQty < line.Qty {
				return nil, ErrStockInsufficient
			}
		}
		unit, ok := new(big.Rat).SetString(sku.Price)
		if !ok {
			return nil, ErrInvalidPrice
		}
		lineAmt := new(big.Rat).Mul(unit, big.NewRat(int64(line.Qty), 1))
		total.Add(total, lineAmt)
		linePts := sku.PricePoints * int64(line.Qty)
		totalPoints += linePts
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
			PricePoints:  sku.PricePoints,
			Qty:          line.Qty,
			LineAmount:   formatMoney(lineAmt),
			LinePoints:   linePts,
		})
	}
	if needAddress {
		if strings.TrimSpace(in.ReceiverName) == "" || strings.TrimSpace(in.ReceiverPhone) == "" || strings.TrimSpace(in.ReceiverAddress) == "" {
			return nil, ErrNeedAddress
		}
	}

	order := &entity.WysMallOrder{
		OrderNo:        "O" + strings.ReplaceAll(uuid.NewString(), "-", "")[:20],
		StoreID:        in.StoreID,
		BuyerUserID:    buyerUserID,
		IdempotencyKey: key,
		Status:         entity.MallOrderUnpaid,
		Amount:         formatMoney(total),
		TotalPoints:    totalPoints,
		PaymentMode:    orderMode,
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

// GetOrder 买家读自己的订单。待支付超时则惰性取消。
func (u *Service) GetOrder(ctx context.Context, buyerUserID string, orderID int64) (*entity.MallOrderDetail, error) {
	order, err := u.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.BuyerUserID != buyerUserID {
		return nil, ErrNotFound
	}
	order, err = u.expireUnpaidIfNeeded(ctx, order)
	if err != nil {
		return nil, err
	}
	items, err := u.repo.ListOrderItems(ctx, orderID)
	if err != nil {
		return nil, err
	}
	pays, err := u.repo.ListPayments(ctx, orderID)
	if err != nil {
		return nil, err
	}
	return &entity.MallOrderDetail{
		Order:         *order,
		Items:         items,
		Payments:      pays,
		PayDeadlineAt: entity.MallOrderPayDeadline(order.Status, order.CreatedAt),
	}, nil
}

// ListOrders 买家分页列出自己的订单。statuses 为空表示全部；每个值必须在 0–4。
func (u *Service) ListOrders(ctx context.Context, buyerUserID string, statuses []int16, page, size int) ([]entity.MallOrderListItem, int64, error) {
	for _, s := range statuses {
		if s < entity.MallOrderUnpaid || s > entity.MallOrderClosed {
			return nil, 0, ErrInvalidStatus
		}
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
	orders, total, err := u.repo.ListOrdersByBuyer(ctx, buyerUserID, statuses, offset, size)
	if err != nil {
		return nil, 0, err
	}
	if len(orders) == 0 {
		return []entity.MallOrderListItem{}, total, nil
	}
	ids := make([]int64, len(orders))
	for i := range orders {
		o, err := u.expireUnpaidIfNeeded(ctx, &orders[i])
		if err != nil {
			return nil, 0, err
		}
		orders[i] = *o
		ids[i] = orders[i].OrderID
	}
	allItems, err := u.repo.ListOrderItemsByOrderIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	byOrder := make(map[int64][]entity.WysMallOrderItem, len(orders))
	for _, it := range allItems {
		byOrder[it.OrderID] = append(byOrder[it.OrderID], it)
	}
	out := make([]entity.MallOrderListItem, 0, len(orders))
	for _, o := range orders {
		items := byOrder[o.OrderID]
		if items == nil {
			items = []entity.WysMallOrderItem{}
		}
		out = append(out, entity.MallOrderListItem{
			OrderID:       o.OrderID,
			OrderNo:       o.OrderNo,
			StoreID:       o.StoreID,
			Status:        o.Status,
			Amount:        o.Amount,
			CreatedAt:     o.CreatedAt,
			PayDeadlineAt: entity.MallOrderPayDeadline(o.Status, o.CreatedAt),
			Items:         items,
		})
	}
	return out, total, nil
}

// PayOrder 支付：积分经 Points 账本扣减；人民币余额渠道经 Wallet 扣减；混合需人民币渠道；失败则退回已扣。
func (u *Service) PayOrder(ctx context.Context, buyerUserID string, orderID int64, channel int16) (*entity.MallOrderDetail, error) {
	if !entity.ValidMallPaymentChannel(channel) {
		return nil, ErrInvalidChannel
	}
	order, err := u.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.BuyerUserID != buyerUserID {
		return nil, ErrNotFound
	}
	if entity.MallOrderPayExpired(order.Status, order.CreatedAt, time.Now()) {
		if _, err := u.repo.CancelUnpaidOrder(ctx, order.OrderID, buyerUserID); err != nil &&
			!errors.Is(err, ErrOrderNotCancelable) {
			return nil, err
		}
		return nil, ErrOrderPayExpired
	}
	needsCNY := !entity.IsZeroMoney(order.Amount)
	needsPoints := order.TotalPoints > 0
	if needsCNY {
		if !entity.ValidMallCNYChannel(channel) {
			return nil, ErrNeedCNYChannel
		}
	} else if needsPoints {
		channel = entity.MallPayPoints
	}

	debitedPts := int64(0)
	debitedFen := int64(0)
	ref := fmt.Sprintf("%d", orderID)
	if needsPoints {
		if u.pointsSvc == nil {
			return nil, points.ErrPointsInsufficient
		}
		if _, err := u.pointsSvc.Debit(ctx, buyerUserID, order.TotalPoints, entity.PointsReasonMallPay, ref); err != nil {
			return nil, err
		}
		debitedPts = order.TotalPoints
	}
	if needsCNY && channel == entity.MallPayBalance {
		if u.walletSvc == nil {
			return nil, wallet.ErrCashInsufficient
		}
		fen, err := entity.ParseYuanToFen(order.Amount)
		if err != nil || fen <= 0 {
			u.restorePayDebits(ctx, buyerUserID, debitedPts, 0, ref)
			return nil, ErrInvalidPrice
		}
		if _, err := u.walletSvc.Debit(ctx, buyerUserID, fen, entity.CashReasonMallPay, ref); err != nil {
			u.restorePayDebits(ctx, buyerUserID, debitedPts, 0, ref)
			return nil, err
		}
		debitedFen = fen
	}
	detail, err := u.repo.PayOrderLocal(ctx, orderID, buyerUserID, channel)
	if err != nil {
		u.restorePayDebits(ctx, buyerUserID, debitedPts, debitedFen, ref)
		return nil, err
	}
	return detail, nil
}

func (u *Service) restorePayDebits(ctx context.Context, userID string, pts, fen int64, ref string) {
	if pts > 0 && u.pointsSvc != nil {
		_, _ = u.pointsSvc.Credit(ctx, userID, pts, entity.PointsReasonMallRefund, ref)
	}
	if fen > 0 && u.walletSvc != nil {
		_, _ = u.walletSvc.Credit(ctx, userID, fen, entity.CashReasonMallRefund, ref)
	}
}

// CancelOrder 取消未支付订单；已支付未履约则取消并按 ADR-0015 将人民币退回钱包、积分退回积分账。
func (u *Service) CancelOrder(ctx context.Context, buyerUserID string, orderID int64) (*entity.WysMallOrder, error) {
	order, err := u.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.BuyerUserID != buyerUserID {
		return nil, ErrNotFound
	}
	if order.Status == entity.MallOrderUnpaid {
		return u.repo.CancelUnpaidOrder(ctx, orderID, buyerUserID)
	}
	if order.Status != entity.MallOrderPaid {
		return nil, ErrOrderNotCancelable
	}
	cancelled, err := u.repo.CancelPaidOrder(ctx, orderID, buyerUserID)
	if err != nil {
		return nil, err
	}
	ref := fmt.Sprintf("%d", orderID)
	if !entity.IsZeroMoney(cancelled.Amount) && u.walletSvc != nil {
		if fen, err := entity.ParseYuanToFen(cancelled.Amount); err == nil && fen > 0 {
			_, _ = u.walletSvc.Credit(ctx, buyerUserID, fen, entity.CashReasonMallRefund, ref)
		}
	}
	if cancelled.TotalPoints > 0 && u.pointsSvc != nil {
		_, _ = u.pointsSvc.Credit(ctx, buyerUserID, cancelled.TotalPoints, entity.PointsReasonMallRefund, ref)
	}
	return cancelled, nil
}

// expireUnpaidIfNeeded 待支付超过 TTL 则取消并返回最新行。
func (u *Service) expireUnpaidIfNeeded(ctx context.Context, order *entity.WysMallOrder) (*entity.WysMallOrder, error) {
	if order == nil {
		return nil, ErrNotFound
	}
	if !entity.MallOrderPayExpired(order.Status, order.CreatedAt, time.Now()) {
		return order, nil
	}
	cancelled, err := u.repo.CancelUnpaidOrder(ctx, order.OrderID, order.BuyerUserID)
	if err != nil {
		if errors.Is(err, ErrOrderNotCancelable) {
			// 并发下已被支付/取消，重读。
			return u.repo.GetOrder(ctx, order.OrderID)
		}
		return nil, err
	}
	return cancelled, nil
}

func (u *Service) requireCatalogWrite(ctx context.Context, actorID string, storeID int) error {
	sid := storeID
	if err := u.accessSvc.RequirePermission(ctx, actorID, entity.PermMallCatalogWrite, &sid); err != nil {
		if errors.Is(err, access.ErrForbidden) {
			return ErrForbidden
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
