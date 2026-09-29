// router.go 售后专区路由。
//
// 按 mashenGOAI 的形态：每域一个 Router struct + Register(engine)，
// 取代原先 router.Options 里传 controller + registerXxxRoutes 的集中式装配。
package after_sales

import (
	"app/internal/shared"
	"github.com/gin-gonic/gin"
)

// Router 挂载 after_sales 域路由。
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

// Register 挂载 /api/v1/after-sales。
func (r *Router) Register(engine *gin.Engine) {
	if r == nil || r.handler == nil || r.sessionAuth == nil {
		return
	}
	v1 := engine.Group("/api/v1")
	chain := shared.SessionAuthChain(r.sessionAuth, r.accountGate)

	g := v1.Group("/after-sales")
	g.Use(chain...)
	{
		g.GET("/pending-appointments", r.handler.ListPendingAppointments)
		g.GET("/records", r.handler.ListRecords)
		g.POST("/records", r.handler.CreateRecord)
		g.GET("/records/:record_id", r.handler.GetRecord)
	}
}
