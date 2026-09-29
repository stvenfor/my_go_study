// resource_error.go 资源型 CRUD 的同构错误映射（跨域共用）。
//
// 原为 delivery/http/controller/resource_error.go 里的 writeResourceCRUDError，
// 供社区与短视频两个 controller 共用。批 3 把这两域切成垂直切片后，两域 handler
// 分属不同包，函数跨包不可见，故上提到共享包并按原语义导出（文案必须一致）。
package shared

import (
	"errors"
	"net/http"

	"app/internal/delivery/http/dto/response"
	"github.com/gin-gonic/gin"
)

// WriteResourceCRUDError 社区/短视频等同构错误映射（文案必须一致）。
func WriteResourceCRUDError(c *gin.Context, err, invalid, notFound, forbidden error) {
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
