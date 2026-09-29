// supabase_context.go 从 Gin 上下文读取 Supabase 鉴权信息。
package controller

import (
	"github.com/gin-gonic/gin"
	"app/internal/delivery/http/middleware"
	pkgauth "common/auth"
)

func supabaseAuthContext(c *gin.Context) (pkgauth.SupabaseUser, string, bool) {
	user, ok := middleware.GetSupabaseUser(c)
	if !ok {
		return pkgauth.SupabaseUser{}, "", false
	}
	token, ok := middleware.GetAccessToken(c)
	if !ok {
		return pkgauth.SupabaseUser{}, "", false
	}
	return user, token, true
}
