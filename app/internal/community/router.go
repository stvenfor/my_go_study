// router.go 社区（话题 / 动态 / 点赞 / 评论 / 关注 / 搜索）路由。
//
// 按 mashenGOAI 的形态：每域一个 Router struct + Register(engine)，
// 取代原先 router.Options 里传 controller + registerXxxRoutes 的集中式装配。
package community

import (
	"app/internal/shared"
	"github.com/gin-gonic/gin"
)

// Router 挂载 community 域路由。
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

// Register 挂载 /api/v1/community。12 条路由，对照切片前 community_routes.go 基线。
func (r *Router) Register(engine *gin.Engine) {
	if r == nil || r.handler == nil || r.sessionAuth == nil {
		return
	}
	v1 := engine.Group("/api/v1")
	chain := shared.SessionAuthChain(r.sessionAuth, r.accountGate)

	g := v1.Group("/community")
	g.Use(chain...)
	{
		g.GET("/topics", r.handler.ListTopics)
		g.GET("/topics/search", r.handler.SearchTopics)
		g.GET("/search", r.handler.Search)

		g.POST("/posts", r.handler.CreatePost)
		g.GET("/posts", r.handler.ListPosts)
		g.DELETE("/posts/:id", r.handler.DeletePost)
		g.POST("/posts/:id/like", r.handler.LikePost)
		g.DELETE("/posts/:id/like", r.handler.UnlikePost)
		g.GET("/posts/:id/comments", r.handler.ListComments)
		g.POST("/posts/:id/comments", r.handler.AddComment)

		g.POST("/users/:id/follow", r.handler.FollowUser)
		g.DELETE("/users/:id/follow", r.handler.UnfollowUser)
	}
}
