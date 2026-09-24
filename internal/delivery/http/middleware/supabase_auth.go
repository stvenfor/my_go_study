// supabase_auth.go — SessionAuth 共用的 user/token context helpers。
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	pkgauth "github.com/stvenfor/my_go_study/pkg/auth"
)

const (
	ContextSupabaseUserKey = "supabaseUser" // c.Get 取用户
	ContextAccessTokenKey  = "accessToken"  // 转发 PostgREST 时需要原 token
)

func GetSupabaseUser(c *gin.Context) (pkgauth.SupabaseUser, bool) {
	value, ok := c.Get(ContextSupabaseUserKey)
	if !ok {
		return pkgauth.SupabaseUser{}, false
	}
	user, ok := value.(pkgauth.SupabaseUser)
	return user, ok
}

func GetAccessToken(c *gin.Context) (string, bool) {
	value, ok := c.Get(ContextAccessTokenKey)
	if !ok {
		return "", false
	}
	token, ok := value.(string)
	return token, ok
}

// bearerToken 解析 "Bearer eyJ..." → "eyJ..."
func bearerToken(header string) string {
	const prefix = "Bearer "
	if strings.HasPrefix(header, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(header, prefix))
	}
	return strings.TrimSpace(header)
}
