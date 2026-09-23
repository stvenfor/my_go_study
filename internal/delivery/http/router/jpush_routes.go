package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/controller"
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
