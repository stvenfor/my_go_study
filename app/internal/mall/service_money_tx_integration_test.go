//go:build integration

// service_money_tx_integration_test.go 真库资金原子性用例（parity-p0-3 硬门槛）。
//
// 跑法（本机 docker PG16，独立 scratch 库，绝不连业务库 my_go_study）：
//
//	docker exec my_go_study_postgres psql -U postgres -d postgres -c "CREATE DATABASE p03_it"
//	GOPROXY=https://goproxy.cn,direct go -C app test -tags=integration ./internal/mall/ -run TestMoneyTx -count=1
//
// 连接：env DATABASE_HOST/PORT/USER/PASSWORD（默认 localhost:5432 postgres/postgres），
// 库名 env P03_PG_DBNAME（默认 p03_it）。清理：每个用例 drop 自己建的那几张表。
// 表结构用 AutoMigrate 现建 —— 这里验的是事务边界，不是迁移保真度（迁移由 P0-1 的闸门覆盖）。
package mall

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"app/internal/points"
	"app/internal/wallet"
	"common/config"
	"core/database"
	"gorm.io/gorm"
	"model/entity"
)

var moneyTxTables = []any{
	&entity.WysCashWallet{}, &entity.WysCashLedger{},
	&entity.WysPointsWallet{}, &entity.WysPointsLedger{},
	&entity.WysMallSKU{}, &entity.WysMallOrder{}, &entity.WysMallOrderItem{},
	&entity.WysMallPayment{}, &entity.WysMallOrderLog{}, &entity.WysMallVirtualCode{},
	&entity.WysMallAuditLog{},
	// 渠道支付流水：pay 包有自己的 EnsureSchema，但这里在同一 scratch 库建表更省事，
	// 且能保证用例结束时和商城表一起被清掉。
	&entity.WysPaymentRecord{}, &entity.WysPaymentReconcileDiff{},
}

