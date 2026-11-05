// router.go 支付预下单路由。
//
// 按 mashenGOAI 的形态：每域一个 Router struct + Register(engine)，
// 取代原先 router.Options 里传 controller + registerXxxRoutes 的集中式装配。
package payment

import (
	"app/internal/shared"
	"github.com/gin-gonic/gin"
)

// Router 挂载 payment 域路由。
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

// Register 挂载 /api/v1/payments。
//
// notify 是公网渠道回调，**不进 SessionAuth 链**（渠道没有会话），
// 唯一信任来源是报文验签；prepay 仍需登录。两者共用一组前缀但链不同。
func (r *Router) Register(engine *gin.Engine) {
	if r == nil || r.handler == nil {
		return
	}
	v1 := engine.Group("/api/v1")

	if r.handler.svc != nil {
		notify := v1.Group("/payments/notify")
		notify.POST("/wechat", r.handler.WechatNotify)
		notify.POST("/alipay", r.handler.AlipayNotify)
	}

	if r.sessionAuth == nil {
		return
	}
	chain := shared.SessionAuthChain(r.sessionAuth, r.accountGate)
	g := v1.Group("/payments", chain...)
	g.POST("/prepay", r.handler.Prepay)
}
