package router

import "github.com/gin-gonic/gin"

// sessionAuthChain 受保护路由：Session Auth（+ 可选账号状态门），身份取自会话 user_id。
func sessionAuthChain(auth gin.HandlerFunc, accountGate gin.HandlerFunc) []gin.HandlerFunc {
	chain := []gin.HandlerFunc{auth}
	if accountGate != nil {
		chain = append(chain, accountGate)
	}
	return chain
}
