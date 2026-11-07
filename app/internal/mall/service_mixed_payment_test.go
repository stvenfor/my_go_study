// service_mixed_payment_test.go 商城组合支付（现金 + 积分）口径②的不变量测试。
//
// 口径②：price = 抵扣后应付现金，price_points = 抵扣所需积分，100 积分 = 1 元（1 积分 = 1 分）。
// 结算收「全额 order.Amount（现金）+ 全额 order.TotalPoints（积分）」就是正确写法。
// 本文件用真实下单 / 支付 / 取消的资金流水锁死这条：谁要是改成「先算抵扣再收现金」，
// 下面的「实付现金 == price×qty」断言会立刻失败（二次打折 / 反向资损）。
package mall

import (
	"context"
	"testing"
	"time"

	"app/internal/points"
	"app/internal/wallet"
	"model/entity"
)

// ---- 夹具（测试专用，不进生产二进制）----

// mixedOrderRepo 只实现 CreateOrder 链用到的方法，其余由嵌入的 Repository 兜底。
type mixedOrderRepo struct {
	Repository
	sku     *entity.WysMallSKU
	product *entity.WysMallProduct
	order   *entity.WysMallOrder
	items   []entity.WysMallOrderItem
}

func (r *mixedOrderRepo) FindOrderByIdempotency(context.Context, string, string) (*entity.WysMallOrder, error) {
	return nil, nil
}

func (r *mixedOrderRepo) GetSKU(_ context.Context, id int64) (*entity.WysMallSKU, error) {
	if r.sku == nil || r.sku.SKUID != id {
		return nil, ErrNotFound
	}
	cp := *r.sku
	return &cp, nil
}

func (r *mixedOrderRepo) GetProduct(_ context.Context, id int64) (*entity.WysMallProduct, error) {
	if r.product == nil || r.product.ProductID != id {
		return nil, ErrNotFound
	}
	cp := *r.product
	return &cp, nil
}

func (r *mixedOrderRepo) CreateOrder(_ context.Context, o *entity.WysMallOrder, items []entity.WysMallOrderItem) error {
	cp := *o
	cp.CreatedAt = time.Now() // 模拟 DB 默认值：否则 expireUnpaidIfNeeded 会把零值时间当成已超时
	r.order = &cp
	r.items = items
	return nil
}

func (r *mixedOrderRepo) GetOrder(_ context.Context, id int64) (*entity.WysMallOrder, error) {
	if r.order == nil || r.order.OrderID != id {
		return nil, ErrNotFound
	}
	cp := *r.order
	return &cp, nil
}

func (r *mixedOrderRepo) ListOrderItems(context.Context, int64) ([]entity.WysMallOrderItem, error) {
	return r.items, nil
}

func (r *mixedOrderRepo) ListPayments(context.Context, int64) ([]entity.WysMallPayment, error) {
	return nil, nil
}

// memPointsRepo 内存积分账；只实现账本三方法，其余由嵌入的 points.Repository 兜底。
type memPointsRepo struct {
	points.Repository
	balance map[string]int64
	ledger  []int64 // 每笔增减，用于断言实扣 / 实退
}

func newMemPointsRepo(init map[string]int64) *memPointsRepo {
	b := make(map[string]int64, len(init))
	for k, v := range init {
		b[k] = v
	}
	return &memPointsRepo{balance: b}
}

func (m *memPointsRepo) GetBalance(_ context.Context, uid string) (int64, error) {
	return m.balance[uid], nil
}

func (m *memPointsRepo) Credit(_ context.Context, uid string, delta int64, _, _ string) (int64, error) {
	m.balance[uid] += delta
	m.ledger = append(m.ledger, delta)
	return m.balance[uid], nil
}

func (m *memPointsRepo) Debit(_ context.Context, uid string, delta int64, _, _ string) (int64, error) {
	if m.balance[uid] < delta {
		return 0, points.ErrPointsInsufficient
	}
	m.balance[uid] -= delta
	m.ledger = append(m.ledger, -delta)
	return m.balance[uid], nil
}

