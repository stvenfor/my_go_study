//go:build integration

// payment_ledger_integration_test.go 渠道支付的流水留痕与跨域核销（parity-p0-2 硬门槛③）。
//
// 跑法（与 P0-3 同一 scratch 库，绝不连业务库 my_go_study）：
//
//	docker exec my_go_study_postgres psql -U postgres -d postgres -c "CREATE DATABASE p03_it"
//	GOPROXY=https://goproxy.cn,direct go -C app test -tags=integration ./internal/mall/ -run TestChannelPay -count=1
//
// 这里验的是「商城 → payment → 商城核销」这条跨域链在真库上成立：
// 门槛③要求走微信渠道的商城订单在 wys_payment_records 必有流水，
// 而流水 paid 之后订单必须同步推进（否则又回到「钱已收、单未付」）。
package mall

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"testing"

	"app/internal/payment"
	"app/internal/points"
	"app/internal/wallet"
	"common/config"
	"gorm.io/gorm"
	"model/entity"
)

// stubWechatGateway 只应答 unifiedorder：返回 SUCCESS + prepay_id。
// 注意本域**不校验 unifiedorder 响应签名**（硬门槛只约束回调报文），
// 所以这里无需按商户 key 签名。
func stubWechatGateway(t *testing.T, prepayID string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pay/unifiedorder" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprintf(w, "<xml><return_code><![CDATA[SUCCESS]]></return_code>"+
			"<result_code><![CDATA[SUCCESS]]></result_code>"+
			"<prepay_id><![CDATA[%s]]></prepay_id></xml>", prepayID)
	}))
}

func wechatPayCfg(endpoint string) config.ThirdPartyConfig {
	return config.ThirdPartyConfig{
		WeChatAppID: "wx-app-id", WeChatMchID: "mch-10086",
		WeChatMchAPIKey:    "it-mch-api-key-0123456789abcdef",
		WeChatMchNotifyURL: "http://127.0.0.1:8081/api/v1/payments/notify/wechat",
		WeChatPayEndpoint:  endpoint,
	}
}

// newChannelPayService 复刻 cmd/api 的装配：payment 持有流水仓储，mall 注册为 mall_order 核销器。
func newChannelPayService(db *gorm.DB, cfg config.ThirdPartyConfig) (*Service, *payment.Service) {
	paySvc := payment.NewService(cfg, payment.NewPostgresRepository(db))
	mallSvc := NewService(NewPostgresRepository(db), nil,
		points.NewService(points.NewPostgresRepository(db), points.NewPointsTaskProgress(db)),
		wallet.NewService(wallet.NewPostgresRepository(db))).WithPayment(paySvc)
	return mallSvc, paySvc
}

