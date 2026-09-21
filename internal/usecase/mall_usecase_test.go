package usecase_test

import (
	"context"
	"testing"

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

func TestValidMallPaymentChannel(t *testing.T) {
	for _, ch := range []int16{1, 2, 3, 4} {
		if !entity.ValidMallPaymentChannel(ch) {
			t.Fatalf("channel %d should be valid", ch)
		}
	}
	for _, ch := range []int16{0, 5, 9, -1} {
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
	}, nil)
	detail, err := on.GetShelfProduct(context.Background(), 1, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.SKUs) != 1 || detail.SKUs[0].Price != "12.00" || detail.SKUs[0].DeliverType == nil {
		t.Fatalf("offer = %+v", detail.SKUs)
	}

	off := usecase.NewMallUsecase(shelfRepo{
		product: &entity.WysMallProduct{ProductID: 7, StoreID: 1, Status: entity.MallProductOffShelf},
	}, nil)
	if _, err := off.GetShelfProduct(context.Background(), 1, 7); err != usecase.ErrMallNotFound {
		t.Fatalf("off shelf got %v", err)
	}
	if _, err := on.GetShelfProduct(context.Background(), 2, 7); err != usecase.ErrMallNotFound {
		t.Fatalf("wrong store got %v", err)
	}
}

func TestPayOrderRejectsInvalidChannel(t *testing.T) {
	uc := usecase.NewMallUsecase(nil, nil)
	_, err := uc.PayOrder(context.Background(), "u1", 1, 9)
	if err != usecase.ErrMallInvalidChannel {
		t.Fatalf("got %v want ErrMallInvalidChannel", err)
	}
}
