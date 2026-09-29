// router.go 二手车业务单路由。
//
// 按 mashenGOAI 的形态：每域一个 Router struct + Register(engine)，
// 取代原先 router.Options 里传 controller + registerXxxRoutes 的集中式装配。
package used_car

import (
	"app/internal/shared"
	"github.com/gin-gonic/gin"
)

// Router 挂载 used_car 域路由。
type Router struct {
	handler     *handler
	sessionAuth gin.HandlerFunc
	accountGate gin.HandlerFunc
}

// NewRouter 创建。sessionAuth 为 nil 表示业务认证未启用，Register 会直接跳过
// （与原先 router.Setup 里 `businessAuth && ctrl != nil && sessionAuth != nil` 等价）。
func NewRouter(svc *Service, sessionAuth, accountGate gin.HandlerFunc) *Router {
	return &Router{
		handler:     newHandler(svc),
		sessionAuth: sessionAuth,
		accountGate: accountGate,
	}
}

// Register 挂载 /api/v1/used-car-orders。
func (r *Router) Register(engine *gin.Engine) {
	if r == nil || r.handler == nil || r.sessionAuth == nil {
		return
	}
	v1 := engine.Group("/api/v1")
	chain := shared.SessionAuthChain(r.sessionAuth, r.accountGate)

	g := v1.Group("/used-car-orders")
	g.Use(chain...)
	{
		g.GET("/summary", r.handler.Summary)
		g.GET("/customers", r.handler.ListCustomers)
		g.GET("", r.handler.List)
		g.POST("", r.handler.Create)
		g.GET("/:order_id", r.handler.Get)
	}
}
