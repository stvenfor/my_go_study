// notify_test.go P0-2 硬门槛单测：
// ①回调重复投递 3 次只核销 1 次；②篡改签名 / 金额不符 → 拒绝且流水状态不变；
// 外加商户号不符、缺密钥不跳过校验、业务核销失败整体回滚、支付宝 RSA2 验签。
package payment

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"net/url"
	"testing"
	"time"

	"common/config"
	"model/entity"
)

const (
	testMchAPIKey = "0123456789abcdef0123456789abcdef"
	testWechatApp = "wx0000000000000001"
	testMchID     = "1900000109"
)

func wechatCfg() config.ThirdPartyConfig {
	return config.ThirdPartyConfig{
		WeChatAppID:        testWechatApp,
		WeChatMchID:        testMchID,
		WeChatMchAPIKey:    testMchAPIKey,
		WeChatMchNotifyURL: "https://example.test/api/v1/payments/notify/wechat",
	}
}

func wechatNotifyVals(overrides map[string]string) map[string]string {
	vals := map[string]string{
		"appid":          testWechatApp,
		"mch_id":         testMchID,
		"out_trade_no":   "OT-1",
		"transaction_id": "wx-txn-1",
		"total_fee":      "1500",
		"result_code":    "SUCCESS",
		"return_code":    "SUCCESS",
		"trade_state":    "SUCCESS",
		"time_end":       "20261104120000",
		"nonce_str":      "abc123",
	}
	for k, v := range overrides {
		vals[k] = v
	}
	return vals
}

// settleSvc 装配一个只注册了 mall 核销器的服务（内存仓储）。
func settleSvc(repo *fakeRepo) (*Service, *fakeSettler) {
	settler := &fakeSettler{}
	svc := NewService(wechatCfg(), repo)
	svc.RegisterSettler(entity.PaymentRefMallOrder, settler)
	return svc, settler
}

// 门槛①：同一份回调投 3 次，业务核销只发生 1 次。
func TestWechatNotifyThreeDeliveriesSettleOnce(t *testing.T) {
	repo := newFakeRepo()
	repo.seedPending("OT-1", entity.PaymentChannelWeChat, entity.PaymentRefMallOrder, "77", 1500, time.Now().Add(-time.Hour))
	svc, settler := settleSvc(repo)
	raw := wechatXML(wechatNotifyVals(nil), testMchAPIKey)
	ctx := context.Background()

	res, err := svc.HandleWechatNotify(ctx, raw)
	mustSettled(t, res, err, true, "settled")

	for i := 0; i < 2; i++ {
		res, err = svc.HandleWechatNotify(ctx, raw)
		mustSettled(t, res, err, false, "duplicate")
	}

	if len(settler.calls) != 1 {
		t.Errorf("业务核销调用 %d 次，期望 1 次：%v", len(settler.calls), settler.calls)
	}
	if repo.settleCalls != 1 {
		t.Errorf("SettlePaid 生效 %d 次，期望 1 次", repo.settleCalls)
	}
	rec, _ := repo.record("OT-1")
	if rec.Status != entity.PaymentStatusPaid {
		t.Errorf("流水状态=%d，期望 paid(%d)", rec.Status, entity.PaymentStatusPaid)
	}
	if rec.ChannelTradeNo == nil || *rec.ChannelTradeNo != "wx-txn-1" {
		t.Errorf("渠道单号未落库：%v", rec.ChannelTradeNo)
	}
	if rec.PaidAt == nil || !rec.PaidAt.Equal(time.Date(2026, 11, 4, 12, 0, 0, 0, channelZone)) {
		t.Errorf("paid_at=%v，期望 2026-11-04 12:00 GMT+8", rec.PaidAt)
	}
}

