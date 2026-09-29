package router

import (
	"github.com/gin-gonic/gin"
	"app/internal/delivery/http/controller"
)

func registerCsAgentRoutes(v1 *gin.RouterGroup, sessionAuth gin.HandlerFunc, ctrl *controller.CsAgentController, accountGate gin.HandlerFunc) {
	chain := sessionAuthChain(sessionAuth, accountGate)
	group := v1.Group("/cs-agent")
	group.Use(chain...)
	{
		group.POST("/chat", ctrl.Chat)
	}
}
