package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

type memHomeTodoRepo struct {
	joins        []entity.WysStoreJoinApplication
	customers    []entity.WysStoreCustomer
	appointments []entity.WysAfterSalesAppointment
	orders       []entity.WysStoreReviewOrder
	nextID       int64
}

func (m *memHomeTodoRepo) CreateJoinApplication(_ context.Context, app *entity.WysStoreJoinApplication) error {
	for _, j := range m.joins {
		if j.StoreID == app.StoreID && j.ApplicantUserID == app.ApplicantUserID && j.Status == entity.JoinStatusPending {
			return repository.ErrJoinApplicationDuplicate
		}
	}
	m.nextID++
	app.ApplicationID = m.nextID
	m.joins = append(m.joins, *app)
	return nil
}

func (m *memHomeTodoRepo) GetJoinApplication(_ context.Context, applicationID int64) (*entity.WysStoreJoinApplication, error) {
	for i := range m.joins {
		if m.joins[i].ApplicationID == applicationID {
			cp := m.joins[i]
			return &cp, nil
		}
	}
	return nil, repository.ErrJoinApplicationNotFound
}

func (m *memHomeTodoRepo) ListPendingJoinApplications(_ context.Context, storeID int) ([]entity.WysStoreJoinApplication, error) {
	var out []entity.WysStoreJoinApplication
	for _, j := range m.joins {
		if j.StoreID == storeID && j.Status == entity.JoinStatusPending {
			out = append(out, j)
		}
	}
	return out, nil
}

func (m *memHomeTodoRepo) CountPendingJoinApplications(_ context.Context, storeID int) (int64, error) {
	var n int64
	for _, j := range m.joins {
		if j.StoreID == storeID && j.Status == entity.JoinStatusPending {
			n++
		}
	}
	return n, nil
}

func (m *memHomeTodoRepo) UpdateJoinApplicationStatus(_ context.Context, applicationID int64, status int16, reviewedBy string) error {
	for i := range m.joins {
		if m.joins[i].ApplicationID == applicationID {
			if m.joins[i].Status != entity.JoinStatusPending {
				return repository.ErrJoinApplicationNotPending
			}
			m.joins[i].Status = status
			m.joins[i].ReviewedBy = &reviewedBy
			return nil
		}
	}
	return repository.ErrJoinApplicationNotPending
}

func (m *memHomeTodoRepo) ListOverdueCustomers(_ context.Context, storeID int, now time.Time) ([]entity.WysStoreCustomer, error) {
	var out []entity.WysStoreCustomer
	for _, c := range m.customers {
		if c.StoreID == storeID && c.NextFollowUpAt != nil && !c.NextFollowUpAt.After(now) {
			out = append(out, c)
		}
	}
	return out, nil
}

func (m *memHomeTodoRepo) CountOverdueCustomers(ctx context.Context, storeID int, now time.Time) (int64, error) {
	rows, err := m.ListOverdueCustomers(ctx, storeID, now)
	return int64(len(rows)), err
}

func (m *memHomeTodoRepo) ListPendingAppointments(_ context.Context, storeID int, today time.Time) ([]entity.WysAfterSalesAppointment, error) {
	day := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	var out []entity.WysAfterSalesAppointment
	for _, a := range m.appointments {
		ad := time.Date(a.AppointmentDate.Year(), a.AppointmentDate.Month(), a.AppointmentDate.Day(), 0, 0, 0, 0, today.Location())
		if a.StoreID == storeID && a.Status == entity.AppointmentStatusPending && !ad.Before(day) {
			out = append(out, a)
		}
	}
	return out, nil
}

func (m *memHomeTodoRepo) CountPendingAppointments(ctx context.Context, storeID int, today time.Time) (int64, error) {
	rows, err := m.ListPendingAppointments(ctx, storeID, today)
	return int64(len(rows)), err
}

func (m *memHomeTodoRepo) ListPendingReviewOrders(_ context.Context, storeID int) ([]entity.WysStoreReviewOrder, error) {
	var out []entity.WysStoreReviewOrder
	for _, o := range m.orders {
		if o.StoreID == storeID && o.Status == entity.ReviewOrderStatusPending {
			out = append(out, o)
		}
	}
	return out, nil
}

func (m *memHomeTodoRepo) CountPendingReviewOrders(ctx context.Context, storeID int) (int64, error) {
	rows, err := m.ListPendingReviewOrders(ctx, storeID)
	return int64(len(rows)), err
}

func (m *memHomeTodoRepo) EnsureSeed(context.Context, int, string) error { return nil }

func (m *memHomeTodoRepo) GetPackingDemoSpec(context.Context, int) (*repository.HomeTodoPackingDemoSpec, error) {
	return nil, nil
}

// --- access mock for home todo ---

type homeTodoAccessRepo struct {
	mockAccessRepo
	currentStore *int
	members      map[string]entity.WysStoreMember // userID|storeID
	storePerms   map[string][]string              // userID -> perms when current store matches
	platformPerm map[string][]string
}

func (m *homeTodoAccessRepo) CurrentStoreID(_ context.Context, _ string) (*int, error) {
	return m.currentStore, nil
}

func (m *homeTodoAccessRepo) GetMember(_ context.Context, userID string, storeID int) (*entity.WysStoreMember, error) {
	key := userID + "|" + itoa(storeID)
	if mem, ok := m.members[key]; ok {
		return &mem, nil
	}
	return nil, repository.ErrAccessMemberNotFound
}

