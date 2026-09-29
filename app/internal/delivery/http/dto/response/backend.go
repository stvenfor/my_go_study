// backend.go Flutter BackendApiClient 兼容的响应格式（直接 JSON，非统一包装）。
package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// BackendErrorBody Flutter 客户端识别的错误结构。
// Code 与统一信封 response.Code* 对齐；客户端分支只认 code，error 仅展示。
type BackendErrorBody struct {
	Error string `json:"error"`
	Code  int    `json:"code,omitempty"`
}

// BackendJSON 返回 Flutter 兼容的直接 JSON 响应。
func BackendJSON(c *gin.Context, status int, payload any) {
	c.JSON(status, payload)
}

// BackendError 返回 Flutter 兼容的错误响应（按 HTTP status 填默认业务码）。
func BackendError(c *gin.Context, status int, message string) {
	BackendErrorWithCode(c, status, defaultBackendCode(status), message)
}

// BackendErrorWithCode 返回带显式业务码的 Flutter 兼容错误。
func BackendErrorWithCode(c *gin.Context, status, code int, message string) {
	c.JSON(status, BackendErrorBody{Error: message, Code: code})
}

func defaultBackendCode(status int) int {
	switch status {
	case http.StatusUnauthorized:
		return CodeTokenInvalid
	case http.StatusForbidden:
		return CodeForbidden
	case http.StatusNotFound:
		return CodeNotFound
	case http.StatusBadRequest:
		return CodeInvalidParams
	default:
		if status >= 500 {
			return CodeInternalError
		}
		return CodeInvalidParams
	}
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
