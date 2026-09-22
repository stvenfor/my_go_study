// profile_routes.go Supabase Profile 用户资料管理路由。
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/controller"
)

// registerProfileRoutes 注册 Profile 相关路由。
// 身份只认 Session Auth；本地模式再拒绝注销或停用账号。
func registerProfileRoutes(v1 *gin.RouterGroup, sbAuth gin.HandlerFunc, profileCtrl *controller.ProfileController, accountGate gin.HandlerFunc) {
	chain := sessionAuthChain(sbAuth, accountGate)
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
		meGroup.GET("/stores", profileCtrl.ListMyStores)
	}
}
