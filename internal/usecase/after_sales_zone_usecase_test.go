package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

type memAfterSalesRepo struct {
	nextRecordID      int64
	nextAppointmentID int64
	records           map[int64]*entity.WysAfterSalesRecord
	appointments      map[int64]*entity.WysAfterSalesAppointment
	customers         map[int64]*entity.WysStoreCustomer
}

func newMemAfterSalesRepo() *memAfterSalesRepo {
	return &memAfterSalesRepo{
		nextRecordID:      1,
		nextAppointmentID: 1,
		records:           map[int64]*entity.WysAfterSalesRecord{},
		appointments:      map[int64]*entity.WysAfterSalesAppointment{},
		customers:         map[int64]*entity.WysStoreCustomer{},
	}
}

func (m *memAfterSalesRepo) ListByStore(_ context.Context, storeID int, offset, limit int) ([]entity.WysAfterSalesRecord, int64, error) {
	var all []entity.WysAfterSalesRecord
	for _, row := range m.records {
		if row.StoreID == storeID {
			all = append(all, *row)
		}
	}
	return pageRecords(all, offset, limit)
}

func (m *memAfterSalesRepo) ListByCustomerUser(_ context.Context, customerUserID string, offset, limit int) ([]entity.WysAfterSalesRecord, int64, error) {
	var all []entity.WysAfterSalesRecord
	for _, row := range m.records {
		if row.CustomerUserID != nil && *row.CustomerUserID == customerUserID {
			all = append(all, *row)
		}
	}
	return pageRecords(all, offset, limit)
}

