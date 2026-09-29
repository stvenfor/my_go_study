// router.go 门店、成员与角色分配路由。
//
// 按 mashenGOAI 的形态：每域一个 Router struct + Register(engine)，
// 取代原先 router.Options 里传 controller 的集中式装配。
package access

import (
	"app/internal/shared"
	"github.com/gin-gonic/gin"
)

// Router 挂载 access 域路由。
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

// Register 挂载 /api/v1/stores、/api/v1/roles、/api/v1/me/permissions。
func (r *Router) Register(engine *gin.Engine) {
	if r == nil || r.handler == nil || r.sessionAuth == nil {
		return
	}
	v1 := engine.Group("/api/v1")
	chain := shared.SessionAuthChain(r.sessionAuth, r.accountGate)

	stores := v1.Group("/stores")
	stores.Use(chain...)
	{
		stores.POST("", r.handler.CreateStore)
		stores.POST("/:store_id/members", r.handler.UpsertMember)
		stores.DELETE("/:store_id/members/:user_id", r.handler.RemoveMember)
	}

	roles := v1.Group("/roles")
	roles.Use(chain...)
	{
		roles.POST("/assignments", r.handler.AssignRole)
		roles.DELETE("/assignments", r.handler.RevokeRole)
	}

	me := v1.Group("/me")
	me.Use(chain...)
	{
		me.GET("/permissions", r.handler.ListMyPermissions)
	}
}
