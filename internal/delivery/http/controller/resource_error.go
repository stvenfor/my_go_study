package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
)

// writeResourceCRUDError 社区/短视频等同构错误映射（文案必须一致）。
func writeResourceCRUDError(c *gin.Context, err, invalid, notFound, forbidden error) {
	switch {
	case errors.Is(err, invalid):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
	case errors.Is(err, notFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "资源不存在")
	case errors.Is(err, forbidden):
		response.Error(c, http.StatusForbidden, response.CodeForbidden, "无权限")
	default:
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "操作失败")
	}
}