func (m *homeTodoAccessRepo) UpsertMember(_ context.Context, member entity.WysStoreMember) error {
	if m.members == nil {
		m.members = map[string]entity.WysStoreMember{}
	}
	key := member.UserID + "|" + itoa(member.StoreID)
	m.members[key] = member
	return nil
}

func (m *homeTodoAccessRepo) PlatformPermissionCodes(_ context.Context, userID string) ([]string, error) {
	return m.platformPerm[userID], nil
}

func (m *homeTodoAccessRepo) StorePermissionCodes(_ context.Context, userID string, _ int) ([]string, error) {
	return m.storePerms[userID], nil
}

func (m *homeTodoAccessRepo) UserExists(_ context.Context, _ string) (bool, error) { return true, nil }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func TestHomeTodoUsecase_ListTodoCards_OrderAndOmit(t *testing.T) {
	storeID := 1
	past := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	repo := &memHomeTodoRepo{
		joins: []entity.WysStoreJoinApplication{
			{ApplicationID: 1, StoreID: storeID, ApplicantUserID: "u2", Status: entity.JoinStatusPending},
		},
		customers: []entity.WysStoreCustomer{
			{CustomerID: 1, StoreID: storeID, DisplayName: "A", NextFollowUpAt: &past},
		},
		appointments: []entity.WysAfterSalesAppointment{
			{AppointmentID: 1, StoreID: storeID, CustomerName: "B", AppointmentDate: time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC), Status: entity.AppointmentStatusPending},
		},
		orders: []entity.WysStoreReviewOrder{
			{OrderID: 1, StoreID: storeID, Title: "O", Status: entity.ReviewOrderStatusPending},
		},
	}
	accessRepo := &homeTodoAccessRepo{
		currentStore: &storeID,
		members: map[string]entity.WysStoreMember{
			"admin|1": {UserID: "admin", StoreID: 1, Position: 2},
		},
		storePerms: map[string][]string{
			"admin": {entity.PermMemberWrite, entity.PermRoleAssignStore},
		},
	}
	access := NewAccessUsecase(accessRepo)
	uc := NewHomeTodoUsecase(repo, access)
	uc.now = func() time.Time { return time.Date(2026, 9, 22, 10, 0, 0, 0, time.UTC) }

	cards, err := uc.ListTodoCards(context.Background(), "admin")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		entity.TodoTypePartnerPending,
		entity.TodoTypeFollowUpCustomer,
		entity.TodoTypeAfterSalesAppointment,
		entity.TodoTypeOrderPendingReview,
	}
	if len(cards) != len(want) {
		t.Fatalf("len=%d cards=%v", len(cards), cards)
	}
	for i, typ := range want {
		if cards[i].Type != typ {
			t.Fatalf("card[%d]=%s want %s", i, cards[i].Type, typ)
		}
		if cards[i].Count < 1 {
			t.Fatalf("card %s count=%d", typ, cards[i].Count)
		}
	}
}

func TestHomeTodoUsecase_ListTodoCards_NoPermOmitsPartner(t *testing.T) {
	storeID := 1
	repo := &memHomeTodoRepo{
		joins: []entity.WysStoreJoinApplication{
			{ApplicationID: 1, StoreID: storeID, ApplicantUserID: "u2", Status: entity.JoinStatusPending},
		},
	}
	accessRepo := &homeTodoAccessRepo{
		currentStore: &storeID,
		members: map[string]entity.WysStoreMember{
			"staff|1": {UserID: "staff", StoreID: 1},
		},
		storePerms: map[string][]string{
			"staff": {entity.PermProfileRead},
		},
	}
	uc := NewHomeTodoUsecase(repo, NewAccessUsecase(accessRepo))
	cards, err := uc.ListTodoCards(context.Background(), "staff")
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 0 {
		t.Fatalf("want empty, got %#v", cards)
	}
}

func TestHomeTodoUsecase_ApproveJoinCreatesMember(t *testing.T) {
	storeID := 1
	repo := &memHomeTodoRepo{
		joins: []entity.WysStoreJoinApplication{
			{ApplicationID: 9, StoreID: storeID, ApplicantUserID: "newbie", Status: entity.JoinStatusPending},
		},
	}
	accessRepo := &homeTodoAccessRepo{
		currentStore: &storeID,
		members: map[string]entity.WysStoreMember{
			"admin|1": {UserID: "admin", StoreID: 1},
		},
		storePerms: map[string][]string{
			"admin": {entity.PermMemberWrite, entity.PermRoleAssignStore},
		},
	}
	uc := NewHomeTodoUsecase(repo, NewAccessUsecase(accessRepo))
	if err := uc.ApproveJoinApplication(context.Background(), "admin", 9); err != nil {
		t.Fatal(err)
	}
	if _, err := accessRepo.GetMember(context.Background(), "newbie", storeID); err != nil {
		t.Fatalf("expected member: %v", err)
	}
	n, _ := repo.CountPendingJoinApplications(context.Background(), storeID)
	if n != 0 {
		t.Fatalf("pending=%d", n)
	}
}

func TestHomeTodoUsecase_ZeroCountOmitsCard(t *testing.T) {
	storeID := 1
	accessRepo := &homeTodoAccessRepo{
		currentStore: &storeID,
		members:      map[string]entity.WysStoreMember{"admin|1": {UserID: "admin", StoreID: 1}},
		storePerms:   map[string][]string{"admin": {entity.PermMemberWrite, entity.PermRoleAssignStore}},
	}
	uc := NewHomeTodoUsecase(&memHomeTodoRepo{}, NewAccessUsecase(accessRepo))
	cards, err := uc.ListTodoCards(context.Background(), "admin")
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 0 {
		t.Fatalf("want empty got %#v", cards)
	}
}
