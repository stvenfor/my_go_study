// router.go transactions（交易流水）路由。Supabase JWT 鉴权。
//
// 按 mashenGOAI 的形态：每域一个 Router struct + Register(engine)，
// 取代原先 router.Options 里传 controller + registerXxxRoutes 的集中式装配。
package transaction

import (
	"app/internal/shared"
	"github.com/gin-gonic/gin"
)

// Router 挂载 transaction 域路由。
type Router struct {
	handler     *handler
	sessionAuth gin.HandlerFunc
	accountGate gin.HandlerFunc
}

// NewRouter 创建。sessionAuth 为 nil 表示业务认证未启用，Register 会直接跳过
// （与原先 router.Setup 里 `businessAuth && opts.TransactionController != nil &&
// sessionAuth != nil` 等价）。
func NewRouter(svc *Service, sessionAuth, accountGate gin.HandlerFunc) *Router {
	return &Router{
		handler:     newHandler(svc),
		sessionAuth: sessionAuth,
		accountGate: accountGate,
	}
}

// Register 挂载 /api/v1/transactions 与 /api/v1/transactions/manage（共 11 条），
// 对照切片前 transaction_routes.go 基线。
//
// 两组风格并存是历史兼容要求，勿合并：
//   - /transactions         → Flutter 二手车 App 实际调用的 legacy 格式
//   - /transactions/manage  → 统一响应格式（page/size 分页）
func (r *Router) Register(engine *gin.Engine) {
	if r == nil || r.handler == nil || r.sessionAuth == nil {
		return
	}
	v1 := engine.Group("/api/v1")
	chain := shared.SessionAuthChain(r.sessionAuth, r.accountGate)

	flutterGroup := v1.Group("/transactions")
	flutterGroup.Use(chain...)
	{
		flutterGroup.GET("", r.handler.List) // 无 page 参数时走 Flutter 格式
		flutterGroup.POST("", r.handler.CreateLegacy)
		flutterGroup.GET("/search", r.handler.Search) // 须在 /:id 之前
		flutterGroup.GET("/:id", r.handler.GetLegacy)
		flutterGroup.PUT("/:id", r.handler.UpdateLegacy)
		flutterGroup.DELETE("/:id", r.handler.DeleteLegacy)
	}

	// 统一管理接口（统一响应格式，分页用 page/size）
	manageGroup := v1.Group("/transactions/manage")
	manageGroup.Use(chain...)
	{
		manageGroup.GET("", r.handler.ListPage)
		manageGroup.POST("", r.handler.Create)
		manageGroup.GET("/:id", r.handler.Get)
		manageGroup.PUT("/:id", r.handler.Update)
		manageGroup.DELETE("/:id", r.handler.Delete)
	}
}
