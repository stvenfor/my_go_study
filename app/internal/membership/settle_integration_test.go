//go:build integration

// settle_integration_test.go h8 真库用例（F2 / 风险 R5）：给 membership.Service 实现
// payment.Settler 后，回调核销即发权益，与 ConfirmBuyout 双链并发只加一个周期。
//
// 跑法（scratch 库，绝不连业务库 my_go_study）：
//
//	docker exec my_go_study_postgres psql -U postgres -d postgres -c "CREATE DATABASE p08_it"
//	DATABASE_PASSWORD=postgres P08_PG_DBNAME=p08_it go -C app test -tags=integration \
//	  ./internal/membership/ -run 'Settle|Buyout' -count=1 -v
//
// 为什么必须真库：内存夹具的 Transaction 是快照/还原假演，无法证明
// 「核销流水置 paid 与发权益在同一 ctx 事务」、以及 MarkOrderPaid 的
// `WHERE status=pending` 条件更新在两条链并发下的行锁互斥。
package membership

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"app/internal/payment"
	"common/config"
	"core/database"
	"gorm.io/gorm"
	"model/entity"
)

const (
	h8MchAPIKey = "0123456789abcdef0123456789abcdef"
	h8WechatApp = "wx0000000000000001"
	h8MchID     = "1900000109"
)

var h8Tables = []any{
	&entity.WysMembershipOrder{},
	&entity.WysMembershipEntitlement{},
	&entity.WysPaymentRecord{},
}

func h8Env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// openSettleDB 连独立 scratch 库 p08_it（与余额用例的 p03_it 分开，零建表冲突）。
func openSettleDB(t *testing.T) *gorm.DB {
	t.Helper()
	port, _ := strconv.Atoi(h8Env("DATABASE_PORT", "5432"))
	db, err := database.NewPostgres(config.DatabaseConfig{
		Host: h8Env("DATABASE_HOST", "127.0.0.1"), Port: port,
		User: h8Env("DATABASE_USER", "postgres"), Password: h8Env("DATABASE_PASSWORD", "postgres"),
		DBName: h8Env("P08_PG_DBNAME", "p08_it"), SSLMode: "disable",
		MaxOpenConns: 8, MaxIdleConns: 4, ConnMaxLifetimeMinutes: 5,
	})
	if err != nil {
		t.Skipf("连不上测试库（%s）：%v —— 需先建库 p08_it", h8Env("P08_PG_DBNAME", "p08_it"), err)
	}
	if err := db.AutoMigrate(h8Tables...); err != nil {
		t.Fatalf("AutoMigrate: %v", err)
	}
	t.Cleanup(func() {
		for i := len(h8Tables) - 1; i >= 0; i-- {
			if err := db.Migrator().DropTable(h8Tables[i]); err != nil {
				t.Errorf("清理失败 %T: %v", h8Tables[i], err)
			}
		}
	})
	return db
}

func h8WechatCfg() config.ThirdPartyConfig {
	return config.ThirdPartyConfig{
		WeChatAppID: h8WechatApp, WeChatMchID: h8MchID, WeChatMchAPIKey: h8MchAPIKey,
	}
}

// h8Services 装配 payment 域 + membership 域。NewService 内部自注册
// membership_order 核销器（零 main.go 改动），所以回调会主动发权益。
func h8Services(db *gorm.DB) (*payment.Service, *Service, *payment.PostgresRepository) {
	payRepo := payment.NewPostgresRepository(db)
	paySvc := payment.NewService(h8WechatCfg(), payRepo)
	memSvc := NewService(NewPostgresRepository(db), nil, paySvc, nil, nil, "dev")
	return paySvc, memSvc, payRepo
}

func h8SeedOrder(t *testing.T, db *gorm.DB, orderID int64, userID, planID string, status int16) {
	t.Helper()
	tier := "svip"
	if p, ok := entity.FindMembershipPlan(planID); ok {
		tier = p.Tier
	}
	order := &entity.WysMembershipOrder{
		OrderID: orderID, UserID: userID, Tier: tier, PlanID: planID,
		Channel: entity.MembershipChannelWeChat, AmountFen: 3000, Status: status,
		OutTradeNo: "H8-OT-" + strconv.FormatInt(orderID, 10), CreatedAt: time.Now(),
	}
	if err := db.WithContext(context.Background()).Create(order).Error; err != nil {
		t.Fatalf("seed 订单: %v", err)
	}
}

