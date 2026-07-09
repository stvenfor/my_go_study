// user_routes.go 自建用户体系路由。
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/handler"
	"github.com/stvenfor/my_go_study/internal/delivery/http/middleware"
	jwtmanager "github.com/stvenfor/my_go_study/pkg/jwt"
)

// registerUserRoutes 注册 /api/v1/user 路由（自建 JWT 用户体系）。
func registerUserRoutes(v1 *gin.RouterGroup, jwtManager *jwtmanager.Manager, userHandler *handler.UserHandler) {
	userGroup := v1.Group("/user")
	{
		userGroup.POST("/register", userHandler.Register)
		userGroup.POST("/login", userHandler.Login)
		userGroup.GET("/list", middleware.Auth(jwtManager), userHandler.List)
		userGroup.GET("/profile", middleware.Auth(jwtManager), userHandler.Profile)
	}
}
