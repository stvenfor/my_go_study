// router.go 现金钱包 / 银行卡 / 充值路由。
package wallet

import (
	"app/internal/shared"
	"github.com/gin-gonic/gin"
)

// Router 挂载 wallet 域路由。
type Router struct {
	handler     *handler
	sessionAuth gin.HandlerFunc
	accountGate gin.HandlerFunc
}

// NewRouter 创建。sessionAuth 为 nil 表示业务认证未启用，Register 会直接跳过。
func NewRouter(svc *Service, sessionAuth, accountGate gin.HandlerFunc) *Router {
	return &Router{
		handler:     newHandler(svc),
		sessionAuth: sessionAuth,
		accountGate: accountGate,
	}
}

// Register 挂载 /api/v1/wallet。
func (r *Router) Register(engine *gin.Engine) {
	if r == nil || r.handler == nil || r.sessionAuth == nil {
		return
	}
	g := engine.Group("/api/v1/wallet")
	g.Use(shared.SessionAuthChain(r.sessionAuth, r.accountGate)...)
	g.GET("", r.handler.GetSummary)
	g.GET("/ledger", r.handler.ListLedger)
	g.GET("/cards", r.handler.ListCards)
	g.POST("/cards", r.handler.BindCard)
	g.POST("/cards/:card_id/default", r.handler.SetDefaultCard)
	g.DELETE("/cards/:card_id", r.handler.DeleteCard)
	g.POST("/recharge", r.handler.Recharge)
}
