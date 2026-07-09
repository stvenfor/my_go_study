// =============================================================================
// 文件：supabase_auth.go
// 层级：Delivery/Middleware —— Gin 请求管道中的「门禁」
//
// 【初学者】中间件执行顺序：
//   请求 → Logger → CORS → SupabaseAuth → Controller
//   SupabaseAuth 失败则 Abort，不会进入 Controller。
// =============================================================================
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	pkgauth "github.com/stvenfor/my_go_study/pkg/auth"
	"github.com/stvenfor/my_go_study/pkg/config"
)

const (
	ContextSupabaseUserKey = "supabaseUser"  // c.Get 取用户
	ContextAccessTokenKey  = "accessToken"   // 转发 PostgREST 时需要原 token
)

// SupabaseAuth 返回 Gin 中间件函数；挂在本需要登录的路由组上。
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

		// 存入 Context，Controller 通过 GetSupabaseUser 读取
		c.Set(ContextSupabaseUserKey, user)
		c.Set(ContextAccessTokenKey, token)
		c.Next() // 放行到下一个 handler
	}
}

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
