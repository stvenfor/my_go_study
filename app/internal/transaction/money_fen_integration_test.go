//go:build integration

// money_fen_integration_test.go 真库用例：transactions 金额口径收敛到整数分（harden-h6b）。
//
// 与 mall/points 那几份集成测试的**关键区别**：这里刻意不用 AutoMigrate 建表。
// 本 Slice 的验收第 1 条要求 schema 由 migrations/ 提供，所以库必须先跑迁移链，
// 用例只做「形状断言 + 行为断言」；库没迁移好就直接 Fail（不是 Skip 成绿）。
//
// 跑法（scratch 库，绝不连业务库 my_go_study）：
//
//	./bin/migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/p06b_it?sslmode=disable" up
//	go -C app test -tags=integration ./internal/transaction/ -run MoneyUnit -count=1
//
// 库名 env P06B_PG_DBNAME（默认 p06b_it）。
package transaction

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"testing"

	"common/config"
	"core/database"
	"gorm.io/gorm"
	"model/entity"
)

const itUserEmailPrefix = "h6b-it@"

func openMoneyFenDB(t *testing.T) *gorm.DB {
	t.Helper()
	port, _ := strconv.Atoi(envOrTest("DATABASE_PORT", "5432"))
	dbName := envOrTest("P06B_PG_DBNAME", "p06b_it")
	db, err := database.NewPostgres(config.DatabaseConfig{
		Host: envOrTest("DATABASE_HOST", "127.0.0.1"), Port: port,
		User: envOrTest("DATABASE_USER", "postgres"), Password: envOrTest("DATABASE_PASSWORD", "postgres"),
		DBName: dbName, SSLMode: "disable",
		MaxOpenConns: 4, MaxIdleConns: 2, ConnMaxLifetimeMinutes: 5,
	})
	if err != nil {
		t.Fatalf("连不上 scratch 库 %s：%v（先建库并跑 make migrate-up 等价的 migrate up）", dbName, err)
	}
	// 形状断言：列必须来自迁移（bigint + NOT NULL + 非负 CHECK），不是 AutoMigrate 的可空无约束列。
	var dataType, isNullable string
	if err := db.Raw(`SELECT data_type, is_nullable FROM information_schema.columns
		WHERE table_name = 'transactions' AND column_name = 'amount_fen'`).
		Row().Scan(&dataType, &isNullable); err != nil {
		t.Fatalf("transactions.amount_fen 列不存在（scratch 库 %s 还没跑本 Slice 的迁移）: %v", dbName, err)
	}
	if dataType != "bigint" || isNullable != "NO" {
		t.Fatalf("amount_fen 形状不符：type=%s nullable=%s，期望 bigint/NO（这是 AutoMigrate 建出来的形状，迁移没生效）",
			dataType, isNullable)
	}
	var checks int64
	if err := db.Raw(`SELECT count(*) FROM pg_constraint c
		JOIN pg_class r ON r.oid = c.conrelid
		WHERE r.relname = 'transactions' AND c.contype = 'c'
		  AND pg_get_constraintdef(c.oid) LIKE '%amount_fen >= 0%'`).
		Scan(&checks).Error; err != nil {
		t.Fatalf("查 CHECK 约束失败: %v", err)
	}
	if checks == 0 {
		t.Fatal("缺 CHECK (amount_fen >= 0)：说明库是 AutoMigrate 建的，不是 migrations/ 建的")
	}
	// 旧列必须还在（本轮「不做」第 4 条：不删列）。
	var oldCols int64
	if err := db.Raw(`SELECT count(*) FROM information_schema.columns
		WHERE table_name = 'transactions' AND column_name = 'amount'`).Scan(&oldCols).Error; err != nil {
		t.Fatalf("查旧列失败: %v", err)
	}
	if oldCols != 1 {
		t.Fatal("旧列 amount 不见了：本 Slice 明令不删列")
	}
	return db
}

