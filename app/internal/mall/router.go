// router.go 门店商城路由。
//
// 按 mashenGOAI 的形态：每域一个 Router struct + Register(engine)，
// 取代原先 router.Options 里传 controller + registerXxxRoutes 的集中式装配。
package mall

import (
	"app/internal/shared"
	"github.com/gin-gonic/gin"
)

// Router 挂载 mall 域路由。
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

// Register 挂载 /api/v1/mall。
//
// 切片前由 router.Setup 里的 `opts.Config.Auth.IsLocalProvider()` 门控（仅本地
// provider 注册），装配层保持同一门控（main.go 只在本地分支构造本域 Service）。
func (r *Router) Register(engine *gin.Engine) {
	if r == nil || r.handler == nil || r.sessionAuth == nil {
		return
	}
	v1 := engine.Group("/api/v1")
	chain := shared.SessionAuthChain(r.sessionAuth, r.accountGate)

	g := v1.Group("/mall")
	g.Use(chain...)
	{
		g.POST("/products", r.handler.CreateProduct)
		g.POST("/products/:product_id/skus", r.handler.CreateSKU)
		g.POST("/skus/:sku_id/codes", r.handler.AddVirtualCodes)

		g.GET("/stores/:store_id/products", r.handler.ListProducts)
		g.GET("/stores/:store_id/products/:product_id", r.handler.GetProduct)

		g.POST("/cart", r.handler.UpsertCart)
		g.GET("/cart", r.handler.ListCart)

		g.GET("/orders", r.handler.ListOrders)
		g.POST("/orders", r.handler.CreateOrder)
		g.GET("/orders/:order_id", r.handler.GetOrder)
		g.POST("/orders/:order_id/pay", r.handler.PayOrder)
		g.POST("/orders/:order_id/cancel", r.handler.CancelOrder)
	}
}
