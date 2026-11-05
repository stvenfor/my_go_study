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
	"strconv"
	"strings"
	"time"

	"app/internal/access"
	"app/internal/payment"
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
	// pay 渠道支付网关（微信/支付宝）。为 nil 时渠道支付直接报错，
	// 不再保留「本地标记已付」的退路。
	pay *payment.Service
}

// NewService 创建。points / wallet 可为 nil。
func NewService(repo Repository, accessSvc *access.Service, pointsSvc *points.Service, walletSvc *wallet.Service) *Service {
	return &Service{repo: repo, accessSvc: accessSvc, pointsSvc: pointsSvc, walletSvc: walletSvc}
}

// WithPayment 注入 payment 域并把自己注册为该 ref_type 的核销器。
func (u *Service) WithPayment(pay *payment.Service) *Service {
	if pay == nil {
		return u
	}
	u.pay = pay
	pay.RegisterSettler(entity.PaymentRefMallOrder, u)
	return u
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

// PayOrder 支付入口。
//
// 微信 / 支付宝是「客户端在 App 里付款」，本地无权判定成功：改走 StartChannelPay
// 落 pending 流水，等渠道回调（或查单）核销后由 SettleChannelPaid 推进订单，
// 消除 P0-2 之前「调用即已付、无流水」的假成功。
// 余额 / 积分 / IAP 占位渠道仍走 settleLocal。
func (u *Service) PayOrder(ctx context.Context, buyerUserID string, orderID int64, channel int16) (*MallPayResult, error) {
	if !entity.ValidMallPaymentChannel(channel) {
		return nil, ErrInvalidChannel
	}
	if channel == entity.MallPayWeChat || channel == entity.MallPayAlipay {
		return u.StartChannelPay(ctx, buyerUserID, orderID, channel)
	}
	detail, err := u.settleLocal(ctx, buyerUserID, orderID, channel)
	if err != nil {
		return nil, err
	}
	return &MallPayResult{Status: "paid", Detail: detail}, nil
}

// MallPayResult 支付返回：本地结单给订单详情；渠道支付给唤起参数与 out_trade_no。
type MallPayResult struct {
	Status        string                  `json:"status"` // paid | pending
	Detail        *entity.MallOrderDetail `json:"detail,omitempty"`
	OutTradeNo    string                  `json:"out_trade_no,omitempty"`
	PrepayChannel string                  `json:"prepay_channel,omitempty"`
	PrepayParams  map[string]any          `json:"prepay_params,omitempty"`
}

// StartChannelPay 落一条 pending 渠道流水并取回 SDK 唤起参数（不改订单状态）。
func (u *Service) StartChannelPay(ctx context.Context, buyerUserID string, orderID int64, channel int16) (*MallPayResult, error) {
	if u.pay == nil {
		return nil, ErrPayGatewayUnavailable
	}
	order, err := u.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.BuyerUserID != buyerUserID {
		return nil, ErrNotFound
	}
	switch order.Status {
	case entity.MallOrderPaid, entity.MallOrderFulfilled:
		return nil, ErrOrderAlreadySettled
	case entity.MallOrderUnpaid:
	default:
		return nil, ErrOrderNotPayable
	}
	if entity.MallOrderPayExpired(order.Status, order.CreatedAt, time.Now()) {
		return nil, ErrOrderPayExpired
	}
	if entity.IsZeroMoney(order.Amount) {
		return nil, ErrNeedCNYChannel
	}
	fen, err := entity.ParseYuanToFen(order.Amount)
	if err != nil || fen <= 0 {
		return nil, ErrInvalidPrice
	}
	channelName, err := mallPayChannelName(channel)
	if err != nil {
		return nil, err
	}
	prepay, err := u.pay.Prepay(ctx, payment.PrepayInput{
		Channel:   channelName,
		AmountFen: fen,
		Subject:   mallPaySubject(order.OrderNo),
		UserID:    buyerUserID,
		RefType:   entity.PaymentRefMallOrder,
		RefID:     strconv.FormatInt(orderID, 10),
	})
	if err != nil {
		return nil, err
	}
	return &MallPayResult{
		Status:        "pending",
		OutTradeNo:    prepay.OutTradeNo,
		PrepayChannel: prepay.Channel,
		PrepayParams:  prepay.Params,
	}, nil
}

// SettleChannelPaid 实现 payment.Settler：渠道核销后把商城单推进到已支付。
//
// 与 payment.settleByFact 同处一个事务：这里失败会连流水一起回滚，
// 不会出现「流水 paid、订单 unpaid」。混合单（积分 + 人民币）的积分在这一步才扣，
// 渠道未收款成功就不能动积分账。
func (u *Service) SettleChannelPaid(ctx context.Context, refType, refID string, rec *entity.WysPaymentRecord) error {
	if refType != entity.PaymentRefMallOrder {
		return ErrNotFound
	}
	orderID, err := strconv.ParseInt(refID, 10, 64)
	if err != nil || orderID <= 0 {
		return ErrNotFound
	}
	channel, err := mallPayChannelByName(rec.Channel)
	if err != nil {
		return err
	}
	return u.repo.WithTx(ctx, func(ctx context.Context) error {
		locked, err := u.repo.LockOrderForSettle(ctx, orderID, rec.UserID)
		if err != nil {
			return err
		}
		if locked.Status == entity.MallOrderPaid || locked.Status == entity.MallOrderFulfilled {
			return nil
		}
		if locked.Status != entity.MallOrderUnpaid {
			return ErrOrderNotPayable
		}
		if locked.TotalPoints > 0 {
			if u.pointsSvc == nil {
				return points.ErrPointsInsufficient
			}
			if _, err := u.pointsSvc.Debit(ctx, rec.UserID, locked.TotalPoints,
				entity.PointsReasonMallPay, refID); err != nil {
				return err
			}
		}
		_, err = u.repo.PayOrderLocal(ctx, orderID, rec.UserID, channel, rec.OutTradeNo)
		return err
	})
}

// mallPayChannelName 商城渠道编号 → payment 域渠道名。
func mallPayChannelName(channel int16) (string, error) {
	switch channel {
	case entity.MallPayWeChat:
		return entity.PaymentChannelWeChat, nil
	case entity.MallPayAlipay:
		return entity.PaymentChannelAlipay, nil
	default:
		return "", ErrInvalidChannel
	}
}

// mallPayChannelByName payment 域渠道名 → 商城渠道编号。
func mallPayChannelByName(name string) (int16, error) {
	switch name {
	case entity.PaymentChannelWeChat:
		return entity.MallPayWeChat, nil
	case entity.PaymentChannelAlipay:
		return entity.MallPayAlipay, nil
	default:
		return 0, ErrInvalidChannel
	}
}

func mallPaySubject(orderNo string) string {
	if strings.TrimSpace(orderNo) == "" {
		return "商城订单"
	}
	return "商城订单-" + orderNo
}

// settleLocal 余额 / 积分 / IAP 占位渠道的本地结单（P0-3 的单事务形态）。
func (u *Service) settleLocal(ctx context.Context, buyerUserID string, orderID int64, channel int16) (*entity.MallOrderDetail, error) {
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

	ref := fmt.Sprintf("%d", orderID)
	var detail *entity.MallOrderDetail
	// 扣积分 + 扣钱包 + 改单/履约在同一事务里完成：任一步失败整体回滚，
	// 不再有「补偿也可能失败还被吞掉」的 restorePayDebits 分支。
	err = u.repo.WithTx(ctx, func(ctx context.Context) error {
		// 资金动作前必须先锁订单行：
		// 1) 并发两笔支付若先扣款后锁单，会各扣一次、只有一单生效；
		// 2) 对已支付订单重复支付时，旧实现会再扣一次款然后幂等返回 —— 现在跳过扣减。
		locked, err := u.repo.LockOrderForSettle(ctx, orderID, buyerUserID)
		if err != nil {
			return err
		}
		alreadySettled := locked.Status == entity.MallOrderPaid || locked.Status == entity.MallOrderFulfilled
		if needsPoints && !alreadySettled {
			if u.pointsSvc == nil {
				return points.ErrPointsInsufficient
			}
			if _, err := u.pointsSvc.Debit(ctx, buyerUserID, order.TotalPoints, entity.PointsReasonMallPay, ref); err != nil {
				return err
			}
		}
		if needsCNY && channel == entity.MallPayBalance && !alreadySettled {
			if u.walletSvc == nil {
				return wallet.ErrCashInsufficient
			}
			fen, err := entity.ParseYuanToFen(order.Amount)
			if err != nil || fen <= 0 {
				return ErrInvalidPrice
			}
			if _, err := u.walletSvc.Debit(ctx, buyerUserID, fen, entity.CashReasonMallPay, ref); err != nil {
				return err
			}
		}
		d, err := u.repo.PayOrderLocal(ctx, orderID, buyerUserID, channel, "")
		if err != nil {
			return err
		}
		detail = d
		return nil
	})
	if err != nil {
		return nil, err
	}
	return detail, nil
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
	ref := fmt.Sprintf("%d", orderID)
	var cancelled *entity.WysMallOrder
	// 改单 + 退款同事务：退款入账失败（含金额解析失败、服务未配置）不再静默跳过，直接回滚取消动作。
	err = u.repo.WithTx(ctx, func(ctx context.Context) error {
		// 与支付一致：先锁订单行，再动钱包/积分。
		if _, err := u.repo.LockOrderForSettle(ctx, orderID, buyerUserID); err != nil {
			return err
		}
		row, err := u.repo.CancelPaidOrder(ctx, orderID, buyerUserID)
		if err != nil {
			return err
		}
		if !entity.IsZeroMoney(row.Amount) {
			fen, err := entity.ParseYuanToFen(row.Amount)
			if err != nil {
				return fmt.Errorf("%w: 订单金额无法解析: %v", ErrRefundUnavailable, err)
			}
			if fen > 0 {
				if u.walletSvc == nil {
					return ErrRefundUnavailable
				}
				if _, err := u.walletSvc.Credit(ctx, buyerUserID, fen, entity.CashReasonMallRefund, ref); err != nil {
					return err
				}
			}
		}
		if row.TotalPoints > 0 {
			if u.pointsSvc == nil {
				return ErrRefundUnavailable
			}
			if _, err := u.pointsSvc.Credit(ctx, buyerUserID, row.TotalPoints, entity.PointsReasonMallRefund, ref); err != nil {
				return err
			}
		}
		cancelled = row
		return nil
	})
	if err != nil {
		return nil, err
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
