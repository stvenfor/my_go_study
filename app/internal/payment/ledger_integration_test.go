//go:build integration

// ledger_integration_test.go 真库并发核销用例（parity-p0-2 硬门槛①在真 PG 下的口径）。
//
// 跑法（本机 docker PG16 的独立 scratch 库，绝不连业务库 my_go_study）：
//
//	docker exec my_go_study_postgres psql -U postgres -d postgres -c "CREATE DATABASE p02_it"
//	GOPROXY=https://goproxy.cn,direct go -C app test -tags=integration ./internal/payment/ -run TestPostgres -count=1
//
// 表结构用 AutoMigrate 现建：这里验的是唯一索引 + FOR UPDATE 锁行为，
// 迁移保真度由 P0-1 的闸门与 migrations/*.sql 覆盖。
package payment

import (
	"context"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"common/config"
	"core/database"
	"gorm.io/gorm"
	"model/entity"
)

var ledgerTables = []any{&entity.WysPaymentRecord{}, &entity.WysPaymentReconcileDiff{}}

func openLedgerDB(t *testing.T) *gorm.DB {
	t.Helper()
	port, _ := strconv.Atoi(envOrLedger("DATABASE_PORT", "5432"))
	db, err := database.NewPostgres(config.DatabaseConfig{
		Host: envOrLedger("DATABASE_HOST", "127.0.0.1"), Port: port,
		User: envOrLedger("DATABASE_USER", "postgres"), Password: envOrLedger("DATABASE_PASSWORD", "postgres"),
		DBName: envOrLedger("P02_PG_DBNAME", "p02_it"), SSLMode: "disable",
		MaxOpenConns: 8, MaxIdleConns: 4, ConnMaxLifetimeMinutes: 5,
	})
	if err != nil {
		t.Skipf("连不上测试库（%s）：%v —— 需先建库 p02_it", envOrLedger("P02_PG_DBNAME", "p02_it"), err)
	}
	if err := EnsureSchema(db); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}
	t.Cleanup(func() {
		for i := len(ledgerTables) - 1; i >= 0; i-- {
			if err := db.Migrator().DropTable(ledgerTables[i]); err != nil {
				t.Errorf("清理失败 %T: %v", ledgerTables[i], err)
			}
		}
	})
	return db
}

func envOrLedger(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// countingSettler 统计业务核销次数（并发下用 atomic）。
type countingSettler struct{ calls int32 }

func (s *countingSettler) SettleChannelPaid(context.Context, string, string, *entity.WysPaymentRecord) error {
	atomic.AddInt32(&s.calls, 1)
	return nil
}

// TestPostgresConcurrentNotifySettlesOnce：同一回调并发投递 3 次，
// 真库必须只有 1 次核销、业务发货也只 1 次，且 paid_at/channel_trade_no 落库。
func TestPostgresConcurrentNotifySettlesOnce(t *testing.T) {
	db := openLedgerDB(t)
	ctx := context.Background()
	repo := NewPostgresRepository(db)

	created := time.Now().Add(-time.Hour)
	if err := repo.CreatePending(ctx, &entity.WysPaymentRecord{
		OutTradeNo: "P02-CONC-1", UserID: "u-1", Channel: entity.PaymentChannelWeChat,
		AmountFen: 1500, RefType: entity.PaymentRefMallOrder, RefID: "42", CreatedAt: created,
	}); err != nil {
		t.Fatal(err)
	}

	svc := NewService(wechatCfg(), repo)
	settler := &countingSettler{}
	svc.RegisterSettler(entity.PaymentRefMallOrder, settler)
	raw := wechatXML(wechatNotifyVals(map[string]string{"out_trade_no": "P02-CONC-1"}), testMchAPIKey)

	var wg sync.WaitGroup
	errs := make([]error, 3)
	start := make(chan struct{})
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, errs[i] = svc.HandleWechatNotify(ctx, raw)
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("并发回调 %d 报错：%v", i, err)
		}
	}

	var paidCount, rowCount int64
	if err := db.Model(&entity.WysPaymentRecord{}).
		Where("out_trade_no = ? AND status = ?", "P02-CONC-1", entity.PaymentStatusPaid).
		Count(&paidCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&entity.WysPaymentRecord{}).Where("out_trade_no = ?", "P02-CONC-1").
		Count(&rowCount).Error; err != nil {
		t.Fatal(err)
	}
	if rowCount != 1 || paidCount != 1 {
		t.Fatalf("总行数=%d paid 行数=%d，期望 1/1", rowCount, paidCount)
	}
	if got := atomic.LoadInt32(&settler.calls); got != 1 {
		t.Errorf("业务核销 %d 次，期望 1 次", got)
	}
	var rec entity.WysPaymentRecord
	if err := db.Where("out_trade_no = ?", "P02-CONC-1").First(&rec).Error; err != nil {
		t.Fatal(err)
	}
	if rec.ChannelTradeNo == nil || *rec.ChannelTradeNo != "wx-txn-1" {
		t.Errorf("渠道单号未落库：%v", rec.ChannelTradeNo)
	}
	if rec.PaidAt == nil {
		t.Error("paid_at 未落库")
	}
	if len(rec.NotifyRaw) == 0 || string(rec.NotifyRaw) == "{}" {
		t.Error("notify_raw 未保存原始回调报文，事后无法对质渠道")
	}
}

