// =============================================================================
// main.go — RabbitMQ Worker 入口（make run-worker → go -C app run ./cmd/worker）
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

	"app/internal/access"
	"app/internal/delivery/worker"
	"app/internal/mall"
	"app/internal/payment"
	"app/internal/points"
	"app/internal/repository/postgres"
	redisrepo "app/internal/repository/redis"
	"app/internal/schemaboot"
	"app/internal/usecase"
	"app/internal/wallet"
	"common/config"
	"common/logger"
	"core/database"
	"core/jpush"
	"core/queue"
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
	pushUC := worker.NewDeliveryPushUsecase(eventRepo, *cfg, publisher)
	queueClient, err := queue.NewClient(*cfg, log)
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

	// Worker 与 API 同一套 schema 闸门（parity-p0-1）：生产路径零 DDL，版本落后即退出。
	if err := schemaboot.Boot(db, schemaboot.Options{
		AppEnv:        cfg.AppEnv,
		AutoMigrate:   cfg.Database.AutoMigrate,
		VersionCheck:  cfg.Database.SchemaVersionCheck,
		MigrationsDir: cfg.Database.MigrationsDir,
		BestEffort: []schemaboot.Step{
			{Name: "极光设备表", Fn: func() error { return postgres.EnsurePushDeviceSchema(db) }},
			{Name: "渠道支付流水表", Fn: func() error { return payment.EnsureSchema(db) }},
		},
	}, log); err != nil {
		return err
	}

	jpushClient := jpush.NewClient(jpush.Config{
		AppKey:         cfg.ThirdParty.JPushAppKey,
		MasterSecret:   cfg.ThirdParty.JPushMasterSecret,
		APNsProduction: cfg.ThirdParty.JPushAPNsProduction,
	})
	jpushUC := usecase.NewJPushUsecase(postgres.NewPushDeviceRepository(db), jpushClient)
	if pushUC != nil {
		pushUC.SetOfflinePusher(jpushUC)
	}

	handler := worker.NewHandler(pushUC, sessionRepo, queueClient, *cfg, redisClient, log).WithJPush(jpushUC)

	// 查单兜底会把 pending 流水核销成已付，核销后必须由本进程做发货，
	// 否则留下「钱已收、单未付」。这里只建核销链上最小的四个服务，不建路由、不跑种子。
	paymentSvc := payment.NewService(cfg.ThirdParty, payment.NewPostgresRepository(db)).
		WithLogger(log).
		WithLocation(cfg.Scheduler.Location())
	mall.NewService(
		mall.NewPostgresRepository(db),
		access.NewService(access.NewPostgresRepository(db)),
		points.NewService(points.NewPostgresRepository(db), points.NewPointsTaskProgress(db)),
		wallet.NewService(wallet.NewPostgresRepository(db)),
	).WithPayment(paymentSvc)
	handler.WithPayment(paymentSvc)

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

	runDone := make(chan struct{})
	go func() {
		defer close(runDone)
		log.Info("RabbitMQ Worker 启动",
			zap.Int("prefetch", consumer.Prefetch()),
			zap.String("queue", consumer.Queue()),
			zap.String("dlq", consumer.DLQ()),
			zap.Bool("dlq_wired", consumer.DLQWired()),
			zap.String("pubsub_channel", cfg.Queue.PubSubChannel()),
		)
		if err := consumer.Run(ctx, handler); err != nil && err != context.Canceled {
			log.Error("RabbitMQ Worker 异常退出", zap.Error(err))
		}
	}()

	go queue.RunHourlyNotifyTicker(ctx, queueClient, *cfg, log)
	go queue.RunPaymentJobsTicker(ctx, queueClient, *cfg, log)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("收到关机信号，停止消费...")
	// 顺序不能变：cancel → 等 Run 退出（此后不再接收新消息）→ 等在途处理完成 → defer 里关连接。
	// 否则 WaitGroup 会与 Run 里的 Add 竞争，或在途消息被半路掐断。
	cancel()
	<-runDone
	consumer.Wait()
	log.Info("Worker 已关闭")
	return nil
}
