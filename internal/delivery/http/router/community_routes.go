package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/controller"
)

func registerCommunityRoutes(v1 *gin.RouterGroup, sessionAuth gin.HandlerFunc, ctrl *controller.CommunityController, accountGate gin.HandlerFunc) {
	chain := sessionAuthChain(sessionAuth, accountGate)
	g := v1.Group("/community")
	g.Use(chain...)
	g.GET("/topics", ctrl.ListTopics)
	g.GET("/topics/search", ctrl.SearchTopics)
	g.POST("/posts", ctrl.CreatePost)
	g.GET("/posts", ctrl.ListPosts)
	g.DELETE("/posts/:id", ctrl.DeletePost)
	g.POST("/posts/:id/like", ctrl.LikePost)
	g.DELETE("/posts/:id/like", ctrl.UnlikePost)
	g.GET("/posts/:id/comments", ctrl.ListComments)
	g.POST("/posts/:id/comments", ctrl.AddComment)
}
