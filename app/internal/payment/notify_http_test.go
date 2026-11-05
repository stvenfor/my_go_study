// notify_http_test.go 回调的 HTTP 层口径：公网免会话、应答格式按渠道要求、被拒可观测。
//
// 域内单测只覆盖 HandleXxxNotify 的逻辑，这三条是路由/适配层的：
//   - notify 不在 SessionAuth 链里（装一个「逢请求即 401」的中间件也要能进）；
//   - 处理结果用**正文**表达（微信 v2 看 return_code，支付宝看 success/failure），
//     状态码恒为 200 —— 所以「错签名」的表现是 FAIL/failure 让渠道重投，而不是 4xx；
//   - 被拒的回调要打进 payment_notify_reject_total，alerts.yml 才有依据。
package payment

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"common/metrics"
	"github.com/gin-gonic/gin"
	"model/entity"
)

// denySession 模拟「没有会话就拒」，用来证明 notify 没挂在这条链上。
func denySession(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未登录"})
}

func newNotifyServer(t *testing.T, svc *Service) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	NewRouter(svc, denySession, denySession).Register(engine)
	engine.GET("/metrics", gin.WrapH(metrics.Handler()))
	srv := httptest.NewServer(engine)
	t.Cleanup(srv.Close)
	return srv
}

func postNotify(t *testing.T, url, contentType string, body []byte) (int, string) {
	t.Helper()
	resp, err := http.Post(url, contentType, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, string(raw)
}

// metricValue 读计数器当前值；序列未出现时按 0 算（Prometheus 不导出未触发的序列）。
func metricValue(t *testing.T, srv *httptest.Server, series string) float64 {
	t.Helper()
	resp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(series) + `\s+(\S+)$`).FindStringSubmatch(string(raw))
	if m == nil {
		return 0
	}
	v, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		t.Fatalf("指标 %s 值解析失败：%q", series, m[1])
	}
	return v
}

// 回调路由免会话：渠道没有登录态，唯一门槛是验签。
func TestWechatNotifyRouteIsPublicAndRepliesSUCCESS(t *testing.T) {
	repo := newFakeRepo()
	repo.seedPending("OT-HTTP", entity.PaymentChannelWeChat, entity.PaymentRefMallOrder, "77", 1500, time.Now().Add(-time.Hour))
	svc, settler := settleSvc(repo)
	srv := newNotifyServer(t, svc)

	code, body := postNotify(t, srv.URL+"/api/v1/payments/notify/wechat", "application/xml",
		wechatXML(wechatNotifyVals(map[string]string{"out_trade_no": "OT-HTTP"}), testMchAPIKey))

	if code != http.StatusOK {
		t.Errorf("状态码=%d，微信 v2 约定恒 200、由正文判定", code)
	}
	if !strings.Contains(body, "SUCCESS") {
		t.Errorf("正文未回 SUCCESS（渠道会无限重投）：%s", body)
	}
	if len(settler.calls) != 1 {
		t.Errorf("业务核销=%v，期望调用 1 次", settler.calls)
	}
	if rec, _ := repo.record("OT-HTTP"); rec.Status != entity.PaymentStatusPaid {
		t.Errorf("流水状态=%d，期望 paid", rec.Status)
	}
}

// 同前缀的业务路由不受影响：prepay 仍走 SessionAuth 链（免鉴权只给回调开洞）。
func TestPrepayRouteStillRequiresSession(t *testing.T) {
	svc, _ := settleSvc(newFakeRepo())
	srv := newNotifyServer(t, svc)

	code, body := postNotify(t, srv.URL+"/api/v1/payments/prepay", "application/json",
		[]byte(`{"channel":"wechat","amount_fen":1500,"subject":"x","ref_type":"mall_order","ref_id":"1"}`))
	if code != http.StatusUnauthorized {
		t.Errorf("prepay code=%d body=%s，期望 401（SessionAuth 链不能被回调改造削弱）", code, body)
	}
}

// 篡改金额 → 验签失败：正文必须是 FAIL（渠道重投），本地状态一点不动，且拒绝指标 +1。
func TestWechatNotifyTamperedRepliesFAILAndCountsReject(t *testing.T) {
	repo := newFakeRepo()
	repo.seedPending("OT-HTTP", entity.PaymentChannelWeChat, entity.PaymentRefMallOrder, "77", 1500, time.Now().Add(-time.Hour))
	svc, settler := settleSvc(repo)
	srv := newNotifyServer(t, svc)

	const series = `payment_notify_reject_total{channel="wechat",reason="sign"}`
	before := metricValue(t, srv, series)

	raw := wechatXML(wechatNotifyVals(map[string]string{"out_trade_no": "OT-HTTP"}), testMchAPIKey)
	tampered := bytes.Replace(raw, []byte("<total_fee><![CDATA[1500]]></total_fee>"),
		[]byte("<total_fee><![CDATA[1]]></total_fee>"), 1)
	if bytes.Equal(raw, tampered) {
		t.Fatal("篡改未生效，用例失效")
	}

	code, body := postNotify(t, srv.URL+"/api/v1/payments/notify/wechat", "application/xml", tampered)
	if code != http.StatusOK {
		t.Errorf("状态码=%d，期望 200", code)
	}
	if !strings.Contains(body, "FAIL") {
		t.Errorf("被拒回调未回 FAIL：%s", body)
	}
	if len(settler.calls) != 0 {
		t.Errorf("被拒回调却发货了：%v", settler.calls)
	}
	if rec, _ := repo.record("OT-HTTP"); rec.Status != entity.PaymentStatusPending {
		t.Errorf("被拒回调把流水推到了 %d", rec.Status)
	}
	if got := metricValue(t, srv, series); got <= before {
		t.Errorf("拒绝指标未增长：before=%v after=%v", before, got)
	}
}

// 支付宝侧：正文只认 success / failure 两个词。
func TestAlipayNotifyRouteRepliesPlainTextVerdict(t *testing.T) {
	repo := newFakeRepo()
	repo.seedPending("OT-ALI", entity.PaymentChannelAlipay, entity.PaymentRefMallOrder, "77", 1500, time.Now().Add(-time.Hour))
	keys := newAlipayKeys(t)
	svc, settler := alipaySvc(repo, keys)
	srv := newNotifyServer(t, svc)

	form := keys.notifyForm(t, map[string]string{"out_trade_no": "OT-ALI"})
	code, body := postNotify(t, srv.URL+"/api/v1/payments/notify/alipay",
		"application/x-www-form-urlencoded", []byte(form.Encode()))
	if code != http.StatusOK || body != "success" {
		t.Fatalf("合法回调应答 code=%d body=%q，期望 200/success", code, body)
	}
	if len(settler.calls) != 1 {
		t.Errorf("业务核销=%v，期望 1 次", settler.calls)
	}

	// 签名之后改金额：验签必失败 → failure，流水保持原状。
	form.Set("total_amount", "0.01")
	if _, body = postNotify(t, srv.URL+"/api/v1/payments/notify/alipay",
		"application/x-www-form-urlencoded", []byte(form.Encode())); body != "failure" {
		t.Fatalf("坏签名应答=%q，期望 failure", body)
	}
	if rec, _ := repo.record("OT-ALI"); rec.Status != entity.PaymentStatusPaid {
		t.Errorf("坏签名改动后的流水状态=%d，应停在已核销的 paid", rec.Status)
	}
	if len(settler.calls) != 1 {
		t.Errorf("坏签名又触发了一次核销：%v", settler.calls)
	}
}