func openMoneyDB(t *testing.T) *gorm.DB {
	t.Helper()
	port, _ := strconv.Atoi(envOr("DATABASE_PORT", "5432"))
	db, err := database.NewPostgres(config.DatabaseConfig{
		Host: envOr("DATABASE_HOST", "127.0.0.1"), Port: port,
		User: envOr("DATABASE_USER", "postgres"), Password: envOr("DATABASE_PASSWORD", "postgres"),
		DBName: envOr("P03_PG_DBNAME", "p03_it"), SSLMode: "disable",
		MaxOpenConns: 8, MaxIdleConns: 4, ConnMaxLifetimeMinutes: 5,
	})
	if err != nil {
		t.Skipf("连不上测试库（%s）：%v —— 需先建库 p03_it", envOr("P03_PG_DBNAME", "p03_it"), err)
	}
	if err := db.AutoMigrate(moneyTxTables...); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	t.Cleanup(func() {
		for i := len(moneyTxTables) - 1; i >= 0; i-- {
			if err := db.Migrator().DropTable(moneyTxTables[i]); err != nil {
				t.Errorf("清理失败 %T: %v", moneyTxTables[i], err)
			}
		}
	})
	return db
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// seedBuyer 钱包 1000 分、积分 500；stock 为 SKU 库存（0 表示支付时必然扣减失败）。
func seedBuyer(t *testing.T, db *gorm.DB, uid string, orderID int64, amount string, stock int) {
	t.Helper()
	ctx := context.Background()
	if err := db.WithContext(ctx).Create(&entity.WysCashWallet{UserID: uid, BalanceFen: 1000, UpdatedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.WithContext(ctx).Create(&entity.WysPointsWallet{UserID: uid, Balance: 500, UpdatedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	dely := entity.MallDeliverContentURL
	url := "https://example.invalid/x"
	if err := db.WithContext(ctx).Create(&entity.WysMallSKU{
		SKUID: orderID*100 + 1, ProductID: 1, Title: "实物", Price: amount,
		StockQty: stock, Status: 1, DeliverType: &dely, ContentURL: &url,
	}).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := db.WithContext(ctx).Create(&entity.WysMallOrder{
		OrderID: orderID, OrderNo: fmt.Sprintf("IT%d", orderID), StoreID: 1,
		BuyerUserID: uid, IdempotencyKey: fmt.Sprintf("idem-%d", orderID),
		Status: entity.MallOrderUnpaid, Amount: amount, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.WithContext(ctx).Create(&entity.WysMallOrderItem{
		OrderID: orderID, SKUID: orderID*100 + 1, ProductID: 1,
		Kind: entity.MallKindPhysical, ProductTitle: "实物", Price: amount,
		Qty: 1, LineAmount: amount, CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
}

func newMoneyService(db *gorm.DB) *Service {
	return NewService(NewPostgresRepository(db), nil,
		points.NewService(points.NewPostgresRepository(db), points.NewPointsTaskProgress(db)),
		wallet.NewService(wallet.NewPostgresRepository(db)))
}

func cashFen(t *testing.T, db *gorm.DB, uid string) int64 {
	t.Helper()
	var w entity.WysCashWallet
	if err := db.Where("user_id = ?", uid).First(&w).Error; err != nil {
		t.Fatal(err)
	}
	return w.BalanceFen
}

func cashLedgerRows(t *testing.T, db *gorm.DB, uid string) int64 {
	t.Helper()
	var n int64
	if err := db.Model(&entity.WysCashLedger{}).Where("user_id = ?", uid).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func orderStatus(t *testing.T, db *gorm.DB, orderID int64) int16 {
	t.Helper()
	var o entity.WysMallOrder
	if err := db.Where("order_id = ?", orderID).First(&o).Error; err != nil {
		t.Fatal(err)
	}
	return o.Status
}

// TestMoneyTxRollbackOnOrderFailure：库存不足让改单失败，钱包扣款必须一起回滚。
func TestMoneyTxRollbackOnOrderFailure(t *testing.T) {
	db := openMoneyDB(t)
	ctx := context.Background()
	uid := "p03-rollback"
	seedBuyer(t, db, uid, 9001, "3.00", 0) // stock=0 → PayOrderLocal 必然 ErrStockInsufficient

	svc := newMoneyService(db)
	_, err := svc.PayOrder(ctx, uid, 9001, entity.MallPayBalance)
	if !errors.Is(err, ErrStockInsufficient) {
		t.Fatalf("期望 ErrStockInsufficient，实际 %v", err)
	}
	if got := cashFen(t, db, uid); got != 1000 {
		t.Fatalf("回滚失败：钱包余额 %d，应为 1000", got)
	}
	if got := cashLedgerRows(t, db, uid); got != 0 {
		t.Fatalf("回滚失败：留下了 %d 条现金流水", got)
	}
	if got := orderStatus(t, db, 9001); got != entity.MallOrderUnpaid {
		t.Fatalf("回滚失败：订单状态 %d，应为未支付 %d", got, entity.MallOrderUnpaid)
	}
}

// TestMoneyTxConcurrentPayDebtsOnce：同一订单两个并发支付，只允许扣一次款。
func TestMoneyTxConcurrentPayDebtsOnce(t *testing.T) {
	db := openMoneyDB(t)
	uid := "p03-concurrent"
	seedBuyer(t, db, uid, 9002, "3.00", 5)
	svc := newMoneyService(db)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = svc.PayOrder(context.Background(), uid, 9002, entity.MallPayBalance)
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("并发支付 %d 失败：%v", i, err)
		}
	}
	if got := cashFen(t, db, uid); got != 700 {
		t.Fatalf("双重扣款：余额 %d，应为 700（只扣 300 一次）", got)
	}
	if got := cashLedgerRows(t, db, uid); got != 1 {
		t.Fatalf("双重扣款：现金流水 %d 条，应为 1 条", got)
	}
	var stock entity.WysMallSKU
	if err := db.Where("sku_id = ?", 9002*100+1).First(&stock).Error; err != nil {
		t.Fatal(err)
	}
	if stock.StockQty != 4 {
		t.Fatalf("库存被重复扣减：%d，应为 4", stock.StockQty)
	}
}

// TestMoneyTxCancelRefundsAtomically：取消已支付订单 → 状态与退款同一事务。
func TestMoneyTxCancelRefundsAtomically(t *testing.T) {
	db := openMoneyDB(t)
	ctx := context.Background()
	uid := "p03-cancel"
	seedBuyer(t, db, uid, 9003, "3.00", 5)
	svc := newMoneyService(db)

	if _, err := svc.PayOrder(ctx, uid, 9003, entity.MallPayBalance); err != nil {
		t.Fatal(err)
	}
	if got := cashFen(t, db, uid); got != 700 {
		t.Fatalf("支付后余额 %d，应为 700", got)
	}
	if _, err := svc.CancelOrder(ctx, uid, 9003); err != nil {
		t.Fatal(err)
	}
	if got := cashFen(t, db, uid); got != 1000 {
		t.Fatalf("退款后余额 %d，应回到 1000", got)
	}
	if got := orderStatus(t, db, 9003); got != entity.MallOrderCancelled {
		t.Fatalf("订单状态 %d，应为已取消", got)
	}

	// 退款服务缺失时：取消必须整体失败，不能出现「单已取消、钱没退」。
	broke := NewService(NewPostgresRepository(db), nil, nil, nil)
	seedBuyer(t, db, "p03-cancel2", 9004, "3.00", 5)
	if _, err := svc.PayOrder(ctx, "p03-cancel2", 9004, entity.MallPayBalance); err != nil {
		t.Fatal(err)
	}
	if _, err := broke.CancelOrder(ctx, "p03-cancel2", 9004); !errors.Is(err, ErrRefundUnavailable) {
		t.Fatalf("期望 ErrRefundUnavailable，实际 %v", err)
	}
	if got := orderStatus(t, db, 9004); got != entity.MallOrderPaid {
		t.Fatalf("退款不可用却把单取消了：状态 %d", got)
	}
}
