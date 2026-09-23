package usecase

import (
	"context"
	"crypto"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stvenfor/my_go_study/pkg/config"
)

var (
	// ErrPayNotConfigured 对应渠道商户密钥未配置。
	ErrPayNotConfigured = errors.New("支付未配置：请填写 WECHAT_MCH_* 或 ALIPAY_APP_ID/ALIPAY_PRIVATE_KEY")
	// ErrPayChannelInvalid 渠道非法。
	ErrPayChannelInvalid = errors.New("支付渠道无效，仅支持 wechat / alipay")
	// ErrPayFailed 渠道下单失败。
	ErrPayFailed = errors.New("预支付失败")
)

// PrepayInput 客户端唤起支付前的下单入参。
type PrepayInput struct {
	Channel   string // wechat | alipay
	AmountFen int64
	Subject   string
	ClientIP  string
}

// PrepayResult 返回给 Flutter WysLoginSharePayService 的 params。
type PrepayResult struct {
	Channel string         `json:"channel"`
	Params  map[string]any `json:"params"`
}

// PaymentUsecase 微信 / 支付宝 App 预支付。
type PaymentUsecase struct {
	cfg    config.ThirdPartyConfig
	client *http.Client
}

// NewPaymentUsecase 创建支付用例。
func NewPaymentUsecase(cfg config.ThirdPartyConfig) *PaymentUsecase {
	return &PaymentUsecase{
		cfg: cfg,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Prepay 按渠道生成客户端 SDK 所需参数。
func (u *PaymentUsecase) Prepay(ctx context.Context, in PrepayInput) (*PrepayResult, error) {
	channel := strings.ToLower(strings.TrimSpace(in.Channel))
	if in.AmountFen <= 0 || strings.TrimSpace(in.Subject) == "" {
		return nil, ErrInvalidParams
	}
	subject := strings.TrimSpace(in.Subject)
	if len(subject) > 120 {
		subject = subject[:120]
	}
	outTradeNo := strings.ReplaceAll(uuid.NewString(), "-", "")

	switch channel {
	case "wechat":
		if !u.cfg.WeChatPayConfigured() {
			return nil, fmt.Errorf("%w（微信：WECHAT_APP_ID / WECHAT_MCH_ID / WECHAT_MCH_API_KEY / WECHAT_MCH_NOTIFY_URL）", ErrPayNotConfigured)
		}
		params, err := u.wechatAppPrepay(ctx, outTradeNo, subject, in.AmountFen, in.ClientIP)
		if err != nil {
			return nil, err
		}
		return &PrepayResult{Channel: "wechat", Params: params}, nil
	case "alipay":
		if !u.cfg.AlipayPayConfigured() {
			return nil, fmt.Errorf("%w（支付宝：ALIPAY_APP_ID / ALIPAY_PRIVATE_KEY）", ErrPayNotConfigured)
		}
		body, err := u.alipayAppOrderInfo(outTradeNo, subject, in.AmountFen)
		if err != nil {
			return nil, err
		}
		return &PrepayResult{
			Channel: "alipay",
			Params: map[string]any{
				"body":      body,
				"orderInfo": body,
			},
		}, nil
	default:
		return nil, ErrPayChannelInvalid
	}
}

func (u *PaymentUsecase) wechatAppPrepay(ctx context.Context, outTradeNo, body string, totalFen int64, clientIP string) (map[string]any, error) {
	if clientIP == "" {
		clientIP = "127.0.0.1"
	}
	nonce := randomHex(16)
	vals := map[string]string{
		"appid":            strings.TrimSpace(u.cfg.WeChatAppID),
		"mch_id":           strings.TrimSpace(u.cfg.WeChatMchID),
		"nonce_str":        nonce,
		"body":             body,
		"out_trade_no":     outTradeNo,
		"total_fee":        fmt.Sprintf("%d", totalFen),
		"spbill_create_ip": clientIP,
		"notify_url":       strings.TrimSpace(u.cfg.WeChatMchNotifyURL),
		"trade_type":       "APP",
	}
	vals["sign"] = wechatSign(vals, strings.TrimSpace(u.cfg.WeChatMchAPIKey))

	xmlBody := wechatMapToXML(vals)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.mch.weixin.qq.com/pay/unifiedorder", strings.NewReader(xmlBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/xml")
	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPayFailed, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: read", ErrPayFailed)
	}
	parsed, err := wechatParseXML(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: xml", ErrPayFailed)
	}
	if parsed["return_code"] != "SUCCESS" || parsed["result_code"] != "SUCCESS" {
		msg := parsed["err_code_des"]
		if msg == "" {
			msg = parsed["return_msg"]
		}
		if msg == "" {
			msg = "unifiedorder failed"
		}
		return nil, fmt.Errorf("%w: %s", ErrPayFailed, msg)
	}
	prepayID := parsed["prepay_id"]
	if prepayID == "" {
		return nil, fmt.Errorf("%w: empty prepay_id", ErrPayFailed)
	}

	ts := fmt.Sprintf("%d", time.Now().Unix())
	pkg := "Sign=WXPay"
	clientNonce := randomHex(16)
	clientVals := map[string]string{
		"appid":     strings.TrimSpace(u.cfg.WeChatAppID),
		"partnerid": strings.TrimSpace(u.cfg.WeChatMchID),
		"prepayid":  prepayID,
		"package":   pkg,
		"noncestr":  clientNonce,
		"timestamp": ts,
	}
	sign := wechatSign(clientVals, strings.TrimSpace(u.cfg.WeChatMchAPIKey))
	return map[string]any{
		"appid":     clientVals["appid"],
		"partnerid": clientVals["partnerid"],
		"prepayid":  prepayID,
		"package":   pkg,
		"noncestr":  clientNonce,
		"timestamp": ts,
		"sign":      sign,
	}, nil
}

