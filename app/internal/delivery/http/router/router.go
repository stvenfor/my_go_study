package router

import (
	"net"
	"net/http"
	"os"
	"strings"

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
	Log                *zap.Logger
	Mode               string
	JWTManager         *jwtmanager.Manager
	UserHandler        *handler.UserHandler
	ProfileController  *controller.ProfileController
	AddressController  *controller.AddressController
	RealtimeController *controller.RealtimeController
	JPushController    *controller.JPushController
	ImController       *controller.ImController
	SseController      *controller.SseController
	CsAgentController  *controller.CsAgentController
	WSHandler          *wshandler.Handler
	Config             config.Config
	Supabase           config.SupabaseConfig
	DeviceSessionUC    *usecase.DeviceSessionUsecase
	AccountGate        gin.HandlerFunc

	// SessionAuth 业务会话中间件（LocalSessionAuth / SupabaseSessionAuth）。
	// 为 nil 时 Setup 按 Config 自行构造，兼容旧调用方。
	SessionAuth gin.HandlerFunc
	// DomainRouters 已切片域的路由模块
	// （access / home / points / wallet / used_car / new_car_follow / after_sales /
	// deal_invoice / purchase_calculator / mall / membership / payment /
	// community / short_video / transaction …）。
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
	// /metrics 不经 SessionAuth；h7 收口为「默认只本机可达」，局域网 IP 取不到指标。
	// Docker 内网 Prometheus 抓取（prometheus.yml 的 app:8080/metrics）由 compose 注入的
	// METRICS_ALLOW_CIDRS 显式放行；loopback 始终放行，保住 Prometheus 抓本机口。
	r.GET("/metrics", metricsAccessGuard(opts.Log), gin.WrapH(metrics.Handler()))

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

	if businessAuth && opts.ProfileController != nil && sessionAuth != nil {
		registerProfileRoutes(v1, sessionAuth, opts.ProfileController, opts.AccountGate)
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

// metricsAccessGuard 把 /metrics 限定为本机可达（h7）：
//   - 只认 TCP 对端 RemoteAddr，不信 X-Forwarded-For / c.ClientIP()（可伪造）；
//   - loopback（127.0.0.0/8、::1）放行，保证 curl 127.0.0.1 与 Prometheus 抓本机口可用；
//   - Docker 内网抓取（prometheus.yml 的 app:8080/metrics）由 METRICS_ALLOW_CIDRS
//     显式放行（compose 注入 Docker 网段）；默认空 => 只本机口；
//   - 其余一律 403，局域网 IP / 经 docker-proxy 转发的外部访问取不到指标。
func metricsAccessGuard(log *zap.Logger) gin.HandlerFunc {
	allow := parseMetricsAllowCIDRs(os.Getenv("METRICS_ALLOW_CIDRS"), log)
	return func(c *gin.Context) {
		if metricsPeerAllowed(c.Request.RemoteAddr, allow) {
			c.Next()
			return
		}
		c.AbortWithStatus(http.StatusForbidden)
	}
}

func parseMetricsAllowCIDRs(raw string, log *zap.Logger) []*net.IPNet {
	var out []*net.IPNet
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		_, cidr, err := net.ParseCIDR(s)
		if err != nil {
			// 非法项忽略：配置写错不应把 /metrics 全开，但必须留痕便于排查抓取失败。
			if log != nil {
				log.Warn("METRICS_ALLOW_CIDRS 含非法 CIDR，已忽略", zap.String("entry", s))
			}
			continue
		}
		out = append(out, cidr)
	}
	return out
}

func metricsPeerAllowed(remoteAddr string, allow []*net.IPNet) bool {
	host := remoteAddr
	if h, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = h
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	for _, cidr := range allow {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}
