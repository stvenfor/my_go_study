// notify.go 渠道异步回调入口的域内编排（验签 → 校验 → 单事务核销 → 指标/日志）。
package payment

import (
	"context"
	"errors"
	"net/url"

	"go.uber.org/zap"
	"model/entity"
)

// HandleWechatNotify 处理微信 v2 回调（raw 为请求体 XML）。
func (u *Service) HandleWechatNotify(ctx context.Context, raw []byte) (*NotifyResult, error) {
	fact, err := VerifyWechatNotify(raw, u.cfg.WeChatMchAPIKey, u.cfg.WeChatAppID, u.cfg.WeChatMchID)
	if err != nil {
		u.observeReject(entity.PaymentChannelWeChat, err)
		return nil, err
	}
	res, err := u.settleByFact(ctx, entity.PaymentChannelWeChat, fact)
	u.observeNotify(entity.PaymentChannelWeChat, res, err)
	return res, err
}

// HandleAlipayNotify 处理支付宝异步通知（form 为 application/x-www-form-urlencoded 解析结果）。
func (u *Service) HandleAlipayNotify(ctx context.Context, form url.Values) (*NotifyResult, error) {
	fact, err := VerifyAlipayNotify(form, u.cfg.AlipayPublicKey, u.cfg.AlipayAppID)
	if err != nil {
		u.observeReject(entity.PaymentChannelAlipay, err)
		return nil, err
	}
	res, err := u.settleByFact(ctx, entity.PaymentChannelAlipay, fact)
	u.observeNotify(entity.PaymentChannelAlipay, res, err)
	return res, err
}

func (u *Service) observeReject(channel string, err error) {
	notifyRejectTotal.WithLabelValues(channel, rejectReason(err)).Inc()
	u.log.Warn("支付回调被拒绝", zap.String("channel", channel), zap.String("reason", rejectReason(err)), zap.Error(err))
}

func (u *Service) observeNotify(channel string, res *NotifyResult, err error) {
	switch {
	case err != nil:
		notifyRejectTotal.WithLabelValues(channel, rejectReason(err)).Inc()
		notifyTotal.WithLabelValues(channel, "rejected").Inc()
		u.log.Error("支付回调核销失败",
			zap.String("channel", channel),
			zap.String("out_trade_no", outTradeNoOf(res)),
			zap.String("reason", rejectReason(err)),
			zap.Error(err))
	case res.Reason == "settled", res.Reason == "settled_no_settler":
		notifyTotal.WithLabelValues(channel, "settled").Inc()
	case res.Reason == "duplicate":
		notifyTotal.WithLabelValues(channel, "duplicate").Inc()
	default:
		notifyTotal.WithLabelValues(channel, "not_paid").Inc()
	}
}

func outTradeNoOf(res *NotifyResult) string {
	if res == nil {
		return ""
	}
	return res.OutTradeNo
}

// rejectReason 把错误归类成有限标签，避免指标基数爆炸。
func rejectReason(err error) string {
	switch {
	case errors.Is(err, ErrSignInvalid):
		return "sign"
	case errors.Is(err, ErrMerchantMismatch):
		return "merchant"
	case errors.Is(err, ErrAmountMismatch):
		return "amount"
	case errors.Is(err, ErrChannelMismatch):
		return "channel"
	case errors.Is(err, ErrRecordNotFound):
		return "no_record"
	case errors.Is(err, ErrRecordNotPending):
		return "terminal_state"
	case errors.Is(err, ErrSettleNotCommitted):
		return "settle_failed"
	default:
		return "internal"
	}
}
