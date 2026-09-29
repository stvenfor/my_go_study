package router

import (
	"github.com/gin-gonic/gin"
	"app/internal/delivery/http/controller"
)

func registerPaymentRoutes(v1 *gin.RouterGroup, sessionAuth gin.HandlerFunc, ctrl *controller.PaymentController, accountGate gin.HandlerFunc) {
	if ctrl == nil || sessionAuth == nil {
		return
	}
	chain := sessionAuthChain(sessionAuth, accountGate)
	g := v1.Group("/payments", chain...)
	g.POST("/prepay", ctrl.Prepay)
}
