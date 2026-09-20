// sse_routes.go SSE HTTP 路由。
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/controller"
)

func registerSseRoutes(v1 *gin.RouterGroup, sessionAuth gin.HandlerFunc, ctrl *controller.SseController) {
	group := v1.Group("/sse")
	group.Use(sessionAuth)
	{
		group.POST("/completions", ctrl.Completions)
	}
}
