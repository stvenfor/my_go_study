package usecase

import (
	"context"
	"errors"
	"testing"

	"model/entity"
)

type stubTxRepo struct{}

func (stubTxRepo) List(context.Context, string, string, entity.TransactionFilter) ([]entity.Transaction, error) {
	return nil, nil
}
func (stubTxRepo) ListPage(context.Context, string, string, entity.TransactionFilter) ([]entity.Transaction, int64, error) {
	return nil, 0, nil
}
func (stubTxRepo) GetByID(context.Context, string, string, int64) (*entity.Transaction, error) {
	return nil, nil
}
func (stubTxRepo) Create(_ context.Context, _, userID string, input entity.CreateTransactionInput) (*entity.Transaction, error) {
	uid := userID
	return &entity.Transaction{ID: 7, UserID: &uid, Type: input.Type, Category: input.Category, Amount: input.Amount, Date: input.Date, Note: input.Note}, nil
}
func (stubTxRepo) Update(_ context.Context, _, userID string, id int64, input entity.UpdateTransactionInput) (*entity.Transaction, error) {
	uid := userID
	t := &entity.Transaction{ID: id, UserID: &uid, Type: "expense", Category: "x", Amount: 1, Date: "2026-01-01"}
	if input.Category != nil {
		t.Category = *input.Category
	}
	return t, nil
}
func (stubTxRepo) Delete(context.Context, string, string, int64) error { return nil }

type mockSearchStore struct {
	indexed   int
	deleted   int
	indexErr  error
	searchQ   string
	searchUID string
}

func (m *mockSearchStore) Index(context.Context, *entity.Transaction) error {
	m.indexed++
	return m.indexErr
}
func (m *mockSearchStore) Delete(_ context.Context, _ int64) error {
	m.deleted++
	return nil
}
func (m *mockSearchStore) Search(_ context.Context, userID, q string, _ int) ([]entity.Transaction, error) {
	m.searchUID = userID
	m.searchQ = q
	uid := userID
	return []entity.Transaction{{ID: 1, UserID: &uid, Type: "expense", Category: "餐饮", Note: &q}}, nil
}

func TestTransactionCreateIndexesBestEffort(t *testing.T) {
	uc := NewTransactionUsecase(stubTxRepo{})
	store := &mockSearchStore{indexErr: errors.New("es down")}
	uc.SetSearchStore(store, nil)

	item, err := uc.Create(context.Background(), "tok", "u1", entity.CreateTransactionInput{
		Type: "expense", Category: "餐饮", Amount: 10, Date: "2026-09-28",
	})
	if err != nil {
		t.Fatalf("create must not fail when ES down: %v", err)
	}
	if item == nil || item.ID != 7 {
		t.Fatalf("item=%v", item)
	}
	if store.indexed != 1 {
		t.Fatalf("indexed=%d", store.indexed)
	}
}

func TestTransactionSearchUsesSessionUser(t *testing.T) {
	uc := NewTransactionUsecase(stubTxRepo{})
	store := &mockSearchStore{}
	uc.SetSearchStore(store, nil)

	items, err := uc.Search(context.Background(), "session-user", "午饭", 10)
	if err != nil {
		t.Fatal(err)
	}
	if store.searchUID != "session-user" || store.searchQ != "午饭" {
		t.Fatalf("store got uid=%q q=%q", store.searchUID, store.searchQ)
	}
	if len(items) != 1 {
		t.Fatalf("len=%d", len(items))
	}
}

func TestTransactionSearchUnavailable(t *testing.T) {
	uc := NewTransactionUsecase(stubTxRepo{})
	_, err := uc.Search(context.Background(), "u", "q", 10)
	if !errors.Is(err, ErrTransactionSearchUnavailable) {
		t.Fatalf("err=%v", err)
	}
}
