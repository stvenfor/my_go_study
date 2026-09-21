// require_active_account.go 已登录请求在账号注销或停用时拒绝。
package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
)

// AccountChecker 判断会话账号是否还能调用业务接口。
type AccountChecker interface {
	AllowsRequest(ctx context.Context, userID string) error
}

// RequireActiveAccount 拒绝已注销或停用的账号。锁定中的已有会话仍放行。
func RequireActiveAccount(check AccountChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		if check == nil {
			c.Next()
			return
		}
		user, ok := GetSupabaseUser(c)
		if !ok || user.ID == "" {
			response.BackendError(c, 401, "未登录")
			c.Abort()
			return
		}
		if err := check.AllowsRequest(c.Request.Context(), user.ID); err != nil {
			response.BackendError(c, 403, "账号不可用")
			c.Abort()
			return
		}
		c.Next()
	}
}
