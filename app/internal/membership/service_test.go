package membership

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"app/internal/wallet"
	"model/entity"
)

type memMembershipRepo struct {
	mu   sync.Mutex
	ents map[string]entity.WysMembershipEntitlement // user|tier
	ords map[int64]entity.WysMembershipOrder
	seq  int64
	// cash 与 wallet 夹具的绑定：夹具的「事务」是快照/还原假演，和 mall 的
	// payOrderRepo.WithTx 同构 —— 真实原子性由 postgres 侧保证，这里只验编排。
	cash *memCashForMembership
	// failExtend 测试开关：让发权益失败，断言扣款跟着回滚。
	failExtend bool
}

func newMemMembershipRepo() *memMembershipRepo {
	return &memMembershipRepo{
		ents: map[string]entity.WysMembershipEntitlement{},
		ords: map[int64]entity.WysMembershipOrder{},
	}
}

func entKey(userID, tier string) string { return userID + "|" + tier }

func (m *memMembershipRepo) ListEntitlements(_ context.Context, userID string) ([]entity.WysMembershipEntitlement, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []entity.WysMembershipEntitlement
	for k, v := range m.ents {
		if len(k) > len(userID) && k[:len(userID)] == userID && k[len(userID)] == '|' {
			out = append(out, v)
		}
	}
	return out, nil
}

func (m *memMembershipRepo) GetEntitlement(_ context.Context, userID, tier string) (*entity.WysMembershipEntitlement, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.ents[entKey(userID, tier)]; ok {
		cp := e
		return &cp, nil
	}
	return nil, nil
}

func (m *memMembershipRepo) ExtendEntitlement(_ context.Context, userID, tier, channel string, months int, now time.Time, huaweiToken, huaweiSubID, appleOriginalTxID string) (*entity.WysMembershipEntitlement, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failExtend {
		return nil, errors.New("mem: extend entitlement failed")
	}
	base := now
	if cur, ok := m.ents[entKey(userID, tier)]; ok && cur.ExpiresAt.After(now) {
		base = cur.ExpiresAt
	}
	row := entity.WysMembershipEntitlement{
		UserID:                     userID,
		Tier:                       tier,
		ExpiresAt:                  base.AddDate(0, months, 0),
		SourceChannel:              channel,
		HuaweiPurchaseToken:        huaweiToken,
		HuaweiSubscriptionID:       huaweiSubID,
		AppleOriginalTransactionID: appleOriginalTxID,
		UpdatedAt:                  now,
	}
	m.ents[entKey(userID, tier)] = row
	return &row, nil
}

func (m *memMembershipRepo) CreateOrder(_ context.Context, order *entity.WysMembershipOrder) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq++
	order.OrderID = m.seq
	m.ords[order.OrderID] = *order
	return nil
}

func (m *memMembershipRepo) GetOrder(_ context.Context, orderID int64) (*entity.WysMembershipOrder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.ords[orderID]
	if !ok {
		return nil, ErrOrderNotFound
	}
	cp := o
	return &cp, nil
}

func (m *memMembershipRepo) MarkOrderPaid(_ context.Context, orderID int64, paidAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.ords[orderID]
	if !ok {
		return ErrOrderNotFound
	}
	if o.Status == entity.MembershipOrderPaid {
		return ErrOrderPaid
	}
	o.Status = entity.MembershipOrderPaid
	o.PaidAt = &paidAt
	m.ords[orderID] = o
	return nil
}