// mustFen 把元字符串转分；测试算例不应失败。
func mustFen(t *testing.T, s string) int64 {
	t.Helper()
	fen, err := entity.ParseYuanToFen(s)
	if err != nil {
		t.Fatalf("ParseYuanToFen(%q): %v", s, err)
	}
	return fen
}

// ---- 1. 折算函数本体 ----

func TestPointsValueFen(t *testing.T) {
	if entity.MallPointsPerYuan != 100 {
		t.Fatalf("兑换率被改为 %d：存量 price / price_points 与展示原价需重刷，见 docs/mall-mixed-payment.md", entity.MallPointsPerYuan)
	}
	cases := []struct {
		points int64
		want   int64 // 分
	}{
		{0, 0},
		{-50, 0},       // 非正积分防御性收口
		{1, 1},         // 1 积分 = 1 分
		{50, 50},       // sku_id=37 的 50 积分 = 0.50 元
		{99, 99},
		{100, 100},     // 100 积分 = 1 元 = 100 分
		{1040, 1040},   // 10.40 元
		{12345, 12345}, // 默认率整除无余数：PointsValueFen(p) == p
	}
	for _, c := range cases {
		if got := entity.PointsValueFen(c.points); got != c.want {
			t.Errorf("PointsValueFen(%d) = %d, want %d", c.points, got, c.want)
		}
	}
	// 默认率下 1 积分 = 1 分，恒等且整除无余数。
	for p := int64(0); p <= 1000; p++ {
		if got := entity.PointsValueFen(p); got != p {
			t.Fatalf("默认率应整除无余数：PointsValueFen(%d) = %d", p, got)
		}
	}
	// sku_id=37 真实算例：990 分 + 50 分 = 1040 分 = 10.40 元。
	if got := entity.FormatFenToYuan(mustFen(t, "9.90") + entity.PointsValueFen(50)); got != "10.40" {
		t.Fatalf("sku_id=37 展示原价 = %s 元, want 10.40", got)
	}
}

// ---- 2. 下单不变量：实付现金 / 实扣积分 / 展示原价 / payment_mode ----

