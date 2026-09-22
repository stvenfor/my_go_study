package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/controller"
	"github.com/stvenfor/my_go_study/internal/delivery/http/handler"
	"github.com/stvenfor/my_go_study/internal/delivery/http/middleware"
	wshandler "github.com/stvenfor/my_go_study/internal/delivery/ws"
	"github.com/stvenfor/my_go_study/internal/usecase"
	"github.com/stvenfor/my_go_study/pkg/config"
	jwtmanager "github.com/stvenfor/my_go_study/pkg/jwt"
	"go.uber.org/zap"
)

// Options 路由依赖。
type Options struct {
	Log                   *zap.Logger
	Mode                  string
	JWTManager            *jwtmanager.Manager
	UserHandler           *handler.UserHandler
	ProfileController     *controller.ProfileController
	AccessController      *controller.AccessController
	MallController        *controller.MallController
	PointsController      *controller.PointsController
	CommunityController   *controller.CommunityController
	ShortVideoController  *controller.ShortVideoController
	AddressController     *controller.AddressController
	HomeTodoController     *controller.HomeTodoController
	DealInvoiceController  *controller.DealInvoiceController
	TransactionController  *controller.TransactionController
	RealtimeController    *controller.RealtimeController
	SseController         *controller.SseController
	WSHandler             *wshandler.Handler
	Config                config.Config
	Supabase              config.SupabaseConfig
	DeviceSessionUC       *usecase.DeviceSessionUsecase
	AccountGate           gin.HandlerFunc
}

// Setup 构建 Gin 路由引擎。
func Setup(opts Options) *gin.Engine {
	gin.SetMode(opts.Mode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.RequestLogger(opts.Log))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")

	businessAuth := opts.Config.Auth.IsLocalProvider() || opts.Supabase.Enabled()
	var sessionAuth gin.HandlerFunc
	if businessAuth && opts.DeviceSessionUC != nil {
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

	if businessAuth && opts.AccessController != nil && sessionAuth != nil {
		registerAccessRoutes(v1, sessionAuth, opts.AccessController, opts.AccountGate)
	}

	if opts.Config.Auth.IsLocalProvider() && opts.MallController != nil && sessionAuth != nil {
		registerMallRoutes(v1, sessionAuth, opts.MallController, opts.AccountGate)
	}

	if opts.Config.Auth.IsLocalProvider() && opts.PointsController != nil && sessionAuth != nil {
		registerPointsRoutes(v1, sessionAuth, opts.PointsController, opts.AccountGate)
	}

	if opts.Config.Auth.IsLocalProvider() && opts.CommunityController != nil && sessionAuth != nil {
		registerCommunityRoutes(v1, sessionAuth, opts.CommunityController, opts.AccountGate)
	}

	if opts.Config.Auth.IsLocalProvider() && opts.ShortVideoController != nil && sessionAuth != nil {
		registerShortVideoRoutes(v1, sessionAuth, opts.ShortVideoController, opts.AccountGate)
	}

	if opts.Config.Auth.IsLocalProvider() && opts.HomeTodoController != nil && sessionAuth != nil {
		registerHomeTodoRoutes(v1, sessionAuth, opts.HomeTodoController, opts.AccountGate)
	}

	if opts.Config.Auth.IsLocalProvider() && opts.DealInvoiceController != nil && sessionAuth != nil {
		registerDealInvoiceRoutes(v1, sessionAuth, opts.DealInvoiceController, opts.AccountGate)
	}

	if businessAuth && opts.RealtimeController != nil && sessionAuth != nil {
		registerRealtimeRoutes(v1, sessionAuth, opts.RealtimeController, opts.AccountGate)
	}

	if businessAuth && opts.SseController != nil && sessionAuth != nil && opts.Config.SSE.Enabled {
		registerSseRoutes(v1, sessionAuth, opts.SseController, opts.AccountGate)
	}

	if opts.WSHandler != nil {
		wsPath := opts.Config.Realtime.WsPath
		if wsPath == "" {
			wsPath = "/realtime/v1/connect"
		}
		r.GET(wsPath, opts.WSHandler.ServeWS)
	}

	return r
}
