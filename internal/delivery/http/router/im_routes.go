package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/controller"
)

func registerImRoutes(v1 *gin.RouterGroup, sessionAuth gin.HandlerFunc, ctrl *controller.ImController, accountGate gin.HandlerFunc) {
	if ctrl == nil || sessionAuth == nil {
		return
	}
	chain := sessionAuthChain(sessionAuth, accountGate)
	g := v1.Group("/im")
	g.Use(chain...)

	g.POST("/session", ctrl.CreateSession)
	g.GET("/users/search", ctrl.SearchUsers)
	g.GET("/users/profile", ctrl.GetUserProfiles)
	g.GET("/friends", ctrl.ListFriends)
	g.GET("/friends/requests", ctrl.ListFriendRequests)
	g.POST("/friends/requests", ctrl.RequestFriend)
	g.POST("/friends/requests/:id/respond", ctrl.RespondFriend)
	g.GET("/private/admission", ctrl.CheckPrivateAdmission)
	g.POST("/groups/free", ctrl.CreateFreeGroup)
	g.POST("/groups/free/:id/invite", ctrl.InviteFreeGroup)
	g.POST("/groups/free/:id/kick", ctrl.KickFreeGroup)
	g.POST("/groups/free/:id/quit", ctrl.QuitFreeGroup)
	g.POST("/groups/free/:id/dismiss", ctrl.DismissFreeGroup)
	g.POST("/groups/store/sync", ctrl.SyncStoreGroup)
	g.POST("/messages/backup", ctrl.BackupMessages)
}
