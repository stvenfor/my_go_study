package usecase_test

import (
	"context"
	"sync"
	"testing"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

type memCashRepo struct {
	mu      sync.Mutex
	balance map[string]int64
	ledger  []entity.WysCashLedger
	cards   map[int64]entity.WysBankCard
	nextID  int64
}

func newMemCashRepo() *memCashRepo {
	return &memCashRepo{
		balance: map[string]int64{},
		cards:   map[int64]entity.WysBankCard{},
		nextID:  1,
	}
}

func (m *memCashRepo) GetBalanceFen(_ context.Context, userID string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.balance[userID], nil
}

func (m *memCashRepo) Credit(_ context.Context, userID string, deltaFen int64, reason, refID string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if deltaFen <= 0 {
		return 0, repository.ErrCashInvalidAmount
	}
	m.balance[userID] += deltaFen
	m.ledger = append(m.ledger, entity.WysCashLedger{
		LedgerID: int64(len(m.ledger) + 1), UserID: userID, DeltaFen: deltaFen,
		BalanceFen: m.balance[userID], Reason: reason, RefID: refID,
	})
	return m.balance[userID], nil
}

func (m *memCashRepo) Debit(_ context.Context, userID string, deltaFen int64, reason, refID string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if deltaFen <= 0 {
		return 0, repository.ErrCashInvalidAmount
	}
	if m.balance[userID] < deltaFen {
		return 0, repository.ErrCashInsufficient
	}
	m.balance[userID] -= deltaFen
	m.ledger = append(m.ledger, entity.WysCashLedger{
		LedgerID: int64(len(m.ledger) + 1), UserID: userID, DeltaFen: -deltaFen,
		BalanceFen: m.balance[userID], Reason: reason, RefID: refID,
	})
	return m.balance[userID], nil
}

func (m *memCashRepo) ListLedger(_ context.Context, userID string, limit, offset int) ([]entity.WysCashLedger, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var all []entity.WysCashLedger
	for i := len(m.ledger) - 1; i >= 0; i-- {
		if m.ledger[i].UserID == userID {
			all = append(all, m.ledger[i])
		}
	}
	total := int64(len(all))
	if offset > len(all) {
		return nil, total, nil
	}
	all = all[offset:]
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all, total, nil
}

func (m *memCashRepo) ListCards(_ context.Context, userID string) ([]entity.WysBankCard, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []entity.WysBankCard
	for _, c := range m.cards {
		if c.UserID == userID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (m *memCashRepo) GetCard(_ context.Context, userID string, cardID int64) (*entity.WysBankCard, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.cards[cardID]
	if !ok || c.UserID != userID {
		return nil, nil
	}
	cp := c
	return &cp, nil
}

func (m *memCashRepo) CreateCard(_ context.Context, card *entity.WysBankCard) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, c := range m.cards {
		if c.UserID == card.UserID {
			n++
		}
	}
	if n == 0 {
		card.IsDefault = true
	}
	card.CardID = m.nextID
	m.nextID++
	if card.IsDefault {
		for id, c := range m.cards {
			if c.UserID == card.UserID {
				c.IsDefault = false
				m.cards[id] = c
			}
		}
	}
	m.cards[card.CardID] = *card
	return nil
}

func (m *memCashRepo) SetDefaultCard(_ context.Context, userID string, cardID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.cards[cardID]
	if !ok || c.UserID != userID {
		return repository.ErrCashCardNotFound
	}
	for id, row := range m.cards {
		if row.UserID == userID {
			row.IsDefault = id == cardID
			m.cards[id] = row
		}
	}
	return nil
}

func (m *memCashRepo) DeleteCard(_ context.Context, userID string, cardID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.cards[cardID]
	if !ok || c.UserID != userID {
		return repository.ErrCashCardNotFound
	}
	delete(m.cards, cardID)
	if c.IsDefault {
		for id, row := range m.cards {
			if row.UserID == userID {
				row.IsDefault = true
				m.cards[id] = row
				break
			}
		}
	}
	return nil
}

func TestCashWallet_RechargeAndBounds(t *testing.T) {
	uc := usecase.NewCashWalletUsecase(newMemCashRepo())
	ctx := context.Background()
	uid := "u1"

	_, err := uc.Recharge(ctx, uid, usecase.RechargeInput{Amount: "0", Channel: entity.WalletRechargeAlipay})
	if err != usecase.ErrCashInvalidAmount {
		t.Fatalf("want invalid amount, got %v", err)
	}
	_, err = uc.Recharge(ctx, uid, usecase.RechargeInput{Amount: "50000.01", Channel: entity.WalletRechargeAlipay})
	if err != usecase.ErrCashInvalidAmount {
		t.Fatalf("want over max, got %v", err)
	}
	res, err := uc.Recharge(ctx, uid, usecase.RechargeInput{Amount: "10.50", Channel: entity.WalletRechargeWeChat})
	if err != nil {
		t.Fatal(err)
	}
	if res.BalanceFen != 1050 || res.Balance != "10.50" {
		t.Fatalf("got %+v", res)
	}
	sum, err := uc.GetSummary(ctx, uid)
	if err != nil || sum.BalanceFen != 1050 {
		t.Fatalf("summary %+v err %v", sum, err)
	}
	page, err := uc.ListLedger(ctx, uid, 10, 0)
	if err != nil || page.Total != 1 || page.Items[0].Reason != entity.CashReasonRecharge {
		t.Fatalf("ledger %+v err %v", page, err)
	}
}

func TestCashWallet_CardDefaultAndRechargeCard(t *testing.T) {
	uc := usecase.NewCashWalletUsecase(newMemCashRepo())
	ctx := context.Background()
	uid := "u2"

	c1, err := uc.BindCard(ctx, uid, usecase.BindCardInput{BankName: "工行", CardLast4: "1234"})
	if err != nil {
		t.Fatal(err)
	}
	if !c1.IsDefault {
		t.Fatal("first card should be default")
	}
	c2, err := uc.BindCard(ctx, uid, usecase.BindCardInput{BankName: "招行", CardLast4: "5678", IsDefault: true})
	if err != nil {
		t.Fatal(err)
	}
	cards, _ := uc.ListCards(ctx, uid)
	var defCount int
	for _, c := range cards {
		if c.IsDefault {
			defCount++
		}
	}
	if defCount != 1 || !c2.IsDefault {
		t.Fatalf("default cards: %+v", cards)
	}
	_, err = uc.BindCard(ctx, uid, usecase.BindCardInput{BankName: "X", CardLast4: "12"})
	if err != usecase.ErrCashInvalidCardLast4 {
		t.Fatalf("want last4 err, got %v", err)
	}
	_, err = uc.Recharge(ctx, uid, usecase.RechargeInput{Amount: "1.00", Channel: entity.WalletRechargeCard})
	if err != usecase.ErrCashCardRequired {
		t.Fatalf("want card required, got %v", err)
	}
	cid := c2.CardID
	res, err := uc.Recharge(ctx, uid, usecase.RechargeInput{
		Amount: "1.00", Channel: entity.WalletRechargeCard, CardID: &cid,
	})
	if err != nil || res.BalanceFen != 100 {
		t.Fatalf("recharge card %+v err %v", res, err)
	}
}

func TestCashWallet_DebitInsufficient(t *testing.T) {
	uc := usecase.NewCashWalletUsecase(newMemCashRepo())
	ctx := context.Background()
	_, err := uc.Debit(ctx, "u3", 100, entity.CashReasonMallPay, "1")
	if err != usecase.ErrCashInsufficient {
		t.Fatalf("got %v", err)
	}
	_, _ = uc.Credit(ctx, "u3", 50, entity.CashReasonRecharge, "x")
	_, err = uc.Debit(ctx, "u3", 100, entity.CashReasonMallPay, "1")
	if err != usecase.ErrCashInsufficient {
		t.Fatalf("got %v", err)
	}
	bal, err := uc.Debit(ctx, "u3", 50, entity.CashReasonMallPay, "1")
	if err != nil || bal != 0 {
		t.Fatalf("bal %d err %v", bal, err)
	}
}

func TestParseYuanToFen(t *testing.T) {
	fen, err := entity.ParseYuanToFen("12.34")
	if err != nil || fen != 1234 {
		t.Fatalf("%d %v", fen, err)
	}
	if entity.FormatFenToYuan(1234) != "12.34" {
		t.Fatal(entity.FormatFenToYuan(1234))
	}
}
