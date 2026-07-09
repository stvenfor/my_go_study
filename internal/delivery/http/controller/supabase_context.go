// supabase_context.go 从 Gin 上下文读取 Supabase 鉴权信息。
package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/middleware"
	pkgauth "github.com/stvenfor/my_go_study/pkg/auth"
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
