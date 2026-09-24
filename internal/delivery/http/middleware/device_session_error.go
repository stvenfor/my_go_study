package middleware

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/usecase"
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
