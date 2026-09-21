// sse_routes.go SSE HTTP 路由。
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/controller"
	"github.com/stvenfor/my_go_study/internal/delivery/http/middleware"
)

func registerSseRoutes(v1 *gin.RouterGroup, sessionAuth gin.HandlerFunc, ctrl *controller.SseController, accountGate gin.HandlerFunc) {
	chain := []gin.HandlerFunc{sessionAuth, middleware.RequireUserID()}
	if accountGate != nil {
		chain = append(chain, accountGate)
	}
	group := v1.Group("/sse")
	group.Use(chain...)
	{
		group.POST("/completions", ctrl.Completions)
	}
}
