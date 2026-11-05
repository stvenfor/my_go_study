//go:build integration

// service_balance_tx_integration_test.go 真库版「余额购买会员」原子性用例（task #12）。
//
// 跑法（与 P0-3 同一个 scratch 库，绝不连业务库 my_go_study）：
//
//	docker exec my_go_study_postgres psql -U postgres -d postgres -c "CREATE DATABASE p03_it"
//	DATABASE_PASSWORD=postgres go -C app test -tags=integration ./internal/membership/ -run TestMembershipBalanceTx -count=1 -v
//
// 为什么值得再写一份真库用例：内存夹具的 Transaction 是快照/还原**假演**原子性，
// 而 ExtendEntitlement 内部本来就自己开了一层 Transaction —— 嵌在外层事务里会落进
// GORM 的 SAVEPOINT，只有真库能证明「内层失败只回滚到保存点、外层整体回滚」这条链路。
package membership

import (
	"context"
	"errors"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"app/internal/wallet"
	"common/config"
	"core/database"
	"gorm.io/gorm"
	"model/entity"
)

var balanceTxTables = []any{
	&entity.WysCashWallet{}, &entity.WysCashLedger{},
	&entity.WysMembershipOrder{}, &entity.WysMembershipEntitlement{},
}

func openBalanceDB(t *testing.T) *gorm.DB {
	t.Helper()
	port, _ := strconv.Atoi(envOrTx("DATABASE_PORT", "5432"))
	db, err := database.NewPostgres(config.DatabaseConfig{
		Host: envOrTx("DATABASE_HOST", "127.0.0.1"), Port: port,
		User: envOrTx("DATABASE_USER", "postgres"), Password: envOrTx("DATABASE_PASSWORD", "postgres"),
		DBName: envOrTx("P03_PG_DBNAME", "p03_it"), SSLMode: "disable",
		MaxOpenConns: 8, MaxIdleConns: 4, ConnMaxLifetimeMinutes: 5,
	})
	if err != nil {
		t.Skipf("连不上测试库（%s）：%v —— 需先建库 p03_it", envOrTx("P03_PG_DBNAME", "p03_it"), err)
	}
	if err := db.AutoMigrate(balanceTxTables...); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	t.Cleanup(func() {
		for i := len(balanceTxTables) - 1; i >= 0; i-- {
			if err := db.Migrator().DropTable(balanceTxTables[i]); err != nil {
				t.Errorf("清理失败 %T: %v", balanceTxTables[i], err)
			}
		}
	})
	return db
}

func envOrTx(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func balanceSvc(db *gorm.DB) *Service {
	return NewService(NewPostgresRepository(db), wallet.NewService(wallet.NewPostgresRepository(db)), nil, nil, nil, "dev")
}

func seedWallet(t *testing.T, db *gorm.DB, uid string, fen int64) {
	t.Helper()
	if err := db.WithContext(context.Background()).Create(&entity.WysCashWallet{
		UserID: uid, BalanceFen: fen, UpdatedAt: time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed 钱包: %v", err)
	}
}

func seedOrder(t *testing.T, db *gorm.DB, orderID int64, uid string, status int16) *entity.WysMembershipOrder {
	t.Helper()
	order := &entity.WysMembershipOrder{
		OrderID: orderID, UserID: uid, Tier: entity.MembershipTierSVIP,
		PlanID: "svip_1m", Channel: entity.MembershipChannelBalance,
		AmountFen: 3000, Status: status, CreatedAt: time.Now(),
	}
	if err := db.WithContext(context.Background()).Create(order).Error; err != nil {
		t.Fatalf("seed 订单: %v", err)
	}
	return order
}

func countRows(t *testing.T, db *gorm.DB, model any) int64 {
	t.Helper()
	var n int64
	if err := db.WithContext(context.Background()).Model(model).Count(&n).Error; err != nil {
		t.Fatalf("count %T: %v", model, err)
	}
	return n
}

// 已支付订单再走一次余额购买：扣款必须在事务里被整体回滚，
// 而不是「扣了钱、MarkOrderPaid 失败、再用 Credit 补一笔」——
// 旧补偿是 `_, _ =` 写的，补失败就是静默吞钱（P0-3 在 mall 域删掉的同一段）。
func TestMembershipBalanceTxRollsBackDebit(t *testing.T) {
	db := openBalanceDB(t)
	ctx := context.Background()
	seedWallet(t, db, "u1", 10000)
	order := seedOrder(t, db, 9001, "u1", entity.MembershipOrderPaid)

	plan, ok := entity.FindMembershipPlan(order.PlanID)
	if !ok {
		t.Fatalf("套餐不存在: %s", order.PlanID)
	}
	_, err := balanceSvc(db).completeBalanceBuyout(ctx, "u1", order, plan)
	if !errors.Is(err, ErrOrderPaid) {
		t.Fatalf("err=%v，期望 ErrOrderPaid", err)
	}

	var bal int64
	if err := db.WithContext(ctx).Model(&entity.WysCashWallet{}).
		Where("user_id = ?", "u1").Pluck("balance_fen", &bal).Error; err != nil {
		t.Fatalf("读余额: %v", err)
	}
	if bal != 10000 {
		t.Errorf("余额=%d，期望 10000（扣款跟着事务回滚）", bal)
	}
	if n := countRows(t, db, &entity.WysCashLedger{}); n != 0 {
		t.Errorf("现金流水 %d 条，期望 0 条（补偿式回滚会留下 2 条：扣 + 补）", n)
	}
	if n := countRows(t, db, &entity.WysMembershipEntitlement{}); n != 0 {
		t.Errorf("权益 %d 条，期望 0 条", n)
	}
}

// 并发对同一笔待支付订单做余额购买：只扣一次钱、只发一次权益。
// 锁序与 mall 一致（先取钱包行锁，再条件更新订单行），后到者拿 ErrOrderPaid 并回滚自己的扣款。
func TestMembershipBalanceTxConcurrentDebtsOnce(t *testing.T) {
	db := openBalanceDB(t)
	ctx := context.Background()
	seedWallet(t, db, "u1", 10000)
	order := seedOrder(t, db, 9002, "u1", entity.MembershipOrderPending)
	plan, _ := entity.FindMembershipPlan(order.PlanID)

	svc := balanceSvc(db)
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = svc.completeBalanceBuyout(ctx, "u1", order, plan)
		}(i)
	}
	wg.Wait()

	if okCount := boolCount(errs); okCount != 1 {
		t.Fatalf("成功 %d 次，期望恰好 1 次：%v", okCount, errs)
	}
	var bal int64
	if err := db.WithContext(ctx).Model(&entity.WysCashWallet{}).
		Where("user_id = ?", "u1").Pluck("balance_fen", &bal).Error; err != nil {
		t.Fatalf("读余额: %v", err)
	}
	if bal != 7000 {
		t.Errorf("余额=%d，期望 7000（只扣一次 3000）", bal)
	}
	if n := countRows(t, db, &entity.WysCashLedger{}); n != 1 {
		t.Errorf("现金流水 %d 条，期望 1 条", n)
	}
	if n := countRows(t, db, &entity.WysMembershipEntitlement{}); n != 1 {
		t.Errorf("权益 %d 条，期望 1 条", n)
	}
}

func boolCount(errs []error) int {
	n := 0
	for _, err := range errs {
		if err == nil {
			n++
		}
	}
	return n
}
