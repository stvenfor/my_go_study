// =============================================================================
// main.go — 应用唯一入口（make run → go run ./cmd/api）
//
// 职责：加载配置 → 连接 PG/Redis → 按 Supabase 开关组装依赖 → 启动 HTTP+WS → 优雅退出
// 详细启动顺序见 AGENTS.md §「应用启动与依赖注入」
// =============================================================================
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stvenfor/my_go_study/internal/delivery/http/controller"
	"github.com/stvenfor/my_go_study/internal/delivery/http/handler"
	"github.com/stvenfor/my_go_study/internal/delivery/http/router"
	wshandler "github.com/stvenfor/my_go_study/internal/delivery/ws"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	redisrepo "github.com/stvenfor/my_go_study/internal/repository/redis"
	sbrepo "github.com/stvenfor/my_go_study/internal/repository/supabase"
	"github.com/stvenfor/my_go_study/internal/repository/postgres"
	"github.com/stvenfor/my_go_study/internal/usecase"
	"github.com/stvenfor/my_go_study/pkg/config"
	"github.com/stvenfor/my_go_study/pkg/database"
	jwtmanager "github.com/stvenfor/my_go_study/pkg/jwt"
	"github.com/stvenfor/my_go_study/pkg/logger"
	pkgsb "github.com/stvenfor/my_go_study/pkg/supabase"
	"github.com/stvenfor/my_go_study/pkg/queue"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "启动失败: %v\n", err)
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

	log, err := logger.Init(cfg.Log)
	if err != nil {
		return err
	}
	defer logger.Sync()

	if !cfg.Supabase.Enabled() {
		log.Warn("Supabase 未启用：请检查 configs/supabase.env 或 SUPABASE_URL / SUPABASE_ANON_KEY 环境变量")
	}

	db, err := database.NewPostgres(cfg.Database)
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	if err := autoMigrate(db); err != nil {
		return err
	}

	redisClient, err := database.NewRedis(cfg.Redis)
	if err != nil {
		return err
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Warn("关闭 Redis 失败", zap.Error(err))
		}
	}()

	jwtMgr := jwtmanager.NewManager(cfg.JWT)
	userRepo := postgres.NewUserRepository(db)
	userUC := usecase.NewUserUsecase(userRepo, jwtMgr)
	sessionRepo := redisrepo.NewSessionRepository(redisClient)
	deviceSessionUC := usecase.NewDeviceSessionUsecase(sessionRepo, cfg.Auth)
	var supabaseAuthUC *usecase.SupabaseAuthUsecase
	var phoneOTPUC *usecase.PhoneOTPUsecase
	var profileController *controller.ProfileController
	var sbClient *pkgsb.Client
	var transactionController *controller.TransactionController
	var realtimeController *controller.RealtimeController
	var wsGateway *wshandler.Handler
	var queueClient *queue.Client
	var fanoutSub *queue.FanoutSubscriber
	if cfg.Supabase.Enabled() {
		var err error
		sbClient, err = pkgsb.New(cfg.Supabase)
		if err != nil {
			return fmt.Errorf("初始化 Supabase 失败: %w", err)
		}
		supabaseAuthUC = usecase.NewSupabaseAuthUsecase(sbClient)
		phoneOTPUC = usecase.NewPhoneOTPUsecase(sbClient, cfg.Auth, cfg.Server.Mode)
		profileRepo := sbrepo.NewProfileRepository(sbClient)
		profileUC := usecase.NewProfileUsecase(profileRepo)
		profileController = controller.NewProfileController(profileUC)
		transactionRepo := sbrepo.NewTransactionRepository(sbClient)
		transactionUC := usecase.NewTransactionUsecase(transactionRepo)
		transactionController = controller.NewTransactionController(transactionUC)
		log.Info("Supabase 已启用（认证 + profile + transactions）", zap.String("url", cfg.Supabase.URL))

		hub := wshandler.NewHub()
		ticketRepo := redisrepo.NewWSTicketRepository(redisClient)
		eventRepo := redisrepo.NewRealtimeEventRepository(redisClient)
		presenceRepo := redisrepo.NewPresenceRepository(redisClient)
		ticketUC := usecase.NewRealtimeTicketUsecase(ticketRepo, *cfg)
		syncUC := usecase.NewRealtimeSyncUsecase(eventRepo, *cfg)
		presenceUC := usecase.NewRealtimePresenceUsecase(presenceRepo, hub)
		wsGateway = wshandler.NewHandler(hub, ticketUC, presenceUC, log)

		var pushEnqueuer usecase.PushEnqueuer
		if cfg.Queue.Enabled {
			queueClient = queue.NewAsynqClient(*cfg)
			pushEnqueuer = queueClient
			fanoutSub = queue.NewFanoutSubscriber(
				redisClient,
				cfg.Queue.PubSubChannel(),
				hub.BroadcastToUser,
			)
			go func() {
				log.Info("Redis Pub/Sub 订阅已启动", zap.String("channel", cfg.Queue.PubSubChannel()))
				if err := fanoutSub.Run(context.Background()); err != nil && err != context.Canceled {
					log.Error("Pub/Sub 订阅异常退出", zap.Error(err))
				}
			}()
			log.Info("异步队列已启用", zap.Bool("queue_enabled", true))
		}

		pushUC := usecase.NewRealtimePushUsecase(eventRepo, *cfg, hub, pushEnqueuer)
		realtimeController = controller.NewRealtimeController(ticketUC, syncUC, pushUC)
		log.Info("Realtime WebSocket 已启用",
			zap.String("ws_path", cfg.Realtime.WsPath),
			zap.String("ws_url", cfg.Realtime.WSURL(cfg.Server.Port)),
		)
	}

	userHandler := handler.NewUserHandler(userUC, supabaseAuthUC, deviceSessionUC, phoneOTPUC)

	engine := router.Setup(router.Options{
		Log:                   log,
		Mode:                  cfg.Server.Mode,
		JWTManager:            jwtMgr,
		UserHandler:           userHandler,
		ProfileController:     profileController,
		TransactionController: transactionController,
		RealtimeController:    realtimeController,
		WSHandler:             wsGateway,
		Config:                *cfg,
		Supabase:              cfg.Supabase,
		DeviceSessionUC:       deviceSessionUC,
	})
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("HTTP 服务启动", zap.Int("port", cfg.Server.Port), zap.String("env", env))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("HTTP 服务异常退出", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("收到关机信号，开始优雅关闭...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if queueClient != nil {
		if err := queueClient.Close(); err != nil {
			log.Warn("关闭 Asynq 客户端失败", zap.Error(err))
		}
	}

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("优雅关机失败: %w", err)
	}
	log.Info("服务已关闭")
	return nil
}

// autoMigrate 自动迁移数据库表结构。
func autoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&entity.User{}, &entity.TransactionRecord{}); err != nil {
		return fmt.Errorf("自动迁移失败: %w", err)
	}
	return nil
}
