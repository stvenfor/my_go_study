// realtime_routes.go Realtime HTTP 路由。
package router

import (
	"app/internal/delivery/http/controller"
	"github.com/gin-gonic/gin"
)

func registerRealtimeRoutes(v1 *gin.RouterGroup, sbAuth gin.HandlerFunc, ctrl *controller.RealtimeController, accountGate gin.HandlerFunc) {
	chain := sessionAuthChain(sbAuth, accountGate)
	group := v1.Group("/realtime")
	group.Use(chain...)
	{
		group.POST("/ws-ticket", ctrl.WSTicket)
		group.POST("/sync", ctrl.Sync)
		group.POST("/push", ctrl.Push)
	}
}
