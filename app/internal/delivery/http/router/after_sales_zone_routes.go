package router

import (
	"github.com/gin-gonic/gin"
	"app/internal/delivery/http/controller"
)

func registerAfterSalesZoneRoutes(v1 *gin.RouterGroup, sbAuth gin.HandlerFunc, ctrl *controller.AfterSalesZoneController, accountGate gin.HandlerFunc) {
	if ctrl == nil {
		return
	}
	chain := sessionAuthChain(sbAuth, accountGate)
	g := v1.Group("/after-sales")
	g.Use(chain...)
	{
		g.GET("/pending-appointments", ctrl.ListPendingAppointments)
		g.GET("/records", ctrl.ListRecords)
		g.POST("/records", ctrl.CreateRecord)
		g.GET("/records/:record_id", ctrl.GetRecord)
	}
}
