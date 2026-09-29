package router

import (
	"app/internal/delivery/http/controller"
	"github.com/gin-gonic/gin"
)

func registerJPushRoutes(v1 *gin.RouterGroup, sessionAuth gin.HandlerFunc, ctrl *controller.JPushController, accountGate gin.HandlerFunc) {
	if ctrl == nil || sessionAuth == nil {
		return
	}
	chain := sessionAuthChain(sessionAuth, accountGate)
	g := v1.Group("/push")
	g.Use(chain...)
	g.POST("/devices", ctrl.RegisterDevice)
	g.POST("/send", ctrl.Send)
}