func (m *memMembershipRepo) FindPaidByOutTradeNo(_ context.Context, outTradeNo string) (*entity.WysMembershipOrder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, o := range m.ords {
		if o.OutTradeNo == outTradeNo && o.Status == entity.MembershipOrderPaid {
			cp := o
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *memMembershipRepo) Transaction(ctx context.Context, fn func(context.Context) error) error {
	m.mu.Lock()
	ents := make(map[string]entity.WysMembershipEntitlement, len(m.ents))
	for k, v := range m.ents {
		ents[k] = v
	}
	ords := make(map[int64]entity.WysMembershipOrder, len(m.ords))
	for k, v := range m.ords {
		ords[k] = v
	}
	m.mu.Unlock()

	var bal map[string]int64
	if m.cash != nil {
		bal = m.cash.snapshotBal()
	}
	if err := fn(ctx); err != nil {
		m.mu.Lock()
		m.ents, m.ords = ents, ords
		m.mu.Unlock()
		if m.cash != nil {
			m.cash.restoreBal(bal)
		}
		return err
	}
	return nil
}

type memCashForMembership struct {
	mu  sync.Mutex
	bal map[string]int64
}

func (m *memCashForMembership) snapshotBal() map[string]int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make(map[string]int64, len(m.bal))
	for k, v := range m.bal {
		cp[k] = v
	}
	return cp
}

func (m *memCashForMembership) restoreBal(bal map[string]int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bal = bal
}

func (m *memCashForMembership) GetBalanceFen(_ context.Context, userID string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.bal[userID], nil
}
func (m *memCashForMembership) Credit(_ context.Context, userID string, deltaFen int64, _, _ string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bal[userID] += deltaFen
	return m.bal[userID], nil
}
func (m *memCashForMembership) Debit(_ context.Context, userID string, deltaFen int64, _, _ string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.bal[userID] < deltaFen {
		return 0, wallet.ErrCashInsufficient
	}
	m.bal[userID] -= deltaFen
	return m.bal[userID], nil
}
func (m *memCashForMembership) ListLedger(context.Context, string, int, int) ([]entity.WysCashLedger, int64, error) {
	return nil, 0, nil
}
func (m *memCashForMembership) ListCards(context.Context, string) ([]entity.WysBankCard, error) {
	return nil, nil
}
func (m *memCashForMembership) GetCard(context.Context, string, int64) (*entity.WysBankCard, error) {
	return nil, nil
}
func (m *memCashForMembership) CreateCard(context.Context, *entity.WysBankCard) error { return nil }
func (m *memCashForMembership) SetDefaultCard(context.Context, string, int64) error   { return nil }
func (m *memCashForMembership) DeleteCard(context.Context, string, int64) error       { return nil }

func TestMembershipBalanceBuyoutAndExtend(t *testing.T) {
	repo := newMemMembershipRepo()
	cash := &memCashForMembership{bal: map[string]int64{"u1": 100000}}
	repo.cash = cash
	walletSvc := wallet.NewService(cash)
	uc := NewService(repo, walletSvc, nil, nil, nil, "dev")

	out, err := uc.Buyout(context.Background(), "u1", "svip_1m", "balance")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "paid" || out.Entitlement == nil || !out.Entitlement.Active {
		t.Fatalf("unexpected: %+v", out)
	}
	if cash.bal["u1"] != 100000-3000 {
		t.Fatalf("balance=%d", cash.bal["u1"])
	}

	// 续费：应从当前 expires 再加 1 月
	firstExp := repo.ents[entKey("u1", "svip")].ExpiresAt
	out2, err := uc.Buyout(context.Background(), "u1", "svip_1m", "balance")
	if err != nil {
		t.Fatal(err)
	}
	secondExp := repo.ents[entKey("u1", "svip")].ExpiresAt
	if !secondExp.After(firstExp) {
		t.Fatalf("expected extend %v -> %v", firstExp, secondExp)
	}
	if out2.Entitlement.CTA != "续费" {
		t.Fatalf("cta=%s", out2.Entitlement.CTA)
	}
}

func TestMembershipBalanceBuyoutRollsBackDebit(t *testing.T) {
	repo := newMemMembershipRepo()
	cash := &memCashForMembership{bal: map[string]int64{"u1": 100000}}
	repo.cash = cash
	repo.failExtend = true // 发权益失败：扣款与改单必须一起回滚
	walletSvc := wallet.NewService(cash)
	uc := NewService(repo, walletSvc, nil, nil, nil, "dev")

	if _, err := uc.Buyout(context.Background(), "u1", "svip_1m", "balance"); err == nil {
		t.Fatal("发权益失败却返回成功")
	}
	if cash.bal["u1"] != 100000 {
		t.Fatalf("余额=%d，期望 100000（补偿式回滚已删，只剩事务回滚）", cash.bal["u1"])
	}
	for _, o := range repo.ords {
		if o.Status != entity.MembershipOrderPending {
			t.Fatalf("订单 %d 状态=%d，期望仍为待支付", o.OrderID, o.Status)
		}
	}
	if len(repo.ents) != 0 {
		t.Fatalf("权益已写入 %v，期望回滚后为空", repo.ents)
	}
}

func TestMembershipBalanceInsufficient(t *testing.T) {
	repo := newMemMembershipRepo()
	cash := &memCashForMembership{bal: map[string]int64{"u1": 100}}
	repo.cash = cash
	walletSvc := wallet.NewService(cash)
	uc := NewService(repo, walletSvc, nil, nil, nil, "dev")
	_, err := uc.Buyout(context.Background(), "u1", "svip_1m", "balance")
	if !errors.Is(err, wallet.ErrCashInsufficient) {
		t.Fatalf("got %v", err)
	}
}

func TestMembershipDualTierIndependent(t *testing.T) {
	repo := newMemMembershipRepo()
	cash := &memCashForMembership{bal: map[string]int64{"u1": 1000000}}
	repo.cash = cash
	walletSvc := wallet.NewService(cash)
	uc := NewService(repo, walletSvc, nil, nil, nil, "dev")
	if _, err := uc.Buyout(context.Background(), "u1", "svip_1m", "balance"); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Buyout(context.Background(), "u1", "ai_svip_1m", "balance"); err != nil {
		t.Fatal(err)
	}
	me, err := uc.Me(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	active := 0
	for _, e := range me.Entitlements {
		if e.Active {
			active++
		}
	}
	if active != 2 {
		t.Fatalf("active=%d entitlements=%+v", active, me.Entitlements)
	}
}

func TestMembershipHuaweiDevVerify(t *testing.T) {
	repo := newMemMembershipRepo()
	uc := NewService(repo, nil, nil, nil, nil, "dev")
	ent, err := uc.VerifyHuawei(context.Background(), "u1", HuaweiVerifyInput{
		ProductID:     "wys_svip_1m",
		PurchaseToken: "tok-dev-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ent.Active || ent.Tier != "svip" {
		t.Fatalf("%+v", ent)
	}
	_, err = uc.VerifyHuawei(context.Background(), "u1", HuaweiVerifyInput{
		ProductID:     "unknown_pid",
		PurchaseToken: "tok",
	})
	if !errors.Is(err, ErrHuaweiMismatch) {
		t.Fatalf("got %v", err)
	}
}

func TestMembershipAppleDevVerify(t *testing.T) {
	repo := newMemMembershipRepo()
	uc := NewService(repo, nil, nil, nil, nil, "dev")
	ent, err := uc.VerifyApple(context.Background(), "u1", AppleVerifyInput{
		ProductID:     "wys_svip_1m",
		TransactionID: "tx-dev-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ent.Active || ent.Tier != "svip" {
		t.Fatalf("%+v", ent)
	}
	_, err = uc.VerifyApple(context.Background(), "u1", AppleVerifyInput{
		ProductID:     "unknown",
		TransactionID: "tx",
	})
	if !errors.Is(err, ErrAppleMismatch) {
		t.Fatalf("got %v", err)
	}
}

// 渠道单不能靠客户端回跳就发权益：没有 payment 域（拿不到已核销流水）必须拒绝。
// P0-2 之前这里「调用即 paid」，正是「无流水的已付单」的来源。
func TestMembershipConfirmRequiresPaymentLedger(t *testing.T) {
	repo := newMemMembershipRepo()
	uc := NewService(repo, nil, nil, nil, nil, "dev")
	order := &entity.WysMembershipOrder{
		UserID: "u1", Tier: "svip", PlanID: "svip_1m", Channel: "wechat",
		AmountFen: 3000, Status: entity.MembershipOrderPending, OutTradeNo: "ot1",
		CreatedAt: time.Now(),
	}
	_ = repo.CreateOrder(context.Background(), order)

	if _, err := uc.ConfirmBuyout(context.Background(), "u1", order.OrderID); !errors.Is(err, ErrPaymentNotConfirmed) {
		t.Fatalf("无流水却确认成功，实得 %v", err)
	}
	if ent, err := repo.GetEntitlement(context.Background(), "u1", "svip"); err != nil || ent != nil {
		t.Fatalf("未核销却发了权益：%+v %v", ent, err)
	}
}

// 已核销（订单已置 paid）后重复确认幂等：不再改状态，只回同一份权益。
func TestMembershipConfirmIdempotent(t *testing.T) {
	repo := newMemMembershipRepo()
	uc := NewService(repo, nil, nil, nil, nil, "dev")
	order := &entity.WysMembershipOrder{
		UserID: "u1", Tier: "svip", PlanID: "svip_1m", Channel: "wechat",
		AmountFen: 3000, Status: entity.MembershipOrderPending, OutTradeNo: "ot2",
		CreatedAt: time.Now(),
	}
	_ = repo.CreateOrder(context.Background(), order)
	if err := repo.MarkOrderPaid(context.Background(), order.OrderID, time.Now()); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		out, err := uc.ConfirmBuyout(context.Background(), "u1", order.OrderID)
		if err != nil {
			t.Fatalf("第 %d 次确认失败：%v", i+1, err)
		}
		if out.Status != "paid" {
			t.Fatalf("第 %d 次确认 %+v", i+1, out)
		}
	}
}
