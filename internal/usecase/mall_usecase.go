// mall_usecase.go 门店商城：目录、购物车、下单、本地模拟支付。
package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

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
	ErrMallInvalidStatus      = errors.New("订单状态无效")
	ErrMallOrderPayExpired    = errors.New("订单支付已超时")
	ErrMallPaymentModeMixed   = errors.New("一笔订单内商品支付方式必须一致")
	ErrMallNeedCNYChannel     = errors.New("混合或人民币订单需使用人民币支付渠道")
)

// MallUsecase 商城用例。
type MallUsecase struct {
	repo   repository.MallRepository
	access *AccessUsecase
	points *PointsUsecase
	wallet *CashWalletUsecase
}

// NewMallUsecase 创建。points / wallet 可为 nil。
func NewMallUsecase(repo repository.MallRepository, access *AccessUsecase, points *PointsUsecase, wallet *CashWalletUsecase) *MallUsecase {
	return &MallUsecase{repo: repo, access: access, points: points, wallet: wallet}
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
	PricePoints int64
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
	if in.PricePoints < 0 {
		return nil, ErrMallInvalidPrice
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
			PricePoints: s.PricePoints,
			PaymentMode: entity.MallSKUPaymentMode(s.Price, s.PricePoints),
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
	var totalPoints int64
	var orderMode int16
	modeSet := false
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
		mode := entity.MallSKUPaymentMode(sku.Price, sku.PricePoints)
		if !modeSet {
			orderMode = mode
			modeSet = true
		} else if mode != orderMode {
			return nil, ErrMallPaymentModeMixed
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
func (u *MallUsecase) GetOrder(ctx context.Context, buyerUserID string, orderID int64) (*entity.MallOrderDetail, error) {
	order, err := u.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.BuyerUserID != buyerUserID {
		return nil, ErrMallNotFound
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
func (u *MallUsecase) ListOrders(ctx context.Context, buyerUserID string, statuses []int16, page, size int) ([]entity.MallOrderListItem, int64, error) {
	for _, s := range statuses {
		if s < entity.MallOrderUnpaid || s > entity.MallOrderClosed {
			return nil, 0, ErrMallInvalidStatus
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
func (u *MallUsecase) PayOrder(ctx context.Context, buyerUserID string, orderID int64, channel int16) (*entity.MallOrderDetail, error) {
	if !entity.ValidMallPaymentChannel(channel) {
		return nil, ErrMallInvalidChannel
	}
	order, err := u.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.BuyerUserID != buyerUserID {
		return nil, ErrMallNotFound
	}
	if entity.MallOrderPayExpired(order.Status, order.CreatedAt, time.Now()) {
		if _, err := u.repo.CancelUnpaidOrder(ctx, order.OrderID, buyerUserID); err != nil &&
			!errors.Is(err, ErrMallOrderNotCancelable) {
			return nil, err
		}
		return nil, ErrMallOrderPayExpired
	}
	needsCNY := !entity.IsZeroMoney(order.Amount)
	needsPoints := order.TotalPoints > 0
	if needsCNY {
		if !entity.ValidMallCNYChannel(channel) {
			return nil, ErrMallNeedCNYChannel
		}
	} else if needsPoints {
		channel = entity.MallPayPoints
	}

	debitedPts := int64(0)
	debitedFen := int64(0)
	ref := fmt.Sprintf("%d", orderID)
	if needsPoints {
		if u.points == nil {
			return nil, ErrPointsInsufficient
		}
		if _, err := u.points.Debit(ctx, buyerUserID, order.TotalPoints, entity.PointsReasonMallPay, ref); err != nil {
			return nil, err
		}
		debitedPts = order.TotalPoints
	}
	if needsCNY && channel == entity.MallPayBalance {
		if u.wallet == nil {
			return nil, ErrCashInsufficient
		}
		fen, err := entity.ParseYuanToFen(order.Amount)
		if err != nil || fen <= 0 {
			u.restorePayDebits(ctx, buyerUserID, debitedPts, 0, ref)
			return nil, ErrMallInvalidPrice
		}
		if _, err := u.wallet.Debit(ctx, buyerUserID, fen, entity.CashReasonMallPay, ref); err != nil {
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

func (u *MallUsecase) restorePayDebits(ctx context.Context, userID string, pts, fen int64, ref string) {
	if pts > 0 && u.points != nil {
		_, _ = u.points.Credit(ctx, userID, pts, entity.PointsReasonMallRefund, ref)
	}
	if fen > 0 && u.wallet != nil {
		_, _ = u.wallet.Credit(ctx, userID, fen, entity.CashReasonMallRefund, ref)
	}
}

// CancelOrder 取消未支付订单；已支付未履约则取消并按 ADR-0015 将人民币退回钱包、积分退回积分账。
func (u *MallUsecase) CancelOrder(ctx context.Context, buyerUserID string, orderID int64) (*entity.WysMallOrder, error) {
	order, err := u.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.BuyerUserID != buyerUserID {
		return nil, ErrMallNotFound
	}
	if order.Status == entity.MallOrderUnpaid {
		return u.repo.CancelUnpaidOrder(ctx, orderID, buyerUserID)
	}
	if order.Status != entity.MallOrderPaid {
		return nil, ErrMallOrderNotCancelable
	}
	cancelled, err := u.repo.CancelPaidOrder(ctx, orderID, buyerUserID)
	if err != nil {
		return nil, err
	}
	ref := fmt.Sprintf("%d", orderID)
	if !entity.IsZeroMoney(cancelled.Amount) && u.wallet != nil {
		if fen, err := entity.ParseYuanToFen(cancelled.Amount); err == nil && fen > 0 {
			_, _ = u.wallet.Credit(ctx, buyerUserID, fen, entity.CashReasonMallRefund, ref)
		}
	}
	if cancelled.TotalPoints > 0 && u.points != nil {
		_, _ = u.points.Credit(ctx, buyerUserID, cancelled.TotalPoints, entity.PointsReasonMallRefund, ref)
	}
	return cancelled, nil
}

// expireUnpaidIfNeeded 待支付超过 TTL 则取消并返回最新行。
func (u *MallUsecase) expireUnpaidIfNeeded(ctx context.Context, order *entity.WysMallOrder) (*entity.WysMallOrder, error) {
	if order == nil {
		return nil, ErrMallNotFound
	}
	if !entity.MallOrderPayExpired(order.Status, order.CreatedAt, time.Now()) {
		return order, nil
	}
	cancelled, err := u.repo.CancelUnpaidOrder(ctx, order.OrderID, order.BuyerUserID)
	if err != nil {
		if errors.Is(err, ErrMallOrderNotCancelable) {
			// 并发下已被支付/取消，重读。
			return u.repo.GetOrder(ctx, order.OrderID)
		}
		return nil, err
	}
	return cancelled, nil
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