// handler.go transactions REST API。
//
// 由 app/internal/delivery/http/controller/transaction_controller.go 迁入。
// 取身份改用 authctx.SupabaseAuthContext；搜索不可用判定改用 shared.ErrSearchUnavailable。
//
// 两套风格（迁移前后一致，勿合并）：
//
//	listLegacy  → GET /api/v1/transactions        → { items: [] }
//	ListPage    → GET /api/v1/transactions/manage → ResultModel 分页
package transaction

import (
	"errors"
	"net/http"
	"strconv"

	"app/internal/delivery/http/authctx"
	"app/internal/delivery/http/dto/request"
	"app/internal/delivery/http/dto/response"
	"app/internal/shared"
	"github.com/gin-gonic/gin"
	"model/entity"
)

// handler Transaction 管理控制器。
type handler struct {
	svc *Service
}

// newHandler 创建 Transaction 控制器。
func newHandler(svc *Service) *handler {
	return &handler{svc: svc}
}

// authContext 从 SupabaseAuth 中间件注入的 Context 取 token 与 userID。
func (h *handler) authContext(c *gin.Context) (accessToken, userID string, ok bool) {
	user, token, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		return "", "", false
	}
	return token, user.ID, true
}

// List 若带 page 参数走 manage 分页，否则走 Flutter legacy limit/offset。
func (h *handler) List(c *gin.Context) {
	if c.Query("page") != "" {
		h.ListPage(c)
		return
	}
	h.listLegacy(c)
}

// ListPage 统一分页列表（page/size，返回 list + pagination）。
// GET /api/v1/transactions/manage
func (h *handler) ListPage(c *gin.Context) {
	accessToken, userID, ok := h.authContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}

	page := response.ParsePageQuery(c, 20)
	filter := entity.TransactionFilter{
		UserID: userID,
		Type:   c.Query("type"),
		Limit:  page.Size,
		Offset: page.Offset(),
	}

	items, total, err := h.svc.ListPage(c.Request.Context(), accessToken, userID, filter)
	if err != nil {
		response.Error(c, http.StatusBadGateway, response.CodeInternalError, err.Error())
		return
	}
	response.SuccessList(c, response.FromTransactions(items), page.Page, page.Size, total)
}

// listLegacy Flutter 二手车列表实际使用的接口。
func (h *handler) listLegacy(c *gin.Context) {
	accessToken, userID, ok := h.authContext(c)
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

	items, err := h.svc.List(c.Request.Context(), accessToken, userID, filter)
	if err != nil {
		response.BackendError(c, http.StatusBadGateway, err.Error())
		return
	}
	if items == nil {
		items = []entity.Transaction{}
	}
	response.BackendJSON(c, http.StatusOK, gin.H{"items": items})
}

// Get 按 ID 查询（统一响应格式）。
// GET /api/v1/transactions/manage/:id
func (h *handler) Get(c *gin.Context) {
	accessToken, userID, ok := h.authContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}

	id, err := parseTransactionID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "无效的 id")
		return
	}

	item, err := h.svc.Get(c.Request.Context(), accessToken, userID, id)
	if err != nil {
		response.Error(c, http.StatusNotFound, response.CodeNotFound, err.Error())
		return
	}
	response.Success(c, response.FromTransaction(item))
}

// GetLegacy Flutter 兼容单条查询。
func (h *handler) GetLegacy(c *gin.Context) {
	accessToken, userID, ok := h.authContext(c)
	if !ok {
		response.BackendError(c, http.StatusUnauthorized, "未授权")
		return
	}

	id, err := parseTransactionID(c)
	if err != nil {
		response.BackendError(c, http.StatusBadRequest, "无效的 id")
		return
	}

	item, err := h.svc.Get(c.Request.Context(), accessToken, userID, id)
	if err != nil {
		response.BackendError(c, http.StatusNotFound, err.Error())
		return
	}
	response.BackendJSON(c, http.StatusOK, item)
}

// Create 创建记录（统一响应格式）。
// POST /api/v1/transactions/manage
func (h *handler) Create(c *gin.Context) {
	accessToken, userID, ok := h.authContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}

	var req request.CreateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}

	item, err := h.svc.Create(c.Request.Context(), accessToken, userID, entity.CreateTransactionInput{
		Type:     req.Type,
		Category: req.Category,
		Amount:   req.Amount,
		Date:     req.Date,
		Note:     req.Note,
	})
	if err != nil {
		response.Error(c, http.StatusBadGateway, response.CodeInternalError, err.Error())
		return
	}
	response.SuccessCreated(c, response.FromTransaction(item))
}

