// backend.go Flutter BackendApiClient 兼容的响应格式（直接 JSON，非统一包装）。
package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// BackendErrorBody Flutter 客户端识别的错误结构。
type BackendErrorBody struct {
	Error string `json:"error"`
}

// BackendJSON 返回 Flutter 兼容的直接 JSON 响应。
func BackendJSON(c *gin.Context, status int, payload any) {
	c.JSON(status, payload)
}

// BackendError 返回 Flutter 兼容的错误响应。
func BackendError(c *gin.Context, status int, message string) {
	c.JSON(status, BackendErrorBody{Error: message})
}

// BackendNoContent 返回 204 无内容。
func BackendNoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// FormatTimePtr 将时间指针格式化为 RFC3339 字符串（Supabase 返回兼容）。
func FormatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}
