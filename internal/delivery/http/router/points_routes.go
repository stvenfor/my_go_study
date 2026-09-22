package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/controller"
)

func registerPointsRoutes(v1 *gin.RouterGroup, sessionAuth gin.HandlerFunc, ctrl *controller.PointsController, accountGate gin.HandlerFunc) {
	if ctrl == nil || sessionAuth == nil {
		return
	}
	chain := sessionAuthChain(sessionAuth, accountGate)
	g := v1.Group("/points")
	g.Use(chain...)
	g.GET("/status", ctrl.GetStatus)
	g.POST("/check-in", ctrl.CheckIn)
	g.GET("/tasks", ctrl.ListTasks)
	g.POST("/tasks/:code/claim", ctrl.ClaimTask)
}
