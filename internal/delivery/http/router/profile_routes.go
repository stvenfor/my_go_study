// profile_routes.go Supabase Profile 用户资料管理路由。
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/controller"
)

// registerProfileRoutes 注册 Profile 相关路由。
func registerProfileRoutes(v1 *gin.RouterGroup, sbAuth gin.HandlerFunc, profileCtrl *controller.ProfileController) {
	// 统一管理接口（统一响应格式 { code, message, data }）
	profilesGroup := v1.Group("/profiles")
	profilesGroup.Use(sbAuth)
	{
		profilesGroup.GET("/me", profileCtrl.GetMe)
		profilesGroup.PATCH("/me", profileCtrl.UpdateMe)
	}

	// Flutter BackendApiClient 兼容（snake_case 直出 JSON）
	meGroup := v1.Group("/me")
	meGroup.Use(sbAuth)
	{
		meGroup.GET("/profile", profileCtrl.GetMeLegacy)
		meGroup.PATCH("/profile", profileCtrl.UpdateMeLegacy)
	}
}