// TestPostgresConcurrentAmountMismatchKeepsPending：金额不符的合法签名并发进来，
// 一条都不能核销成功，流水必须仍为 pending。
func TestPostgresConcurrentAmountMismatchKeepsPending(t *testing.T) {
	db := openLedgerDB(t)
	ctx := context.Background()
	repo := NewPostgresRepository(db)
	if err := repo.CreatePending(ctx, &entity.WysPaymentRecord{
		OutTradeNo: "P02-AMT-1", UserID: "u-1", Channel: entity.PaymentChannelWeChat,
		AmountFen: 1500, RefType: entity.PaymentRefMallOrder, RefID: "43",
		CreatedAt: time.Now().Add(-time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	svc := NewService(wechatCfg(), repo)
	settler := &countingSettler{}
	svc.RegisterSettler(entity.PaymentRefMallOrder, settler)

	raw := wechatXML(wechatNotifyVals(map[string]string{
		"out_trade_no": "P02-AMT-1",
		"total_fee":    "1",
	}), testMchAPIKey)
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := svc.HandleWechatNotify(ctx, raw); err == nil {
				t.Errorf("金额不符却返回成功")
			}
		}()
	}
	wg.Wait()

	var rec entity.WysPaymentRecord
	if err := db.Where("out_trade_no = ?", "P02-AMT-1").First(&rec).Error; err != nil {
		t.Fatal(err)
	}
	if rec.Status != entity.PaymentStatusPending {
		t.Errorf("流水状态=%d，期望 pending", rec.Status)
	}
	if got := atomic.LoadInt32(&settler.calls); got != 0 {
		t.Errorf("金额不符却发货 %d 次", got)
	}
}

// TestPostgresReconcileDiffUnique：对账差异重跑不翻倍（唯一索引 + ON CONFLICT DO NOTHING）。
func TestPostgresReconcileDiffUnique(t *testing.T) {
	db := openLedgerDB(t)
	ctx := context.Background()
	repo := NewPostgresRepository(db)
	// 每轮用全新结构体，避免上一轮回填的主键制造「假唯一」。
	newDiff := func() *entity.WysPaymentReconcileDiff {
		return &entity.WysPaymentReconcileDiff{
			ReconDate: "2026-11-03", Channel: entity.PaymentChannelWeChat,
			OutTradeNo: "P02-DIFF-1", Kind: entity.PaymentDiffAmountMismatch,
		}
	}
	for i := 0; i < 3; i++ {
		if err := repo.CreateDiff(ctx, newDiff()); err != nil {
			t.Fatalf("第 %d 次写入差异失败: %v", i, err)
		}
	}
	var n int64
	if err := db.Model(&entity.WysPaymentReconcileDiff{}).
		Where("recon_date = ? AND out_trade_no = ?", "2026-11-03", "P02-DIFF-1").
		Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("差异行数=%d，期望 1", n)
	}
}
