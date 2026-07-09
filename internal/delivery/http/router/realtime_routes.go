// realtime_routes.go Realtime HTTP 路由。
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/controller"
)

func registerRealtimeRoutes(v1 *gin.RouterGroup, sbAuth gin.HandlerFunc, ctrl *controller.RealtimeController) {
	group := v1.Group("/realtime")
	group.Use(sbAuth)
	{
		group.POST("/ws-ticket", ctrl.WSTicket)
		group.POST("/sync", ctrl.Sync)
		group.POST("/push", ctrl.Push)
	}
}
