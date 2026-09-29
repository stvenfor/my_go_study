package router

import (
	"app/internal/delivery/http/controller"
	"github.com/gin-gonic/gin"
)

func registerCsAgentRoutes(v1 *gin.RouterGroup, sessionAuth gin.HandlerFunc, ctrl *controller.CsAgentController, accountGate gin.HandlerFunc) {
	chain := sessionAuthChain(sessionAuth, accountGate)
	group := v1.Group("/cs-agent")
	group.Use(chain...)
	{
		group.POST("/chat", ctrl.Chat)
	}
}
