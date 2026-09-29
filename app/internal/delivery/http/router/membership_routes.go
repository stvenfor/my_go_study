package router

import (
	"app/internal/delivery/http/controller"
	"github.com/gin-gonic/gin"
)

func registerMembershipRoutes(v1 *gin.RouterGroup, sessionAuth gin.HandlerFunc, ctrl *controller.MembershipController, accountGate gin.HandlerFunc) {
	if ctrl == nil || sessionAuth == nil {
		return
	}
	chain := sessionAuthChain(sessionAuth, accountGate)
	g := v1.Group("/membership")
	g.Use(chain...)
	g.GET("/me", ctrl.Me)
	g.POST("/buyout", ctrl.Buyout)
	g.POST("/buyout/confirm", ctrl.ConfirmBuyout)
	g.POST("/huawei/verify", ctrl.VerifyHuawei)
	g.POST("/apple/verify", ctrl.VerifyApple)
	// 关键事件通知可不带用户 Session（华为服务器回调）；MVP stub 仍挂同组便于联调。
	g.POST("/huawei/notifications", ctrl.HuaweiNotifications)
}
