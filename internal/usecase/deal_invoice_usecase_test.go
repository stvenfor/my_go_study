package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

type memDealInvoiceRepo struct {
	nextID     int64
	invoices   map[int64]*entity.WysDealInvoice
	customers  map[int64]*entity.WysStoreCustomer
	userName   string
	avatarURL  string
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
		return nil, repository.ErrDealInvoiceNotFound
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
		return repository.ErrDealInvoiceNotFound
	}
	cp := *row
	m.invoices[row.InvoiceID] = &cp
	return nil
}

func (m *memDealInvoiceRepo) GetCustomer(_ context.Context, id int64) (*entity.WysStoreCustomer, error) {
	c, ok := m.customers[id]
	if !ok {
		return nil, repository.ErrDealInvoiceBadCustomer
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

type stubAccessForDeal struct {
	storeID int
	member  *entity.WysStoreMember
	store   *entity.WysStore
}

func (s *stubAccessForDeal) asUsecase() *AccessUsecase {
	return &AccessUsecase{repo: s}
}

func (s *stubAccessForDeal) CreateStore(context.Context, entity.WysStore) error { return nil }
func (s *stubAccessForDeal) GetStore(_ context.Context, storeID int) (*entity.WysStore, error) {
	if s.store != nil && s.store.StoreID == storeID {
		return s.store, nil
	}
	return nil, repository.ErrAccessStoreNotFound
}
func (s *stubAccessForDeal) UpsertMember(context.Context, entity.WysStoreMember) error { return nil }
func (s *stubAccessForDeal) RemoveMember(context.Context, string, int) error           { return nil }
func (s *stubAccessForDeal) GetMember(_ context.Context, userID string, storeID int) (*entity.WysStoreMember, error) {
	if s.member != nil && s.member.UserID == userID && s.member.StoreID == storeID {
		return s.member, nil
	}
	return nil, repository.ErrAccessMemberNotFound
}
func (s *stubAccessForDeal) GetRole(context.Context, string) (*entity.Role, error) { return nil, nil }
func (s *stubAccessForDeal) UserExists(context.Context, string) (bool, error)      { return true, nil }
func (s *stubAccessForDeal) CurrentStoreID(context.Context, string) (*int, error) {
	if s.storeID <= 0 {
		return nil, nil
	}
	id := s.storeID
	return &id, nil
}
func (s *stubAccessForDeal) ClearCurrentStoreIf(context.Context, string, int) error { return nil }
func (s *stubAccessForDeal) PlatformPermissionCodes(context.Context, string) ([]string, error) {
	return nil, nil
}
func (s *stubAccessForDeal) StorePermissionCodes(context.Context, string, int) ([]string, error) {
	return nil, nil
}
func (s *stubAccessForDeal) AssignRole(context.Context, entity.UserRole) error { return nil }
func (s *stubAccessForDeal) RevokeRole(context.Context, string, string, *int) error {
	return nil
}
func (s *stubAccessForDeal) HasRoleAssignment(context.Context, string, string, *int) (bool, error) {
	return false, nil
}
func (s *stubAccessForDeal) CountPlatformAdmins(context.Context) (int, error) { return 0, nil }

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

	access := &stubAccessForDeal{
		storeID: 1,
		store:   &entity.WysStore{StoreID: 1, Name: "[4S]北京沃德龙鼎吉利"},
		member:  &entity.WysStoreMember{UserID: "u1", StoreID: 1, Position: entity.StoreRoleManager},
	}
	uc := NewDealInvoiceUsecase(repo, access.asUsecase())
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
	access := &stubAccessForDeal{storeID: 1, store: &entity.WysStore{StoreID: 1, Name: "店"}}
	uc := NewDealInvoiceUsecase(repo, access.asUsecase())
	fixed := time.Date(2026, 9, 22, 15, 0, 0, 0, time.UTC)
	uc.now = func() time.Time { return fixed }

	created, err := uc.Create(context.Background(), "u1", CreateDealInvoiceInput{CustomerID: 10})
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != "pending_review" || created.Phone != "13612345678" {
		t.Fatalf("created=%+v", created)
	}

	id, _ := ParseDealInvoiceID(created.InvoiceID)
	row := repo.invoices[id]
	row.Status = entity.DealInvoiceRejected
	reason := "模糊"
	row.RejectReason = &reason

	again, err := uc.Resubmit(context.Background(), "u1", id, ResubmitDealInvoiceInput{})
	if err != nil {
		t.Fatal(err)
	}
	if again.Status != "pending_review" || again.RejectReason != nil {
		t.Fatalf("resubmit=%+v", again)
	}

	if _, err := uc.Resubmit(context.Background(), "u1", id, ResubmitDealInvoiceInput{}); err != ErrDealInvoiceInvalidStatus {
		t.Fatalf("expected invalid status, got %v", err)
	}
}

func TestDealInvoiceNoStore(t *testing.T) {
	repo := newMemDealRepo()
	access := &stubAccessForDeal{storeID: 0}
	uc := NewDealInvoiceUsecase(repo, access.asUsecase())
	if _, err := uc.Summary(context.Background(), "u1"); err != ErrDealInvoiceNoStore {
		t.Fatalf("got %v", err)
	}
}
