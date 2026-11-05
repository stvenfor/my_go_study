// jobs_test.go 预支付落流水 / 主动查单 / T+1 对账：
// 打真实 HTTP 往返到假网关（httptest），签名与验签走真代码，不是打桩跳过。
package payment

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"common/config"
	"model/entity"
)

// wechatFact 假网关对某一单的答复（orderquery 响应字段）。
type wechatFact map[string]string

// fakeWechatGateway 同时扮演 unifiedorder 与 orderquery。
// prepayID 为空表示下单失败；facts 里没有的单号按「渠道查无此单」回错。
func fakeWechatGateway(t *testing.T, apiKey, prepayID string, facts map[string]wechatFact) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/pay/unifiedorder", func(w http.ResponseWriter, r *http.Request) {
		if prepayID == "" {
			_, _ = w.Write(wechatXML(map[string]string{
				"return_code": "FAIL", "return_msg": "merchant not exist",
			}, ""))
			return
		}
		_, _ = w.Write(wechatXML(map[string]string{
			"return_code": "SUCCESS", "result_code": "SUCCESS", "prepay_id": prepayID,
		}, apiKey))
	})
	mux.HandleFunc("/pay/orderquery", func(w http.ResponseWriter, r *http.Request) {
		vals, err := wechatParseXML(readAll(t, r))
		if err != nil {
			t.Fatalf("假网关解析查单请求: %v", err)
		}
		fact, ok := facts[vals["out_trade_no"]]
		if !ok {
			_, _ = w.Write(wechatXML(map[string]string{
				"return_code": "SUCCESS", "result_code": "FAIL",
				"err_code": "ORDER_NOT_EXIST", "err_code_des": "order not exist",
			}, apiKey))
			return
		}
		resp := map[string]string{
			"appid": testWechatApp, "mch_id": testMchID,
			"return_code": "SUCCESS", "result_code": "SUCCESS",
			"out_trade_no": vals["out_trade_no"],
		}
		for k, v := range fact {
			resp[k] = v
		}
		_, _ = w.Write(wechatXML(resp, apiKey))
	})
	return httptest.NewServer(mux)
}

func readAll(t *testing.T, r *http.Request) []byte {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("读请求体: %v", err)
	}
	return body
}

func wechatCfgWithGateway(gatewayURL string) config.ThirdPartyConfig {
	cfg := wechatCfg()
	cfg.WeChatPayEndpoint = gatewayURL
	return cfg
}

// 门槛③（域内单测口径）：走微信渠道下单必须先落一条 pending 流水，
// 且商户单号回给客户端（真库 SQL 校验见 repo_postgres_integration_test.go）。
func TestPrepayWechatWritesPendingLedger(t *testing.T) {
	gw := fakeWechatGateway(t, testMchAPIKey, "wx-prepay-id-1", nil)
	defer gw.Close()
	repo := newFakeRepo()
	svc := NewService(wechatCfgWithGateway(gw.URL), repo)

	res, err := svc.Prepay(context.Background(), PrepayInput{
		Channel: "wechat", AmountFen: 1500, Subject: "商城订单",
		UserID: "user-1", RefType: entity.PaymentRefMallOrder, RefID: "77",
	})
	if err != nil {
		t.Fatalf("Prepay: %v", err)
	}
	if res.OutTradeNo == "" {
		t.Fatal("未回传 out_trade_no")
	}
	if len(repo.records) != 1 || repo.createCalls != 1 {
		t.Fatalf("流水条数=%d createCalls=%d，期望 1/1", len(repo.records), repo.createCalls)
	}
	rec, ok := repo.record(res.OutTradeNo)
	if !ok {
		t.Fatalf("流水未按 out_trade_no=%s 落库", res.OutTradeNo)
	}
	if rec.Status != entity.PaymentStatusPending || rec.AmountFen != 1500 ||
		rec.RefType != entity.PaymentRefMallOrder || rec.RefID != "77" {
		t.Errorf("流水内容不符：%+v", rec)
	}
	if res.Params["prepayid"] != "wx-prepay-id-1" {
		t.Errorf("SDK 参数缺 prepayid：%v", res.Params)
	}
}

// 渠道网络失败不能把流水判死：渠道可能已建单，留在 pending 由查单收敛。
func TestPrepayChannelErrorKeepsPending(t *testing.T) {
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer gw.Close()
	repo := newFakeRepo()
	svc := NewService(wechatCfgWithGateway(gw.URL), repo)

	_, err := svc.Prepay(context.Background(), PrepayInput{
		Channel: "wechat", AmountFen: 1500, Subject: "商城订单",
		UserID: "user-1", RefType: entity.PaymentRefMallOrder, RefID: "77",
	})
	if !strings.Contains(fmt.Sprint(err), "预支付失败") {
		t.Fatalf("期望渠道失败错误，实得 %v", err)
	}
	for _, rec := range repo.records {
		if rec.Status != entity.PaymentStatusPending {
			t.Errorf("流水 %s 状态=%d，期望仍为 pending", rec.OutTradeNo, rec.Status)
		}
	}
}

