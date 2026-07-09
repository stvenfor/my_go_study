// router.go 注册 HTTP 路由与中间件。
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/controller"
	"github.com/stvenfor/my_go_study/internal/delivery/http/handler"
	"github.com/stvenfor/my_go_study/internal/delivery/http/middleware"
	"github.com/stvenfor/my_go_study/pkg/config"
	jwtmanager "github.com/stvenfor/my_go_study/pkg/jwt"
	"go.uber.org/zap"
)

// Options 路由依赖。
type Options struct {
	Log                  *zap.Logger
	Mode                 string
	JWTManager           *jwtmanager.Manager
	UserHandler          *handler.UserHandler
	ProfileController    *controller.ProfileController
	TransactionController *controller.TransactionController
	Supabase             config.SupabaseConfig
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
	registerUserRoutes(v1, opts.JWTManager, opts.UserHandler)

	if opts.Supabase.Enabled() && opts.TransactionController != nil {
		sbAuth := middleware.SupabaseAuth(opts.Supabase)
		registerTransactionRoutes(v1, sbAuth, opts.TransactionController)
	}

	if opts.Supabase.Enabled() && opts.ProfileController != nil {
		sbAuth := middleware.SupabaseAuth(opts.Supabase)
		registerProfileRoutes(v1, sbAuth, opts.ProfileController)
	}

	return r
}
