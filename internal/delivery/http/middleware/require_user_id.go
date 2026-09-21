// require_user_id.go 要求已登录请求携带与会话一致的 user_id。
package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
)

// RequireUserID 从 query 或 JSON body 读取 user_id，必须等于当前会话用户。
// 调用前需已写入 supabaseUser。会把 body 放回，供后续 handler 再次绑定。
func RequireUserID() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, ok := GetSupabaseUser(c)
		if !ok || strings.TrimSpace(user.ID) == "" {
			response.BackendError(c, 401, "未登录")
			c.Abort()
			return
		}
		got := requestUserID(c)
		if got == "" {
			response.BackendError(c, 400, "缺少 user_id")
			c.Abort()
			return
		}
		if got != user.ID {
			response.BackendError(c, 400, "user_id 与当前登录身份不一致")
			c.Abort()
			return
		}
		c.Next()
	}
}

func requestUserID(c *gin.Context) string {
	if v := strings.TrimSpace(c.Query("user_id")); v != "" {
		return v
	}
	if c.Request == nil || c.Request.Body == nil {
		return ""
	}
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return ""
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(raw))
	if len(bytes.TrimSpace(raw)) == 0 {
		return ""
	}
	var payload struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.UserID)
}
