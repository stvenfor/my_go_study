// reconcile.go T+1 对账：逐笔比对本日流水与渠道侧事实，差异落 wys_payment_reconcile_diff。
//
// 只做「本地有 → 问渠道」这一向：反向（渠道有、本地无）需要下载渠道账单文件，
// 本片未实现，已在验收记录里登记为 Deferred（不写成看起来做了的假实现）。
package payment

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"model/entity"
)

// Reconcile 对 day 当天（本地时区自然日）的流水做对账，返回写入的差异条数。
func (u *Service) Reconcile(ctx context.Context, day time.Time) (int, error) {
	if u.repo == nil {
		return 0, ErrPayNotConfigured
	}
	from := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, u.loc)
	to := from.AddDate(0, 0, 1)
	rows, err := u.repo.ListWindow(ctx, "", from, to)
	if err != nil {
		return 0, err
	}
	reconDate := from.Format("2006-01-02")
	diffs := 0
	for i := range rows {
		rec := rows[i]
		diff := u.reconcileOne(ctx, reconDate, &rec)
		if diff == nil {
			continue
		}
		if err := u.repo.CreateDiff(ctx, diff); err != nil {
			u.log.Error("写入对账差异失败",
				zap.String("out_trade_no", rec.OutTradeNo), zap.Error(err))
			continue
		}
		reconcileDiffTotal.WithLabelValues(diff.Kind).Inc()
		diffs++
	}
	return diffs, nil
}

// reconcileOne 单笔比对；无差异返回 nil。
func (u *Service) reconcileOne(ctx context.Context, reconDate string, rec *entity.WysPaymentRecord) *entity.WysPaymentReconcileDiff {
	base := &entity.WysPaymentReconcileDiff{
		ReconDate:      reconDate,
		Channel:        rec.Channel,
		OutTradeNo:     rec.OutTradeNo,
		LocalStatus:    rec.Status,
		LocalAmountFen: rec.AmountFen,
	}
	if rec.ChannelTradeNo != nil {
		base.ChannelTradeNo = *rec.ChannelTradeNo
	}
	fact, err := u.queryChannel(ctx, rec.Channel, rec.OutTradeNo)
	if err != nil {
		// 渠道查无此单：本地记为已付却查不到，是要人工追的差异。
		base.Kind = entity.PaymentDiffChannelMissing
		base.ChannelStatus = "query_error"
		if errors.Is(err, ErrSignInvalid) || errors.Is(err, ErrPayNotConfigured) {
			// 配置/验签问题不算资金差异，交给日志与指标。
			u.log.Warn("对账查单不可用",
				zap.String("out_trade_no", rec.OutTradeNo), zap.Error(err))
			return nil
		}
		return base
	}
	base.ChannelTradeNo = fact.ChannelTradeNo
	base.ChannelStatus = fact.TradeState
	base.ChannelAmountFen = fact.AmountFen

	if fact.AmountFen != rec.AmountFen {
		base.Kind = entity.PaymentDiffAmountMismatch
		return base
	}
	if rec.IsPaid() && !fact.Paid {
		base.Kind = entity.PaymentDiffStatusMismatch
		return base
	}
	if !rec.IsPaid() && fact.Paid {
		// 回调与查单任务都没补上：此刻补核销，并留一条差异说明「晚到核销」。
		if _, err := u.settleByFact(ctx, rec.Channel, fact); err != nil {
			u.log.Error("对账补核销失败",
				zap.String("out_trade_no", rec.OutTradeNo), zap.Error(err))
		}
		base.Kind = entity.PaymentDiffStatusMismatch
		base.ResolveNote = "late_settled_by_reconcile"
		return base
	}
	return nil
}
