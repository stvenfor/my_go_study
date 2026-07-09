// transaction_handler.go 处理 /api/v1/transactions（Flutter 兼容，自建 JWT 鉴权）。
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

// TransactionHandler Transaction API 处理器。
type TransactionHandler struct {
	transactionUC *usecase.TransactionUsecase
}

// NewTransactionHandler 创建 Transaction 处理器。
func NewTransactionHandler(transactionUC *usecase.TransactionUsecase) *TransactionHandler {
	return &TransactionHandler{transactionUC: transactionUC}
}

// List 列表查询，返回 { items: [...] }。
func (h *TransactionHandler) List(c *gin.Context) {
	userID, ok := jwtAuthContext(c)
	if !ok {
		response.BackendError(c, http.StatusUnauthorized, "未授权")
		return
	}

	filter := entity.TransactionFilter{
		UserID: userID,
		Type:   c.Query("type"),
	}
	if limit, err := strconv.Atoi(c.Query("limit")); err == nil {
		filter.Limit = limit
	}
	if offset, err := strconv.Atoi(c.Query("offset")); err == nil {
		filter.Offset = offset
	}

	items, err := h.transactionUC.List(c.Request.Context(), "", userID, filter)
	if err != nil {
		response.BackendError(c, http.StatusBadGateway, err.Error())
		return
	}
	if items == nil {
		items = []entity.Transaction{}
	}
	response.BackendJSON(c, http.StatusOK, gin.H{"items": items})
}

// Get 按 ID 查询。
func (h *TransactionHandler) Get(c *gin.Context) {
	userID, ok := jwtAuthContext(c)
	if !ok {
		response.BackendError(c, http.StatusUnauthorized, "未授权")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BackendError(c, http.StatusBadRequest, "无效的 id")
		return
	}

	item, err := h.transactionUC.Get(c.Request.Context(), "", userID, id)
	if err != nil {
		response.BackendError(c, http.StatusNotFound, err.Error())
		return
	}
	response.BackendJSON(c, http.StatusOK, item)
}

// Create 创建记录。
func (h *TransactionHandler) Create(c *gin.Context) {
	userID, ok := jwtAuthContext(c)
	if !ok {
		response.BackendError(c, http.StatusUnauthorized, "未授权")
		return
	}

	var input entity.CreateTransactionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BackendError(c, http.StatusBadRequest, "请求体格式错误")
		return
	}
	if input.Type == "" || input.Category == "" || input.Date == "" {
		response.BackendError(c, http.StatusBadRequest, "type、category、date 为必填")
		return
	}

	item, err := h.transactionUC.Create(c.Request.Context(), "", userID, input)
	if err != nil {
		response.BackendError(c, http.StatusBadGateway, err.Error())
		return
	}
	response.BackendJSON(c, http.StatusCreated, item)
}

// Update 更新记录。
func (h *TransactionHandler) Update(c *gin.Context) {
	userID, ok := jwtAuthContext(c)
	if !ok {
		response.BackendError(c, http.StatusUnauthorized, "未授权")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BackendError(c, http.StatusBadRequest, "无效的 id")
		return
	}

	var input entity.UpdateTransactionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BackendError(c, http.StatusBadRequest, "请求体格式错误")
		return
	}

	item, err := h.transactionUC.Update(c.Request.Context(), "", userID, id, input)
	if err != nil {
		response.BackendError(c, http.StatusBadGateway, err.Error())
		return
	}
	response.BackendJSON(c, http.StatusOK, item)
}

// Delete 删除记录。
func (h *TransactionHandler) Delete(c *gin.Context) {
	userID, ok := jwtAuthContext(c)
	if !ok {
		response.BackendError(c, http.StatusUnauthorized, "未授权")
		return
	}

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BackendError(c, http.StatusBadRequest, "无效的 id")
		return
	}

	if err := h.transactionUC.Delete(c.Request.Context(), "", userID, id); err != nil {
		response.BackendError(c, http.StatusBadGateway, err.Error())
		return
	}
	response.BackendNoContent(c)
}
