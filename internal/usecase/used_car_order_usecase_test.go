package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

type memUsedCarRepo struct {
	nextID    int64
	orders    map[int64]*entity.WysUsedCarOrder
	customers map[int64]*entity.WysStoreCustomer
	userName  string
	avatarURL string
}

func newMemUsedCarRepo() *memUsedCarRepo {
	return &memUsedCarRepo{
		nextID:    1,
		orders:    map[int64]*entity.WysUsedCarOrder{},
		customers: map[int64]*entity.WysStoreCustomer{},
		userName:  "东东枪",
		avatarURL: "https://example.com/a.png",
	}
}

func (m *memUsedCarRepo) CountStats(_ context.Context, storeID int, uploader string) (entity.UsedCarOrderStats, error) {
	var s entity.UsedCarOrderStats
	for _, row := range m.orders {
		if row.StoreID != storeID || row.UploaderUserID != uploader {
			continue
		}
		s.Submitted++
		switch row.Status {
		case entity.UsedCarOrderPendingReview:
			s.PendingReview++
		case entity.UsedCarOrderApprovedPendingRating, entity.UsedCarOrderRated:
			s.Approved++
		case entity.UsedCarOrderRejected:
			s.Rejected++
		}
	}
	return s, nil
}

func (m *memUsedCarRepo) CountPendingByStore(_ context.Context, storeID int) (int64, error) {
	var n int64
	for _, row := range m.orders {
		if row.StoreID == storeID && row.Status == entity.UsedCarOrderPendingReview {
			n++
		}
	}
	return n, nil
}

