package mall

import (
	"context"
	"testing"
	"time"

	"app/internal/points"
	"app/internal/wallet"
	"model/entity"
)

type payOrderRepo struct {
	Repository
	order   *entity.WysMallOrder
	failPay bool
}

func (r *payOrderRepo) GetOrder(_ context.Context, orderID int64) (*entity.WysMallOrder, error) {
	if r.order == nil || r.order.OrderID != orderID {
		return nil, ErrNotFound
	}
	cp := *r.order
	return &cp, nil
}

func (r *payOrderRepo) PayOrderLocal(_ context.Context, orderID int64, buyerUserID string, channel int16) (*entity.MallOrderDetail, error) {
	if r.failPay {
		return nil, ErrStockInsufficient
	}
	r.order.Status = entity.MallOrderPaid
	ch := channel
	r.order.PaymentChannel = &ch
	return &entity.MallOrderDetail{Order: *r.order}, nil
}

func (r *payOrderRepo) CancelPaidOrder(_ context.Context, orderID int64, buyerUserID string) (*entity.WysMallOrder, error) {
	if r.order == nil || r.order.OrderID != orderID || r.order.BuyerUserID != buyerUserID {
		return nil, ErrNotFound
	}
	if r.order.Status != entity.MallOrderPaid {
		return nil, ErrOrderNotCancelable
	}
	r.order.Status = entity.MallOrderCancelled
	cp := *r.order
	return &cp, nil
}

func (r *payOrderRepo) CancelUnpaidOrder(_ context.Context, orderID int64, buyerUserID string) (*entity.WysMallOrder, error) {
	return nil, ErrOrderNotCancelable
}

func TestMallBalancePayAndCancelRefund(t *testing.T) {
	cash := wallet.NewService(newMemCashRepo())
	ctx := context.Background()
	uid := "buyer1"
	_, err := cash.Recharge(ctx, uid, wallet.RechargeInput{Amount: "20.00", Channel: entity.WalletRechargeAlipay})
	if err != nil {
		t.Fatal(err)
	}
	repo := &payOrderRepo{order: &entity.WysMallOrder{
		OrderID: 11, BuyerUserID: uid, Status: entity.MallOrderUnpaid,
		Amount: "10.00", TotalPoints: 0, CreatedAt: time.Now(),
	}}
	uc := NewService(repo, nil, nil, cash)

	_, err = uc.PayOrder(ctx, uid, 11, entity.MallPayBalance)
	if err != nil {
		t.Fatal(err)
	}
	bal, _ := cash.GetBalanceFen(ctx, uid)
	if bal != 1000 {
		t.Fatalf("after pay bal=%d", bal)
	}

	_, err = uc.CancelOrder(ctx, uid, 11)
	if err != nil {
		t.Fatal(err)
	}
	bal, _ = cash.GetBalanceFen(ctx, uid)
	if bal != 2000 {
		t.Fatalf("after cancel refund bal=%d", bal)
	}
}

func TestMallBalancePayInsufficient(t *testing.T) {
	cash := wallet.NewService(newMemCashRepo())
	ctx := context.Background()
	uid := "buyer2"
	repo := &payOrderRepo{order: &entity.WysMallOrder{
		OrderID: 12, BuyerUserID: uid, Status: entity.MallOrderUnpaid,
		Amount: "5.00", CreatedAt: time.Now(),
	}}
	uc := NewService(repo, nil, nil, cash)
	_, err := uc.PayOrder(ctx, uid, 12, entity.MallPayBalance)
	if err != wallet.ErrCashInsufficient {
		t.Fatalf("got %v", err)
	}
}

func TestMallBalancePayRestoresOnLocalFail(t *testing.T) {
	cash := wallet.NewService(newMemCashRepo())
	ctx := context.Background()
	uid := "buyer3"
	_, _ = cash.Recharge(ctx, uid, wallet.RechargeInput{Amount: "10.00", Channel: entity.WalletRechargeWeChat})
	repo := &payOrderRepo{
		order: &entity.WysMallOrder{
			OrderID: 13, BuyerUserID: uid, Status: entity.MallOrderUnpaid,
			Amount: "3.00", CreatedAt: time.Now(),
		},
		failPay: true,
	}
	uc := NewService(repo, nil, nil, cash)
	_, err := uc.PayOrder(ctx, uid, 13, entity.MallPayBalance)
	if err != ErrStockInsufficient {
		t.Fatalf("got %v", err)
	}
	bal, _ := cash.GetBalanceFen(ctx, uid)
	if bal != 1000 {
		t.Fatalf("restored bal=%d", bal)
	}
}

func TestMallRejectsBalanceOnPointsOnly(t *testing.T) {
	cash := wallet.NewService(newMemCashRepo())
	ctx := context.Background()
	uid := "buyer4"
	_, _ = cash.Recharge(ctx, uid, wallet.RechargeInput{Amount: "10.00", Channel: entity.WalletRechargeAlipay})
	repo := &payOrderRepo{order: &entity.WysMallOrder{
		OrderID: 14, BuyerUserID: uid, Status: entity.MallOrderUnpaid,
		Amount: "0.00", TotalPoints: 100, CreatedAt: time.Now(),
	}}
	uc := NewService(repo, nil, nil, cash)
	// points-only forces channel to points; balance channel with needsCNY false uses points path
	// Valid: needsCNY false → channel overwritten to points. Calling with balance still ok for points-only.
	// Spec: points-only must not use balance to pay points price — PayOrder rewrites channel to points.
	_, err := uc.PayOrder(ctx, uid, 14, entity.MallPayBalance)
	// points is nil → ErrPointsInsufficient
	if err != points.ErrPointsInsufficient {
		t.Fatalf("got %v", err)
	}
	bal, _ := cash.GetBalanceFen(ctx, uid)
	if bal != 1000 {
		t.Fatalf("cash should be untouched bal=%d", bal)
	}
}
