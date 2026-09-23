package usecase_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

type memMembershipRepo struct {
	mu   sync.Mutex
	ents map[string]entity.WysMembershipEntitlement // user|tier
	ords map[int64]entity.WysMembershipOrder
	seq  int64
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
		return nil, repository.ErrMembershipOrderNotFound
	}
	cp := o
	return &cp, nil
}

func (m *memMembershipRepo) MarkOrderPaid(_ context.Context, orderID int64, paidAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.ords[orderID]
	if !ok {
		return repository.ErrMembershipOrderNotFound
	}
	if o.Status == entity.MembershipOrderPaid {
		return repository.ErrMembershipOrderPaid
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

type memCashForMembership struct {
	mu  sync.Mutex
	bal map[string]int64
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
		return 0, repository.ErrCashInsufficient
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
	wallet := usecase.NewCashWalletUsecase(cash)
	uc := usecase.NewMembershipUsecase(repo, wallet, nil, nil, nil, "debug")

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

func TestMembershipBalanceInsufficient(t *testing.T) {
	repo := newMemMembershipRepo()
	cash := &memCashForMembership{bal: map[string]int64{"u1": 100}}
	wallet := usecase.NewCashWalletUsecase(cash)
	uc := usecase.NewMembershipUsecase(repo, wallet, nil, nil, nil, "debug")
	_, err := uc.Buyout(context.Background(), "u1", "svip_1m", "balance")
	if !errors.Is(err, usecase.ErrCashInsufficient) {
		t.Fatalf("got %v", err)
	}
}

func TestMembershipDualTierIndependent(t *testing.T) {
	repo := newMemMembershipRepo()
	cash := &memCashForMembership{bal: map[string]int64{"u1": 1000000}}
	wallet := usecase.NewCashWalletUsecase(cash)
	uc := usecase.NewMembershipUsecase(repo, wallet, nil, nil, nil, "debug")
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
	uc := usecase.NewMembershipUsecase(repo, nil, nil, nil, nil, "debug")
	ent, err := uc.VerifyHuawei(context.Background(), "u1", usecase.HuaweiVerifyInput{
		ProductID:     "wys_svip_1m",
		PurchaseToken: "tok-dev-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ent.Active || ent.Tier != "svip" {
		t.Fatalf("%+v", ent)
	}
	_, err = uc.VerifyHuawei(context.Background(), "u1", usecase.HuaweiVerifyInput{
		ProductID:     "unknown_pid",
		PurchaseToken: "tok",
	})
	if !errors.Is(err, usecase.ErrMembershipHuaweiMismatch) {
		t.Fatalf("got %v", err)
	}
}

func TestMembershipAppleDevVerify(t *testing.T) {
	repo := newMemMembershipRepo()
	uc := usecase.NewMembershipUsecase(repo, nil, nil, nil, nil, "debug")
	ent, err := uc.VerifyApple(context.Background(), "u1", usecase.AppleVerifyInput{
		ProductID:     "wys_svip_1m",
		TransactionID: "tx-dev-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ent.Active || ent.Tier != "svip" {
		t.Fatalf("%+v", ent)
	}
	_, err = uc.VerifyApple(context.Background(), "u1", usecase.AppleVerifyInput{
		ProductID:     "unknown",
		TransactionID: "tx",
	})
	if !errors.Is(err, usecase.ErrMembershipAppleMismatch) {
		t.Fatalf("got %v", err)
	}
}

func TestMembershipConfirmIdempotent(t *testing.T) {
	repo := newMemMembershipRepo()
	uc := usecase.NewMembershipUsecase(repo, nil, nil, nil, nil, "debug")
	order := &entity.WysMembershipOrder{
		UserID: "u1", Tier: "svip", PlanID: "svip_1m", Channel: "wechat",
		AmountFen: 3000, Status: entity.MembershipOrderPending, OutTradeNo: "ot1",
		CreatedAt: time.Now(),
	}
	_ = repo.CreateOrder(context.Background(), order)
	out1, err := uc.ConfirmBuyout(context.Background(), "u1", order.OrderID)
	if err != nil {
		t.Fatal(err)
	}
	out2, err := uc.ConfirmBuyout(context.Background(), "u1", order.OrderID)
	if err != nil {
		t.Fatal(err)
	}
	if out1.Status != "paid" || out2.Status != "paid" {
		t.Fatalf("%+v %+v", out1, out2)
	}
}
