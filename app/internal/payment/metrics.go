// metrics.go payment 域自定义指标，注册进 common/metrics 的进程级 Registry。
//
// 之所以在本域声明而非改 common/metrics：P0-2 的文件白名单只含 payment/，
// 且这些指标名与 deploy/observability/alerts.yml 的 payment 组一一对应。
package payment

import (
	"github.com/prometheus/client_golang/prometheus"

	"common/metrics"
)

var (
	// notifyTotal 回调处理次数：result 取 settled / duplicate / not_paid / rejected。
	notifyTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "payment_notify_total",
		Help: "Payment notify callbacks by channel and result.",
	}, []string{"channel", "result"})

	// notifyRejectTotal 被拒绝的回调，按拒绝原因分类（验签/商户号/金额/渠道/无流水）。
	notifyRejectTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "payment_notify_reject_total",
		Help: "Rejected payment notify callbacks by channel and reject reason.",
	}, []string{"channel", "reason"})

	// queryTotal 主动查单结果计数。
	queryTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "payment_query_total",
		Help: "Channel order queries by channel and result.",
	}, []string{"channel", "result"})

	// pendingStale 本轮扫到的超时未核销流水数（>0 说明回调丢失或渠道未收款）。
	pendingStale = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "payment_pending_stale",
		Help: "Pending payment records older than the query threshold.",
	})

	// reconcileDiffTotal 对账写出的差异条数，按类型。
	reconcileDiffTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "payment_reconcile_diff_total",
		Help: "Reconciliation differences written by kind.",
	}, []string{"kind"})
)

func init() {
	metrics.Registry.MustRegister(notifyTotal, notifyRejectTotal, queryTotal, pendingStale, reconcileDiffTotal)
}
