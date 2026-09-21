package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/controller"
	"github.com/stvenfor/my_go_study/internal/delivery/http/middleware"
)

// registerAccessRoutes 门店、成员与角色分配。仅 local 有完整后端。
func registerAccessRoutes(v1 *gin.RouterGroup, sbAuth gin.HandlerFunc, accessCtrl *controller.AccessController, accountGate gin.HandlerFunc) {
	if accessCtrl == nil {
		return
	}
	chain := []gin.HandlerFunc{sbAuth, middleware.RequireUserID()}
	if accountGate != nil {
		chain = append(chain, accountGate)
	}

	stores := v1.Group("/stores")
	stores.Use(chain...)
	{
		stores.POST("", accessCtrl.CreateStore)
		stores.POST("/:store_id/members", accessCtrl.UpsertMember)
		stores.DELETE("/:store_id/members/:user_id", accessCtrl.RemoveMember)
	}

	roles := v1.Group("/roles")
	roles.Use(chain...)
	{
		roles.POST("/assignments", accessCtrl.AssignRole)
		roles.DELETE("/assignments", accessCtrl.RevokeRole)
	}

	me := v1.Group("/me")
	me.Use(chain...)
	{
		me.GET("/permissions", accessCtrl.ListMyPermissions)
	}
}