// 确定没到渠道（未配置商户）→ 置 failed，避免 pending 无限堆积。
func TestPrepayNotConfiguredMarksFailed(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(config.ThirdPartyConfig{WeChatAppID: testWechatApp}, repo)

	_, err := svc.Prepay(context.Background(), PrepayInput{
		Channel: "wechat", AmountFen: 1500, Subject: "商城订单",
		UserID: "user-1", RefType: entity.PaymentRefMallOrder, RefID: "77",
	})
	if !strings.Contains(fmt.Sprint(err), "支付未配置") {
		t.Fatalf("期望未配置错误，实得 %v", err)
	}
	for _, rec := range repo.records {
		if rec.Status != entity.PaymentStatusFailed {
			t.Errorf("流水 %s 状态=%d，期望 failed", rec.OutTradeNo, rec.Status)
		}
	}
}

// 回调丢失：查单兜底把 pending 补成 paid 并触发业务核销。
func TestQueryStalePendingSettlesLostCallback(t *testing.T) {
	gw := fakeWechatGateway(t, testMchAPIKey, "", map[string]wechatFact{
		"OT-1": {"trade_state": "SUCCESS", "total_fee": "1500", "transaction_id": "wx-txn-1", "time_end": "20261104123000"},
	})
	defer gw.Close()
	repo := newFakeRepo()
	repo.seedPending("OT-1", entity.PaymentChannelWeChat, entity.PaymentRefMallOrder, "77", 1500, time.Now().Add(-time.Hour))
	settler := &fakeSettler{}
	svc := NewService(wechatCfgWithGateway(gw.URL), repo)
	svc.RegisterSettler(entity.PaymentRefMallOrder, settler)

	scanned, err := svc.QueryStalePending(context.Background(), 10*time.Minute, 200)
	if err != nil {
		t.Fatalf("QueryStalePending: %v", err)
	}
	if scanned != 1 {
		t.Errorf("扫描条数=%d，期望 1", scanned)
	}
	if got := repo.status(t, "OT-1"); got != entity.PaymentStatusPaid {
		t.Errorf("流水状态=%d，期望 paid", got)
	}
	if len(settler.calls) != 1 {
		t.Errorf("业务核销 %v，期望调用 1 次", settler.calls)
	}
}

// 渠道明确关单 → 置 failed，否则每轮重复扫同一条。
func TestQueryStalePendingClosesFailedOrders(t *testing.T) {
	gw := fakeWechatGateway(t, testMchAPIKey, "", map[string]wechatFact{
		"OT-2": {"trade_state": "CLOSED", "total_fee": "1500", "transaction_id": ""},
	})
	defer gw.Close()
	repo := newFakeRepo()
	repo.seedPending("OT-2", entity.PaymentChannelWeChat, entity.PaymentRefMallOrder, "78", 1500, time.Now().Add(-time.Hour))
	settler := &fakeSettler{}
	svc := NewService(wechatCfgWithGateway(gw.URL), repo)
	svc.RegisterSettler(entity.PaymentRefMallOrder, settler)

	if _, err := svc.QueryStalePending(context.Background(), 10*time.Minute, 200); err != nil {
		t.Fatal(err)
	}
	rec, _ := repo.record("OT-2")
	if rec.Status != entity.PaymentStatusFailed || rec.FailReason != "channel_CLOSED" {
		t.Errorf("关单未收敛：%+v", rec)
	}
	if len(settler.calls) != 0 {
		t.Errorf("渠道未收款却发货：%v", settler.calls)
	}
}

// 渠道查无此单（网络/商户错误）时不得误判 failed。
func TestQueryStalePendingKeepsUnknownOrders(t *testing.T) {
	gw := fakeWechatGateway(t, testMchAPIKey, "", nil) // 所有单号都回 ORDER_NOT_EXIST
	defer gw.Close()
	repo := newFakeRepo()
	repo.seedPending("OT-3", entity.PaymentChannelWeChat, entity.PaymentRefMallOrder, "79", 1500, time.Now().Add(-time.Hour))
	svc := NewService(wechatCfgWithGateway(gw.URL), repo)

	if _, err := svc.QueryStalePending(context.Background(), 10*time.Minute, 200); err != nil {
		t.Fatal(err)
	}
	if got := repo.status(t, "OT-3"); got != entity.PaymentStatusPending {
		t.Errorf("流水状态=%d，期望仍为 pending（查单失败不等于没收到钱）", got)
	}
}

