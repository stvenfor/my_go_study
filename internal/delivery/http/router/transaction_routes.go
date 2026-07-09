// transaction_routes.go Transaction 交易管理路由（Supabase JWT 鉴权）。
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/controller"
)

// registerTransactionRoutes 注册 Transaction 相关路由。
func registerTransactionRoutes(v1 *gin.RouterGroup, sbAuth gin.HandlerFunc, txCtrl *controller.TransactionController) {
	// Flutter 兼容：limit/offset 列表 + snake_case CRUD
	flutterGroup := v1.Group("/transactions")
	flutterGroup.Use(sbAuth)
	{
		flutterGroup.GET("", txCtrl.List) // 无 page 参数时走 Flutter 格式
		flutterGroup.POST("", txCtrl.CreateLegacy)
		flutterGroup.GET("/:id", txCtrl.GetLegacy)
		flutterGroup.PUT("/:id", txCtrl.UpdateLegacy)
		flutterGroup.DELETE("/:id", txCtrl.DeleteLegacy)
	}

	// 统一管理接口（统一响应格式，分页用 page/size）
	manageGroup := v1.Group("/transactions/manage")
	manageGroup.Use(sbAuth)
	{
		manageGroup.GET("", txCtrl.ListPage)
		manageGroup.POST("", txCtrl.Create)
		manageGroup.GET("/:id", txCtrl.Get)
		manageGroup.PUT("/:id", txCtrl.Update)
		manageGroup.DELETE("/:id", txCtrl.Delete)
	}
}
