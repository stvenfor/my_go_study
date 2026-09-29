package router

import (
	"app/internal/delivery/http/controller"
	"github.com/gin-gonic/gin"
)

func registerPaymentRoutes(v1 *gin.RouterGroup, sessionAuth gin.HandlerFunc, ctrl *controller.PaymentController, accountGate gin.HandlerFunc) {
	if ctrl == nil || sessionAuth == nil {
		return
	}
	chain := sessionAuthChain(sessionAuth, accountGate)
	g := v1.Group("/payments", chain...)
	g.POST("/prepay", ctrl.Prepay)
}
