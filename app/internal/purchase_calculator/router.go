// router.go 购车计算器路由。
//
// 本域是唯一公开域：不取身份、无 SessionAuth，因此 Router 只持有 handler，
// NewRouter 也不接收 sessionAuth / accountGate（注册与否由装配层决定）。
package purchase_calculator

import "github.com/gin-gonic/gin"

// Router 挂载 purchase_calculator 域路由。
type Router struct {
	handler *handler
}

// NewRouter 创建。
func NewRouter(svc *Service) *Router {
	return &Router{handler: newHandler(svc)}
}

// Register 挂载 /api/v1/purchase-calculator（公开，无需 SessionAuth）。
func (r *Router) Register(engine *gin.Engine) {
	if r == nil || r.handler == nil {
		return
	}
	g := engine.Group("/api/v1/purchase-calculator")
	g.GET("/products", r.handler.ListProducts)
	g.POST("/quote", r.handler.Quote)
}
