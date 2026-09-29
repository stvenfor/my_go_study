// router.go 积分 / 签到 / 成长任务路由。
package points

import (
	"app/internal/shared"
	"github.com/gin-gonic/gin"
)

// Router 挂载 points 域路由。
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

// Register 挂载 /api/v1/points。
func (r *Router) Register(engine *gin.Engine) {
	if r == nil || r.handler == nil || r.sessionAuth == nil {
		return
	}
	g := engine.Group("/api/v1/points")
	g.Use(shared.SessionAuthChain(r.sessionAuth, r.accountGate)...)
	g.GET("/status", r.handler.GetStatus)
	g.POST("/check-in", r.handler.CheckIn)
	g.GET("/tasks", r.handler.ListTasks)
	g.POST("/tasks/:code/claim", r.handler.ClaimTask)
}
