// user_routes.go 自建用户体系路由。
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/controller"
	"github.com/stvenfor/my_go_study/internal/delivery/http/handler"
	jwtmanager "github.com/stvenfor/my_go_study/pkg/jwt"
)

// registerUserRoutes 注册 /api/v1/user 路由（自建 JWT 用户体系 + Supabase 认证）。
func registerUserRoutes(v1 *gin.RouterGroup, jwtManager *jwtmanager.Manager, userHandler *handler.UserHandler, sbAuth gin.HandlerFunc, accountGate gin.HandlerFunc, addressCtrl *controller.AddressController) {
	userGroup := v1.Group("/user")
	{
		userGroup.POST("/register", userHandler.Register)
		userGroup.POST("/login", userHandler.Login)
		userGroup.POST("/refresh", userHandler.Refresh)
		userGroup.POST("/phone/otp/send", userHandler.SendPhoneOTP)
		userGroup.POST("/phone/otp/verify", userHandler.VerifyPhoneOTP)
		if sbAuth != nil {
			// 身份只认 Session Auth 写入的 user_id，不要求客户端再传。
			chain := sessionAuthChain(sbAuth, accountGate)
			userGroup.POST("/logout", append(chain, userHandler.Logout)...)
			userGroup.POST("/deactivate", append(chain, userHandler.Deactivate)...)
			if addressCtrl != nil {
				addrs := userGroup.Group("/addresses", chain...)
				addrs.GET("", addressCtrl.List)
				addrs.POST("", addressCtrl.Create)
				addrs.GET("/:address_id", addressCtrl.Get)
				addrs.PATCH("/:address_id", addressCtrl.Update)
				addrs.POST("/:address_id/default", addressCtrl.SetDefault)
				addrs.DELETE("/:address_id", addressCtrl.Delete)
			}
		}
	}
}
