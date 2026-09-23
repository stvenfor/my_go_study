package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/controller"
)

func registerNewCarFollowRoutes(v1 *gin.RouterGroup, sbAuth gin.HandlerFunc, ctrl *controller.NewCarFollowController, accountGate gin.HandlerFunc) {
	if ctrl == nil {
		return
	}
	chain := sessionAuthChain(sbAuth, accountGate)
	g := v1.Group("/new-car-follow-files")
	g.Use(chain...)
	{
		g.GET("/summary", ctrl.Summary)
		g.GET("/customers", ctrl.ListCustomers)
		g.GET("", ctrl.List)
		g.POST("", ctrl.Create)
		g.GET("/:file_id", ctrl.Get)
		g.PATCH("/:file_id", ctrl.Patch)
		g.GET("/:file_id/logs", ctrl.ListLogs)
		g.POST("/:file_id/logs", ctrl.CreateLog)
	}
}
