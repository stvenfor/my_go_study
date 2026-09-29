package router

import (
	"app/internal/delivery/http/controller"
	"github.com/gin-gonic/gin"
)

func registerDealInvoiceRoutes(v1 *gin.RouterGroup, sbAuth gin.HandlerFunc, ctrl *controller.DealInvoiceController, accountGate gin.HandlerFunc) {
	if ctrl == nil {
		return
	}
	chain := sessionAuthChain(sbAuth, accountGate)
	g := v1.Group("/deal-invoices")
	g.Use(chain...)
	{
		g.GET("/summary", ctrl.Summary)
		g.GET("/customers", ctrl.ListCustomers)
		g.GET("", ctrl.List)
		g.POST("", ctrl.Create)
		g.GET("/:invoice_id", ctrl.Get)
		g.PATCH("/:invoice_id/resubmit", ctrl.Resubmit)
	}
}
