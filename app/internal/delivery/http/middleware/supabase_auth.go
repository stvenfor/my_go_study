// supabase_auth.go — 会话身份 helper 的转发层。
//
// 真正的实现在 authctx 包（叶子包，不与 usecase 成环，域 handler 直接用）。
// 这里保留同名导出符号，是为了让尚未切片的 controller 不必一次性全改；
// 批 2~4 切完后可删除本文件的转发函数。
package middleware

import (
	"app/internal/delivery/http/authctx"
	pkgauth "common/auth"
	"github.com/gin-gonic/gin"
)

const (
	ContextSupabaseUserKey = authctx.ContextSupabaseUserKey // c.Get 取用户
	ContextAccessTokenKey  = authctx.ContextAccessTokenKey  // 转发 PostgREST 时需要原 token
)

// GetSupabaseUser 见 authctx.GetSupabaseUser。
func GetSupabaseUser(c *gin.Context) (pkgauth.SupabaseUser, bool) {
	return authctx.GetSupabaseUser(c)
}

// GetAccessToken 见 authctx.GetAccessToken。
func GetAccessToken(c *gin.Context) (string, bool) {
	return authctx.GetAccessToken(c)
}

// SupabaseAuthContext 见 authctx.SupabaseAuthContext。
func SupabaseAuthContext(c *gin.Context) (pkgauth.SupabaseUser, string, bool) {
	return authctx.SupabaseAuthContext(c)
}

// bearerToken 解析 "Bearer eyJ..." → "eyJ..."
func bearerToken(header string) string {
	return authctx.BearerToken(header)
}
