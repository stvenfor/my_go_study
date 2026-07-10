// user_routes.go 自建用户体系路由。
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/handler"
	"github.com/stvenfor/my_go_study/internal/delivery/http/middleware"
	jwtmanager "github.com/stvenfor/my_go_study/pkg/jwt"
)

// registerUserRoutes 注册 /api/v1/user 路由（自建 JWT 用户体系 + Supabase 认证）。
func registerUserRoutes(v1 *gin.RouterGroup, jwtManager *jwtmanager.Manager, userHandler *handler.UserHandler, sbAuth gin.HandlerFunc) {
	userGroup := v1.Group("/user")
	{
		userGroup.POST("/register", userHandler.Register)
		userGroup.POST("/login", userHandler.Login)
		userGroup.POST("/refresh", userHandler.Refresh)
		userGroup.POST("/phone/otp/send", userHandler.SendPhoneOTP)
		userGroup.POST("/phone/otp/verify", userHandler.VerifyPhoneOTP)
		if sbAuth != nil {
			userGroup.POST("/logout", sbAuth, userHandler.Logout)
		}
		userGroup.GET("/list", middleware.Auth(jwtManager), userHandler.List)
		userGroup.GET("/profile", middleware.Auth(jwtManager), userHandler.Profile)
	}
}
