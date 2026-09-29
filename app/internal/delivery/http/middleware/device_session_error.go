package middleware

import (
	"errors"
	"log"
	"net/http"

	"app/internal/delivery/http/dto/response"
	"app/internal/usecase"
	"github.com/gin-gonic/gin"
)

// abortDeviceSessionError 单设备 session 失败：统一 {code,message,data}，供 Flutter 按 code 分支。
func abortDeviceSessionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrSessionReplaced):
		log.Printf("[auth] session replaced path=%s", c.Request.URL.Path)
		response.Error(c, http.StatusUnauthorized, response.CodeSessionReplaced, usecase.MsgSessionReplaced)
	default:
		log.Printf("[auth] session invalid path=%s err=%v", c.Request.URL.Path, err)
		response.Error(c, http.StatusUnauthorized, response.CodeSessionInvalid, usecase.MsgSessionInvalid)
	}
	c.Abort()
}
