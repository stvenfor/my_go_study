package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/controller"
)

func registerUsedCarOrderRoutes(v1 *gin.RouterGroup, sbAuth gin.HandlerFunc, ctrl *controller.UsedCarOrderController, accountGate gin.HandlerFunc) {
	if ctrl == nil {
		return
	}
	chain := sessionAuthChain(sbAuth, accountGate)
	g := v1.Group("/used-car-orders")
	g.Use(chain...)
	{
		g.GET("/summary", ctrl.Summary)
		g.GET("/customers", ctrl.ListCustomers)
		g.GET("", ctrl.List)
		g.POST("", ctrl.Create)
		g.GET("/:order_id", ctrl.Get)
	}
}