func h8SeedPayment(t *testing.T, db *gorm.DB, outTradeNo, refID, userID string, status int16) {
	t.Helper()
	rec := &entity.WysPaymentRecord{
		OutTradeNo: outTradeNo, UserID: userID, Channel: entity.PaymentChannelWeChat,
		AmountFen: 3000, Status: status, RefType: entity.PaymentRefMembershipOrder,
		RefID: refID, CreatedAt: time.Now(),
	}
	if err := db.WithContext(context.Background()).Create(rec).Error; err != nil {
		t.Fatalf("seed 流水: %v", err)
	}
}

// 以下三个是 payment 域 wechatSign / wechatXML 的本地副本（payment 内部函数不可导出给本包），
// 用来构造一份能通过 VerifyWechatNotify 的真实签名回调报文。
func h8WechatSign(vals map[string]string, apiKey string) string {
	keys := make([]string, 0, len(vals))
	for k, v := range vals {
		if k == "sign" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(vals[k])
	}
	b.WriteString("&key=")
	b.WriteString(apiKey)
	sum := md5.Sum([]byte(b.String()))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

func h8WechatXML(vals map[string]string, apiKey string) []byte {
	signed := make(map[string]string, len(vals)+1)
	for k, v := range vals {
		signed[k] = v
	}
	if apiKey != "" {
		signed["sign"] = h8WechatSign(signed, apiKey)
	}
	keys := make([]string, 0, len(signed))
	for k := range signed {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	b.WriteString("<xml>")
	for _, k := range keys {
		b.WriteString("<")
		b.WriteString(k)
		b.WriteString("><![CDATA[")
		b.WriteString(signed[k])
		b.WriteString("]]></")
		b.WriteString(k)
		b.WriteString(">")
	}
	b.WriteString("</xml>")
	return []byte(b.String())
}

func h8NotifyVals(overrides map[string]string) map[string]string {
	vals := map[string]string{
		"appid": h8WechatApp, "mch_id": h8MchID,
		"out_trade_no": "H8-OT-9101", "transaction_id": "wx-txn-h8",
		"total_fee": "3000", "result_code": "SUCCESS", "return_code": "SUCCESS",
		"trade_state": "SUCCESS", "time_end": "20261107120000", "nonce_str": "abc123",
	}
	for k, v := range overrides {
		vals[k] = v
	}
	return vals
}

func h8Entitlement(t *testing.T, db *gorm.DB, userID, tier string) *entity.WysMembershipEntitlement {
	t.Helper()
	var row entity.WysMembershipEntitlement
	err := db.WithContext(context.Background()).
		Where("user_id = ? AND tier = ?", userID, tier).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		t.Fatalf("读权益: %v", err)
	}
	return &row
}

func h8EntCount(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	if err := db.WithContext(context.Background()).Model(&entity.WysMembershipEntitlement{}).Count(&n).Error; err != nil {
		t.Fatalf("count 权益: %v", err)
	}
	return n
}

func h8OrderStatus(t *testing.T, db *gorm.DB, orderID int64) int16 {
	t.Helper()
	var st int16
	if err := db.WithContext(context.Background()).Model(&entity.WysMembershipOrder{}).
		Where("order_id = ?", orderID).Pluck("status", &st).Error; err != nil {
		t.Fatalf("读订单状态: %v", err)
	}
	return st
}

func h8PaymentStatus(t *testing.T, db *gorm.DB, outTradeNo string) int16 {
	t.Helper()
	var st int16
	if err := db.WithContext(context.Background()).Model(&entity.WysPaymentRecord{}).
		Where("out_trade_no = ?", outTradeNo).Pluck("status", &st).Error; err != nil {
		t.Fatalf("读流水状态: %v", err)
	}
	return st
}

// h8AssertOnePeriod：一个周期 ≈ now+1 月（≈30 天），两个周期 ≈ now+2 月（≈60 天）。
// 用 20~45 天窗口区分，与具体月长/时刻无关；叠加两次必然落在窗口外。
func h8AssertOnePeriod(t *testing.T, ent *entity.WysMembershipEntitlement, before time.Time) {
	t.Helper()
	if ent.ExpiresAt.Before(before.AddDate(0, 0, 20)) || ent.ExpiresAt.After(before.AddDate(0, 0, 45)) {
		t.Fatalf("expires=%v，期望约 +1 个月（单个周期）；落在 2 周期窗口说明被叠加了", ent.ExpiresAt)
	}
}

// TestSettleChannelPaidGrantsEntitlementAndIdempotent：
// 真实回调 HandleWechatNotify 核销 membership_order 即发权益，Reason=settled
// （settled_no_settler 对该 ref_type 不再出现）；连投两次幂等；
// 回调已发权益后客户端回跳 ConfirmBuyout 不再发第二周期。
func TestSettleChannelPaidGrantsEntitlementAndIdempotent(t *testing.T) {
	db := openSettleDB(t)
	ctx := context.Background()
	const orderID int64 = 9101
	const ot = "H8-OT-9101"
	h8SeedOrder(t, db, orderID, "u1", "svip_1m", entity.MembershipOrderPending)
	h8SeedPayment(t, db, ot, strconv.FormatInt(orderID, 10), "u1", entity.PaymentStatusPending)
	paySvc, memSvc, _ := h8Services(db)

	raw := h8WechatXML(h8NotifyVals(map[string]string{"out_trade_no": ot}), h8MchAPIKey)
	before := time.Now()
	res, err := paySvc.HandleWechatNotify(ctx, raw)
	if err != nil {
		t.Fatalf("回调核销失败: %v", err)
	}
	if !res.NewlySettled || res.Reason != "settled" {
		t.Fatalf("Reason=%q NewlySettled=%v，期望 settled/true（settled_no_settler 对 membership_order 不应再出现）", res.Reason, res.NewlySettled)
	}
	if h8PaymentStatus(t, db, ot) != entity.PaymentStatusPaid {
		t.Fatal("流水未置 paid")
	}
	if h8OrderStatus(t, db, orderID) != entity.MembershipOrderPaid {
		t.Fatal("订单未置 paid")
	}
	ent := h8Entitlement(t, db, "u1", "svip")
	if ent == nil {
		t.Fatal("核销后未发权益")
	}
	h8AssertOnePeriod(t, ent, before)

	// 回调重投：payment 层 SettlePaid 返回 newly=false，权益不再叠加。
	res2, err := paySvc.HandleWechatNotify(ctx, raw)
	if err != nil {
		t.Fatalf("重复回调失败: %v", err)
	}
	if res2.NewlySettled || res2.Reason != "duplicate" {
		t.Fatalf("重复回调 Reason=%q NewlySettled=%v，期望 duplicate/false", res2.Reason, res2.NewlySettled)
	}
	ent2 := h8Entitlement(t, db, "u1", "svip")
	if ent2 == nil || !ent2.ExpiresAt.Equal(ent.ExpiresAt) {
		t.Fatalf("重复回调叠加了权益: %v -> %+v", ent.ExpiresAt, ent2)
	}
	if n := h8EntCount(t, db); n != 1 {
		t.Fatalf("权益行数=%d，期望 1", n)
	}

	// 对照：回调已发权益后客户端回跳不得再发第二周期。
	out, err := memSvc.ConfirmBuyout(ctx, "u1", orderID)
	if err != nil {
		t.Fatalf("回调后 ConfirmBuyout 报错: %v", err)
	}
	if out.Status != "paid" {
		t.Fatalf("ConfirmBuyout status=%s", out.Status)
	}
	ent3 := h8Entitlement(t, db, "u1", "svip")
	if ent3 == nil || !ent3.ExpiresAt.Equal(ent.ExpiresAt) {
		t.Fatalf("ConfirmBuyout 叠加了权益: %v -> %+v", ent.ExpiresAt, ent3)
	}
}

// TestSettleChannelPaidVsConfirmBuyoutConcurrent（风险 R5）：
// 一条走 SettleChannelPaid（回调链，用 payRepo.WithTx 复刻 settleByFact 的事务 ctx），
// 一条走 ConfirmBuyout（客户端回跳链），同一 order 并发跑；
// MarkOrderPaid 的条件更新 + 行锁保证只有一条链走到 ExtendEntitlement。
func TestSettleChannelPaidVsConfirmBuyoutConcurrent(t *testing.T) {
	db := openSettleDB(t)
	ctx := context.Background()
	const orderID int64 = 9102
	refID := strconv.FormatInt(orderID, 10)
	const ot = "H8-OT-9102"
	h8SeedOrder(t, db, orderID, "u1", "svip_1m", entity.MembershipOrderPending)
	// 已核销流水：让 ConfirmBuyout→ConfirmRefPaid 直接返回（不触发渠道查单/HTTP）。
	h8SeedPayment(t, db, ot, refID, "u1", entity.PaymentStatusPaid)
	_, memSvc, payRepo := h8Services(db)

	rec := &entity.WysPaymentRecord{
		OutTradeNo: ot, UserID: "u1", Channel: entity.PaymentChannelWeChat,
		AmountFen: 3000, Status: entity.PaymentStatusPaid,
		RefType: entity.PaymentRefMembershipOrder, RefID: refID,
	}
	before := time.Now()
	var wg sync.WaitGroup
	errs := make([]error, 2)
	start := make(chan struct{})
	// 链 A：回调核销。
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		errs[0] = payRepo.WithTx(ctx, func(txCtx context.Context) error {
			return memSvc.SettleChannelPaid(txCtx, entity.PaymentRefMembershipOrder, refID, rec)
		})
	}()
	// 链 B：客户端回跳确认。
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-start
		_, errs[1] = memSvc.ConfirmBuyout(ctx, "u1", orderID)
	}()
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("链 %d 返回错误: %v", i, err)
		}
	}
	ent := h8Entitlement(t, db, "u1", "svip")
	if ent == nil {
		t.Fatal("双链并发后未发权益")
	}
	h8AssertOnePeriod(t, ent, before)
	if n := h8EntCount(t, db); n != 1 {
		t.Fatalf("权益行数=%d，期望 1（不重复叠加）", n)
	}
	if h8OrderStatus(t, db, orderID) != entity.MembershipOrderPaid {
		t.Fatal("订单未置 paid")
	}
}

