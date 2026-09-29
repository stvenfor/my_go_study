package deal_invoice

import (
	"context"
	"testing"
	"time"

	"app/internal/testsupport"
	"model/entity"
)

type memDealInvoiceRepo struct {
	nextID    int64
	invoices  map[int64]*entity.WysDealInvoice
	customers map[int64]*entity.WysStoreCustomer
	userName  string
	avatarURL string
}

func newMemDealRepo() *memDealInvoiceRepo {
	return &memDealInvoiceRepo{
		nextID:    1,
		invoices:  map[int64]*entity.WysDealInvoice{},
		customers: map[int64]*entity.WysStoreCustomer{},
		userName:  "东东枪",
		avatarURL: "https://example.com/a.png",
	}
}

func (m *memDealInvoiceRepo) CountStats(_ context.Context, storeID int, uploader string) (entity.DealInvoiceStats, error) {
	var s entity.DealInvoiceStats
	for _, row := range m.invoices {
		if row.StoreID != storeID || row.UploaderUserID != uploader {
			continue
		}
		s.Uploaded++
		switch row.Status {
		case entity.DealInvoicePendingReview:
			s.PendingReview++
		case entity.DealInvoiceApprovedPendingRating, entity.DealInvoiceRated:
			s.Approved++
		case entity.DealInvoiceRejected:
			s.Rejected++
		}
	}
	return s, nil
}

func (m *memDealInvoiceRepo) ListInvoices(
	_ context.Context, storeID int, uploader string, statuses []int16, offset, limit int,
) ([]entity.WysDealInvoice, int64, error) {
	var all []entity.WysDealInvoice
	for _, row := range m.invoices {
		if row.StoreID != storeID || row.UploaderUserID != uploader {
			continue
		}
		if len(statuses) > 0 {
			ok := false
			for _, st := range statuses {
				if row.Status == st {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		all = append(all, *row)
	}
	total := int64(len(all))
	if offset >= len(all) {
		return nil, total, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}

func (m *memDealInvoiceRepo) GetInvoice(_ context.Context, id int64) (*entity.WysDealInvoice, error) {
	row, ok := m.invoices[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *row
	return &cp, nil
}

func (m *memDealInvoiceRepo) CreateInvoice(_ context.Context, row *entity.WysDealInvoice) error {
	row.InvoiceID = m.nextID
	m.nextID++
	cp := *row
	m.invoices[row.InvoiceID] = &cp
	return nil
}

func (m *memDealInvoiceRepo) UpdateInvoice(_ context.Context, row *entity.WysDealInvoice) error {
	if _, ok := m.invoices[row.InvoiceID]; !ok {
		return ErrNotFound
	}
	cp := *row
	m.invoices[row.InvoiceID] = &cp
	return nil
}

func (m *memDealInvoiceRepo) GetCustomer(_ context.Context, id int64) (*entity.WysStoreCustomer, error) {
	c, ok := m.customers[id]
	if !ok {
		return nil, ErrBadCustomer
	}
	cp := *c
	return &cp, nil
}

func (m *memDealInvoiceRepo) ListCustomers(
	_ context.Context, storeID int, _ string, offset, limit int,
) ([]entity.WysStoreCustomer, int64, error) {
	var all []entity.WysStoreCustomer
	for _, c := range m.customers {
		if c.StoreID == storeID {
			all = append(all, *c)
		}
	}
	total := int64(len(all))
	if offset >= len(all) {
		return nil, total, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}

func (m *memDealInvoiceRepo) GetUserBrief(_ context.Context, _ string) (string, string, error) {
	return m.userName, m.avatarURL, nil
}

func TestDealInvoiceSummaryAndListFilter(t *testing.T) {
	repo := newMemDealRepo()
	repo.customers[1] = &entity.WysStoreCustomer{
		CustomerID: 1, StoreID: 1, DisplayName: "小张女士", Phone: "13812345678",
	}
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	repo.invoices[1] = &entity.WysDealInvoice{
		InvoiceID: 1, StoreID: 1, UploaderUserID: "u1",
		CustomerID: 1, CustomerPhone: "13812345678", CustomerName: "小张女士",
		Status: entity.DealInvoicePendingReview, SubmittedAt: now,
	}
	repo.invoices[2] = &entity.WysDealInvoice{
		InvoiceID: 2, StoreID: 1, UploaderUserID: "u1",
		CustomerID: 1, CustomerPhone: "13812345678", CustomerName: "小张女士",
		Status: entity.DealInvoiceRejected, SubmittedAt: now,
	}
	repo.invoices[3] = &entity.WysDealInvoice{
		InvoiceID: 3, StoreID: 1, UploaderUserID: "other",
		CustomerID: 1, CustomerPhone: "13812345678", CustomerName: "小张女士",
		Status: entity.DealInvoicePendingReview, SubmittedAt: now,
	}

	stub := &testsupport.AccessStub{
		StoreID: 1,
		Store:   &entity.WysStore{StoreID: 1, Name: "[4S]北京沃德龙鼎吉利"},
		Member:  &entity.WysStoreMember{UserID: "u1", StoreID: 1, Position: entity.StoreRoleManager},
	}
	uc := NewService(repo, stub.Service())
	uc.now = func() time.Time { return now }

	sum, err := uc.Summary(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if sum.DisplayName != "东东枪" || sum.PositionLabel != "销售经理" {
		t.Fatalf("summary persona=%+v", sum)
	}
	if sum.Stats.Uploaded != 2 || sum.Stats.PendingReview != 1 || sum.Stats.Rejected != 1 {
		t.Fatalf("stats=%+v", sum.Stats)
	}

	list, total, err := uc.List(context.Background(), "u1", "pending_review", 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 || list[0].Status != "pending_review" {
		t.Fatalf("list total=%d len=%d item=%+v", total, len(list), list)
	}
}

func TestDealInvoiceCreateAndResubmit(t *testing.T) {
	repo := newMemDealRepo()
	repo.customers[10] = &entity.WysStoreCustomer{
		CustomerID: 10, StoreID: 1, DisplayName: "王先生", Phone: "13612345678",
	}
	stub := &testsupport.AccessStub{StoreID: 1, Store: &entity.WysStore{StoreID: 1, Name: "店"}}
	uc := NewService(repo, stub.Service())
	fixed := time.Date(2026, 9, 22, 15, 0, 0, 0, time.UTC)
	uc.now = func() time.Time { return fixed }

	created, err := uc.Create(context.Background(), "u1", CreateInput{CustomerID: 10})
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != "pending_review" || created.Phone != "13612345678" {
		t.Fatalf("created=%+v", created)
	}

	id, _ := parseInvoiceID(created.InvoiceID)
	row := repo.invoices[id]
	row.Status = entity.DealInvoiceRejected
	reason := "模糊"
	row.RejectReason = &reason

	again, err := uc.Resubmit(context.Background(), "u1", id, ResubmitInput{})
	if err != nil {
		t.Fatal(err)
	}
	if again.Status != "pending_review" || again.RejectReason != nil {
		t.Fatalf("resubmit=%+v", again)
	}

	if _, err := uc.Resubmit(context.Background(), "u1", id, ResubmitInput{}); err != ErrInvalidStatus {
		t.Fatalf("expected invalid status, got %v", err)
	}
}

func TestDealInvoiceNoStore(t *testing.T) {
	repo := newMemDealRepo()
	stub := &testsupport.AccessStub{StoreID: 0}
	uc := NewService(repo, stub.Service())
	if _, err := uc.Summary(context.Background(), "u1"); err != ErrNoStore {
		t.Fatalf("got %v", err)
	}
}
