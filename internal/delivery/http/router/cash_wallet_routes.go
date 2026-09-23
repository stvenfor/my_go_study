package router

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/controller"
)

func registerCashWalletRoutes(v1 *gin.RouterGroup, sessionAuth gin.HandlerFunc, ctrl *controller.CashWalletController, accountGate gin.HandlerFunc) {
	if ctrl == nil || sessionAuth == nil {
		return
	}
	chain := sessionAuthChain(sessionAuth, accountGate)
	g := v1.Group("/wallet")
	g.Use(chain...)
	g.GET("", ctrl.GetSummary)
	g.GET("/ledger", ctrl.ListLedger)
	g.GET("/cards", ctrl.ListCards)
	g.POST("/cards", ctrl.BindCard)
	g.POST("/cards/:card_id/default", ctrl.SetDefaultCard)
	g.DELETE("/cards/:card_id", ctrl.DeleteCard)
	g.POST("/recharge", ctrl.Recharge)
}
