// verify.go 渠道回调报文验签与字段校验。
//
// notify 路由是公网免鉴权入口，**验签是唯一的信任来源**，因此：
//   - 缺密钥/缺公钥 → 一律拒绝（而不是跳过校验）；
//   - 商户号 / app_id / 金额 / 渠道 四项全部要和本地流水对上；
//   - 任何一项不符都只返回错误，由上层落指标与日志，绝不标记已付。
package payment

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"model/entity"
)

// notifyFact 验签通过后得到的渠道事实。
type notifyFact struct {
	OutTradeNo     string
	ChannelTradeNo string
	AmountFen      int64
	Paid           bool
	TradeState     string
	PaidAt         time.Time
	Raw            string
}

// VerifyWechatNotify 校验微信 v2 回调 XML 的 MD5 签名，并取出核销所需事实。
// apiKey 为空直接拒绝：无法验签的部署不应接受回调。
func VerifyWechatNotify(raw []byte, apiKey, appID, mchID string) (*notifyFact, error) {
	key := strings.TrimSpace(apiKey)
	if key == "" {
		return nil, fmt.Errorf("%w：微信商户密钥未配置", ErrSignInvalid)
	}
	vals, err := wechatParseXML(raw)
	if err != nil {
		return nil, fmt.Errorf("%w：xml 解析失败", ErrSignInvalid)
	}
	if got := strings.TrimSpace(vals["sign"]); got == "" || !strings.EqualFold(wechatSign(vals, key), got) {
		return nil, ErrSignInvalid
	}
	if appID != "" && vals["appid"] != strings.TrimSpace(appID) {
		return nil, fmt.Errorf("%w：appid=%s", ErrMerchantMismatch, vals["appid"])
	}
	if mchID != "" && vals["mch_id"] != strings.TrimSpace(mchID) {
		return nil, fmt.Errorf("%w：mch_id=%s", ErrMerchantMismatch, vals["mch_id"])
	}
	if vals["return_code"] != "" && vals["return_code"] != "SUCCESS" {
		return nil, fmt.Errorf("%w：return_code=%s", ErrSignInvalid, vals["return_code"])
	}
	fen, err := strconv.ParseInt(strings.TrimSpace(vals["total_fee"]), 10, 64)
	if err != nil || fen <= 0 {
		return nil, fmt.Errorf("%w：total_fee=%q", ErrAmountMismatch, vals["total_fee"])
	}
	rc := vals["result_code"]
	tradeState := vals["trade_state"]
	paid := rc == "SUCCESS" && (tradeState == "" || tradeState == "SUCCESS" || tradeState == "REFUND")
	paidAt := parseWechatTime(vals["time_end"])
	return &notifyFact{
		OutTradeNo:     vals["out_trade_no"],
		ChannelTradeNo: vals["transaction_id"],
		AmountFen:      fen,
		Paid:           paid,
		TradeState:     tradeState,
		PaidAt:         paidAt,
		Raw:            string(raw),
	}, nil
}

// parseWechatTime 微信 time_end 形如 20141030122531（东八区）。
func parseWechatTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if t, err := time.ParseInLocation("20060102150405", s, channelZone); err == nil {
		return t
	}
	return time.Now()
}

// channelZone 微信 time_end / 支付宝 gmt_payment 都是东八区字符串；
// 容器里 time.Local 常为 UTC，按本地解析会把 PaidAt 整体挪 8 小时。
var channelZone = time.FixedZone("GMT+8", 8*3600)

// VerifyAlipayNotify 校验支付宝异步通知（RSA2）并取出核销所需事实。
// publicKeyPEM 为支付宝公钥（控制台那串 base64 或完整 PEM 均可）。
func VerifyAlipayNotify(form url.Values, publicKeyPEM, appID string) (*notifyFact, error) {
	pub := strings.TrimSpace(publicKeyPEM)
	if pub == "" {
		return nil, fmt.Errorf("%w：ALIPAY_PUBLIC_KEY 未配置", ErrSignInvalid)
	}
	sig := strings.TrimSpace(form.Get("sign"))
	if sig == "" {
		return nil, ErrSignInvalid
	}
	if signType := strings.TrimSpace(form.Get("sign_type")); signType != "" && signType != "RSA2" {
		return nil, fmt.Errorf("%w：sign_type=%s 仅支持 RSA2", ErrSignInvalid, signType)
	}
	content := alipayNotifyContent(form)
	if err := alipayVerifyRSA2(content, sig, pub); err != nil {
		return nil, err
	}
	if appID != "" && form.Get("app_id") != strings.TrimSpace(appID) {
		return nil, fmt.Errorf("%w：app_id=%s", ErrMerchantMismatch, form.Get("app_id"))
	}
	out := strings.TrimSpace(form.Get("out_trade_no"))
	if out == "" {
		return nil, fmt.Errorf("%w：out_trade_no 为空", ErrSignInvalid)
	}
	yuan := strings.TrimSpace(form.Get("total_amount"))
	fen, err := entity.ParseYuanToFen(yuan)
	if err != nil || fen <= 0 {
		return nil, fmt.Errorf("%w：total_amount=%q", ErrAmountMismatch, yuan)
	}
	status := strings.TrimSpace(form.Get("trade_status"))
	paidAt := parseAlipayTime(form.Get("gmt_payment"))
	raw := form.Encode()
	return &notifyFact{
		OutTradeNo:     out,
		ChannelTradeNo: strings.TrimSpace(form.Get("trade_no")),
		AmountFen:      fen,
		Paid:           status == "TRADE_SUCCESS" || status == "TRADE_FINISHED",
		TradeState:     status,
		PaidAt:         paidAt,
		Raw:            raw,
	}, nil
}

// alipayNotifyContent 待验签串：除 sign / sign_type 外全部参数按 key 字典序拼 k=v&。
func alipayNotifyContent(form url.Values) string {
	keys := make([]string, 0, len(form))
	for k := range form {
		if k == "sign" || k == "sign_type" {
			continue
		}
		if strings.TrimSpace(form.Get(k)) == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+form.Get(k))
	}
	return strings.Join(parts, "&")
}

func alipayVerifyRSA2(content, signB64, publicKeyPEM string) error {
	block, _ := pem.Decode([]byte(normalizePublicKeyPEM(publicKeyPEM)))
	if block == nil {
		return fmt.Errorf("%w：公钥 PEM 解析失败", ErrSignInvalid)
	}
	var pub *rsa.PublicKey
	switch block.Type {
	case "RSA PUBLIC KEY":
		k, err := x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return fmt.Errorf("%w：pkcs1 公钥", ErrSignInvalid)
		}
		pub = k
	default:
		k, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return fmt.Errorf("%w：pkix 公钥", ErrSignInvalid)
		}
		rk, ok := k.(*rsa.PublicKey)
		if !ok {
			return fmt.Errorf("%w：非 RSA 公钥", ErrSignInvalid)
		}
		pub = rk
	}
	sig, err := base64.StdEncoding.DecodeString(signB64)
	if err != nil {
		return fmt.Errorf("%w：sign base64", ErrSignInvalid)
	}
	digest := sha256.Sum256([]byte(content))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, digest[:], sig); err != nil {
		return ErrSignInvalid
	}
	return nil
}

func normalizePublicKeyPEM(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.ReplaceAll(s, `\n`, "\n")
	if strings.Contains(s, "BEGIN") {
		return s
	}
	return "-----BEGIN PUBLIC KEY-----\n" + s + "\n-----END PUBLIC KEY-----"
}

func parseAlipayTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", s, channelZone); err == nil {
		return t
	}
	return time.Now()
}
