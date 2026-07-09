// scheduler.go Asynq 定时调度注册。
package queue

import (
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/stvenfor/my_go_study/pkg/config"
)

// RegisterHourlyNotifyScheduler 注册每小时系统通知 Cron 任务。
func RegisterHourlyNotifyScheduler(scheduler *asynq.Scheduler, cfg config.Config) (string, error) {
	if !cfg.Scheduler.Enabled || !cfg.Scheduler.HourlyNotify.Enabled {
		return "", nil
	}
	task, err := NewBroadcastNotifyTask(BroadcastNotifyPayload{})
	if err != nil {
		return "", err
	}
	entryID, err := scheduler.Register(cfg.Scheduler.HourlyNotify.CronSpec(), task)
	if err != nil {
		return "", fmt.Errorf("register hourly notify cron: %w", err)
	}
	return entryID, nil
}

// NewAsynqScheduler 创建 Asynq Scheduler。
func NewAsynqScheduler(cfg config.Config) *asynq.Scheduler {
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}
	return asynq.NewScheduler(redisOpt, &asynq.SchedulerOpts{
		Location: cfg.Scheduler.Location(),
	})
}

// RedisClientOpt 返回 Asynq 使用的 Redis 配置。
func RedisClientOpt(cfg config.Config) asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}
}