// TestConfirmBuyoutRequiresSettledRecord（对照）：
// 即便已自注册 Settler，ConfirmBuyout 的「必须有已核销流水」仍成立 ——
// 无 wys_payment_records 流水时拒绝且绝不发权益。
func TestConfirmBuyoutRequiresSettledRecord(t *testing.T) {
	db := openSettleDB(t)
	ctx := context.Background()
	const orderID int64 = 9103
	h8SeedOrder(t, db, orderID, "u1", "svip_1m", entity.MembershipOrderPending)
	// 故意不落任何流水。
	_, memSvc, _ := h8Services(db)

	out, err := memSvc.ConfirmBuyout(ctx, "u1", orderID)
	if !errors.Is(err, ErrPaymentNotConfirmed) {
		t.Fatalf("无已核销流水却发权益：out=%+v err=%v", out, err)
	}
	if ent := h8Entitlement(t, db, "u1", "svip"); ent != nil {
		t.Fatalf("无流水却发了权益: %+v", ent)
	}
	if h8OrderStatus(t, db, orderID) != entity.MembershipOrderPending {
		t.Fatal("无流水却把订单置 paid")
	}
}

// TestSettleChannelPaidSharesPaymentTx（事务边界）：
// 复刻 settleByFact 的事务 ctx，SettleChannelPaid 成功后强制外层回滚。
// 若其写真的落在同一 payment 事务里，权益 / 订单都应随回滚消失；
// 若它错误地另开独立事务，权益会残留 → 断言失败。
func TestSettleChannelPaidSharesPaymentTx(t *testing.T) {
	db := openSettleDB(t)
	ctx := context.Background()
	const orderID int64 = 9104
	refID := strconv.FormatInt(orderID, 10)
	const ot = "H8-OT-9104"
	h8SeedOrder(t, db, orderID, "u1", "svip_1m", entity.MembershipOrderPending)
	h8SeedPayment(t, db, ot, refID, "u1", entity.PaymentStatusPending)
	_, memSvc, payRepo := h8Services(db)

	rec := &entity.WysPaymentRecord{
		OutTradeNo: ot, UserID: "u1", Channel: entity.PaymentChannelWeChat,
		AmountFen: 3000, Status: entity.PaymentStatusPaid,
		RefType: entity.PaymentRefMembershipOrder, RefID: refID,
	}
	err := payRepo.WithTx(ctx, func(txCtx context.Context) error {
		if err := memSvc.SettleChannelPaid(txCtx, entity.PaymentRefMembershipOrder, refID, rec); err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	if err == nil {
		t.Fatal("期望强制回滚错误")
	}
	if ent := h8Entitlement(t, db, "u1", "svip"); ent != nil {
		t.Fatalf("SettleChannelPaid 的权益写没随 payment 事务回滚（说明另开了事务）: %+v", ent)
	}
	if h8OrderStatus(t, db, orderID) != entity.MembershipOrderPending {
		t.Fatal("订单写没随 payment 事务回滚（说明另开了事务）")
	}
}