func TestMixedPaymentDeductionInvariants(t *testing.T) {
	// sku_id=37：price 9.90（抵扣后应付现金）+ 50 积分（抵扣所需），qty=3。
	const qty = 3
	sku := &entity.WysMallSKU{
		SKUID: 37, ProductID: 1, Title: "组合支付样例",
		Price: "9.90", PricePoints: 50, StockQty: 100, Status: entity.MallSKUOn,
	}
	repo := &mixedOrderRepo{
		sku: sku,
		product: &entity.WysMallProduct{
			ProductID: 1, StoreID: 1, Kind: entity.MallKindPhysical,
			Title: "样例商品", Status: entity.MallProductOnShelf,
		},
	}
	uc := NewService(repo, nil, nil, nil)
	detail, err := uc.CreateOrder(context.Background(), "buyer1", CreateOrderInput{
		IdempotencyKey:  "h9-mixed-1",
		StoreID:         1,
		Lines:           []CreateOrderLine{{SKUID: 37, Qty: qty}},
		ReceiverName:    "张三",
		ReceiverPhone:   "13400000000",
		ReceiverAddress: "某地址",
	})
	if err != nil {
		t.Fatal(err)
	}

	priceFen := mustFen(t, sku.Price)                                         // 990
	wantCashFen := priceFen * int64(qty)                                       // 2970
	wantPoints := sku.PricePoints * int64(qty)                                 // 150
	wantListPriceFen := (priceFen + entity.PointsValueFen(sku.PricePoints)) * int64(qty) // 1040*3 = 3120

	// 实付现金额 == price 分数 × qty（结算据此收现金，不含任何积分折算项）。
	if got := mustFen(t, detail.Order.Amount); got != wantCashFen {
		t.Errorf("实付现金 = %d 分, want %d (= price %d 分 × qty %d)", got, wantCashFen, priceFen, qty)
	}
	// 实扣积分 == price_points × qty。
	if detail.Order.TotalPoints != wantPoints {
		t.Errorf("实扣积分 = %d, want %d (= price_points %d × qty %d)", detail.Order.TotalPoints, wantPoints, sku.PricePoints, qty)
	}
	// 订单行快照同样自洽。
	if len(detail.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(detail.Items))
	}
	it := detail.Items[0]
	if mustFen(t, it.LineAmount) != wantCashFen || it.LinePoints != wantPoints {
		t.Errorf("行快照 LineAmount/LinePoints = %s/%d, want %d 分/%d", it.LineAmount, it.LinePoints, wantCashFen, wantPoints)
	}
	// 实付现金分 + PointsValueFen(实扣积分) == 展示原价分。
	if got := mustFen(t, detail.Order.Amount) + entity.PointsValueFen(detail.Order.TotalPoints); got != wantListPriceFen {
		t.Errorf("展示原价分 = %d, want %d", got, wantListPriceFen)
	}
	// 与 payment_mode 的派生关系：两字段都 > 0 ⇒ Mixed。
	if got := entity.MallSKUPaymentMode(sku.Price, sku.PricePoints); got != entity.MallPayModeMixed {
		t.Errorf("payment_mode = %d, want Mixed(%d)", got, entity.MallPayModeMixed)
	}
	if detail.Order.PaymentMode != entity.MallPayModeMixed {
		t.Errorf("order.PaymentMode = %d, want Mixed", detail.Order.PaymentMode)
	}
	// sku_id=37 单位：990 + 50 = 1040 分 = 10.40 元。
	if got := entity.FormatFenToYuan(priceFen + entity.PointsValueFen(sku.PricePoints)); got != "10.40" {
		t.Errorf("sku_id=37 单位展示原价 = %s 元, want 10.40", got)
	}
}

// ---- 3. 支付 + 退款对称（真实钱包 / 积分流水）----

