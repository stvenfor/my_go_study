// profile_routes.go Supabase Profile 用户资料管理路由。
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/controller"
	"github.com/stvenfor/my_go_study/internal/delivery/http/middleware"
)

// registerProfileRoutes 注册 Profile 相关路由。
// 已登录请求必须带与会话一致的 user_id。本地模式再拒绝注销或停用账号。
func registerProfileRoutes(v1 *gin.RouterGroup, sbAuth gin.HandlerFunc, profileCtrl *controller.ProfileController, accountGate gin.HandlerFunc) {
	chain := []gin.HandlerFunc{sbAuth, middleware.RequireUserID()}
	if accountGate != nil {
		chain = append(chain, accountGate)
	}
	profilesGroup := v1.Group("/profiles")
	profilesGroup.Use(chain...)
	{
		profilesGroup.GET("/me", profileCtrl.GetMe)
		profilesGroup.PATCH("/me", profileCtrl.UpdateMe)
		profilesGroup.POST("/me/store", profileCtrl.SwitchStore)
	}

	meGroup := v1.Group("/me")
	meGroup.Use(chain...)
	{
		meGroup.GET("/profile", profileCtrl.GetMeLegacy)
		meGroup.PATCH("/profile", profileCtrl.UpdateMeLegacy)
	}
}
