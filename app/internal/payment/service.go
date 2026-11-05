// service.go 微信 / 支付宝 App 预支付 + 回调核销 —— 域内业务逻辑。
//
// 由 app/internal/usecase/payment_usecase.go 迁入。原实现只做「通道封装 + 签名」，
// out_trade_no 生成即丢；P0-2 起本域自有仓储（wys_payment_records），
// 预支付先落流水、回调验签后在同一事务里核销（见 settle.go / repo_postgres.go）。
// ErrInvalidParams 原先复用 usecase 包的同名哨兵（按 HTTP 400 映射），
// 切片后本域自带一份，语义一致。
package payment

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

	"common/config"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"model/entity"
)

var (
	// ErrPayNotConfigured 对应渠道商户密钥未配置。
	ErrPayNotConfigured = errors.New("支付未配置：请填写 WECHAT_MCH_* 或 ALIPAY_APP_ID/ALIPAY_PRIVATE_KEY")
	// ErrPayChannelInvalid 渠道非法。
	ErrPayChannelInvalid = errors.New("支付渠道无效，仅支持 wechat / alipay")
	// ErrPayFailed 渠道下单失败。
	ErrPayFailed = errors.New("预支付失败")
	// ErrInvalidParams 入参非法（金额 / 商品描述）。
	ErrInvalidParams = errors.New("invalid params")
)

// PrepayInput 客户端唤起支付前的下单入参。
type PrepayInput struct {
	Channel   string // wechat | alipay
	AmountFen int64
	Subject   string
	ClientIP  string
	// 业务单引用：落库 wys_payment_records 后，notify 核销据此回调对应域。
	// UserID/RefType/RefID 任一为空表示不落流水（仅本地单测路径），
	// 线上装配必须带齐，否则「无流水的已付单」又回来了。
	UserID  string
	RefType string
	RefID   string
	// OutTradeNo 业务侧已有的商户单号（如会员单 out_trade_no 已落库）；
	// 为空则本域生成。传进来是为了让业务单与流水共用同一个商户单号，对账才能对上。
	OutTradeNo string
}

// PrepayResult 返回给 Flutter WysLoginSharePayService 的 params。
type PrepayResult struct {
	Channel    string         `json:"channel"`
	OutTradeNo string         `json:"out_trade_no"`
	Params     map[string]any `json:"params"`
}

// Service 微信 / 支付宝 App 预支付 + 回调核销 + 查单/对账。
type Service struct {
	cfg      config.ThirdPartyConfig
	client   *http.Client
	repo     Repository
	settlers map[string]Settler
	log      *zap.Logger
	// loc 决定「自然日」边界：容器里 time.Local 常是 UTC，
	// 而渠道账单/对账按东八区自然日，必须可由调度时区注入。
	loc *time.Location
}

