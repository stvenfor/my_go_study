package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

type shelfRepo struct {
	repository.MallRepository
	product *entity.WysMallProduct
	skus    []entity.WysMallSKU
}

func (s shelfRepo) GetProduct(context.Context, int64) (*entity.WysMallProduct, error) {
	if s.product == nil {
		return nil, repository.ErrMallNotFound
	}
	return s.product, nil
}

func (s shelfRepo) ListOnShelfSKUs(context.Context, int64) ([]entity.WysMallSKU, error) {
	return s.skus, nil
}

func TestMallSKUPaymentMode(t *testing.T) {
	cases := []struct {
		cny    string
		points int64
		want   int16
	}{
		{"12.00", 0, entity.MallPayModeCNY},
		{"0.00", 100, entity.MallPayModePoints},
		{"9.90", 50, entity.MallPayModeMixed},
	}
	for _, c := range cases {
		if got := entity.MallSKUPaymentMode(c.cny, c.points); got != c.want {
			t.Fatalf("%s/%d got %d want %d", c.cny, c.points, got, c.want)
		}
	}
}

func TestValidMallPaymentChannel(t *testing.T) {
	for _, ch := range []int16{1, 2, 3, 4, 5} {
		if !entity.ValidMallPaymentChannel(ch) {
			t.Fatalf("channel %d should be valid", ch)
		}
	}
	for _, ch := range []int16{0, 9, -1} {
		if entity.ValidMallPaymentChannel(ch) {
			t.Fatalf("channel %d should be invalid", ch)
		}
	}
}

func TestGetShelfProductHidesOffShelfAndDeliveryURL(t *testing.T) {
	url := "https://example.com/secret"
	dt := entity.MallDeliverContentURL
	on := usecase.NewMallUsecase(shelfRepo{
		product: &entity.WysMallProduct{ProductID: 7, StoreID: 1, Status: entity.MallProductOnShelf, Title: "卡"},
		skus: []entity.WysMallSKU{{
			SKUID: 9, Title: "虚拟发放", Price: "12.00", ContentURL: &url, DeliverType: &dt,
		}},
	}, nil, nil, nil)
	detail, err := on.GetShelfProduct(context.Background(), 1, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.SKUs) != 1 || detail.SKUs[0].Price != "12.00" || detail.SKUs[0].DeliverType == nil {
		t.Fatalf("offer = %+v", detail.SKUs)
	}

	off := usecase.NewMallUsecase(shelfRepo{
		product: &entity.WysMallProduct{ProductID: 7, StoreID: 1, Status: entity.MallProductOffShelf},
	}, nil, nil, nil)
	if _, err := off.GetShelfProduct(context.Background(), 1, 7); err != usecase.ErrMallNotFound {
		t.Fatalf("off shelf got %v", err)
	}
	if _, err := on.GetShelfProduct(context.Background(), 2, 7); err != usecase.ErrMallNotFound {
		t.Fatalf("wrong store got %v", err)
	}
}

func TestPayOrderRejectsInvalidChannel(t *testing.T) {
	uc := usecase.NewMallUsecase(nil, nil, nil, nil)
	_, err := uc.PayOrder(context.Background(), "u1", 1, 9)
	if err != usecase.ErrMallInvalidChannel {
		t.Fatalf("got %v want ErrMallInvalidChannel", err)
	}
}

func TestMallOrderPayDeadline(t *testing.T) {
	created := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	d := entity.MallOrderPayDeadline(entity.MallOrderUnpaid, created)
	if d == nil || !d.Equal(created.Add(15*time.Minute)) {
		t.Fatalf("deadline = %v", d)
	}
	if entity.MallOrderPayDeadline(entity.MallOrderPaid, created) != nil {
		t.Fatal("paid should have no deadline")
	}
	if !entity.MallOrderPayExpired(entity.MallOrderUnpaid, created, created.Add(15*time.Minute)) {
		t.Fatal("at TTL should be expired")
	}
	if entity.MallOrderPayExpired(entity.MallOrderUnpaid, created, created.Add(14*time.Minute)) {
		t.Fatal("before TTL should not expire")
	}
}

type listOrdersRepo struct {
	repository.MallRepository
	orders map[string][]entity.WysMallOrder
	items  map[int64][]entity.WysMallOrderItem
}

