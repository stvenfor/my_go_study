// repository.go payment 域的仓储端口与错误哨兵。
package payment

import (
	"context"
	"errors"
	"time"

	"model/entity"
)

var (
	// ErrRecordNotFound out_trade_no / ref 无对应流水。
	ErrRecordNotFound = errors.New("支付流水不存在")
	// ErrSignInvalid 回调验签失败（任何人可伪造标记已付，本片最高风险，必须拒绝）。
	ErrSignInvalid = errors.New("支付回调验签失败")
	// ErrMerchantMismatch 回调里的商户号/APPID 与本地配置不符。
	ErrMerchantMismatch = errors.New("支付回调商户标识不匹配")
	// ErrAmountMismatch 渠道金额与本地流水金额不符（分）。
	ErrAmountMismatch = errors.New("支付回调金额与流水不符")
	// ErrChannelMismatch 流水渠道与回调渠道不符。
	ErrChannelMismatch = errors.New("支付回调渠道与流水不符")
	// ErrRecordNotPending 流水已是终态，拒绝再次核销。
	ErrRecordNotPending = errors.New("支付流水已处于终态")
	// ErrSettleNotCommitted 渠道已收款但业务核销失败：流水保持 pending，等查单/重投。
	ErrSettleNotCommitted = errors.New("业务核销未成功，流水保持待核销")
)

// Repository 支付流水与对账差异存取。
type Repository interface {
	// CreatePending 落一条 pending 流水（prepay 阶段）。
	CreatePending(ctx context.Context, rec *entity.WysPaymentRecord) error
	GetByOutTradeNo(ctx context.Context, outTradeNo string) (*entity.WysPaymentRecord, error)
	// GetPaidByRef 取某业务单已核销的流水；没有返回 ErrRecordNotFound。
	GetPaidByRef(ctx context.Context, refType, refID string) (*entity.WysPaymentRecord, error)
	// GetLatestByRef 取某业务单最近一条流水（不论状态），用于客户端「支付完成」回跳时补查单。
	GetLatestByRef(ctx context.Context, refType, refID string) (*entity.WysPaymentRecord, error)
	// SettlePaid 锁行做 pending→paid 单向流转；amountFen 必须等于流水金额，否则拒绝。
	// 返回 newlySettled=false 表示该单早已核销（幂等重投）。
	SettlePaid(ctx context.Context, outTradeNo, channelTradeNo string, amountFen int64, notifyRaw string, paidAt time.Time) (rec *entity.WysPaymentRecord, newlySettled bool, err error)
	// MarkFailed 核销判定不通过（验签/金额/商户不符且确认非重放）时置失败。
	MarkFailed(ctx context.Context, outTradeNo, reason string) error
	// ListStalePending 列出 created_at 早于 olderThan 的 pending 流水（主动查单）。
	ListStalePending(ctx context.Context, olderThan time.Time, limit int) ([]entity.WysPaymentRecord, error)
	// ListWindow 列出时间窗内的流水（对账）。
	ListWindow(ctx context.Context, channel string, from, to time.Time) ([]entity.WysPaymentRecord, error)
	CreateDiff(ctx context.Context, diff *entity.WysPaymentReconcileDiff) error
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
