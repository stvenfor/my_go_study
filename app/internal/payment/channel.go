// channel.go 渠道侧主动查询（查单兜底与 T+1 对账的唯一事实来源）。
//
// 端点走配置项，默认官方域名；单测与本机联调把 endpoint 指到假网关，
// 仍然跑真实 HTTP 往返 + 真实验签，不是打桩返回。
package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"model/entity"
)

// queryWechatOrder 微信 v2 orderquery：响应必须同样验签。
func (u *Service) queryWechatOrder(ctx context.Context, outTradeNo string) (*notifyFact, error) {
	if !u.cfg.WeChatPayConfigured() {
		return nil, fmt.Errorf("%w（微信查单需商户密钥）", ErrPayNotConfigured)
	}
	vals := map[string]string{
		"appid":        strings.TrimSpace(u.cfg.WeChatAppID),
		"mch_id":       strings.TrimSpace(u.cfg.WeChatMchID),
		"out_trade_no": outTradeNo,
		"nonce_str":    randomHex(16),
	}
	vals["sign"] = wechatSign(vals, strings.TrimSpace(u.cfg.WeChatMchAPIKey))
	endpoint := u.cfg.WeChatPayEndpointOrDefault() + "/pay/orderquery"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(wechatMapToXML(vals)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/xml")
	raw, err := u.do(ctx, req)
	if err != nil {
		return nil, err
	}
	fact, err := VerifyWechatNotify(raw, u.cfg.WeChatMchAPIKey, u.cfg.WeChatAppID, u.cfg.WeChatMchID)
	if err != nil {
		return nil, err
	}
	if fact.TradeState == "CLOSED" || fact.TradeState == "PAYERROR" {
		fact.Paid = false
	}
	return fact, nil
}

// queryAlipayOrder 支付宝 alipay.trade.query：按 out_trade_no 查单笔。
func (u *Service) queryAlipayOrder(ctx context.Context, outTradeNo string) (*notifyFact, error) {
	if !u.cfg.AlipayPayConfigured() {
		return nil, fmt.Errorf("%w（支付宝查单需应用私钥）", ErrPayNotConfigured)
	}
	biz := fmt.Sprintf(`{"out_trade_no":"%s"}`, outTradeNo)
	params := map[string]string{
		"app_id":      strings.TrimSpace(u.cfg.AlipayAppID),
		"method":      "alipay.trade.query",
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"biz_content": biz,
	}
	sign, err := alipaySignRSA2(params, strings.TrimSpace(u.cfg.AlipayPrivateKey))
	if err != nil {
		return nil, err
	}
	params["sign"] = sign
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.cfg.AlipayGatewayOrDefault(),
		strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")
	raw, err := u.do(ctx, req)
	if err != nil {
		return nil, err
	}
	return parseAlipayQueryResponse(raw, strings.TrimSpace(u.cfg.AlipayPublicKey))
}

// parseAlipayQueryResponse 校验响应 sign（对 alipay_trade_query_response 原始 JSON 串验签）。
func parseAlipayQueryResponse(raw []byte, publicKeyPEM string) (*notifyFact, error) {
	var envelope struct {
		Sign string          `json:"sign"`
		Resp json.RawMessage `json:"alipay_trade_query_response"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("%w：查单响应非法 JSON", ErrSignInvalid)
	}
	if len(envelope.Resp) == 0 {
		return nil, fmt.Errorf("%w：查单无业务响应体", ErrSignInvalid)
	}
	fact, err := verifyAlipaySignedBody(envelope.Resp, envelope.Sign, publicKeyPEM)
	if err != nil {
		return nil, err
	}
	return fact, nil
}

func verifyAlipaySignedBody(body json.RawMessage, sign, publicKeyPEM string) (*notifyFact, error) {
	if err := alipayVerifyRSA2(string(body), sign, publicKeyPEM); err != nil {
		return nil, err
	}
	var resp struct {
		Code        string `json:"code"`
		Msg         string `json:"msg"`
		SubCode     string `json:"sub_code"`
		SubMsg      string `json:"sub_msg"`
		OutTradeNo  string `json:"out_trade_no"`
		TradeNo     string `json:"trade_no"`
		TradeStatus string `json:"trade_status"`
		TotalAmount string `json:"total_amount"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("%w：业务体解析失败", ErrSignInvalid)
	}
	if resp.Code != "10000" {
		return nil, fmt.Errorf("%w：code=%s %s %s", ErrPayFailed, resp.Code, resp.SubCode, resp.SubMsg)
	}
	fen, err := entity.ParseYuanToFen(resp.TotalAmount)
	if err != nil || fen <= 0 {
		return nil, fmt.Errorf("%w：total_amount=%q", ErrAmountMismatch, resp.TotalAmount)
	}
	return &notifyFact{
		OutTradeNo:     resp.OutTradeNo,
		ChannelTradeNo: resp.TradeNo,
		AmountFen:      fen,
		Paid:           resp.TradeStatus == "TRADE_SUCCESS" || resp.TradeStatus == "TRADE_FINISHED",
		TradeState:     resp.TradeStatus,
		PaidAt:         time.Now(),
		Raw:            string(body),
	}, nil
}

func (u *Service) do(ctx context.Context, req *http.Request) ([]byte, error) {
	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPayFailed, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: read body", ErrPayFailed)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: http %d", ErrPayFailed, resp.StatusCode)
	}
	return raw, nil
}