// 门槛②a：签名被篡改（改了金额没重签）→ 拒绝，流水仍 pending。
func TestWechatNotifyTamperedSignatureRejected(t *testing.T) {
	repo := newFakeRepo()
	repo.seedPending("OT-1", entity.PaymentChannelWeChat, entity.PaymentRefMallOrder, "77", 1500, time.Now().Add(-time.Hour))
	svc, settler := settleSvc(repo)

	signed := wechatXML(wechatNotifyVals(nil), testMchAPIKey)
	tampered := bytes.Replace(signed,
		[]byte(`<total_fee><![CDATA[1500]]></total_fee>`),
		[]byte(`<total_fee><![CDATA[1]]></total_fee>`), 1)
	if bytes.Equal(tampered, signed) {
		t.Fatal("篡改未生效，用例失效")
	}

	_, err := svc.HandleWechatNotify(context.Background(), tampered)
	if !errors.Is(err, ErrSignInvalid) {
		t.Fatalf("期望 ErrSignInvalid，实得 %v", err)
	}
	if got := repo.status(t, "OT-1"); got != entity.PaymentStatusPending {
		t.Errorf("流水状态=%d，期望仍为 pending", got)
	}
	if len(settler.calls) != 0 {
		t.Errorf("验签失败却调了业务核销：%v", settler.calls)
	}
}

// 门槛②b：签名合法但金额与流水不符 → 拒绝，流水仍 pending。
func TestWechatNotifyAmountMismatchRejected(t *testing.T) {
	repo := newFakeRepo()
	repo.seedPending("OT-1", entity.PaymentChannelWeChat, entity.PaymentRefMallOrder, "77", 1500, time.Now().Add(-time.Hour))
	svc, settler := settleSvc(repo)

	raw := wechatXML(wechatNotifyVals(map[string]string{"total_fee": "1"}), testMchAPIKey)
	_, err := svc.HandleWechatNotify(context.Background(), raw)
	if !errors.Is(err, ErrAmountMismatch) {
		t.Fatalf("期望 ErrAmountMismatch，实得 %v", err)
	}
	if got := repo.status(t, "OT-1"); got != entity.PaymentStatusPending {
		t.Errorf("流水状态=%d，期望仍为 pending", got)
	}
	if len(settler.calls) != 0 {
		t.Errorf("金额不符却发货：%v", settler.calls)
	}
}

// 商户号不符（别人家的合法签名）→ 拒绝。
func TestWechatNotifyMerchantMismatchRejected(t *testing.T) {
	repo := newFakeRepo()
	repo.seedPending("OT-1", entity.PaymentChannelWeChat, entity.PaymentRefMallOrder, "77", 1500, time.Now().Add(-time.Hour))
	svc, _ := settleSvc(repo)

	raw := wechatXML(wechatNotifyVals(map[string]string{"mch_id": "1900000999"}), testMchAPIKey)
	if _, err := svc.HandleWechatNotify(context.Background(), raw); !errors.Is(err, ErrMerchantMismatch) {
		t.Fatalf("期望 ErrMerchantMismatch，实得 %v", err)
	}
	if got := repo.status(t, "OT-1"); got != entity.PaymentStatusPending {
		t.Errorf("流水状态=%d，期望仍为 pending", got)
	}
}

// 未配商户密钥时必须拒绝回调，而不是「没密钥就跳过验签」。
func TestWechatNotifyWithoutAPIKeyAlwaysRejects(t *testing.T) {
	repo := newFakeRepo()
	repo.seedPending("OT-1", entity.PaymentChannelWeChat, entity.PaymentRefMallOrder, "77", 1500, time.Now().Add(-time.Hour))
	raw := wechatXML(wechatNotifyVals(nil), testMchAPIKey)

	svc := NewService(config.ThirdPartyConfig{WeChatAppID: testWechatApp, WeChatMchID: testMchID}, repo)
	if _, err := svc.HandleWechatNotify(context.Background(), raw); !errors.Is(err, ErrSignInvalid) {
		t.Fatalf("缺密钥时期望拒绝，实得 %v", err)
	}
	if got := repo.status(t, "OT-1"); got != entity.PaymentStatusPending {
		t.Errorf("流水状态=%d，期望仍为 pending", got)
	}
}

