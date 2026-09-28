// hourly_ticker.go worker 侧整点广播（替代 Asynq Scheduler）。
package queue

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/stvenfor/my_go_study/pkg/config"
	"go.uber.org/zap"
)

// RunHourlyNotifyTicker 每分钟检查一次；整点且命中 cron 小时范围时入队 broadcast。
func RunHourlyNotifyTicker(ctx context.Context, client *Client, cfg config.Config, log *zap.Logger) {
	if log == nil {
		log = zap.NewNop()
	}
	if !cfg.Scheduler.Enabled || !cfg.Scheduler.HourlyNotify.Enabled {
		log.Info("定时广播未启用（scheduler.hourly_notify.enabled=false）")
		return
	}
	loc := cfg.Scheduler.Location()
	cron := cfg.Scheduler.HourlyNotify.CronSpec()
	log.Info("HourlyNotify ticker 启动",
		zap.String("cron", cron),
		zap.String("timezone", cfg.Scheduler.Timezone),
	)

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	var lastSlot string

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			now = now.In(loc)
			if now.Minute() != 0 {
				continue
			}
			if !hourMatchesCron(cron, now.Hour()) {
				continue
			}
			slot := now.Truncate(time.Hour).Format(time.RFC3339)
			if slot == lastSlot {
				continue
			}
			if _, err := client.EnqueueBroadcastNotify(ctx, BroadcastNotifyPayload{ScheduleSlot: slot}); err != nil {
				log.Error("hourly broadcast enqueue failed", zap.Error(err), zap.String("slot", slot))
				continue
			}
			lastSlot = slot
			log.Info("hourly broadcast enqueued by ticker", zap.String("slot", slot))
		}
	}
}

// hourMatchesCron 解析简易 cron（如 "0 10-19 * * *"）的小时字段。
func hourMatchesCron(cron string, hour int) bool {
	parts := strings.Fields(strings.TrimSpace(cron))
	if len(parts) < 2 {
		return hour >= 10 && hour <= 19
	}
	field := parts[1]
	if field == "*" {
		return true
	}
	if a, b, ok := strings.Cut(field, "-"); ok {
		lo, err1 := strconv.Atoi(a)
		hi, err2 := strconv.Atoi(b)
		if err1 != nil || err2 != nil {
			return false
		}
		return hour >= lo && hour <= hi
	}
	h, err := strconv.Atoi(field)
	if err != nil {
		return false
	}
	return hour == h
}
