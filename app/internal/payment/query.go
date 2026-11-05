// query.go 主动查单兜底：回调丢失时把 pending 流水补齐（或确认渠道未收款）。
package payment

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"model/entity"
)

// DefaultQueryThreshold 待核销超过该时长即认为回调可能丢失。
const DefaultQueryThreshold = 10 * time.Minute

// DefaultQueryBatch 单轮最多查单条数，避免把渠道打爆。
const DefaultQueryBatch = 200

// QueryStalePending 扫描 created_at 早于 now-threshold 的 pending 流水，逐笔向渠道查单。
// 返回扫描条数；单笔失败只记日志与指标，不影响其余（幂等，下一轮还会扫到）。
func (u *Service) QueryStalePending(ctx context.Context, threshold time.Duration, limit int) (int, error) {
	if u.repo == nil {
		return 0, ErrPayNotConfigured
	}
	if threshold <= 0 {
		threshold = DefaultQueryThreshold
	}
	if limit <= 0 {
		limit = DefaultQueryBatch
	}
	rows, err := u.repo.ListStalePending(ctx, time.Now().Add(-threshold), limit)
	if err != nil {
		return 0, err
	}
	pendingStale.Set(float64(len(rows)))
	for i := range rows {
		rec := rows[i]
		fact, err := u.queryChannel(ctx, rec.Channel, rec.OutTradeNo)
		if err != nil {
			queryTotal.WithLabelValues(rec.Channel, "query_failed").Inc()
			u.log.Warn("支付查单失败",
				zap.String("out_trade_no", rec.OutTradeNo),
				zap.String("channel", rec.Channel),
				zap.Error(err))
			continue
		}
		if !fact.Paid {
			queryTotal.WithLabelValues(rec.Channel, "not_paid").Inc()
			// 渠道明确关单/失败 → 置 failed，否则这条 pending 会被每轮重复扫到。
			switch fact.TradeState {
			case "CLOSED", "PAYERROR", "REVOKED":
				if err := u.repo.MarkFailed(ctx, rec.OutTradeNo, "channel_"+fact.TradeState); err != nil {
					u.log.Error("关单置 failed 失败",
						zap.String("out_trade_no", rec.OutTradeNo), zap.Error(err))
				}
			}
			continue
		}
		res, err := u.settleByFact(ctx, rec.Channel, fact)
		if err != nil {
			queryTotal.WithLabelValues(rec.Channel, "settle_failed").Inc()
			u.log.Error("支付查单后补核销失败",
				zap.String("out_trade_no", rec.OutTradeNo),
				zap.Error(err))
			continue
		}
		if res.NewlySettled {
			queryTotal.WithLabelValues(rec.Channel, "settled").Inc()
		} else {
			queryTotal.WithLabelValues(rec.Channel, "duplicate").Inc()
		}
	}
	return len(rows), nil
}

func (u *Service) queryChannel(ctx context.Context, channel, outTradeNo string) (*notifyFact, error) {
	switch channel {
	case entity.PaymentChannelWeChat:
		return u.queryWechatOrder(ctx, outTradeNo)
	case entity.PaymentChannelAlipay:
		return u.queryAlipayOrder(ctx, outTradeNo)
	default:
		return nil, fmt.Errorf("%w：%s", ErrPayChannelInvalid, channel)
	}
}