func pageRecords(all []entity.WysAfterSalesRecord, offset, limit int) ([]entity.WysAfterSalesRecord, int64, error) {
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

func (m *memAfterSalesRepo) GetByID(_ context.Context, recordID int64) (*entity.WysAfterSalesRecord, error) {
	row, ok := m.records[recordID]
	if !ok {
		return nil, repository.ErrAfterSalesRecordNotFound
	}
	cp := *row
	return &cp, nil
}

func (m *memAfterSalesRepo) GetStoreCustomer(_ context.Context, storeID int, customerID int64) (*entity.WysStoreCustomer, error) {
	row, ok := m.customers[customerID]
	if !ok || row.StoreID != storeID {
		return nil, repository.ErrAfterSalesBadCustomer
	}
	cp := *row
	return &cp, nil
}

func (m *memAfterSalesRepo) GetAppointment(_ context.Context, appointmentID int64) (*entity.WysAfterSalesAppointment, error) {
	row, ok := m.appointments[appointmentID]
	if !ok {
		return nil, repository.ErrAfterSalesBadAppointment
	}
	cp := *row
	return &cp, nil
}

func (m *memAfterSalesRepo) ListPendingAppointments(_ context.Context, storeID int, today time.Time) ([]entity.WysAfterSalesAppointment, error) {
	day := today.Format("2006-01-02")
	var out []entity.WysAfterSalesAppointment
	for _, row := range m.appointments {
		if row.StoreID != storeID || row.Status != entity.AppointmentStatusPending {
			continue
		}
		if row.AppointmentDate.Format("2006-01-02") < day {
			continue
		}
		out = append(out, *row)
	}
	return out, nil
}

func (m *memAfterSalesRepo) CreateRecord(_ context.Context, row *entity.WysAfterSalesRecord) error {
	if row.AppointmentID != nil {
		appt, ok := m.appointments[*row.AppointmentID]
		if !ok || appt.StoreID != row.StoreID || appt.Status != entity.AppointmentStatusPending {
			return repository.ErrAfterSalesBadAppointment
		}
		for _, r := range m.records {
			if r.AppointmentID != nil && *r.AppointmentID == *row.AppointmentID {
				return repository.ErrAfterSalesAppointmentTaken
			}
		}
		appt.Status = entity.AppointmentStatusDone
	}
	row.RecordID = m.nextRecordID
	m.nextRecordID++
	cp := *row
	m.records[row.RecordID] = &cp
	return nil
}

func (m *memAfterSalesRepo) countPending(storeID int, today time.Time) int {
	rows, _ := m.ListPendingAppointments(context.Background(), storeID, today)
	return len(rows)
}

func TestAfterSalesCreateWithAppointmentMarksDone(t *testing.T) {
	repo := newMemAfterSalesRepo()
	today := time.Date(2026, 9, 23, 0, 0, 0, 0, time.FixedZone("CST", 8*3600))
	repo.appointments[1] = &entity.WysAfterSalesAppointment{
		AppointmentID: 1, StoreID: 1, CustomerName: "王先生",
		AppointmentDate: today, Status: entity.AppointmentStatusPending,
	}
	access := &stubAccessForDeal{
		storeID: 1,
		member:  &entity.WysStoreMember{UserID: "staff1", StoreID: 1, Position: entity.StoreRoleAdvisor},
	}
	uc := NewAfterSalesZoneUsecase(repo, access.asUsecase())
	uc.now = func() time.Time { return today.Add(10 * time.Hour) }

	if n := repo.countPending(1, today); n != 1 {
		t.Fatalf("before create pending=%d", n)
	}
	apptID := int64(1)
	dto, err := uc.Create(context.Background(), "staff1", CreateAfterSalesRecordInput{
		AppointmentID: &apptID,
		CustomerName:  "王先生",
		CustomerPhone: "13900001111",
		ServiceKind:   "maintenance",
		Title:         "首保",
		ServiceDate:   "2026-09-23",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if dto.AppointmentID == nil || *dto.AppointmentID != 1 {
		t.Fatalf("appointment link %+v", dto.AppointmentID)
	}
	if repo.appointments[1].Status != entity.AppointmentStatusDone {
		t.Fatalf("appointment status=%d", repo.appointments[1].Status)
	}
	if n := repo.countPending(1, today); n != 0 {
		t.Fatalf("after create pending=%d", n)
	}
}

func TestAfterSalesCustomerCannotCreate(t *testing.T) {
	repo := newMemAfterSalesRepo()
	access := &stubAccessForDeal{storeID: 1} // no member
	uc := NewAfterSalesZoneUsecase(repo, access.asUsecase())
	_, err := uc.Create(context.Background(), "customer1", CreateAfterSalesRecordInput{
		CustomerName:  "客",
		CustomerPhone: "139",
		ServiceKind:   "0",
		Title:         "修",
		ServiceDate:   "2026-09-23",
	})
	if err != ErrAfterSalesNotMember {
		t.Fatalf("want NotMember got %v", err)
	}
}

func TestAfterSalesMemberListIsolatedByStore(t *testing.T) {
	repo := newMemAfterSalesRepo()
	uid := "cust-a"
	repo.records[1] = &entity.WysAfterSalesRecord{
		RecordID: 1, StoreID: 1, CustomerName: "A", CustomerPhone: "1",
		ServiceKind: entity.ServiceKindRepair, Title: "s1", CreatedBy: "s",
		ServiceDate: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	repo.records[2] = &entity.WysAfterSalesRecord{
		RecordID: 2, StoreID: 2, CustomerName: "B", CustomerPhone: "2",
		CustomerUserID: &uid,
		ServiceKind: entity.ServiceKindRepair, Title: "s2", CreatedBy: "s",
		ServiceDate: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	access := &stubAccessForDeal{
		storeID: 1,
		member:  &entity.WysStoreMember{UserID: "staff1", StoreID: 1},
	}
	uc := NewAfterSalesZoneUsecase(repo, access.asUsecase())
	list, total, canCreate, err := uc.List(context.Background(), "staff1", 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if !canCreate || total != 1 || len(list) != 1 || list[0].RecordID != 1 {
		t.Fatalf("member list %+v total=%d can=%v", list, total, canCreate)
	}
}

func TestAfterSalesCustomerListOwnOnly(t *testing.T) {
	repo := newMemAfterSalesRepo()
	me := "cust-me"
	other := "cust-other"
	repo.records[1] = &entity.WysAfterSalesRecord{
		RecordID: 1, StoreID: 1, CustomerUserID: &me, CustomerName: "me",
		CustomerPhone: "1", ServiceKind: 0, Title: "a", CreatedBy: "s",
		ServiceDate: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	repo.records[2] = &entity.WysAfterSalesRecord{
		RecordID: 2, StoreID: 1, CustomerUserID: &other, CustomerName: "o",
		CustomerPhone: "2", ServiceKind: 0, Title: "b", CreatedBy: "s",
		ServiceDate: time.Now(), CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	access := &stubAccessForDeal{storeID: 0} // no current store → customer path
	uc := NewAfterSalesZoneUsecase(repo, access.asUsecase())
	list, total, canCreate, err := uc.List(context.Background(), me, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if canCreate || total != 1 || len(list) != 1 || list[0].RecordID != 1 {
		t.Fatalf("customer list %+v total=%d can=%v", list, total, canCreate)
	}
}