func (u *PaymentUsecase) alipayAppOrderInfo(outTradeNo, subject string, amountFen int64) (string, error) {
	amountYuan := fmt.Sprintf("%.2f", float64(amountFen)/100.0)
	biz := fmt.Sprintf(
		`{"out_trade_no":"%s","total_amount":"%s","subject":"%s","product_code":"QUICK_MSECURITY_PAY"}`,
		outTradeNo, amountYuan, escapeJSON(subject),
	)
	params := map[string]string{
		"app_id":      strings.TrimSpace(u.cfg.AlipayAppID),
		"method":      "alipay.trade.app.pay",
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"biz_content": biz,
	}
	if n := strings.TrimSpace(u.cfg.AlipayNotifyURL); n != "" {
		params["notify_url"] = n
	}
	sign, err := alipaySignRSA2(params, strings.TrimSpace(u.cfg.AlipayPrivateKey))
	if err != nil {
		return "", err
	}
	params["sign"] = sign

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+url.QueryEscape(params[k]))
	}
	return strings.Join(parts, "&"), nil
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

func wechatSign(vals map[string]string, apiKey string) string {
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

func wechatMapToXML(vals map[string]string) string {
	var b strings.Builder
	b.WriteString("<xml>")
	for k, v := range vals {
		b.WriteString("<")
		b.WriteString(k)
		b.WriteString("><![CDATA[")
		b.WriteString(v)
		b.WriteString("]]></")
		b.WriteString(k)
		b.WriteString(">")
	}
	b.WriteString("</xml>")
	return b.String()
}

func wechatParseXML(raw []byte) (map[string]string, error) {
	type kv struct {
		XMLName xml.Name
		Value   string `xml:",chardata"`
	}
	type root struct {
		XMLName xml.Name
		Nodes   []kv `xml:",any"`
	}
	var r root
	if err := xml.Unmarshal(raw, &r); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(r.Nodes))
	for _, n := range r.Nodes {
		out[n.XMLName.Local] = strings.TrimSpace(n.Value)
	}
	return out, nil
}

func alipaySignRSA2(params map[string]string, privateKeyPEM string) (string, error) {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k == "sign" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var content strings.Builder
	for i, k := range keys {
		if i > 0 {
			content.WriteByte('&')
		}
		content.WriteString(k)
		content.WriteByte('=')
		content.WriteString(params[k])
	}

	key := normalizePEM(privateKeyPEM)
	block, _ := pem.Decode([]byte(key))
	if block == nil {
		return "", fmt.Errorf("%w: invalid alipay private key PEM", ErrPayNotConfigured)
	}
	var priv *rsa.PrivateKey
	if parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		var ok bool
		priv, ok = parsed.(*rsa.PrivateKey)
		if !ok {
			return "", fmt.Errorf("%w: not RSA private key", ErrPayNotConfigured)
		}
	} else if parsed, err2 := x509.ParsePKCS1PrivateKey(block.Bytes); err2 == nil {
		priv = parsed
	} else {
		return "", fmt.Errorf("%w: parse private key", ErrPayNotConfigured)
	}

	h := crypto.SHA256.New()
	h.Write([]byte(content.String()))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, h.Sum(nil))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

func normalizePEM(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.ReplaceAll(s, `\n`, "\n")
	if strings.Contains(s, "BEGIN") {
		return s
	}
	return "-----BEGIN PRIVATE KEY-----\n" + s + "\n-----END PRIVATE KEY-----"
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