func (r listOrdersRepo) ListOrdersByBuyer(_ context.Context, buyerUserID string, statuses []int16, offset, limit int) ([]entity.WysMallOrder, int64, error) {
	all := r.orders[buyerUserID]
	filtered := make([]entity.WysMallOrder, 0, len(all))
	for _, o := range all {
		if len(statuses) == 0 {
			filtered = append(filtered, o)
			continue
		}
		for _, s := range statuses {
			if o.Status == s {
				filtered = append(filtered, o)
				break
			}
		}
	}
	total := int64(len(filtered))
	if offset >= len(filtered) {
		return nil, total, nil
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], total, nil
}

func (r listOrdersRepo) ListOrderItemsByOrderIDs(_ context.Context, orderIDs []int64) ([]entity.WysMallOrderItem, error) {
	var out []entity.WysMallOrderItem
	for _, id := range orderIDs {
		out = append(out, r.items[id]...)
	}
	return out, nil
}

func (r listOrdersRepo) CancelUnpaidOrder(_ context.Context, orderID int64, buyerUserID string) (*entity.WysMallOrder, error) {
	for uid, list := range r.orders {
		for i := range list {
			o := &list[i]
			if o.OrderID != orderID {
				continue
			}
			if o.BuyerUserID != buyerUserID || o.Status != entity.MallOrderUnpaid {
				return nil, repository.ErrMallOrderNotCancelable
			}
			o.Status = entity.MallOrderCancelled
			r.orders[uid] = list
			cp := *o
			return &cp, nil
		}
	}
	return nil, repository.ErrMallNotFound
}

func (r listOrdersRepo) GetOrder(_ context.Context, orderID int64) (*entity.WysMallOrder, error) {
	for _, list := range r.orders {
		for i := range list {
			if list[i].OrderID == orderID {
				cp := list[i]
				return &cp, nil
			}
		}
	}
	return nil, repository.ErrMallNotFound
}

func TestListOrdersOwnAndStatusFilter(t *testing.T) {
	now := time.Now()
	repo := listOrdersRepo{
		orders: map[string][]entity.WysMallOrder{
			"u1": {
				{OrderID: 1, OrderNo: "A", BuyerUserID: "u1", Status: entity.MallOrderUnpaid, Amount: "10.00", CreatedAt: now},
				{OrderID: 2, OrderNo: "B", BuyerUserID: "u1", Status: entity.MallOrderPaid, Amount: "20.00", CreatedAt: now},
			},
			"u2": {
				{OrderID: 3, OrderNo: "C", BuyerUserID: "u2", Status: entity.MallOrderPaid, Amount: "30.00", CreatedAt: now},
			},
		},
		items: map[int64][]entity.WysMallOrderItem{
			1: {{ItemID: 11, OrderID: 1, ProductTitle: "茶", Qty: 1, LineAmount: "10.00"}},
			2: {{ItemID: 12, OrderID: 2, ProductTitle: "杯", Qty: 2, LineAmount: "20.00"}},
			3: {{ItemID: 13, OrderID: 3, ProductTitle: "他", Qty: 1, LineAmount: "30.00"}},
		},
	}
	uc := usecase.NewMallUsecase(repo, nil, nil, nil)

	all, total, err := uc.ListOrders(context.Background(), "u1", nil, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(all) != 2 {
		t.Fatalf("own all: total=%d len=%d", total, len(all))
	}
	for _, row := range all {
		if row.OrderID == 3 {
			t.Fatal("leaked other buyer order")
		}
		if len(row.Items) == 0 {
			t.Fatalf("missing items on order %d", row.OrderID)
		}
	}
	if all[0].OrderID == 1 && all[0].PayDeadlineAt == nil {
		t.Fatal("unpaid row should expose pay_deadline_at")
	}

	unpaid, total, err := uc.ListOrders(context.Background(), "u1", []int16{entity.MallOrderUnpaid}, 1, 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(unpaid) != 1 || unpaid[0].OrderID != 1 {
		t.Fatalf("unpaid filter = %+v total=%d", unpaid, total)
	}

	if _, _, err := uc.ListOrders(context.Background(), "u1", []int16{9}, 1, 10); err != usecase.ErrMallInvalidStatus {
		t.Fatalf("invalid status got %v", err)
	}
}
