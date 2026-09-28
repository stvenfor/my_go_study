// =============================================================================
// main.go — RabbitMQ Worker 入口（make run-worker → go run ./cmd/worker）
//
// 职责：消费 RabbitMQ 任务（Push/SMS/JPush/定时广播），经 Redis Pub/Sub 广播到各 BFF 实例。
// =============================================================================
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/stvenfor/my_go_study/internal/repository/postgres"
	redisrepo "github.com/stvenfor/my_go_study/internal/repository/redis"
	"github.com/stvenfor/my_go_study/internal/usecase"
	"github.com/stvenfor/my_go_study/pkg/config"
	"github.com/stvenfor/my_go_study/pkg/database"
	"github.com/stvenfor/my_go_study/pkg/jpush"
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
	queueClient, err := queue.NewClient(*cfg)
	if err != nil {
		return fmt.Errorf("连接 RabbitMQ 失败: %w", err)
	}
	defer func() {
		if err := queueClient.Close(); err != nil {
			log.Warn("关闭 RabbitMQ 客户端失败", zap.Error(err))
		}
	}()

	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		return fmt.Errorf("连接 Postgres 失败: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer func() { _ = sqlDB.Close() }()

	var jpushUC *usecase.JPushUsecase
	pushDeviceRepo := postgres.NewPushDeviceRepository(db)
	if err := postgres.EnsurePushDeviceSchema(db); err != nil {
		log.Warn("极光设备表初始化失败", zap.Error(err))
	} else {
		jpushClient := jpush.NewClient(jpush.Config{
			AppKey:         cfg.ThirdParty.JPushAppKey,
			MasterSecret:   cfg.ThirdParty.JPushMasterSecret,
			APNsProduction: cfg.ThirdParty.JPushAPNsProduction,
		})
		jpushUC = usecase.NewJPushUsecase(pushDeviceRepo, jpushClient)
		if pushUC != nil {
			pushUC.SetOfflinePusher(jpushUC)
		}
	}

	handler := queue.NewHandler(pushUC, sessionRepo, queueClient, *cfg, redisClient, log).WithJPush(jpushUC)
	consumer, err := queue.NewConsumer(*cfg, log)
	if err != nil {
		return fmt.Errorf("创建 RabbitMQ 消费者失败: %w", err)
	}
	defer func() {
		if err := consumer.Close(); err != nil {
			log.Warn("关闭 RabbitMQ 消费者失败", zap.Error(err))
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Info("RabbitMQ Worker 启动",
			zap.Int("prefetch", consumer.Prefetch()),
			zap.String("queue", consumer.Queue()),
			zap.String("pubsub_channel", cfg.Queue.PubSubChannel()),
		)
		if err := consumer.Run(ctx, handler); err != nil && err != context.Canceled {
			log.Fatal("RabbitMQ Worker 异常退出", zap.Error(err))
		}
	}()

	go queue.RunHourlyNotifyTicker(ctx, queueClient, *cfg, log)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("收到关机信号，停止 Worker...")
	cancel()
	log.Info("Worker 已关闭")
	return nil
}