// T+1 对账：晚到核销、金额不符、渠道查无此单各写一条差异；两边一致的不写。
func TestReconcileWritesDiffsAndLateSettles(t *testing.T) {
	gw := fakeWechatGateway(t, testMchAPIKey, "", map[string]wechatFact{
		// 本地 pending、渠道已付 → 补核销 + status_mismatch
		"OT-LATE": {"trade_state": "SUCCESS", "total_fee": "1500", "transaction_id": "wx-late", "time_end": "20261104090000"},
		// 本地已付、渠道金额不同 → amount_mismatch
		"OT-AMT": {"trade_state": "SUCCESS", "total_fee": "100", "transaction_id": "wx-amt", "time_end": "20261104090000"},
		// 本地已付、渠道未付 → status_mismatch
		"OT-UNPAID": {"trade_state": "NOTPAY", "total_fee": "1500", "transaction_id": "", "time_end": "20261104090000"},
		// OT-MISSING 不在表里 → 渠道查无此单 → channel_missing
		// OT-OK 两边一致 → 不写差异
		"OT-OK": {"trade_state": "SUCCESS", "total_fee": "1500", "transaction_id": "wx-ok", "time_end": "20261104090000"},
	})
	defer gw.Close()

	repo := newFakeRepo()
	now := time.Now()
	repo.seedPending("OT-LATE", entity.PaymentChannelWeChat, entity.PaymentRefMallOrder, "1", 1500, now)
	repo.seedPending("OT-AMT", entity.PaymentChannelWeChat, entity.PaymentRefMallOrder, "2", 1500, now)
	if _, _, err := repo.SettlePaid(context.Background(), "OT-AMT", "wx-amt-local", 1500, "", now); err != nil {
		t.Fatal(err)
	}
	repo.seedPending("OT-UNPAID", entity.PaymentChannelWeChat, entity.PaymentRefMallOrder, "3", 1500, now)
	if _, _, err := repo.SettlePaid(context.Background(), "OT-UNPAID", "wx-unpaid", 1500, "", now); err != nil {
		t.Fatal(err)
	}
	repo.seedPending("OT-MISSING", entity.PaymentChannelWeChat, entity.PaymentRefMallOrder, "4", 1500, now)
	if _, _, err := repo.SettlePaid(context.Background(), "OT-MISSING", "wx-missing", 1500, "", now); err != nil {
		t.Fatal(err)
	}
	repo.seedPending("OT-OK", entity.PaymentChannelWeChat, entity.PaymentRefMallOrder, "5", 1500, now)
	if _, _, err := repo.SettlePaid(context.Background(), "OT-OK", "wx-ok", 1500, "", now); err != nil {
		t.Fatal(err)
	}

	settler := &fakeSettler{}
	svc := NewService(wechatCfgWithGateway(gw.URL), repo)
	svc.RegisterSettler(entity.PaymentRefMallOrder, settler)

	diffs, err := svc.Reconcile(context.Background(), now)
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if diffs != 4 {
		t.Errorf("差异条数=%d，期望 4（late/amount/status/channel_missing）", diffs)
	}
	got := map[string]string{}
	for _, d := range repo.diffs {
		got[d.OutTradeNo] = d.Kind
	}
	want := map[string]string{
		"OT-LATE":    entity.PaymentDiffStatusMismatch,
		"OT-AMT":     entity.PaymentDiffAmountMismatch,
		"OT-UNPAID":  entity.PaymentDiffStatusMismatch,
		"OT-MISSING": entity.PaymentDiffChannelMissing,
	}
	for out, kind := range want {
		if got[out] != kind {
			t.Errorf("%s 差异=%q，期望 %q", out, got[out], kind)
		}
	}
	if _, dup := got["OT-OK"]; dup {
		t.Errorf("两边一致的单却写了差异：%+v", got)
	}
	// 晚到核销必须真的把业务补上（否则差异只是记录、钱还是没发货）
	if got["OT-LATE"] == entity.PaymentDiffStatusMismatch && len(settler.calls) != 1 {
		t.Errorf("晚到核销未触发发货：%v", settler.calls)
	}
	if rec, _ := repo.record("OT-LATE"); !rec.IsPaid() {
		t.Errorf("OT-LATE 未被补核销：%+v", rec)
	}
}