// CreateLegacy Flutter 兼容创建。
func (h *handler) CreateLegacy(c *gin.Context) {
	accessToken, userID, ok := h.authContext(c)
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

	item, err := h.svc.Create(c.Request.Context(), accessToken, userID, input)
	if err != nil {
		response.BackendError(c, http.StatusBadGateway, err.Error())
		return
	}
	response.BackendJSON(c, http.StatusCreated, item)
}

// Update 更新记录（统一响应格式）。
// PUT /api/v1/transactions/manage/:id
func (h *handler) Update(c *gin.Context) {
	accessToken, userID, ok := h.authContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}

	id, err := parseTransactionID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "无效的 id")
		return
	}

	var req request.UpdateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}

	item, err := h.svc.Update(c.Request.Context(), accessToken, userID, id, entity.UpdateTransactionInput{
		Type:     req.Type,
		Category: req.Category,
		Amount:   req.Amount,
		Date:     req.Date,
		Note:     req.Note,
	})
	if err != nil {
		response.Error(c, http.StatusBadGateway, response.CodeInternalError, err.Error())
		return
	}
	response.Success(c, response.FromTransaction(item))
}

// UpdateLegacy Flutter 兼容更新。
func (h *handler) UpdateLegacy(c *gin.Context) {
	accessToken, userID, ok := h.authContext(c)
	if !ok {
		response.BackendError(c, http.StatusUnauthorized, "未授权")
		return
	}

	id, err := parseTransactionID(c)
	if err != nil {
		response.BackendError(c, http.StatusBadRequest, "无效的 id")
		return
	}

	var input entity.UpdateTransactionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BackendError(c, http.StatusBadRequest, "请求体格式错误")
		return
	}

	item, err := h.svc.Update(c.Request.Context(), accessToken, userID, id, input)
	if err != nil {
		response.BackendError(c, http.StatusBadGateway, err.Error())
		return
	}
	response.BackendJSON(c, http.StatusOK, item)
}

// Delete 删除记录（统一响应格式）。
// DELETE /api/v1/transactions/manage/:id
func (h *handler) Delete(c *gin.Context) {
	accessToken, userID, ok := h.authContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}

	id, err := parseTransactionID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "无效的 id")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), accessToken, userID, id); err != nil {
		response.Error(c, http.StatusBadGateway, response.CodeInternalError, err.Error())
		return
	}
	response.SuccessWithMessage(c, "删除成功", gin.H{})
}

// Search 全文搜索当前会话用户的交易（Elasticsearch）。
// GET /api/v1/transactions/search?q=&limit=
func (h *handler) Search(c *gin.Context) {
	_, userID, ok := h.authContext(c)
	if !ok {
		response.BackendError(c, http.StatusUnauthorized, "未授权")
		return
	}
	q := c.Query("q")
	limit := 20
	if n, err := strconv.Atoi(c.Query("limit")); err == nil {
		limit = n
	}
	items, err := h.svc.Search(c.Request.Context(), userID, q, limit)
	if err != nil {
		if errors.Is(err, shared.ErrSearchUnavailable) {
			response.BackendError(c, http.StatusServiceUnavailable, "搜索暂不可用")
			return
		}
		response.BackendError(c, http.StatusBadGateway, err.Error())
		return
	}
	if items == nil {
		items = []entity.Transaction{}
	}
	response.BackendJSON(c, http.StatusOK, gin.H{"items": items})
}

// DeleteLegacy Flutter 兼容删除（204）。
func (h *handler) DeleteLegacy(c *gin.Context) {
	accessToken, userID, ok := h.authContext(c)
	if !ok {
		response.BackendError(c, http.StatusUnauthorized, "未授权")
		return
	}

	id, err := parseTransactionID(c)
	if err != nil {
		response.BackendError(c, http.StatusBadRequest, "无效的 id")
		return
	}

	if err := h.svc.Delete(c.Request.Context(), accessToken, userID, id); err != nil {
		response.BackendError(c, http.StatusBadGateway, err.Error())
		return
	}
	response.BackendNoContent(c)
}

func parseTransactionID(c *gin.Context) (int64, error) {
	return strconv.ParseInt(c.Param("id"), 10, 64)
}
