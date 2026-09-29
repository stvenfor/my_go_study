package router

import (
	"app/internal/delivery/http/controller"
	"app/internal/delivery/http/handler"
	"app/internal/delivery/http/middleware"
	wshandler "app/internal/delivery/ws"
	"app/internal/shared"
	"app/internal/usecase"
	"common/config"
	jwtmanager "common/jwt"
	"common/metrics"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Options 路由依赖。
//
// 迁移期同时存在两套装配方式：
//   - 上面的 *Controller 字段：尚未切片的域，仍由 registerXxxRoutes 集中挂载；
//   - DomainRouters：已切成垂直切片的域，各自实现 Register(engine)。
//
// 全部域切完后，*Controller 字段与 registerXxxRoutes 一起删除，Setup 只剩
// 「公共中间件 + 遍历 DomainRouters」。
type Options struct {
	Log                          *zap.Logger
	Mode                         string
	JWTManager                   *jwtmanager.Manager
	UserHandler                  *handler.UserHandler
	ProfileController            *controller.ProfileController
	MallController               *controller.MallController
	MembershipController         *controller.MembershipController
	CommunityController          *controller.CommunityController
	ShortVideoController         *controller.ShortVideoController
	AddressController            *controller.AddressController
	DealInvoiceController        *controller.DealInvoiceController
	UsedCarOrderController       *controller.UsedCarOrderController
	NewCarFollowController       *controller.NewCarFollowController
	AfterSalesZoneController     *controller.AfterSalesZoneController
	PurchaseCalculatorController *controller.PurchaseCalculatorController
	PaymentController            *controller.PaymentController
	TransactionController        *controller.TransactionController
	RealtimeController           *controller.RealtimeController
	JPushController              *controller.JPushController
	ImController                 *controller.ImController
	SseController                *controller.SseController
	CsAgentController            *controller.CsAgentController
	WSHandler                    *wshandler.Handler
	Config                       config.Config
	Supabase                     config.SupabaseConfig
	DeviceSessionUC              *usecase.DeviceSessionUsecase
	AccountGate                  gin.HandlerFunc

	// SessionAuth 业务会话中间件（LocalSessionAuth / SupabaseSessionAuth）。
	// 为 nil 时 Setup 按 Config 自行构造，兼容旧调用方。
	SessionAuth gin.HandlerFunc
	// DomainRouters 已切片域的路由模块（access / home / points / wallet …）。
	DomainRouters []shared.Registrable
}

// Setup 构建 Gin 路由引擎。
func Setup(opts Options) *gin.Engine {
	gin.SetMode(opts.Mode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.PrometheusHTTP())
	r.Use(middleware.RequestLogger(opts.Log))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	// /metrics 不经 SessionAuth；仅本机 / Docker 内网可达即可。
	r.GET("/metrics", gin.WrapH(metrics.Handler()))

	v1 := r.Group("/api/v1")

	businessAuth := opts.Config.Auth.IsLocalProvider() || opts.Supabase.Enabled()
	sessionAuth := opts.SessionAuth
	if sessionAuth == nil && businessAuth && opts.DeviceSessionUC != nil {
		if opts.Config.Auth.IsLocalProvider() {
			sessionAuth = middleware.LocalSessionAuth(opts.JWTManager, opts.DeviceSessionUC)
		} else {
			sessionAuth = middleware.SupabaseSessionAuth(opts.Supabase, opts.DeviceSessionUC)
		}
	}

	registerUserRoutes(v1, opts.JWTManager, opts.UserHandler, sessionAuth, opts.AccountGate, opts.AddressController)

	if businessAuth && opts.TransactionController != nil && sessionAuth != nil {
		registerTransactionRoutes(v1, sessionAuth, opts.TransactionController, opts.AccountGate)
	}

	if businessAuth && opts.ProfileController != nil && sessionAuth != nil {
		registerProfileRoutes(v1, sessionAuth, opts.ProfileController, opts.AccountGate)
	}

	if opts.Config.Auth.IsLocalProvider() && opts.MallController != nil && sessionAuth != nil {
		registerMallRoutes(v1, sessionAuth, opts.MallController, opts.AccountGate)
	}

	if opts.Config.Auth.IsLocalProvider() && opts.MembershipController != nil && sessionAuth != nil {
		registerMembershipRoutes(v1, sessionAuth, opts.MembershipController, opts.AccountGate)
	}

	if opts.Config.Auth.IsLocalProvider() && opts.CommunityController != nil && sessionAuth != nil {
		registerCommunityRoutes(v1, sessionAuth, opts.CommunityController, opts.AccountGate)
	}

	if opts.Config.Auth.IsLocalProvider() && opts.ShortVideoController != nil && sessionAuth != nil {
		registerShortVideoRoutes(v1, sessionAuth, opts.ShortVideoController, opts.AccountGate)
	}

	if opts.Config.Auth.IsLocalProvider() && opts.DealInvoiceController != nil && sessionAuth != nil {
		registerDealInvoiceRoutes(v1, sessionAuth, opts.DealInvoiceController, opts.AccountGate)
	}

	if opts.Config.Auth.IsLocalProvider() && opts.UsedCarOrderController != nil && sessionAuth != nil {
		registerUsedCarOrderRoutes(v1, sessionAuth, opts.UsedCarOrderController, opts.AccountGate)
	}

	if opts.Config.Auth.IsLocalProvider() && opts.NewCarFollowController != nil && sessionAuth != nil {
		registerNewCarFollowRoutes(v1, sessionAuth, opts.NewCarFollowController, opts.AccountGate)
	}

	if opts.Config.Auth.IsLocalProvider() && opts.AfterSalesZoneController != nil && sessionAuth != nil {
		registerAfterSalesZoneRoutes(v1, sessionAuth, opts.AfterSalesZoneController, opts.AccountGate)
	}

	if opts.Config.Auth.IsLocalProvider() {
		registerPurchaseCalculatorRoutes(v1, opts.PurchaseCalculatorController)
	}

	if businessAuth && opts.PaymentController != nil && sessionAuth != nil {
		registerPaymentRoutes(v1, sessionAuth, opts.PaymentController, opts.AccountGate)
	}

	if businessAuth && opts.RealtimeController != nil && sessionAuth != nil {
		registerRealtimeRoutes(v1, sessionAuth, opts.RealtimeController, opts.AccountGate)
	}

	if businessAuth && opts.JPushController != nil && sessionAuth != nil {
		registerJPushRoutes(v1, sessionAuth, opts.JPushController, opts.AccountGate)
	}

	if businessAuth && opts.ImController != nil && sessionAuth != nil {
		registerImRoutes(v1, sessionAuth, opts.ImController, opts.AccountGate)
	}

	if businessAuth && opts.SseController != nil && sessionAuth != nil && opts.Config.SSE.Enabled {
		registerSseRoutes(v1, sessionAuth, opts.SseController, opts.AccountGate)
	}

	if businessAuth && opts.CsAgentController != nil && sessionAuth != nil && opts.Config.CsAgent.Enabled {
		registerCsAgentRoutes(v1, sessionAuth, opts.CsAgentController, opts.AccountGate)
	}

	if opts.WSHandler != nil {
		wsPath := opts.Config.Realtime.WsPath
		if wsPath == "" {
			wsPath = "/realtime/v1/connect"
		}
		r.GET(wsPath, opts.WSHandler.ServeWS)
	}

	// 已切成垂直切片的域自行挂载路由（每域一个 Router + Register）。
	// 各域在构造时就带好了自己的 sessionAuth / accountGate，sessionAuth 为 nil
	// 时域内会跳过注册。
	for _, domain := range opts.DomainRouters {
		if domain != nil {
			domain.Register(r)
		}
	}

	return r
}
