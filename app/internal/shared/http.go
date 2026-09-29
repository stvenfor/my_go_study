// http.go 跨域共用的 HTTP 装配片段。
package shared

import "github.com/gin-gonic/gin"

// SessionAuthChain 受保护路由的中间件链：Session Auth（必选）+ 账号状态门（可选）。
//
// 各域 Router 用它拼自己的路由组；identity 由 Session Auth 写入上下文，
// 业务侧只读不传。
func SessionAuthChain(auth gin.HandlerFunc, accountGate gin.HandlerFunc) []gin.HandlerFunc {
	chain := []gin.HandlerFunc{auth}
	if accountGate != nil {
		chain = append(chain, accountGate)
	}
	return chain
}

// Registrable 可挂载到 HTTP 引擎的路由模块（每域一个）。
//
// 迁移期 router.Setup 通过它接收已切片域的路由；未切片域仍走旧的
// registerXxxRoutes 函数。全部域切完后 Setup 只剩"遍历 + Register"。
type Registrable interface {
	Register(engine *gin.Engine)
}