func ledgerRows(t *testing.T, db *gorm.DB, orderID int64) []entity.WysPaymentRecord {
	t.Helper()
	var rows []entity.WysPaymentRecord
	err := db.Where("ref_type = ? AND ref_id = ?", entity.PaymentRefMallOrder,
		strconv.FormatInt(orderID, 10)).Order("id").Find(&rows).Error
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

// TestChannelPayWritesPendingLedger：微信渠道下单必须先落一条 pending 流水，
// 且订单仍停在未支付（本地无权判定「已付」）。
func TestChannelPayWritesPendingLedger(t *testing.T) {
	db := openMoneyDB(t)
	ctx := context.Background()
	uid := "p02-mall-prepay"
	seedBuyer(t, db, uid, 9101, "15.00", 5)

	gw := stubWechatGateway(t, "wx-prepay-it-1")
	defer gw.Close()
	svc, _ := newChannelPayService(db, wechatPayCfg(gw.URL))

	res, err := svc.PayOrder(ctx, uid, 9101, entity.MallPayWeChat)
	if err != nil {
		t.Fatalf("渠道下单失败：%v", err)
	}
	if res.Status != "pending" || res.OutTradeNo == "" {
		t.Fatalf("返回=%+v，期望 status=pending 且带 out_trade_no", res)
	}
	if got := res.PrepayParams["prepayid"]; got != "wx-prepay-it-1" {
		t.Errorf("未把渠道 prepay_id 透给客户端：params=%v", res.PrepayParams)
	}

	rows := ledgerRows(t, db, 9101)
	if len(rows) != 1 {
		t.Fatalf("流水 %d 条，应为 1 条", len(rows))
	}
	rec := rows[0]
	if rec.OutTradeNo != res.OutTradeNo {
		t.Errorf("返回的单号 %q 与流水 %q 不一致", res.OutTradeNo, rec.OutTradeNo)
	}
	if rec.Status != entity.PaymentStatusPending {
		t.Errorf("流水状态 %d，应为 pending %d", rec.Status, entity.PaymentStatusPending)
	}
	if rec.Channel != entity.PaymentChannelWeChat || rec.AmountFen != 1500 || rec.UserID != uid {
		t.Errorf("流水内容不符：%+v", rec)
	}
	if got := orderStatus(t, db, 9101); got != entity.MallOrderUnpaid {
		t.Fatalf("渠道支付把本地订单改成了已付 %d（应为未支付 %d）", got, entity.MallOrderUnpaid)
	}
}

// TestChannelPayGatewayUnreachableKeepsPending：渠道调用失败时流水必须留在 pending，
// 不能置 failed——渠道可能已建单，置 failed 就是「渠道收了钱、本地判死」。
func TestChannelPayGatewayUnreachableKeepsPending(t *testing.T) {
	db := openMoneyDB(t)
	ctx := context.Background()
	uid := "p02-mall-down"
	seedBuyer(t, db, uid, 9102, "15.00", 5)

	gw := stubWechatGateway(t, "wx-prepay-it-2")
	url := gw.URL
	gw.Close() // 先记下地址再关，模拟渠道网络不可达
	svc, _ := newChannelPayService(db, wechatPayCfg(url))

	if _, err := svc.PayOrder(ctx, uid, 9102, entity.MallPayWeChat); err == nil {
		t.Fatal("渠道不可达却返回成功")
	}
	rows := ledgerRows(t, db, 9102)
	if len(rows) != 1 {
		t.Fatalf("流水 %d 条，应为 1 条（下单失败也要留可查单的锚点）", len(rows))
	}
	if rows[0].Status != entity.PaymentStatusPending {
		t.Errorf("流水状态 %d，期望留在 pending（由查单任务收敛）", rows[0].Status)
	}
}

// TestChannelNotifySettlesMallOrder：真回调（签名合法）→ 流水 paid + 订单推进，同一事务。
// 重复投递第二次不重复发货（门槛①在跨域真库下的形态）。
func TestChannelNotifySettlesMallOrder(t *testing.T) {
	db := openMoneyDB(t)
	ctx := context.Background()
	uid := "p02-mall-notify"
	seedBuyer(t, db, uid, 9103, "15.00", 5)

	gw := stubWechatGateway(t, "wx-prepay-it-3")
	defer gw.Close()
	cfg := wechatPayCfg(gw.URL)
	mallSvc, paySvc := newChannelPayService(db, cfg)

	res, err := mallSvc.PayOrder(ctx, uid, 9103, entity.MallPayWeChat)
	if err != nil {
		t.Fatalf("下单失败：%v", err)
	}
	body := signedWechatNotify(t, cfg, map[string]string{
		"out_trade_no": res.OutTradeNo, "transaction_id": "wx-txn-3",
		"total_fee": "1500", "time_end": "20261104120000",
	})

	notifyRes, err := paySvc.HandleWechatNotify(ctx, body)
	if err != nil {
		t.Fatalf("合法回调被拒：%v", err)
	}
	if !notifyRes.NewlySettled {
		t.Fatalf("首次回调未触发业务核销：res=%+v", notifyRes)
	}
	if got := orderStatus(t, db, 9103); got != entity.MallOrderPaid {
		t.Fatalf("订单状态 %d，应为已支付 %d", got, entity.MallOrderPaid)
	}
	rows := ledgerRows(t, db, 9103)
	if len(rows) != 1 || rows[0].Status != entity.PaymentStatusPaid {
		t.Fatalf("核销未落库：%+v", rows)
	}
	if rows[0].ChannelTradeNo == nil || *rows[0].ChannelTradeNo != "wx-txn-3" {
		t.Errorf("渠道单号未回填：%+v", rows[0])
	}

	// 同一回调再投两次：流水仍 1 条 paid，订单不被重复推进。
	for i := 0; i < 2; i++ {
		res, err := paySvc.HandleWechatNotify(ctx, body)
		if err != nil {
			t.Fatalf("重复回调 %d 报错：%v", i, err)
		}
		if res.NewlySettled {
			t.Fatalf("重复回调 %d 又核销了一次", i)
		}
	}
	if got := len(ledgerRows(t, db, 9103)); got != 1 {
		t.Fatalf("重复回调写出 %d 条流水", got)
	}
	var payCount int64
	if err := db.Model(&entity.WysMallPayment{}).Where("order_id = ?", 9103).Count(&payCount).Error; err != nil {
		t.Fatal(err)
	}
	if payCount != 1 {
		t.Fatalf("重复回调重复发货：wys_mall_payment %d 条，应为 1 条", payCount)
	}
	var sku entity.WysMallSKU
	if err := db.Where("sku_id = ?", 9103*100+1).First(&sku).Error; err != nil {
		t.Fatal(err)
	}
	if sku.StockQty != 4 {
		t.Fatalf("库存被重复扣减：%d，应为 4", sku.StockQty)
	}
}

// TestChannelNotifyTamperedAmountKeepsUnpaid：金额不符的回调必须被拒，
// 流水与订单都停在原状态（门槛②的跨域形态）。
func TestChannelNotifyTamperedAmountKeepsUnpaid(t *testing.T) {
	db := openMoneyDB(t)
	ctx := context.Background()
	uid := "p02-mall-tamper"
	seedBuyer(t, db, uid, 9104, "15.00", 5)

	gw := stubWechatGateway(t, "wx-prepay-it-4")
	defer gw.Close()
	cfg := wechatPayCfg(gw.URL)
	mallSvc, paySvc := newChannelPayService(db, cfg)

	res, err := mallSvc.PayOrder(ctx, uid, 9104, entity.MallPayWeChat)
	if err != nil {
		t.Fatalf("下单失败：%v", err)
	}
	// 按 1 分签名后把金额改成 1500：模拟中间人抬价。
	body := signedWechatNotify(t, cfg, map[string]string{
		"out_trade_no": res.OutTradeNo, "transaction_id": "wx-txn-4",
		"total_fee": "1", "time_end": "20261104120000",
	})
	body = []byte(strings.Replace(string(body), "<total_fee><![CDATA[1]]></total_fee>",
		"<total_fee><![CDATA[1500]]></total_fee>", 1))

	if _, err := paySvc.HandleWechatNotify(ctx, body); err == nil {
		t.Fatal("篡改金额的回调通过了")
	}
	if got := orderStatus(t, db, 9104); got != entity.MallOrderUnpaid {
		t.Errorf("订单状态 %d，应仍未支付", got)
	}
	if got := ledgerRows(t, db, 9104)[0].Status; got != entity.PaymentStatusPending {
		t.Errorf("流水状态 %d，应仍为 pending", got)
	}
}

// signedWechatNotify 按微信 v2 规则造一份签名合法的回调报文。
// 签名算法与 payment.wechatSign 一致（字典序 k=v& + &key=商户key，MD5 大写）；
// 这里是测试侧的独立实现，用「另一个实现签出来的包能被域内验通」反过来锁住算法。
func signedWechatNotify(t *testing.T, cfg config.ThirdPartyConfig, vals map[string]string) []byte {
	t.Helper()
	all := map[string]string{
		"appid":       cfg.WeChatAppID,
		"mch_id":      cfg.WeChatMchID,
		"sub_isv":     "",
		"nonce_str":   "0123456789abcdef0123456789abcdef",
		"sign_type":   "MD5",
		"result_code": "SUCCESS",
		"trade_state": "SUCCESS",
	}
	for k, v := range vals {
		all[k] = v
	}
	keys := make([]string, 0, len(all))
	for k, v := range all {
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
		b.WriteString(all[k])
	}
	b.WriteString("&key=")
	b.WriteString(cfg.WeChatMchAPIKey)
	sum := md5.Sum([]byte(b.String()))
	all["sign"] = strings.ToUpper(hex.EncodeToString(sum[:]))

	var xml strings.Builder
	xml.WriteString("<xml>")
	for _, k := range append(keys, "sign") {
		fmt.Fprintf(&xml, "<%s><![CDATA[%s]]></%s>", k, all[k], k)
	}
	xml.WriteString("</xml>")
	return []byte(xml.String())
}
