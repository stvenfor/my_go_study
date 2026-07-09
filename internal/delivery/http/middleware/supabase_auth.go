// supabase_auth.go Supabase JWT 鉴权中间件。
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	pkgauth "github.com/stvenfor/my_go_study/pkg/auth"
	"github.com/stvenfor/my_go_study/pkg/config"
)

const (
	// ContextSupabaseUserKey 上下文中 Supabase 用户的键。
	ContextSupabaseUserKey = "supabaseUser"
	// ContextAccessTokenKey 上下文中 access token 的键。
	ContextAccessTokenKey = "accessToken"
)

// SupabaseAuth 校验 Supabase access token 并注入用户信息。
func SupabaseAuth(cfg config.SupabaseConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := bearerToken(c.GetHeader("Authorization"))
		if token == "" {
			response.BackendError(c, 401, "未提供 Authorization 头")
			c.Abort()
			return
		}

		user, err := pkgauth.ValidateAccessToken(c.Request.Context(), cfg, token)
		if err != nil {
			response.BackendError(c, 401, err.Error())
			c.Abort()
			return
		}

		c.Set(ContextSupabaseUserKey, user)
		c.Set(ContextAccessTokenKey, token)
		c.Next()
	}
}

// GetSupabaseUser 从上下文读取 Supabase 用户。
func GetSupabaseUser(c *gin.Context) (pkgauth.SupabaseUser, bool) {
	value, ok := c.Get(ContextSupabaseUserKey)
	if !ok {
		return pkgauth.SupabaseUser{}, false
	}
	user, ok := value.(pkgauth.SupabaseUser)
	return user, ok
}

// GetAccessToken 从上下文读取 access token。
func GetAccessToken(c *gin.Context) (string, bool) {
	value, ok := c.Get(ContextAccessTokenKey)
	if !ok {
		return "", false
	}
	token, ok := value.(string)
	return token, ok
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if strings.HasPrefix(header, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(header, prefix))
	}
	return strings.TrimSpace(header)
}
