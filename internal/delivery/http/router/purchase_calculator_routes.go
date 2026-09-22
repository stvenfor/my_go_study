package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/controller"
)

// registerPurchaseCalculatorRoutes 购车计算器（公开，无需 SessionAuth）。
func registerPurchaseCalculatorRoutes(v1 *gin.RouterGroup, ctrl *controller.PurchaseCalculatorController) {
	if ctrl == nil {
		return
	}
	g := v1.Group("/purchase-calculator")
	g.GET("/products", ctrl.ListProducts)
	g.POST("/quote", ctrl.Quote)
}
