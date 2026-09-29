// =============================================================================
// main.go — 应用唯一入口（make run → go -C app run ./cmd/api）
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
	"strings"
	"syscall"
	"time"

	grpcdelivery "app/internal/delivery/grpc"
	grpcauth "app/internal/delivery/grpc/interceptor"
	"app/internal/delivery/http/controller"
	"app/internal/delivery/http/handler"
	"app/internal/delivery/http/middleware"
	"app/internal/delivery/http/router"
	wshandler "app/internal/delivery/ws"
	"github.com/gin-gonic/gin"

	// 已切成垂直切片的域（阶段 3：批 1 通用域 + 批 2 汽车业务域 + 批 3 电商/社区域）。
	// 每域自带 Service + Router。
	"app/internal/access"
	"app/internal/after_sales"
	"app/internal/community"
	"app/internal/deal_invoice"
	"app/internal/home"
	"app/internal/mall"
	"app/internal/membership"
	"app/internal/new_car_follow"
	"app/internal/payment"
	"app/internal/points"
	"app/internal/purchase_calculator"
	"app/internal/shared"
	"app/internal/short_video"
	"app/internal/transaction"
	"app/internal/used_car"
	"app/internal/wallet"

	llmrepo "app/internal/repository/llm"
	"app/internal/repository/postgres"
	repoqueue "app/internal/repository/queue"
	redisrepo "app/internal/repository/redis"
	sbrepo "app/internal/repository/supabase"
	"app/internal/usecase"
	"common/config"
	jwtmanager "common/jwt"
	"common/logger"
	"core/database"
	"core/jpush"
	"core/queue"
	"core/rongcloud"
	pkgsb "core/supabase"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"model/entity"
	"model/provider"
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

	if !cfg.Supabase.Enabled() && !cfg.Auth.IsLocalProvider() {
		log.Warn("Supabase 未启用：请检查 configs/supabase.env，或设置 auth.provider=local")
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
	sessionRepo := redisrepo.NewSessionRepository(redisClient)
	deviceSessionUC := usecase.NewDeviceSessionUsecase(sessionRepo, cfg.Auth, cfg.AppEnv)
	var sessionAuthUC usecase.SessionAuth
	var phoneOTPUC usecase.PhoneOTPAuth
	var wechatLoginUC usecase.WeChatLoginAuth
	var huaweiLoginUC usecase.HuaweiLoginAuth
	var profileController *controller.ProfileController
	var profileUC *usecase.ProfileUsecase

	// 批 1 已切片的域 Service（原 usecase.AccessUsecase / PointsUsecase /
	// CashWalletUsecase / HomeTodoUsecase 一并下沉到各域）。
	var accessSvc *access.Service
	var homeSvc *home.Service
	var pointsSvc *points.Service
	var walletSvc *wallet.Service
	// 批 2 已切片的汽车业务 5 域 Service（原 usecase.UsedCarOrderUsecase /
	// NewCarFollowUsecase / AfterSalesZoneUsecase / DealInvoiceUsecase /
	// PurchaseQuoteUsecase 一并下沉到各域）。
	var usedCarSvc *used_car.Service
	var newCarFollowSvc *new_car_follow.Service
	var afterSalesSvc *after_sales.Service
	var dealInvoiceSvc *deal_invoice.Service
	var purchaseCalcSvc *purchase_calculator.Service
	// 批 3 已切片的电商 + 社区 6 域 Service（原 usecase.MallUsecase /
	// MembershipUsecase / PaymentUsecase / CommunityUsecase / ShortVideoUsecase /
	// TransactionUsecase 一并下沉到各域）。
	var mallSvc *mall.Service
	var membershipSvc *membership.Service
	var paymentSvc *payment.Service
	var communitySvc *community.Service
	var shortVideoSvc *short_video.Service
	var transactionSvc *transaction.Service
	var addressController *controller.AddressController
	var sbClient *pkgsb.Client
	var jpushController *controller.JPushController
	var imController *controller.ImController
	var realtimeController *controller.RealtimeController
	var pushUC *usecase.RealtimePushUsecase
	var sseController *controller.SseController
	var csAgentController *controller.CsAgentController
	var wsGateway *wshandler.Handler
	var queueClient *queue.Client
	var fanoutSub *queue.FanoutSubscriber
	var communityRepo *community.PostgresRepository

	var accountGate gin.HandlerFunc
	businessEnabled := cfg.Auth.IsLocalProvider() || cfg.Supabase.Enabled()
	if cfg.Auth.IsLocalProvider() {
		localAuth := usecase.NewLocalAuthUsecase(db, jwtMgr, cfg.Auth, cfg.AppEnv)
		sessionAuthUC = localAuth
		phoneOTPUC = localAuth
		wechatLoginUC = usecase.NewWeChatAuthUsecase(cfg.ThirdParty, localAuth)
		huaweiLoginUC = usecase.NewHuaweiAuthUsecase(cfg.ThirdParty, localAuth)
		accountGate = middleware.RequireActiveAccount(localAuth)
		profileRepo := postgres.NewProfileRepository(db)
		storeStatsRepo := postgres.NewStoreStatsRepository(db)
		profileUC = usecase.NewProfileUsecase(profileRepo, storeStatsRepo)
		profileController = controller.NewProfileController(profileUC)
		accessSvc = access.NewService(access.NewPostgresRepository(db))
		pointsSvc = points.NewService(points.NewPostgresRepository(db), points.NewPointsTaskProgress(db))
		walletSvc = wallet.NewService(wallet.NewPostgresRepository(db))
		mallSvc = mall.NewService(mall.NewPostgresRepository(db), accessSvc, pointsSvc, walletSvc)
		membershipRepo := membership.NewPostgresRepository(db)
		if err := membership.EnsureSchema(db); err != nil {
			log.Warn("会员订阅表准备失败", zap.Error(err))
		}
		// membership 依赖 payment（会员下单复用预支付通道），payment 先切；
		// 这里与下方 paymentSvc 各建一次实例（切片前同为两次 NewPaymentUsecase，
		// Service 只持有 cfg + http.Client，无状态，保持原结构不动）。
		paymentSvcForMembership := payment.NewService(cfg.ThirdParty)
		huaweiVerifier := membership.NewLiveHuaweiSubscriptionVerifier(cfg.ThirdParty.HuaweiIAP)
		appleVerifier := membership.NewLiveAppleReceiptVerifier(cfg.ThirdParty.AppleIAP)
		membershipSvc = membership.NewService(membershipRepo, walletSvc, paymentSvcForMembership, huaweiVerifier, appleVerifier, cfg.AppEnv)
		communityRepo = community.NewPostgresRepository(db)
		shortVideoSvc = short_video.NewService(short_video.NewPostgresRepository(db), cfg.ShortVideo.ReviewDelayOrDefault())
		if err := short_video.EnsureSeed(db); err != nil {
			log.Warn("小视频种子写入失败", zap.Error(err))
		} else {
			log.Info("小视频种子已就绪（13400000000 / 16 条）")
		}
		addressRepo := postgres.NewAddressRepository(db)
		addressUC := usecase.NewAddressUsecase(addressRepo)
		addressController = controller.NewAddressController(addressUC)
		homeSvc = home.NewService(home.NewPostgresRepository(db), accessSvc)
		if err := homeSvc.EnsureSeed(context.Background(), 1, ""); err != nil {
			log.Warn("首页待办种子写入失败", zap.Error(err))
		} else {
			log.Info("首页待办种子已就绪（门店1）")
		}
		if err := home.EnsurePackingDemo(db); err != nil {
			log.Warn("首页待办装箱演示种子失败", zap.Error(err))
		} else {
			log.Info("首页待办装箱演示已就绪（13400000000 / 1大3中4小）")
		}
		dealInvoiceSvc = deal_invoice.NewService(deal_invoice.NewPostgresRepository(db), accessSvc)
		if err := deal_invoice.EnsureSchema(db); err != nil {
			log.Warn("成交发票表准备失败", zap.Error(err))
		}
		if err := deal_invoice.EnsureSeed(db); err != nil {
			log.Warn("成交发票种子写入失败", zap.Error(err))
		} else {
			log.Info("成交发票种子已就绪（13400000000）")
		}
		usedCarSvc = used_car.NewService(used_car.NewPostgresRepository(db), accessSvc)
		if err := used_car.EnsureSchema(db); err != nil {
			log.Warn("二手车业务单表准备失败", zap.Error(err))
		}
		if err := used_car.EnsureSeed(db); err != nil {
			log.Warn("二手车业务单种子写入失败", zap.Error(err))
		} else {
			log.Info("二手车业务单种子已就绪（13400000000）")
		}
		newCarFollowSvc = new_car_follow.NewService(new_car_follow.NewPostgresRepository(db), accessSvc)
		if err := new_car_follow.EnsureSchema(db); err != nil {
			log.Warn("新车跟进档案表准备失败", zap.Error(err))
		}
		if err := new_car_follow.EnsureSeed(db); err != nil {
			log.Warn("新车跟进档案种子写入失败", zap.Error(err))
		} else {
			log.Info("新车跟进档案种子已就绪（13400000000）")
		}
		afterSalesSvc = after_sales.NewService(after_sales.NewPostgresRepository(db), accessSvc)
		if err := after_sales.EnsureSchema(db); err != nil {
			log.Warn("售后专区表准备失败", zap.Error(err))
		}
		purchaseCalcSvc = purchase_calculator.NewService(purchase_calculator.NewPostgresRepository(db))
		if err := purchase_calculator.EnsureSeed(db); err != nil {
			log.Warn("购车金融产品种子写入失败", zap.Error(err))
		} else {
			log.Info("购车金融产品种子已就绪")
		}
		if err := mall.EnsureSeed(db); err != nil {
			log.Warn("商城种子商品写入失败", zap.Error(err))
		} else {
			log.Info("商城种子商品已就绪（门店1 ≥35 条）")
		}
		transactionSvc = transaction.NewService(transaction.NewPostgresRepository(db))
		log.Info("Auth provider=local（本机 Postgres Auth + 业务表）")
		if phones := cfg.Auth.DevTestPhones(); len(phones) > 0 {
			log.Info("测试 OTP 账号已启用（多端 session 豁免）",
				zap.Strings("phones", phones),
				zap.Int("session_whitelist_emails", len(cfg.Auth.SessionWhitelistEmails)),
			)
		}
	} else if cfg.Supabase.Enabled() {
		var err error
		sbClient, err = pkgsb.New(cfg.Supabase)
		if err != nil {
			return fmt.Errorf("初始化 Supabase 失败: %w", err)
		}
		supabaseAuthUC := usecase.NewSupabaseAuthUsecase(sbClient)
		sessionAuthUC = supabaseAuthUC
		phoneOTPUC = usecase.NewPhoneOTPUsecase(sbClient, cfg.Auth, cfg.AppEnv)
		profileRepo := sbrepo.NewProfileRepository(sbClient)
		storeStatsRepo := postgres.NewStoreStatsRepository(db)
		profileUC = usecase.NewProfileUsecase(profileRepo, storeStatsRepo)
		profileController = controller.NewProfileController(profileUC)
		transactionSvc = transaction.NewService(transaction.NewSupabaseRepository(sbClient))
		log.Info("Auth provider=supabase（认证 + profile + transactions）", zap.String("url", cfg.Supabase.URL))
	} else {
		log.Warn("未启用业务认证：请设置 auth.provider=local，或配置 SUPABASE_URL / SUPABASE_ANON_KEY")
	}

	if transactionSvc != nil && cfg.Elasticsearch.Enabled() {
		txES := transaction.NewESStore(cfg.Elasticsearch.BaseURL())
		if err := txES.EnsureIndex(context.Background()); err != nil {
			log.Warn("Elasticsearch transactions 索引准备失败（写路径降级，搜索可能 503）", zap.Error(err))
		} else {
			log.Info("Elasticsearch transactions 索引已就绪", zap.String("url", cfg.Elasticsearch.BaseURL()))
		}
		transactionSvc.SetSearchStore(txES, log)
	}

	if businessEnabled {
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
			var qErr error
			queueClient, qErr = queue.NewClient(*cfg)
			if qErr != nil {
				return fmt.Errorf("连接 RabbitMQ 失败: %w", qErr)
			}
			pushEnqueuer = repoqueue.NewPushEnqueuer(queueClient)
			fanoutSub = queue.NewFanoutSubscriber(
				redisClient,
				cfg.Queue.PubSubChannel(),
				hub.BroadcastToUser,
				log,
			)
			go func() {
				log.Info("Redis Pub/Sub 订阅已启动", zap.String("channel", cfg.Queue.PubSubChannel()))
				if err := fanoutSub.Run(context.Background()); err != nil && err != context.Canceled {
					log.Error("Pub/Sub 订阅异常退出", zap.Error(err))
				}
			}()
			log.Info("异步队列已启用", zap.Bool("queue_enabled", true))
		}

		pushUC = usecase.NewRealtimePushUsecase(eventRepo, *cfg, hub, pushEnqueuer)
		realtimeController = controller.NewRealtimeController(ticketUC, syncUC, pushUC)
		if communityRepo != nil {
			// community 属批 3、realtime（pushUC）属批 4：本域只声明消费端最小接口
			// PushNotifier，装配处用适配器把 PushRequest 桥到 RealtimePushInput。
			// 批 4 切完 realtime 后删除适配器，直接传 realtime 域的公开 Service。
			communitySvc = community.NewService(communityRepo, communityPushAdapter{uc: pushUC}, cfg.Community.AskEveryoneInviteUserIDs)
			if err := community.EnsureSearchIndexes(db); err != nil {
				log.Warn("社区搜索索引准备失败（ILIKE 仍可用）", zap.Error(err))
			}
			if err := communitySvc.EnsureSeed(context.Background()); err != nil {
				log.Warn("社区种子数据写入失败", zap.Error(err))
			} else {
				log.Info("社区话题/动态种子已就绪")
			}
		}
		log.Info("Realtime WebSocket 已启用",
			zap.String("ws_path", cfg.Realtime.WsPath),
			zap.String("ws_url", cfg.Realtime.WSURL(cfg.Server.Port)),
		)

		if cfg.SSE.Enabled {
			streamProvider := buildSSEProvider(cfg.SSE, log)
			convRepo := redisrepo.NewConversationRepository(redisClient)
			limiter := usecase.NewMemoryRateLimiter(cfg.SSE.RateLimitPerUserPerMinute)
			completionUC := usecase.NewCompletionUsecase(streamProvider, convRepo, limiter, cfg.SSE)
			sseController = controller.NewSseController(completionUC, cfg.SSE)
			log.Info("SSE completions 已启用",
				zap.String("provider", cfg.SSE.ProviderName()),
				zap.Int("conversation_ttl_seconds", cfg.SSE.ConversationTTLSeconds),
				zap.Int("max_turns", cfg.SSE.MaxTurnsOrDefault()),
			)
		}

		if cfg.CsAgent.Enabled {
			csUC := usecase.NewCsAgentUsecase(transactionSvc, cfg.CsAgent)
			csAgentController = controller.NewCsAgentController(csUC, cfg.CsAgent)
			log.Info("客服 Agent 已启用",
				zap.String("provider", cfg.CsAgent.Provider),
				zap.String("model", cfg.CsAgent.ModelName()),
			)
		}
	}

	if businessEnabled {
		pushDeviceRepo := postgres.NewPushDeviceRepository(db)
		if err := postgres.EnsurePushDeviceSchema(db); err != nil {
			log.Warn("极光设备表初始化失败", zap.Error(err))
		} else {
			log.Info("极光设备表已就绪")
		}
		jpushClient := jpush.NewClient(jpush.Config{
			AppKey:         cfg.ThirdParty.JPushAppKey,
			MasterSecret:   cfg.ThirdParty.JPushMasterSecret,
			APNsProduction: cfg.ThirdParty.JPushAPNsProduction,
		})
		jpushUC := usecase.NewJPushUsecase(pushDeviceRepo, jpushClient)
		jpushController = controller.NewJPushController(jpushUC)
		if pushUC != nil {
			pushUC.SetOfflinePusher(jpushUC)
		}
		if cfg.ThirdParty.JPushConfigured() {
			log.Info("JPush 服务端已配置")
		} else {
			log.Info("JPush 服务端未配置（跳过真实下发；登记 API 仍可用）")
		}

		if err := postgres.EnsureImSchema(db); err != nil {
			log.Warn("融云 IM 表初始化失败", zap.Error(err))
		} else {
			log.Info("融云 IM 表已就绪")
		}
		rcClient := rongcloud.NewClient(rongcloud.Config{
			AppKey:     cfg.ThirdParty.RongCloudAppKey,
			AppSecret:  cfg.ThirdParty.RongCloudAppSecret,
			APIBaseURL: cfg.ThirdParty.RongCloudAPIBaseURL,
		})
		imUserLookup := postgres.NewImUserLookupRepository(db)
		imSessionUC := usecase.NewImSessionUsecase(rcClient, imUserLookup)
		imFriendUC := usecase.NewImFriendUsecase(postgres.NewImFriendRepository(db), imUserLookup)
		imGroupUC := usecase.NewImGroupUsecase(postgres.NewImGroupRepository(db), rcClient)
		imBackupUC := usecase.NewImBackupUsecase(postgres.NewImBackupRepository(db))
		imController = controller.NewImController(imSessionUC, imFriendUC, imGroupUC, imBackupUC)
		if profileUC != nil {
			profileUC.SetStoreGroupSync(imGroupUC)
		}
		if accessSvc != nil {
			accessSvc.SetStoreGroupSync(imGroupUC)
		}
		if cfg.ThirdParty.RongCloudConfigured() {
			log.Info("融云 IM 服务端已配置")
		} else {
			log.Info("融云 IM 服务端未配置（/im/session 将返回未配置）")
		}
	}

	userHandler := handler.NewUserHandler(sessionAuthUC, deviceSessionUC, phoneOTPUC, wechatLoginUC, huaweiLoginUC)

	if businessEnabled {
		paymentSvc = payment.NewService(cfg.ThirdParty)
	}

	var analyticsUC *usecase.AnalyticsUsecase
	analyticsRepo := postgres.NewAnalyticsRepository(db)
	analyticsUC = usecase.NewAnalyticsUsecase(analyticsRepo)
	if err := analyticsUC.EnsureSeedData(context.Background()); err != nil {
		log.Warn("analytics 种子数据写入失败", zap.Error(err))
	} else {
		log.Info("analytics 数据表已就绪（含种子数据）")
	}

	// 业务会话中间件在装配层构造一次，既给 router.Setup 里的遗留路由用，
	// 也给各域 Router 用（域内 sessionAuth 为 nil 时会跳过注册）。
	var sessionAuth gin.HandlerFunc
	if businessEnabled && deviceSessionUC != nil {
		if cfg.Auth.IsLocalProvider() {
			sessionAuth = middleware.LocalSessionAuth(jwtMgr, deviceSessionUC)
		} else {
			sessionAuth = middleware.SupabaseSessionAuth(cfg.Supabase, deviceSessionUC)
		}
	}

	// 批 1 已切片的域路由（各自 Register + 自带 auth 链）。
	// sessionAuth 为 nil 时域内跳过，无需在此判 businessAuth。
	var domainRouters []shared.Registrable
	if accessSvc != nil {
		domainRouters = append(domainRouters, access.NewRouter(accessSvc, sessionAuth, accountGate))
	}
	if homeSvc != nil {
		domainRouters = append(domainRouters, home.NewRouter(homeSvc, sessionAuth, accountGate))
	}
	if pointsSvc != nil {
		domainRouters = append(domainRouters, points.NewRouter(pointsSvc, sessionAuth, accountGate))
	}
	if walletSvc != nil {
		domainRouters = append(domainRouters, wallet.NewRouter(walletSvc, sessionAuth, accountGate))
	}

	// 批 2 已切片的汽车业务 5 域路由。
	// purchase_calculator 是公开域（无身份、无 SessionAuth），只在本地
	// provider 下装配 —— 与切片前 registerPurchaseCalculatorRoutes 的
	// `IsLocalProvider()` 门控一致（该 Service 只在本地分支构造）。
	if usedCarSvc != nil {
		domainRouters = append(domainRouters, used_car.NewRouter(usedCarSvc, sessionAuth, accountGate))
	}
	if newCarFollowSvc != nil {
		domainRouters = append(domainRouters, new_car_follow.NewRouter(newCarFollowSvc, sessionAuth, accountGate))
	}
	if afterSalesSvc != nil {
		domainRouters = append(domainRouters, after_sales.NewRouter(afterSalesSvc, sessionAuth, accountGate))
	}
	if dealInvoiceSvc != nil {
		domainRouters = append(domainRouters, deal_invoice.NewRouter(dealInvoiceSvc, sessionAuth, accountGate))
	}
	if purchaseCalcSvc != nil {
		domainRouters = append(domainRouters, purchase_calculator.NewRouter(purchaseCalcSvc))
	}

	// 批 3 已切片的电商 + 社区 6 域路由。
	// mall / membership / community / short_video 切片前由 router.Setup 里的
	// `opts.Config.Auth.IsLocalProvider()` 门控（仅本地 provider 注册），装配层
	// 保持同一门控：只在本地分支构造了 Service，故此处按 svc != nil 判断即可。
	// transaction / payment 为 businessAuth 门控，本地与 supabase 两种 provider
	// 都会构造 Service。
	if mallSvc != nil {
		domainRouters = append(domainRouters, mall.NewRouter(mallSvc, sessionAuth, accountGate))
	}
	if membershipSvc != nil {
		domainRouters = append(domainRouters, membership.NewRouter(membershipSvc, sessionAuth, accountGate))
	}
	if communitySvc != nil {
		domainRouters = append(domainRouters, community.NewRouter(communitySvc, sessionAuth, accountGate))
	}
	if shortVideoSvc != nil {
		domainRouters = append(domainRouters, short_video.NewRouter(shortVideoSvc, sessionAuth, accountGate))
	}
	if transactionSvc != nil {
		domainRouters = append(domainRouters, transaction.NewRouter(transactionSvc, sessionAuth, accountGate))
	}
	if paymentSvc != nil {
		domainRouters = append(domainRouters, payment.NewRouter(paymentSvc, sessionAuth, accountGate))
	}

	engine := router.Setup(router.Options{
		Log:                log,
		Mode:               cfg.Server.Mode,
		JWTManager:         jwtMgr,
		UserHandler:        userHandler,
		ProfileController:  profileController,
		AddressController:  addressController,
		RealtimeController: realtimeController,
		JPushController:    jpushController,
		ImController:       imController,
		SseController:      sseController,
		CsAgentController:  csAgentController,
		WSHandler:          wsGateway,
		Config:             *cfg,
		Supabase:           cfg.Supabase,
		DeviceSessionUC:    deviceSessionUC,
		AccountGate:        accountGate,
		SessionAuth:        sessionAuth,
		DomainRouters:      domainRouters,
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

	var grpcServer *grpcdelivery.Server
	if cfg.GRPC.Enabled && analyticsUC != nil && deviceSessionUC != nil && businessEnabled {
		var authenticator *grpcauth.Authenticator
		if cfg.Auth.IsLocalProvider() {
			authenticator = grpcauth.NewLocal(jwtMgr, deviceSessionUC)
		} else if cfg.Supabase.Enabled() {
			authenticator = grpcauth.NewSupabase(cfg.Supabase, deviceSessionUC)
		}
		if authenticator != nil {
			gs, err := grpcdelivery.NewServer(grpcdelivery.Options{
				Port:          cfg.GRPC.Port,
				Log:           log,
				Authenticator: authenticator,
				AnalyticsUC:   analyticsUC,
			})
			if err != nil {
				return fmt.Errorf("初始化 gRPC 失败: %w", err)
			}
			grpcServer = gs
			go func() {
				if err := grpcServer.Serve(); err != nil {
					log.Fatal("gRPC 服务异常退出", zap.Error(err))
				}
			}()
		} else {
			log.Warn("gRPC 已启用但缺少鉴权后端，跳过启动")
		}
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("收到关机信号，开始优雅关闭...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if grpcServer != nil {
		grpcServer.GracefulStop()
		log.Info("gRPC 已关闭")
	}

	if queueClient != nil {
		if err := queueClient.Close(); err != nil {
			log.Warn("关闭 RabbitMQ 客户端失败", zap.Error(err))
		}
	}

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("优雅关机失败: %w", err)
	}
	log.Info("服务已关闭")
	return nil
}

// buildSSEProvider 按配置装配 StreamProvider；openai_compatible 无 key 时回退 mock 并打日志。
func buildSSEProvider(sseCfg config.SSEConfig, log *zap.Logger) provider.StreamProvider {
	switch sseCfg.ProviderName() {
	case config.SSEProviderOpenAICompatible:
		p := llmrepo.NewOpenAICompatibleProvider(llmrepo.OpenAICompatibleConfig{
			BaseURL: sseCfg.OpenAI.BaseURL,
			APIKey:  sseCfg.OpenAI.APIKey,
			Model:   sseCfg.OpenAI.Model,
			Timeout: sseCfg.RequestTimeout(),
		})
		if strings.TrimSpace(sseCfg.OpenAI.APIKey) == "" {
			log.Warn("sse.provider=openai_compatible 但未配置 SSE_OPENAI_API_KEY，回退 mock")
			return llmrepo.NewMockStreamProvider()
		}
		log.Info("SSE Provider=openai_compatible", zap.String("model", sseCfg.OpenAI.Model))
		return p
	default:
		return llmrepo.NewMockStreamProvider()
	}
}

// autoMigrate 自动迁移数据库表结构。
func autoMigrate(db *gorm.DB) error {
	var dataType string
	_ = db.Raw(`
		SELECT data_type FROM information_schema.columns
		WHERE table_schema = CURRENT_SCHEMA()
		  AND table_name = 'transactions'
		  AND column_name = 'user_id'
	`).Scan(&dataType).Error
	if dataType == "bigint" || dataType == "integer" {
		if err := db.Exec(`ALTER TABLE transactions RENAME TO transactions_legacy_uint`).Error; err != nil {
			return fmt.Errorf("重命名旧 transactions 失败: %w", err)
		}
	}
	if err := postgres.ConsolidateLocalUsers(db); err != nil {
		return err
	}
	if err := db.AutoMigrate(
		&entity.User{},
		&entity.AuthRefreshToken{},
		&entity.Transaction{},
		&entity.TransactionRecord{},
		&entity.AnalyticsRecord{},
		&entity.WysUserStoreStats{},
		&entity.WysTopic{},
		&entity.WysPost{},
		&entity.WysPostLike{},
		&entity.WysPostComment{},
		&entity.WysUserFollow{},
		&entity.WysShortVideo{},
		&entity.WysShortVideoLike{},
	); err != nil {
		return fmt.Errorf("自动迁移失败: %w", err)
	}
	return nil
}

// communityPushAdapter 把 usecase.RealtimePushUsecase 适配成 community.PushNotifier。
//
// community 域属批 3、realtime 域属批 4。为免过早切片 realtime 引发大爆炸，
// community 侧只声明消费端所需的最小接口（PushRequest / PushNotifier），
// 由本适配器做逐字段拷贝。批 4 切完 realtime 后删除本类型，直接把 realtime
// 域的公开 Service 传进 community.NewService。
//
// 字段名刻意保持与 RealtimePushInput 一致（尤其 Name），避免桥接时错位。
type communityPushAdapter struct {
	uc *usecase.RealtimePushUsecase
}

// Notify 实现 community.PushNotifier。
func (a communityPushAdapter) Notify(ctx context.Context, req community.PushRequest) error {
	_, err := a.uc.PushToUser(ctx, usecase.RealtimePushInput{
		UserID: req.UserID,
		Topic:  req.Topic,
		Title:  req.Title,
		Body:   req.Body,
		Name:   req.Name,
		Extra:  req.Extra,
	})
	return err
}
