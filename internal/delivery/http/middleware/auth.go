// auth.go JWT 鉴权中间件，解析 token 并注入用户信息到上下文。
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	jwtmanager "github.com/stvenfor/my_go_study/pkg/jwt"
)

const (
	// ContextUserIDKey 上下文中用户 ID 的键。
	ContextUserIDKey = "userID"
	// ContextUsernameKey 上下文中用户名的键。
	ContextUsernameKey = "username"
)

// Auth JWT 鉴权中间件。
func Auth(jwtManager *jwtmanager.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Error(c, 401, response.CodeUnauthorized, "未提供 Authorization 头")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			response.Error(c, 401, response.CodeUnauthorized, "Authorization 格式错误")
			c.Abort()
			return
		}

		claims, err := jwtManager.Parse(parts[1])
		if err != nil {
			response.Error(c, 401, response.CodeUnauthorized, "token 无效或已过期")
			c.Abort()
			return
		}

		c.Set(ContextUserIDKey, claims.UserID)
		c.Set(ContextUsernameKey, claims.Username)
		c.Next()
	}
}

// GetUserID 从上下文读取当前用户 ID。
func GetUserID(c *gin.Context) (uint, bool) {
	value, ok := c.Get(ContextUserIDKey)
	if !ok {
		return 0, false
	}
	userID, ok := value.(uint)
	return userID, ok
}