// 流水不存在的回调（随便挑一个商户号打过来）→ 拒绝，且不得凭空造记录。
func TestWechatNotifyUnknownOutTradeNoRejected(t *testing.T) {
	repo := newFakeRepo()
	svc, settler := settleSvc(repo)

	raw := wechatXML(wechatNotifyVals(map[string]string{"out_trade_no": "OT-NOPE"}), testMchAPIKey)
	if _, err := svc.HandleWechatNotify(context.Background(), raw); !errors.Is(err, ErrRecordNotFound) {
		t.Fatalf("期望 ErrRecordNotFound，实得 %v", err)
	}
	if len(repo.records) != 0 || len(settler.calls) != 0 {
		t.Errorf("无流水却产生了记录/核销：%d 条 / %v", len(repo.records), settler.calls)
	}
}

// 业务核销失败 → 整体回滚，流水回到 pending（渠道重投还能补）。
func TestSettleRollsbackWhenSettlerFails(t *testing.T) {
	repo := newFakeRepo()
	repo.seedPending("OT-1", entity.PaymentChannelWeChat, entity.PaymentRefMallOrder, "77", 1500, time.Now().Add(-time.Hour))
	settler := &fakeSettler{err: errors.New("库存不足")}
	svc := NewService(wechatCfg(), repo)
	svc.RegisterSettler(entity.PaymentRefMallOrder, settler)

	raw := wechatXML(wechatNotifyVals(nil), testMchAPIKey)
	_, err := svc.HandleWechatNotify(context.Background(), raw)
	if !errors.Is(err, ErrSettleNotCommitted) {
		t.Fatalf("期望 ErrSettleNotCommitted，实得 %v", err)
	}
	if got := repo.status(t, "OT-1"); got != entity.PaymentStatusPending {
		t.Errorf("核销失败后流水状态=%d，期望回滚为 pending", got)
	}

	// 修好业务侧后重投同一份回调，应当正常核销（幂等不挡重试）。
	settler.err = nil
	res, err := svc.HandleWechatNotify(context.Background(), raw)
	mustSettled(t, res, err, true, "settled")
}

// 已置 failed 的流水不能被回调翻成已付（状态机单向）。
func TestSettleRejectsTerminalFailedRecord(t *testing.T) {
	repo := newFakeRepo()
	repo.seedPending("OT-1", entity.PaymentChannelWeChat, entity.PaymentRefMallOrder, "77", 1500, time.Now().Add(-time.Hour))
	if err := repo.MarkFailed(context.Background(), "OT-1", "prepay_config_missing"); err != nil {
		t.Fatal(err)
	}
	svc, settler := settleSvc(repo)

	raw := wechatXML(wechatNotifyVals(nil), testMchAPIKey)
	if _, err := svc.HandleWechatNotify(context.Background(), raw); !errors.Is(err, ErrRecordNotPending) {
		t.Fatalf("期望 ErrRecordNotPending，实得 %v", err)
	}
	if got := repo.status(t, "OT-1"); got != entity.PaymentStatusFailed {
		t.Errorf("failed 流水被改成 %d", got)
	}
	if len(settler.calls) != 0 {
		t.Errorf("终态流水却发货：%v", settler.calls)
	}
}

// 未注册核销器的 ref_type：流水照实置 paid，但必须留痕（不能装作发货成功）。
func TestSettledWithoutSettlerIsMarkedNotNewly(t *testing.T) {
	repo := newFakeRepo()
	repo.seedPending("OT-9", entity.PaymentChannelWeChat, entity.PaymentRefMembershipOrder, "9", 1500, time.Now().Add(-time.Hour))
	svc := NewService(wechatCfg(), repo)

	raw := wechatXML(wechatNotifyVals(map[string]string{"out_trade_no": "OT-9"}), testMchAPIKey)
	res, err := svc.HandleWechatNotify(context.Background(), raw)
	mustSettled(t, res, err, false, "settled_no_settler")
	if got := repo.status(t, "OT-9"); got != entity.PaymentStatusPaid {
		t.Errorf("流水状态=%d，期望 paid（渠道确实收了钱）", got)
	}
}

