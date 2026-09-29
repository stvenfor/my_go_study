package router

import (
	"app/internal/delivery/http/controller"
	"github.com/gin-gonic/gin"
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
