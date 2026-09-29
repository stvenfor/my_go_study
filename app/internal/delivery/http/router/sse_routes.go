// sse_routes.go SSE HTTP 路由。
package router

import (
	"github.com/gin-gonic/gin"
	"app/internal/delivery/http/controller"
)

func registerSseRoutes(v1 *gin.RouterGroup, sessionAuth gin.HandlerFunc, ctrl *controller.SseController, accountGate gin.HandlerFunc) {
	chain := sessionAuthChain(sessionAuth, accountGate)
	group := v1.Group("/sse")
	group.Use(chain...)
	{
		group.POST("/completions", ctrl.Completions)
	}
}