// --- 支付宝 RSA2 ---

// alipayKeys 一对测试密钥：pubBare 是控制台那种「纯 base64 一串」形态，
// 顺带覆盖 normalizePublicKeyPEM；privPEM 给需要签请求的用例（查单）。
type alipayKeys struct {
	key     *rsa.PrivateKey
	privPEM string
	pubBare string
}

func newAlipayKeys(t *testing.T) alipayKeys {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	privDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return alipayKeys{
		key:     key,
		privPEM: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})),
		pubBare: base64.StdEncoding.EncodeToString(pubDER),
	}
}

// sign 按支付宝 RSA2 规则对待验签串签名。
func (k alipayKeys) sign(t *testing.T, content string) string {
	t.Helper()
	digest := sha256.Sum256([]byte(content))
	sig, err := rsa.SignPKCS1v15(rand.Reader, k.key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("签名失败: %v", err)
	}
	return base64.StdEncoding.EncodeToString(sig)
}

func alipayCfg(pubBare string) config.ThirdPartyConfig {
	return config.ThirdPartyConfig{
		AlipayAppID:      "2021000000000000",
		AlipayPrivateKey: "unused-in-notify-tests",
		AlipayPublicKey:  pubBare,
	}
}

func (k alipayKeys) notifyForm(t *testing.T, overrides map[string]string) url.Values {
	t.Helper()
	form := url.Values{}
	set := func(name, v string) { form.Set(name, v) }
	set("app_id", "2021000000000000")
	set("out_trade_no", "OT-1")
	set("trade_no", "ali-txn-1")
	set("trade_status", "TRADE_SUCCESS")
	set("total_amount", "15.00")
	set("gmt_payment", "2026-11-04 12:00:00")
	set("seller_id", "2088000000000000")
	set("notify_time", "2026-11-04 12:00:05")
	set("notify_type", "trade_status_sync")
	set("charset", "utf-8")
	for name, v := range overrides {
		set(name, v)
	}
	form.Set("sign_type", "RSA2")
	form.Set("sign", k.sign(t, alipayNotifyContent(form)))
	return form
}

func alipaySvc(repo *fakeRepo, keys alipayKeys) (*Service, *fakeSettler) {
	settler := &fakeSettler{}
	svc := NewService(alipayCfg(keys.pubBare), repo)
	svc.RegisterSettler(entity.PaymentRefMallOrder, settler)
	return svc, settler
}

func TestAlipayNotifySettlesWithRSA2(t *testing.T) {
	repo := newFakeRepo()
	repo.seedPending("OT-1", entity.PaymentChannelAlipay, entity.PaymentRefMallOrder, "77", 1500, time.Now().Add(-time.Hour))
	keys := newAlipayKeys(t)
	svc, settler := alipaySvc(repo, keys)

	form := keys.notifyForm(t, nil)
	res, err := svc.HandleAlipayNotify(context.Background(), form)
	if err != nil {
		t.Fatalf("合法回调被拒: %v", err)
	}
	if res.Reason != "settled" || len(settler.calls) != 1 {
		t.Errorf("res=%+v settler=%v", res, settler.calls)
	}
	rec, _ := repo.record("OT-1")
	if rec.Status != entity.PaymentStatusPaid || rec.ChannelTradeNo == nil || *rec.ChannelTradeNo != "ali-txn-1" {
		t.Errorf("核销未落库：%+v", rec)
	}
}

// 待验签串必须是「除 sign/sign_type 外、按 key 字典序、k=v&」——错了就永远验不过。
func TestAlipayNotifyContentIsSortedAndExcludesSign(t *testing.T) {
	form := url.Values{}
	form.Set("sign", "whatever")
	form.Set("sign_type", "RSA2")
	form.Set("total_amount", "15.00")
	form.Set("app_id", "2021000000000000")
	form.Set("out_trade_no", "OT-1")
	form.Set("empty_field", "")
	want := "app_id=2021000000000000&out_trade_no=OT-1&total_amount=15.00"
	if got := alipayNotifyContent(form); got != want {
		t.Errorf("待验签串=%q，期望 %q", got, want)
	}
}

