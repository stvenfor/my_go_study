// auth_context.go 从 Gin 上下文读取鉴权信息。
package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/middleware"
)

func jwtAuthContext(c *gin.Context) (userID string, ok bool) {
	id, ok := middleware.GetUserID(c)
	if !ok {
		return "", false
	}
	return strconv.FormatUint(uint64(id), 10), true
}