// newItUser 建一个只服务本用例的 users 行（transactions.user_id 有外键）。
func newItUser(t *testing.T, db *gorm.DB) string {
	t.Helper()
	uid := fmt.Sprintf("00000000-0000-4000-8000-%012d", os.Getpid()%1000000000000)
	res := db.Exec(`INSERT INTO users (user_id, email, password_hash, user_name)
		VALUES (?, ?, 'it', 'h6b-it') ON CONFLICT (user_id) DO NOTHING`,
		uid, itUserEmailPrefix+uid[:8])
	if res.Error != nil {
		t.Fatalf("建测试用户失败: %v", res.Error)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM transactions WHERE user_id = ?`, uid)
		db.Exec(`DELETE FROM users WHERE user_id = ?`, uid)
	})
	return uid
}

// TestMoneyUnitCreateReadAggregate 0.1 + 0.2 记两笔，按分聚合必须等于 0.30。
func TestMoneyUnitCreateReadAggregate(t *testing.T) {
	db := openMoneyFenDB(t)
	repo := NewPostgresRepository(db)
	uid := newItUser(t, db)
	ctx := context.Background()

	for _, yuan := range []float64{0.1, 0.2} {
		if _, err := repo.Create(ctx, "", uid, entity.CreateTransactionInput{
			Type: "expense", Category: "GATE", Amount: yuan, Date: "2026-01-01",
		}); err != nil {
			t.Fatalf("Create(%v) 失败: %v", yuan, err)
		}
	}

	items, total, err := repo.ListPage(ctx, "", uid, entity.TransactionFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListPage: %v", err)
	}
	if total != 2 {
		t.Fatalf("期望 2 行，实得 %d", total)
	}
	var sumFen int64
	for _, it := range items {
		if it.AmountFen == nil {
			t.Fatalf("id=%d 的 amount_fen 为 NULL：写路径双写没生效", it.ID)
		}
		sumFen += *it.AmountFen
	}
	if sumFen != 30 {
		t.Fatalf("分聚合 = %d，期望 30", sumFen)
	}
	if got := entity.YuanFromAmountFen(sumFen); got != 0.3 {
		t.Fatalf("分聚合换元 = %v，期望 0.3", got)
	}
	// 对照：同一批金额按旧口径（float 元）运行期累加，得到的是 0.30000000000000004。
	var sumYuan float64
	for _, v := range []float64{0.1, 0.2} {
		sumYuan += v
	}
	if sumYuan == 0.3 {
		t.Fatal("对照组失效：float 累加竟然精确等于 0.3，本用例的对比失去意义")
	}

	// 双写一致性：两列必须互等（旧列 amount、新列 amount_fen）。
	var drift int64
	if err := db.Raw(`SELECT count(*) FROM transactions WHERE user_id = ?
		AND amount_fen IS DISTINCT FROM round(amount * 100)::bigint`, uid).Scan(&drift).Error; err != nil {
		t.Fatalf("漂移查询失败: %v", err)
	}
	if drift != 0 {
		t.Fatalf("有 %d 行 amount 与 amount_fen 不一致（双写漂移）", drift)
	}

	// 读回的元必须仍带两位小数口径，且 JSON 字段名不变（amount 存在、amount_fen 不外泄）。
	item, err := repo.GetByID(ctx, "", uid, items[0].ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if item.Amount != entity.YuanFromAmountFen(*item.AmountFen) {
		t.Fatalf("GetByID 读回的 amount=%v 未按 amount_fen=%d 还原", item.Amount, *item.AmountFen)
	}
}

// TestMoneyUnitRejectsSubCent 超过两位小数的入参必须被拒，且不留半行数据。
func TestMoneyUnitRejectsSubCent(t *testing.T) {
	db := openMoneyFenDB(t)
	repo := NewPostgresRepository(db)
	uid := newItUser(t, db)
	ctx := context.Background()

	_, err := repo.Create(ctx, "", uid, entity.CreateTransactionInput{
		Type: "expense", Category: "GATE", Amount: 1.005, Date: "2026-01-01",
	})
	if !errors.Is(err, ErrAmountPrecision) {
		t.Fatalf("1.005 应返回 ErrAmountPrecision，实得 %v", err)
	}
	_, err = repo.Create(ctx, "", uid, entity.CreateTransactionInput{
		Type: "expense", Category: "GATE", Amount: -1, Date: "2026-01-01",
	})
	if !errors.Is(err, ErrAmountPrecision) {
		t.Fatalf("负数应返回 ErrAmountPrecision（正负由 type 表达），实得 %v", err)
	}

	var n int64
	if err := db.Raw(`SELECT count(*) FROM transactions WHERE user_id = ?`, uid).Scan(&n).Error; err != nil {
		t.Fatalf("计数失败: %v", err)
	}
	if n != 0 {
		t.Fatalf("被拒的写入留下了 %d 行", n)
	}
}

// TestMoneyUnitUpdateDualWrites 改金额必须同时改两列。
func TestMoneyUnitUpdateDualWrites(t *testing.T) {
	db := openMoneyFenDB(t)
	repo := NewPostgresRepository(db)
	uid := newItUser(t, db)
	ctx := context.Background()

	created, err := repo.Create(ctx, "", uid, entity.CreateTransactionInput{
		Type: "expense", Category: "GATE", Amount: 0.10, Date: "2026-01-01",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	newAmount := 0.15
	updated, err := repo.Update(ctx, "", uid, created.ID, entity.UpdateTransactionInput{Amount: &newAmount})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.AmountFen == nil || *updated.AmountFen != 15 {
		t.Fatalf("Update 后 amount_fen=%v，期望 15", updated.AmountFen)
	}
	if updated.Amount != 0.15 {
		t.Fatalf("Update 后 amount=%v，期望 0.15（对外口径仍是元）", updated.Amount)
	}
	bad := 1.005
	if _, err := repo.Update(ctx, "", uid, created.ID, entity.UpdateTransactionInput{Amount: &bad}); !errors.Is(err, ErrAmountPrecision) {
		t.Fatalf("Update 到 1.005 应被拒，实得 %v", err)
	}
	// 被拒后原值不变。
	after, err := repo.GetByID(ctx, "", uid, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if after.AmountFen == nil || *after.AmountFen != 15 {
		t.Fatalf("失败的 Update 改动了 amount_fen=%v", after.AmountFen)
	}
}

// TestMoneyUnitCheckConstraintRejectsNegative 绕过 Go 直接写库也必须被约束拦住。
func TestMoneyUnitCheckConstraintRejectsNegative(t *testing.T) {
	db := openMoneyFenDB(t)
	uid := newItUser(t, db)
	err := db.Exec(`INSERT INTO transactions (user_id, type, category, amount, amount_fen, date)
		VALUES (?, 'income', 'GATE', -1, -100, '2026-01-01')`, uid).Error
	if err == nil {
		t.Fatal("CHECK (amount_fen >= 0) 没拦住负数分")
	}
}

func envOrTest(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// TestMoneyUnitAutoMigrateShapeFallsBack 另一库形态：AutoMigrate 建出来的 amount_fen 是
// **可空、无 CHECK**（与 migrations/ 路径不同形），历史行/绕过 Go 的写入会留下 NULL 分。
// 本用例锁死「NULL 不能被读成 0 元」：读路径必须回落旧列 amount。
//
// 库：P06B_AM_DBNAME（默认 p06b_am，空库起步，只服务本用例）。
func TestMoneyUnitAutoMigrateShapeFallsBack(t *testing.T) {
	port, _ := strconv.Atoi(envOrTest("DATABASE_PORT", "5432"))
	dbName := envOrTest("P06B_AM_DBNAME", "p06b_am")
	db, err := database.NewPostgres(config.DatabaseConfig{
		Host: envOrTest("DATABASE_HOST", "127.0.0.1"), Port: port,
		User: envOrTest("DATABASE_USER", "postgres"), Password: envOrTest("DATABASE_PASSWORD", "postgres"),
		DBName: dbName, SSLMode: "disable",
		MaxOpenConns: 4, MaxIdleConns: 2, ConnMaxLifetimeMinutes: 5,
	})
	if err != nil {
		t.Fatalf("连不上 scratch 库 %s: %v", dbName, err)
	}
	if err := db.AutoMigrate(&entity.Transaction{}); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}

	// 形状必须与迁移路径不同（这条断言是「差异存在」的证据，不是巧合）。
	var nullable string
	if err := db.Raw(`SELECT is_nullable FROM information_schema.columns
		WHERE table_name = 'transactions' AND column_name = 'amount_fen'`).Row().Scan(&nullable); err != nil {
		t.Fatalf("查列失败: %v", err)
	}
	if nullable != "YES" {
		t.Fatalf("该库的 amount_fen 是 NOT NULL（说明它跑过迁移，不是 AutoMigrate 形态），本用例前提不成立")
	}

	uid := "h6b-automigrate-" + strconv.Itoa(os.Getpid())
	t.Cleanup(func() { db.Exec(`DELETE FROM transactions WHERE user_id = ?`, uid) })
	if err := db.Exec(`INSERT INTO transactions (user_id, type, category, amount, date)
		VALUES (?, 'expense', 'GATE', 88.50, '2026-01-01')`, uid).Error; err != nil {
		t.Fatalf("未回填行插入失败（旧列单写）: %v", err)
	}
	if err := db.Exec(`INSERT INTO transactions (user_id, type, category, amount, amount_fen, date)
		VALUES (?, 'expense', 'GATE', 12.34, 1234, '2026-01-01')`, uid).Error; err != nil {
		t.Fatalf("已回填行插入失败: %v", err)
	}

	repo := NewPostgresRepository(db)
	items, total, err := repo.ListPage(context.Background(), "", uid, entity.TransactionFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListPage: %v", err)
	}
	if total != 2 {
		t.Fatalf("期望 2 行，实得 %d", total)
	}
	byFen := map[bool]float64{}
	for _, it := range items {
		byFen[it.AmountFen == nil] = it.Amount
	}
	if byFen[true] != 88.50 {
		t.Fatalf("amount_fen 为 NULL 的行读回 amount=%v，期望回落旧列 88.50（读成 0 就是丢钱）", byFen[true])
	}
	if byFen[false] != 12.34 {
		t.Fatalf("amount_fen=1234 的行读回 amount=%v，期望 12.34", byFen[false])
	}
}