// 门槛②（支付宝侧）：签名合法后改金额 → 要么验签失败要么金额不符，绝不能核销。
func TestAlipayNotifyTamperedAmountRejected(t *testing.T) {
	repo := newFakeRepo()
	repo.seedPending("OT-1", entity.PaymentChannelAlipay, entity.PaymentRefMallOrder, "77", 1500, time.Now().Add(-time.Hour))
	keys := newAlipayKeys(t)
	svc, settler := alipaySvc(repo, keys)

	form := keys.notifyForm(t, nil)
	form.Set("total_amount", "0.01") // 签名没跟着改
	if _, err := svc.HandleAlipayNotify(context.Background(), form); !errors.Is(err, ErrSignInvalid) {
		t.Fatalf("篡改金额期望 ErrSignInvalid，实得 %v", err)
	}
	if got := repo.status(t, "OT-1"); got != entity.PaymentStatusPending {
		t.Errorf("流水状态=%d，期望仍为 pending", got)
	}
	if len(settler.calls) != 0 {
		t.Errorf("验签失败却发货：%v", settler.calls)
	}
}

// 签名合法但金额与流水不符（渠道侧真改了价）→ ErrAmountMismatch。
func TestAlipayNotifyAmountMismatchRejected(t *testing.T) {
	repo := newFakeRepo()
	repo.seedPending("OT-1", entity.PaymentChannelAlipay, entity.PaymentRefMallOrder, "77", 1500, time.Now().Add(-time.Hour))
	keys := newAlipayKeys(t)
	svc, settler := alipaySvc(repo, keys)

	form := keys.notifyForm(t, map[string]string{"total_amount": "1.00"})
	if _, err := svc.HandleAlipayNotify(context.Background(), form); !errors.Is(err, ErrAmountMismatch) {
		t.Fatalf("期望 ErrAmountMismatch，实得 %v", err)
	}
	if got := repo.status(t, "OT-1"); got != entity.PaymentStatusPending {
		t.Errorf("流水状态=%d，期望仍为 pending", got)
	}
	if len(settler.calls) != 0 {
		t.Errorf("金额不符却发货：%v", settler.calls)
	}
}

// 非 RSA2 一律拒绝（RSA 弱签名算法不接受）。
func TestAlipayNotifyRejectsWeakSignType(t *testing.T) {
	repo := newFakeRepo()
	repo.seedPending("OT-1", entity.PaymentChannelAlipay, entity.PaymentRefMallOrder, "77", 1500, time.Now().Add(-time.Hour))
	keys := newAlipayKeys(t)
	svc, _ := alipaySvc(repo, keys)

	form := keys.notifyForm(t, nil)
	form.Set("sign_type", "RSA") // 改在签名之后，模拟换算法重放
	if _, err := svc.HandleAlipayNotify(context.Background(), form); !errors.Is(err, ErrSignInvalid) {
		t.Fatalf("期望拒绝非 RSA2，实得 %v", err)
	}
}

// 缺支付宝公钥 → 无法验签，必须拒绝而不是跳过。
func TestAlipayNotifyWithoutPublicKeyRejected(t *testing.T) {
	repo := newFakeRepo()
	repo.seedPending("OT-1", entity.PaymentChannelAlipay, entity.PaymentRefMallOrder, "77", 1500, time.Now().Add(-time.Hour))
	keys := newAlipayKeys(t)
	svc, _ := alipaySvc(repo, keys)
	svc.cfg.AlipayPublicKey = ""

	form := keys.notifyForm(t, nil)
	if _, err := svc.HandleAlipayNotify(context.Background(), form); !errors.Is(err, ErrSignInvalid) {
		t.Fatalf("缺公钥时期望拒绝，实得 %v", err)
	}
	if got := repo.status(t, "OT-1"); got != entity.PaymentStatusPending {
		t.Errorf("流水状态=%d，期望仍为 pending", got)
	}
}
