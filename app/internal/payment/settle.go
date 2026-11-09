// settle.go 核销编排：流水置 paid + 业务发货/发放权益在**同一事务**内完成。
//
// 任一侧失败整体回滚，流水保持 pending：渠道还会重投回调、查单任务也会补，
// 不会出现「渠道已收款、业务无记录」或反之的半状态（复用 P0-3 的 txdb）。
package payment

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"model/entity"
)

// Settler 业务域收到「渠道已收款」后的核销动作。
// refID 由各域自己解析（商城是 order_id，会员是 membership order_id）。
type Settler interface {
	SettleChannelPaid(ctx context.Context, refType, refID string, rec *entity.WysPaymentRecord) error
}

// NotifyResult 回调处理结论（HTTP 层据此决定应答内容与指标）。
type NotifyResult struct {
	OutTradeNo   string
	Accepted     bool // 是否需要渠道重投：false 表示本地判定不通过
	NewlySettled bool // 首次核销（据此触发业务发货）
	Reason       string
	Record       *entity.WysPaymentRecord
}

// RegisterSettler 按 ref_type 注册业务核销器（装配期由 main 调用）。
func (u *Service) RegisterSettler(refType string, s Settler) {
	if s == nil {
		return
	}
	u.settlers[refType] = s
}

// SettledRecord 返回某业务单已核销的流水；没有则 ErrRecordNotFound。
// 供 membership 的 ConfirmBuyout 做「必须有已核销流水才发权益」的校验。
func (u *Service) SettledRecord(ctx context.Context, refType, refID string) (*entity.WysPaymentRecord, error) {
	if u.repo == nil {
		return nil, ErrRecordNotFound
	}
	return u.repo.GetPaidByRef(ctx, refType, refID)
}

// ConfirmRefPaid 客户端支付回跳时的同步确认：本地已核销直接返回；
// 仍 pending 则立刻向渠道查单补齐（异步回调经常晚于 App 回跳）。
// 查不到已付事实返回 ErrSettleNotCommitted，调用方据此回「支付确认中」而不是发货。
func (u *Service) ConfirmRefPaid(ctx context.Context, refType, refID string) (*entity.WysPaymentRecord, error) {
	if u.repo == nil {
		return nil, ErrRecordNotFound
	}
	rec, err := u.repo.GetLatestByRef(ctx, refType, refID)
	if err != nil {
		return nil, err
	}
	if rec.IsPaid() {
		return rec, nil
	}
	fact, err := u.queryChannel(ctx, rec.Channel, rec.OutTradeNo)
	if err != nil {
		return nil, errors.Join(ErrSettleNotCommitted, err)
	}
	if !fact.Paid {
		return nil, ErrSettleNotCommitted
	}
	res, err := u.settleByFact(ctx, rec.Channel, fact)
	if err != nil {
		return nil, err
	}
	return res.Record, nil
}

// settleByFact 把渠道事实落到流水并核销；幂等：重复投递返回 NewlySettled=false。
func (u *Service) settleByFact(ctx context.Context, channel string, fact *notifyFact) (*NotifyResult, error) {
	if fact.OutTradeNo == "" {
		return nil, ErrSignInvalid
	}
	rec, err := u.repo.GetByOutTradeNo(ctx, fact.OutTradeNo)
	if err != nil {
		return nil, err
	}
	if rec.Channel != channel {
		return nil, ErrChannelMismatch
	}
	res := &NotifyResult{OutTradeNo: fact.OutTradeNo, Record: rec, Accepted: true}
	if !fact.Paid {
		res.Reason = "not_paid"
		return res, nil
	}
	err = u.repo.WithTx(ctx, func(ctx context.Context) error {
		settled, newly, err := u.repo.SettlePaid(ctx, fact.OutTradeNo, fact.ChannelTradeNo, fact.AmountFen,
			fact.Raw, fact.PaidAt)
		if err != nil {
			return err
		}
		res.Record = settled
		if !newly {
			res.Reason = "duplicate"
			return nil
		}
		settler, ok := u.settlers[settled.RefType]
		if !ok {
			// 兜底：该 ref_type 没有可用业务核销器时，流水照实置 paid 并留痕，
			// 业务侧由 ConfirmRefPaid/查单收敛，不把未知 ref_type 塞进本事务。
			// membership_order 已由 membership 的 Settler 自注册、仓储也已支持
			// txdb，正常路径走下面的 settler 分支；这里只兜「真未知」的 ref_type。
			u.log.Warn("渠道已收款但该 ref_type 无可用业务核销器（兜底留痕）",
				zap.String("ref_type", settled.RefType),
				zap.String("ref_id", settled.RefID),
				zap.String("out_trade_no", settled.OutTradeNo))
			res.Reason = "settled_no_settler"
			return nil
		}
		if err := settler.SettleChannelPaid(ctx, settled.RefType, settled.RefID, settled); err != nil {
			// 业务侧没做成，流水回滚为 pending（渠道重投/查单仍可补齐）。
			return errors.Join(ErrSettleNotCommitted, err)
		}
		res.NewlySettled = true
		res.Reason = "settled"
		return nil
	})
	if err != nil {
		return res, err
	}
	return res, nil
}
