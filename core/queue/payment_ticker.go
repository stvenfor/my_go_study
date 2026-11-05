// payment_ticker.go worker 侧支付兜底任务入队（替代 Asynq Scheduler）。
//
// 与 hourly_ticker.go 同构：本包只做「到点 → 入队」，真正的查单/对账逻辑在
// app/internal/payment，由 delivery/worker 的 Dispatch 执行。
// 基建层不能反向依赖业务层，所以这里不出现任何支付语义。
package queue

import (
	"context"
	"time"

	"common/config"
	"go.uber.org/zap"
)

// RunPaymentJobsTicker 按 query_interval_minutes 入队主动查单；
// 命中 reconcile_hour 那个整点时入队一次 T+1 对账（对昨天）。
func RunPaymentJobsTicker(ctx context.Context, client *Client, cfg config.Config, log *zap.Logger) {
	if log == nil {
		log = zap.NewNop()
	}
	job := cfg.Scheduler.Payment
	if !cfg.Scheduler.Enabled || !job.Enabled {
		log.Info("支付兜底任务未启用（需 scheduler.enabled 与 scheduler.payment.enabled 同时为 true）")
		return
	}
	loc := cfg.Scheduler.Location()
	interval := time.Duration(job.QueryIntervalMinutesOrDefault()) * time.Minute
	log.Info("Payment jobs ticker 启动",
		zap.Duration("query_interval", interval),
		zap.Bool("reconcile_enabled", job.ReconcileEnabled),
		zap.Int("reconcile_hour", job.ReconcileHourOrDefault()),
		zap.String("timezone", cfg.Scheduler.Timezone),
	)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var lastReconcileDate string

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			queryEnv, err := NewPaymentQueryPendingTask(PaymentQueryPendingPayload{})
			if err != nil {
				log.Error("build payment query task failed", zap.Error(err))
				continue
			}
			if _, err := client.Enqueue(ctx, queryEnv); err != nil {
				log.Error("payment query enqueue failed", zap.Error(err))
			}

			// 同一天只入队一次；入队失败不记 lastReconcileDate，下一轮自然重试。
			local := now.In(loc)
			if !job.ReconcileEnabled || local.Hour() != job.ReconcileHourOrDefault() {
				continue
			}
			reconDate := local.AddDate(0, 0, -1).Format("2006-01-02")
			if reconDate == lastReconcileDate {
				continue
			}
			reconcileEnv, err := NewPaymentReconcileTask(PaymentReconcilePayload{Date: reconDate})
			if err != nil {
				log.Error("build payment reconcile task failed", zap.Error(err))
				continue
			}
			if _, err := client.Enqueue(ctx, reconcileEnv); err != nil {
				log.Error("payment reconcile enqueue failed", zap.Error(err), zap.String("recon_date", reconDate))
				continue
			}
			lastReconcileDate = reconDate
			log.Info("payment reconcile enqueued by ticker", zap.String("recon_date", reconDate))
		}
	}
}
