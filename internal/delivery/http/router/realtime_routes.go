// realtime_routes.go Realtime HTTP 路由。
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/controller"
	"github.com/stvenfor/my_go_study/internal/delivery/http/middleware"
)

func registerRealtimeRoutes(v1 *gin.RouterGroup, sbAuth gin.HandlerFunc, ctrl *controller.RealtimeController, accountGate gin.HandlerFunc) {
	chain := []gin.HandlerFunc{sbAuth, middleware.RequireUserID()}
	if accountGate != nil {
		chain = append(chain, accountGate)
	}
	group := v1.Group("/realtime")
	group.Use(chain...)
	{
		group.POST("/ws-ticket", ctrl.WSTicket)
		group.POST("/sync", ctrl.Sync)
		group.POST("/push", ctrl.Push)
	}
}
