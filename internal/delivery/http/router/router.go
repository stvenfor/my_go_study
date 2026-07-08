// router.go 注册 HTTP 路由与中间件。
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/handler"
	"github.com/stvenfor/my_go_study/internal/delivery/http/middleware"
	jwtmanager "github.com/stvenfor/my_go_study/pkg/jwt"
	"go.uber.org/zap"
)

// Setup 构建 Gin 路由引擎。
func Setup(log *zap.Logger, jwtManager *jwtmanager.Manager, userHandler *handler.UserHandler, mode string) *gin.Engine {
	gin.SetMode(mode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.RequestLogger(log))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")
	{
		userGroup := v1.Group("/user")
		{
			userGroup.POST("/register", userHandler.Register)
			userGroup.POST("/login", userHandler.Login)
			userGroup.GET("/profile", middleware.Auth(jwtManager), userHandler.Profile)
		}
	}

	return r
}
