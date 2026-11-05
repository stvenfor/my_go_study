// router.go 会员订阅路由。
//
// 按 mashenGOAI 的形态：每域一个 Router struct + Register(engine)，
// 取代原先 router.Options 里传 controller + registerXxxRoutes 的集中式装配。
package membership

import (
	"app/internal/shared"
	"github.com/gin-gonic/gin"
)

// Router 挂载 membership 域路由。
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

// Register 挂载 /api/v1/membership。
//
// 切片前由 router.Setup 里的 `opts.Config.Auth.IsLocalProvider()` 门控（仅本地
// provider 注册），装配层保持同一门控（main.go 只在本地分支构造本域 Service）。
func (r *Router) Register(engine *gin.Engine) {
	if r == nil || r.handler == nil || r.sessionAuth == nil {
		return
	}
	v1 := engine.Group("/api/v1")
	chain := shared.SessionAuthChain(r.sessionAuth, r.accountGate)

	g := v1.Group("/membership")
	g.Use(chain...)
	g.GET("/me", r.handler.Me)
	g.POST("/buyout", r.handler.Buyout)
	g.POST("/buyout/confirm", r.handler.ConfirmBuyout)
	g.POST("/huawei/verify", r.handler.VerifyHuawei)
	g.POST("/apple/verify", r.handler.VerifyApple)
	// 关键事件通知可不带用户 Session（华为服务器回调）；回调未接入前该路由恒返 501，
	// 挂同组仅保留占位与联调可见性（见 docs/placeholder-inventory.md）。
	g.POST("/huawei/notifications", r.handler.HuaweiNotifications)
}
