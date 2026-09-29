package router

import (
	"app/internal/delivery/http/controller"
	"github.com/gin-gonic/gin"
)

func registerShortVideoRoutes(v1 *gin.RouterGroup, sessionAuth gin.HandlerFunc, ctrl *controller.ShortVideoController, accountGate gin.HandlerFunc) {
	if ctrl == nil {
		return
	}
	chain := sessionAuthChain(sessionAuth, accountGate)
	g := v1.Group("/short-videos")
	g.Use(chain...)
	g.POST("", ctrl.Create)
	g.GET("", ctrl.List)
	g.GET("/profile", ctrl.Profile)
	g.GET("/:id", ctrl.Get)
	g.DELETE("/:id", ctrl.Delete)
	g.POST("/:id/like", ctrl.Like)
	g.DELETE("/:id/like", ctrl.Unlike)
	g.POST("/:id/view", ctrl.View)
}
