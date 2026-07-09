// common.go 定义统一 HTTP 响应结构与错误码。
package response

import (
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	CodeSuccess       = 0
	CodeInvalidParams = 10001
	CodeUnauthorized  = 10002
	CodeForbidden     = 10003
	CodeNotFound      = 10004
	CodeInternalError = 50000
)

// Response 统一 API 响应格式。
type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// Pagination 列表分页信息（仅列表接口返回）。
type Pagination struct {
	Page       int   `json:"page"`
	Size       int   `json:"size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
}

// ListData 列表数据包装，包含 list 与 pagination。
type ListData[T any] struct {
	List       []T        `json:"list"`
	Pagination Pagination `json:"pagination"`
}

// PageQuery 分页查询参数。
type PageQuery struct {
	Page int
	Size int
}

// ParsePageQuery 从 query 解析分页参数，page 从 1 开始。
func ParsePageQuery(c *gin.Context, defaultSize int) PageQuery {
	page := parsePositiveInt(c.Query("page"), 1)
	size := parsePositiveInt(c.Query("size"), defaultSize)
	if size > 100 {
		size = 100
	}
	return PageQuery{Page: page, Size: size}
}

// Offset 计算数据库 offset。
func (p PageQuery) Offset() int {
	return (p.Page - 1) * p.Size
}

// NewPagination 根据分页参数与总数生成分页元数据。
func NewPagination(page, size int, total int64) Pagination {
	totalPages := 0
	if size > 0 && total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(size)))
	}
	return Pagination{
		Page:       page,
		Size:       size,
		Total:      total,
		TotalPages: totalPages,
	}
}

// Success 返回非列表成功响应（data 不含 pagination）。
func Success(c *gin.Context, data interface{}) {
	if data == nil {
		data = gin.H{}
	}
	write(c, http.StatusOK, CodeSuccess, "success", data)
}

// SuccessWithMessage 返回带自定义消息的非列表成功响应。
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	if data == nil {
		data = gin.H{}
	}
	write(c, http.StatusOK, CodeSuccess, message, data)
}

// SuccessCreated 返回 201 创建成功响应。
func SuccessCreated(c *gin.Context, data interface{}) {
	if data == nil {
		data = gin.H{}
	}
	write(c, http.StatusCreated, CodeSuccess, "创建成功", data)
}

// SuccessList 返回列表成功响应（data 含 list + pagination）。
func SuccessList[T any](c *gin.Context, list []T, page, size int, total int64) {
	if list == nil {
		list = make([]T, 0)
	}
	write(c, http.StatusOK, CodeSuccess, "success", ListData[T]{
		List:       list,
		Pagination: NewPagination(page, size, total),
	})
}

// Error 返回错误响应。
func Error(c *gin.Context, httpStatus, code int, message string) {
	write(c, httpStatus, code, message, gin.H{})
}

func write(c *gin.Context, httpStatus, code int, message string, data interface{}) {
	c.JSON(httpStatus, Response{
		Code:      code,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Unix(),
	})
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	var n int
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return fallback
		}
		n = n*10 + int(ch-'0')
	}
	if n <= 0 {
		return fallback
	}
	return n
}