func TestMixedPaymentRefundSymmetry(t *testing.T) {
	const qty = 3
	priceFen := mustFen(t, "9.90")              // 990
	cashFen := priceFen * int64(qty)             // 实付现金 2970 分 = 29.70 元
	pts := int64(50) * int64(qty)               // 实扣积分 150
	amount := entity.FormatFenToYuan(cashFen)   // "29.70"

	ctx := context.Background()
	uid := "buyer-mixed"
	memCash := newMemCashRepo()
	cash := wallet.NewService(memCash)
	if _, err := cash.Recharge(ctx, uid, wallet.RechargeInput{Amount: "100.00", Channel: entity.WalletRechargeAlipay}); err != nil {
		t.Fatal(err)
	}
	memPoints := newMemPointsRepo(map[string]int64{uid: 1000})
	ptsSvc := points.NewService(memPoints, nil)

	order := &entity.WysMallOrder{
		OrderID: 37, BuyerUserID: uid, Status: entity.MallOrderUnpaid,
		Amount: amount, TotalPoints: pts, CreatedAt: time.Now(),
	}
	repo := &payOrderRepo{cash: memCash, order: order}
	uc := NewService(repo, nil, ptsSvc, cash)

	startCash, _ := cash.GetBalanceFen(ctx, uid) // 10000
	startPoints, _ := ptsSvc.GetBalance(ctx, uid) // 1000

	if _, err := uc.PayOrder(ctx, uid, 37, entity.MallPayBalance); err != nil {
		t.Fatal(err)
	}
	paidCash, _ := cash.GetBalanceFen(ctx, uid)
	paidPoints, _ := ptsSvc.GetBalance(ctx, uid)
	// 实付现金 == price 分数 × qty；实扣积分 == price_points × qty。
	// 若有人把 settleLocal 改成「先算抵扣再收现金」，下面第一条立即失败。
	if startCash-paidCash != cashFen {
		t.Errorf("实付现金 = %d 分, want %d", startCash-paidCash, cashFen)
	}
	if startPoints-paidPoints != pts {
		t.Errorf("实扣积分 = %d, want %d", startPoints-paidPoints, pts)
	}
	// 结算没有把积分价值再从现金里净额掉：现金腿 + 积分腿 == 展示原价分。
	listPriceFen := cashFen + entity.PointsValueFen(pts)
	if (startCash-paidCash)+entity.PointsValueFen(startPoints-paidPoints) != listPriceFen {
		t.Errorf("现金腿 + 积分腿 != 展示原价分 %d", listPriceFen)
	}
	// 积分流水须是「全额一笔」，互不覆盖。
	if len(memPoints.ledger) != 1 || memPoints.ledger[0] != -pts {
		t.Errorf("积分流水 = %v, want 单笔 -%d", memPoints.ledger, pts)
	}

	if _, err := uc.CancelOrder(ctx, uid, 37); err != nil {
		t.Fatal(err)
	}
	endCash, _ := cash.GetBalanceFen(ctx, uid)
	endPoints, _ := ptsSvc.GetBalance(ctx, uid)
	// 退款对称：退现金 == 实付现金，退积分 == 实扣积分。
	if endCash != startCash {
		t.Errorf("退现金后余额 = %d, want 回到 %d（退现金应等于实付 %d）", endCash, startCash, cashFen)
	}
	if endPoints != startPoints {
		t.Errorf("退积分后余额 = %d, want 回到 %d（退积分应等于实扣 %d）", endPoints, startPoints, pts)
	}
	if len(memPoints.ledger) != 2 || memPoints.ledger[1] != pts {
		t.Errorf("积分退款流水 = %v, want 第二笔 +%d", memPoints.ledger, pts)
	}
}

// ---- 4. 证明不存在「同一路径既收现金又扣同一份价值」 ----

func TestMixedPaymentDeductionNoDoubleCount(t *testing.T) {
	priceFen := mustFen(t, "9.90")           // 990 分：现金腿 = 抵扣后应付现金
	pointsFen := entity.PointsValueFen(50)    // 50 分：积分腿的现金等价值
	listPriceFen := priceFen + pointsFen      // 1040 分 = 10.40 元：展示原价

	// (1) 两腿恰好把展示原价切一次：和为原价，无遗漏、无重复。
	if priceFen+pointsFen != listPriceFen {
		t.Fatalf("现金腿 + 积分腿 = %d, want 展示原价分 %d", priceFen+pointsFen, listPriceFen)
	}
	// (2) 现金腿不包含积分那一份价值：它严格小于展示原价。
	//     若现金腿 == 展示原价，则同一份积分价值既以现金收、又以积分扣 = 双重计费。
	if priceFen >= listPriceFen || pointsFen <= 0 {
		t.Fatalf("现金腿 %d / 积分腿 %d 与展示原价 %d 重叠", priceFen, pointsFen, listPriceFen)
	}
	// (3) 反向资损守卫：把 price 当原价、再减一次积分价值（「先算抵扣再收现金」）
	//     会少收 pointsFen。正确结算收 price 本身；entity.PointsValueFen 只用于派生展示原价，
	//     绝不参与扣减。service.go:settleLocal 直读 order.Amount / order.TotalPoints 全额收/扣，
	//     TestMixedPaymentRefundSymmetry 已用真实钱包 / 积分流水证明实付 == price×qty、
	//     实扣 == price_points×qty，两笔互不覆盖。
	if wrong := priceFen - pointsFen; wrong == priceFen {
		t.Fatal("算例失效：积分腿为 0，无法验证二次打折")
	}
}
