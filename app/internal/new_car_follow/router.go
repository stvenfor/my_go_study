// router.go 新车跟进档案路由。
//
// 按 mashenGOAI 的形态：每域一个 Router struct + Register(engine)，
// 取代原先 router.Options 里传 controller + registerXxxRoutes 的集中式装配。
package new_car_follow

import (
	"app/internal/shared"
	"github.com/gin-gonic/gin"
)

// Router 挂载 new_car_follow 域路由。
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

// Register 挂载 /api/v1/new-car-follow-files。
func (r *Router) Register(engine *gin.Engine) {
	if r == nil || r.handler == nil || r.sessionAuth == nil {
		return
	}
	v1 := engine.Group("/api/v1")
	chain := shared.SessionAuthChain(r.sessionAuth, r.accountGate)

	g := v1.Group("/new-car-follow-files")
	g.Use(chain...)
	{
		g.GET("/summary", r.handler.Summary)
		g.GET("/customers", r.handler.ListCustomers)
		g.GET("", r.handler.List)
		g.POST("", r.handler.Create)
		g.GET("/:file_id", r.handler.Get)
		g.PATCH("/:file_id", r.handler.Patch)
		g.GET("/:file_id/logs", r.handler.ListLogs)
		g.POST("/:file_id/logs", r.handler.CreateLog)
	}
}
