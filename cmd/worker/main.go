// =============================================================================
// main.go — Asynq Worker 入口（make run-worker → go run ./cmd/worker）
//
// 职责：消费 Asynq 任务（Push/SMS/JPush/定时广播），经 Redis Pub/Sub 广播到各 BFF 实例。
// =============================================================================
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"
	redisrepo "github.com/stvenfor/my_go_study/internal/repository/redis"
	"github.com/stvenfor/my_go_study/pkg/config"
	"github.com/stvenfor/my_go_study/pkg/database"
	"github.com/stvenfor/my_go_study/pkg/logger"
	"github.com/stvenfor/my_go_study/pkg/queue"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Worker 启动失败: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	cfg, err := config.Load(config.ResolveConfigDir(), env)
	if err != nil {
		return err
	}
	if !cfg.Queue.Enabled {
		return fmt.Errorf("queue.enabled=false，无需启动 Worker；如需异步 Push 请在 config.dev.yaml 启用 queue.enabled")
	}

	log, err := logger.Init(cfg.Log)
	if err != nil {
		return err
	}
	defer logger.Sync()

	redisClient, err := database.NewRedis(cfg.Redis)
	if err != nil {
		return err
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Warn("关闭 Redis 失败", zap.Error(err))
		}
	}()

	eventRepo := redisrepo.NewRealtimeEventRepository(redisClient)
	sessionRepo := redisrepo.NewSessionRepository(redisClient)
	publisher := queue.NewFanoutPublisher(redisClient, cfg.Queue.PubSubChannel())
	pushUC := queue.NewDeliveryPushUsecase(eventRepo, *cfg, publisher)
	queueClient := queue.NewAsynqClient(*cfg)
	defer func() {
		if err := queueClient.Close(); err != nil {
			log.Warn("关闭 Asynq 客户端失败", zap.Error(err))
		}
	}()

	server := queue.NewAsynqServer(*cfg)
	mux := asynq.NewServeMux()
	queue.NewHandler(pushUC, sessionRepo, queueClient, *cfg, redisClient, log).Register(mux)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Info("Asynq Worker 启动",
			zap.Int("concurrency", cfg.Queue.AsynqConcurrency()),
			zap.String("pubsub_channel", cfg.Queue.PubSubChannel()),
		)
		if err := server.Run(mux); err != nil {
			log.Fatal("Asynq Worker 异常退出", zap.Error(err))
		}
	}()

	if cfg.Scheduler.Enabled && cfg.Scheduler.HourlyNotify.Enabled {
		scheduler := queue.NewAsynqScheduler(*cfg)
		entryID, err := queue.RegisterHourlyNotifyScheduler(scheduler, *cfg)
		if err != nil {
			return fmt.Errorf("注册定时任务失败: %w", err)
		}
		go func() {
			log.Info("Asynq Scheduler 启动",
				zap.String("cron", cfg.Scheduler.HourlyNotify.CronSpec()),
				zap.String("timezone", cfg.Scheduler.Timezone),
				zap.String("entry_id", entryID),
			)
			if err := scheduler.Run(); err != nil {
				log.Error("Asynq Scheduler 异常退出", zap.Error(err))
			}
		}()
	} else {
		log.Info("定时广播未启用（scheduler.hourly_notify.enabled=false）")
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
		log.Info("收到关机信号，停止 Worker...")
	case <-ctx.Done():
	}

	server.Shutdown()
	log.Info("Worker 已关闭")
	return nil
}
