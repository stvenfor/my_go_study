// router.go 首页待办 / 入店申请路由。
package home

import (
	"app/internal/shared"
	"github.com/gin-gonic/gin"
)

// Router 挂载 home 域路由。
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

// Register 挂载 /api/v1/home 与 /api/v1/stores/:store_id/join-applications。
func (r *Router) Register(engine *gin.Engine) {
	if r == nil || r.handler == nil || r.sessionAuth == nil {
		return
	}
	v1 := engine.Group("/api/v1")
	chain := shared.SessionAuthChain(r.sessionAuth, r.accountGate)

	home := v1.Group("/home")
	home.Use(chain...)
	{
		home.GET("/todo-cards", r.handler.ListTodoCards)
		home.GET("/join-applications", r.handler.ListPendingJoinApplications)
		home.POST("/join-applications/:application_id/approve", r.handler.ApproveJoinApplication)
		home.POST("/join-applications/:application_id/reject", r.handler.RejectJoinApplication)
		home.GET("/follow-up-customers", r.handler.ListOverdueCustomers)
		home.GET("/after-sales-appointments", r.handler.ListPendingAppointments)
		home.GET("/store-review-orders", r.handler.ListPendingReviewOrders)
	}

	stores := v1.Group("/stores")
	stores.Use(chain...)
	{
		stores.POST("/:store_id/join-applications", r.handler.ApplyToStore)
	}
}
