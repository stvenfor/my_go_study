// memwallet_test.go mall 域测试用的内存钱包仓储（测试专用夹具，不进生产二进制）。
//
// 由 app/internal/usecase/memwallet_test.go 随 mall 域切片迁入，供商城余额支付
// 用例构造 wallet.Service。wallet 域自带同名副本（wallet/service_test.go），
// 两份都是测试夹具；错误哨兵统一用 wallet.Err*，与 wallet.Service 的比较口径一致。
package mall

import (
	"context"
	"sync"

	"app/internal/wallet"
	"model/entity"
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

// snapshot/restore 只服务于 payOrderRepo.WithTx：内存仓储没有真事务，
// 用「进闭包前快照、出错还原」模拟原子性，让「失败必回滚」这条在单测里也有断言点。
// 真正的 PG 回滚语义由 integration tag 的用例覆盖（见 service_money_tx_integration_test.go）。
func (m *memCashRepo) snapshot() (map[string]int64, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make(map[string]int64, len(m.balance))
	for k, v := range m.balance {
		cp[k] = v
	}
	return cp, len(m.ledger)
}

func (m *memCashRepo) restore(bal map[string]int64, ledgerLen int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.balance = bal
	if ledgerLen <= len(m.ledger) {
		m.ledger = m.ledger[:ledgerLen]
	}
}

func (m *memCashRepo) Credit(_ context.Context, userID string, deltaFen int64, reason, refID string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if deltaFen <= 0 {
		return 0, wallet.ErrCashInvalidAmount
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
		return 0, wallet.ErrCashInvalidAmount
	}
	if m.balance[userID] < deltaFen {
		return 0, wallet.ErrCashInsufficient
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
		return wallet.ErrCashCardNotFound
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
		return wallet.ErrCashCardNotFound
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