// NewService 创建支付服务。repo 为 nil 时退化为「只做渠道签单、不落流水」，
// 仅用于单测；装配期由 main 传入 Postgres 仓储。
func NewService(cfg config.ThirdPartyConfig, repo Repository) *Service {
	return &Service{
		cfg:      cfg,
		repo:     repo,
		settlers: make(map[string]Settler),
		log:      zap.NewNop(),
		loc:      time.Local,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// WithLocation 注入自然日时区（装配期传 scheduler.timezone）。
func (u *Service) WithLocation(loc *time.Location) *Service {
	if loc != nil {
		u.loc = loc
	}
	return u
}

// WithLogger 注入日志（回调被拒、核销失败必须可追）。
func (u *Service) WithLogger(log *zap.Logger) *Service {
	if log != nil {
		u.log = log
	}
	return u
}

// Prepay 按渠道生成客户端 SDK 所需参数，并落一条 pending 流水。
// 顺序是「先落流水再问渠道」：渠道一旦受理，回调随时可能到，
// 本地必须先有可寻址的 out_trade_no。
//
// 渠道调用失败**不置 failed**：网络超时等情况下渠道可能已经建单，
// 置 failed 就成了「渠道收了钱、本地判死」的死路；留在 pending 由查单任务收敛。
// 只有确定没到渠道（未配置 / 参数非法 / 渠道名非法）时才置 failed。
func (u *Service) Prepay(ctx context.Context, in PrepayInput) (*PrepayResult, error) {
	channel := strings.ToLower(strings.TrimSpace(in.Channel))
	if in.AmountFen <= 0 || strings.TrimSpace(in.Subject) == "" {
		return nil, ErrInvalidParams
	}
	subject := strings.TrimSpace(in.Subject)
	if len(subject) > 120 {
		subject = subject[:120]
	}
	outTradeNo := strings.TrimSpace(in.OutTradeNo)
	if outTradeNo == "" {
		outTradeNo = strings.ReplaceAll(uuid.NewString(), "-", "")
	}
	persist := u.repo != nil && in.UserID != "" && in.RefType != "" && in.RefID != ""
	if persist {
		rec := &entity.WysPaymentRecord{
			OutTradeNo: outTradeNo,
			UserID:     in.UserID,
			Channel:    channel,
			AmountFen:  in.AmountFen,
			RefType:    in.RefType,
			RefID:      in.RefID,
		}
		if err := u.repo.CreatePending(ctx, rec); err != nil {
			return nil, err
		}
	}
	result, err := u.prepayOnChannel(ctx, channel, outTradeNo, subject, in)
	if err != nil {
		if persist && neverReachedChannel(err) {
			u.failPending(ctx, outTradeNo, err)
		}
		return nil, err
	}
	return result, nil
}

// neverReachedChannel 判定这次失败一定没产生渠道侧订单。
func neverReachedChannel(err error) bool {
	return errors.Is(err, ErrPayNotConfigured) ||
		errors.Is(err, ErrPayChannelInvalid) ||
		errors.Is(err, ErrInvalidParams)
}

func (u *Service) failPending(ctx context.Context, outTradeNo string, cause error) {
	if u.repo == nil {
		return
	}
	if err := u.repo.MarkFailed(ctx, outTradeNo, cause.Error()); err != nil {
		u.log.Error("标记支付流水失败时再错",
			zap.String("out_trade_no", outTradeNo), zap.Error(err))
	}
}

func (u *Service) prepayOnChannel(ctx context.Context, channel, outTradeNo, subject string, in PrepayInput) (*PrepayResult, error) {
	switch channel {
	case entity.PaymentChannelWeChat:
		if !u.cfg.WeChatPayConfigured() {
			return nil, fmt.Errorf("%w（微信：WECHAT_APP_ID / WECHAT_MCH_ID / WECHAT_MCH_API_KEY / WECHAT_MCH_NOTIFY_URL）", ErrPayNotConfigured)
		}
		params, err := u.wechatAppPrepay(ctx, outTradeNo, subject, in.AmountFen, in.ClientIP)
		if err != nil {
			return nil, err
		}
		return &PrepayResult{Channel: entity.PaymentChannelWeChat, OutTradeNo: outTradeNo, Params: params}, nil
	case entity.PaymentChannelAlipay:
		if !u.cfg.AlipayPayConfigured() {
			return nil, fmt.Errorf("%w（支付宝：ALIPAY_APP_ID / ALIPAY_PRIVATE_KEY）", ErrPayNotConfigured)
		}
		body, err := u.alipayAppOrderInfo(outTradeNo, subject, in.AmountFen)
		if err != nil {
			return nil, err
		}
		return &PrepayResult{
			Channel:    entity.PaymentChannelAlipay,
			OutTradeNo: outTradeNo,
			Params: map[string]any{
				"body":      body,
				"orderInfo": body,
			},
		}, nil
	default:
		return nil, ErrPayChannelInvalid
	}
}

func (u *Service) wechatAppPrepay(ctx context.Context, outTradeNo, body string, totalFen int64, clientIP string) (map[string]any, error) {
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
	endpoint := u.cfg.WeChatPayEndpointOrDefault() + "/pay/unifiedorder"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(xmlBody))
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

func (u *Service) alipayAppOrderInfo(outTradeNo, subject string, amountFen int64) (string, error) {
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