// 对账重跑不得翻倍计差（唯一索引 + ON CONFLICT DO NOTHING 的内存等价口径）。
func TestReconcileRerunDoesNotDuplicateDiffs(t *testing.T) {
	gw := fakeWechatGateway(t, testMchAPIKey, "", map[string]wechatFact{
		"OT-AMT": {"trade_state": "SUCCESS", "total_fee": "100", "transaction_id": "wx-amt", "time_end": "20261104090000"},
	})
	defer gw.Close()
	repo := newFakeRepo()
	now := time.Now()
	repo.seedPending("OT-AMT", entity.PaymentChannelWeChat, entity.PaymentRefMallOrder, "2", 1500, now)
	svc := NewService(wechatCfgWithGateway(gw.URL), repo)

	if _, err := svc.Reconcile(context.Background(), now); err != nil {
		t.Fatal(err)
	}
	// 第二轮（假网关答复不变）：同一 (recon_date, out_trade_no, kind) 不应新增行。
	if _, err := svc.Reconcile(context.Background(), now); err != nil {
		t.Fatal(err)
	}
	if n := repo.diffCount("OT-AMT"); n != 1 {
		t.Errorf("差异行数=%d，期望 1", n)
	}
}

// --- 支付宝查单（响应必须验签，签名对不上就当不可信） ---

func TestQueryStalePendingAlipaySettles(t *testing.T) {
	keys := newAlipayKeys(t)
	gw := fakeAlipayGateway(t, keys, map[string]string{
		"OT-A": `{"code":"10000","msg":"Success","out_trade_no":"OT-A","trade_no":"ali-1","trade_status":"TRADE_SUCCESS","total_amount":"15.00"}`,
	})
	defer gw.Close()

	repo := newFakeRepo()
	repo.seedPending("OT-A", entity.PaymentChannelAlipay, entity.PaymentRefMallOrder, "10", 1500, time.Now().Add(-time.Hour))
	settler := &fakeSettler{}
	svc := NewService(alipayCfgWithGateway(gw.URL, keys), repo)
	svc.RegisterSettler(entity.PaymentRefMallOrder, settler)

	if _, err := svc.QueryStalePending(context.Background(), 10*time.Minute, 200); err != nil {
		t.Fatal(err)
	}
	if got := repo.status(t, "OT-A"); got != entity.PaymentStatusPaid {
		t.Errorf("流水状态=%d，期望 paid", got)
	}
	if len(settler.calls) != 1 {
		t.Errorf("支付宝查单后未发货：%v", settler.calls)
	}
}

// 假网关用另一把钥匙签响应 → 验签失败，不得核销。
func TestQueryStalePendingAlipayRejectsForeignSignature(t *testing.T) {
	signer := newAlipayKeys(t)
	verifier := newAlipayKeys(t)
	gw := fakeAlipayGateway(t, signer, map[string]string{
		"OT-A": `{"code":"10000","msg":"Success","out_trade_no":"OT-A","trade_no":"ali-1","trade_status":"TRADE_SUCCESS","total_amount":"15.00"}`,
	})
	defer gw.Close()

	repo := newFakeRepo()
	repo.seedPending("OT-A", entity.PaymentChannelAlipay, entity.PaymentRefMallOrder, "10", 1500, time.Now().Add(-time.Hour))
	svc := NewService(alipayCfgWithGateway(gw.URL, verifier), repo)

	if _, err := svc.QueryStalePending(context.Background(), 10*time.Minute, 200); err != nil {
		t.Fatal(err)
	}
	if got := repo.status(t, "OT-A"); got != entity.PaymentStatusPending {
		t.Errorf("流水状态=%d，期望仍为 pending", got)
	}
}

func alipayCfgWithGateway(gatewayURL string, keys alipayKeys) config.ThirdPartyConfig {
	return config.ThirdPartyConfig{
		AlipayAppID:      "2021000000000000",
		AlipayPrivateKey: keys.privPEM,
		AlipayPublicKey:  keys.pubBare,
		AlipayGateway:    gatewayURL,
	}
}

// fakeAlipayGateway 按 biz_content 里的 out_trade_no 查表，返回带签名的响应信封。
func fakeAlipayGateway(t *testing.T, keys alipayKeys, responses map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		biz := r.Form.Get("biz_content")
		out := ""
		if idx := strings.Index(biz, "OT-"); idx >= 0 {
			end := idx + 3
			for end < len(biz) && biz[end] != '"' {
				end++
			}
			out = biz[idx:end]
		}
		body, ok := responses[out]
		if !ok {
			body = `{"code":"40004","msg":"Business Failed","sub_code":"ACQ.TRADE_NOT_EXIST","sub_msg":"交易不存在"}`
		}
		envelope := fmt.Sprintf(`{"sign":%q,"alipay_trade_query_response":%s}`, keys.sign(t, body), body)
		w.Header().Set("Content-Type", "application/json;charset=utf-8")
		_, _ = w.Write([]byte(envelope))
	}))
}