func (m *memUsedCarRepo) ListOrders(
	_ context.Context, storeID int, uploader string, f repository.UsedCarOrderListFilter, offset, limit int,
) ([]entity.WysUsedCarOrder, int64, error) {
	var all []entity.WysUsedCarOrder
	for _, row := range m.orders {
		if row.StoreID != storeID || row.UploaderUserID != uploader {
			continue
		}
		if len(f.Statuses) > 0 {
			ok := false
			for _, st := range f.Statuses {
				if row.Status == st {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		if f.FilterKind && row.Kind != f.Kind {
			continue
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

func (m *memUsedCarRepo) GetOrder(_ context.Context, id int64) (*entity.WysUsedCarOrder, error) {
	row, ok := m.orders[id]
	if !ok {
		return nil, repository.ErrUsedCarOrderNotFound
	}
	cp := *row
	return &cp, nil
}

func (m *memUsedCarRepo) CreateOrder(_ context.Context, row *entity.WysUsedCarOrder) error {
	if row.OrderID == 0 {
		row.OrderID = m.nextID
		m.nextID++
	}
	cp := *row
	m.orders[row.OrderID] = &cp
	return nil
}

func (m *memUsedCarRepo) GetCustomer(_ context.Context, id int64) (*entity.WysStoreCustomer, error) {
	c, ok := m.customers[id]
	if !ok {
		return nil, repository.ErrUsedCarOrderBadCustomer
	}
	cp := *c
	return &cp, nil
}

func (m *memUsedCarRepo) ListCustomers(
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

func (m *memUsedCarRepo) GetUserBrief(_ context.Context, _ string) (string, string, error) {
	return m.userName, m.avatarURL, nil
}

func TestUsedCarOrderSummaryListAndCreate(t *testing.T) {
	repo := newMemUsedCarRepo()
	repo.customers[1] = &entity.WysStoreCustomer{
		CustomerID: 1, StoreID: 1, DisplayName: "小张女士", Phone: "13812345678",
	}
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	repo.orders[1] = &entity.WysUsedCarOrder{
		OrderID: 1, StoreID: 1, UploaderUserID: "u1", Kind: entity.UsedCarKindTradeIn,
		CustomerID: 1, CustomerPhone: "13812345678", CustomerName: "小张女士",
		VehicleModel: "凯美瑞", PlateNo: "京A1", VIN: "VIN1", MileageKm: 1, ModelYear: 2020, Amount: 1,
		Status: entity.UsedCarOrderPendingReview, SubmittedAt: now,
	}
	repo.orders[2] = &entity.WysUsedCarOrder{
		OrderID: 2, StoreID: 1, UploaderUserID: "u1", Kind: entity.UsedCarKindConsign,
		CustomerID: 1, CustomerPhone: "13812345678", CustomerName: "小张女士",
		VehicleModel: "宝马", PlateNo: "京A2", VIN: "VIN2", MileageKm: 1, ModelYear: 2020, Amount: 1,
		Status: entity.UsedCarOrderRejected, SubmittedAt: now,
	}
	repo.orders[3] = &entity.WysUsedCarOrder{
		OrderID: 3, StoreID: 1, UploaderUserID: "other", Kind: entity.UsedCarKindPurchase,
		CustomerID: 1, CustomerPhone: "13812345678", CustomerName: "小张女士",
		VehicleModel: "帕萨特", PlateNo: "京A3", VIN: "VIN3", MileageKm: 1, ModelYear: 2020, Amount: 1,
		Status: entity.UsedCarOrderPendingReview, SubmittedAt: now,
	}

	access := &stubAccessForDeal{
		storeID: 1,
		store:   &entity.WysStore{StoreID: 1, Name: "[4S]北京沃德龙鼎吉利"},
		member:  &entity.WysStoreMember{UserID: "u1", StoreID: 1, Position: entity.StoreRoleManager},
	}
	uc := NewUsedCarOrderUsecase(repo, access.asUsecase())
	uc.now = func() time.Time { return now }

	sum, err := uc.Summary(context.Background(), "u1")
	if err != nil {
		t.Fatal(err)
	}
	if sum.Stats.Submitted != 2 || sum.Stats.PendingReview != 1 || sum.Stats.Rejected != 1 {
		t.Fatalf("stats=%+v", sum.Stats)
	}

	list, total, err := uc.List(context.Background(), "u1", "pending_review", "trade_in", 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 || list[0].Kind != "trade_in" || list[0].AmountLabel != "补差价" {
		t.Fatalf("list total=%d item=%+v", total, list)
	}

	got, err := uc.Get(context.Background(), "u1", 1)
	if err != nil || got.VehicleModel != "凯美瑞" {
		t.Fatalf("get=%+v err=%v", got, err)
	}
	if _, err := uc.Get(context.Background(), "u1", 3); err != ErrUsedCarOrderNotFound {
		t.Fatalf("expected not found for peer order, got %v", err)
	}

	created, err := uc.Create(context.Background(), "u1", CreateUsedCarOrderInput{
		Kind: "purchase", CustomerID: 1,
		VehicleModel: "雅阁", PlateNo: "京D1", VIN: "VINNEW",
		MileageKm: 10000, ModelYear: 2019, Amount: 88000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Status != "pending_review" || created.Kind != "purchase" || created.AmountLabel != "收车价" {
		t.Fatalf("created=%+v", created)
	}

	if _, err := uc.Create(context.Background(), "u1", CreateUsedCarOrderInput{
		Kind: "nope", CustomerID: 1, VehicleModel: "x", PlateNo: "y", VIN: "z",
		MileageKm: 1, ModelYear: 2020, Amount: 1,
	}); err == nil {
		t.Fatal("expected bad kind")
	}
	if _, err := uc.Create(context.Background(), "u1", CreateUsedCarOrderInput{
		Kind: "trade_in", CustomerID: 1, VehicleModel: "", PlateNo: "y", VIN: "z",
		MileageKm: 1, ModelYear: 2020, Amount: 1,
	}); err == nil {
		t.Fatal("expected bad vehicle")
	}
}

func TestUsedCarOrderNoStore(t *testing.T) {
	repo := newMemUsedCarRepo()
	access := &stubAccessForDeal{storeID: 0}
	uc := NewUsedCarOrderUsecase(repo, access.asUsecase())
	if _, err := uc.Summary(context.Background(), "u1"); err != ErrUsedCarOrderNoStore {
		t.Fatalf("got %v", err)
	}
}
