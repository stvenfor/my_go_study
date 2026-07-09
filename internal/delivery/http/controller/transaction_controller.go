// =============================================================================
// 文件：transaction_controller.go
// 层级：Delivery/HTTP —— transactions REST API
//
// 两套风格：
//   listLegacy  → GET /api/v1/transactions        → { items: [] }
//   ListPage    → GET /api/v1/transactions/manage → ResultModel 分页
// =============================================================================
package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/request"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

// TransactionController Transaction 管理控制器。
type TransactionController struct {
	transactionUC *usecase.TransactionUsecase
}

// NewTransactionController 创建 Transaction 控制器。
func NewTransactionController(transactionUC *usecase.TransactionUsecase) *TransactionController {
	return &TransactionController{transactionUC: transactionUC}
}

// authContext 从 SupabaseAuth 中间件注入的 Context 取 token 与 userID。
func (ctrl *TransactionController) authContext(c *gin.Context) (accessToken, userID string, ok bool) {
	user, token, ok := supabaseAuthContext(c)
	if !ok {
		return "", "", false
	}
	return token, user.ID, true
}

// List 若带 page 参数走 manage 分页，否则走 Flutter legacy limit/offset。
func (ctrl *TransactionController) List(c *gin.Context) {
	if c.Query("page") != "" {
		ctrl.ListPage(c)
		return
	}
	ctrl.listLegacy(c)
}

// ListPage 统一分页列表（page/size，返回 list + pagination）。
// GET /api/v1/transactions/manage
func (ctrl *TransactionController) ListPage(c *gin.Context) {
	accessToken, userID, ok := ctrl.authContext(c)
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

	items, total, err := ctrl.transactionUC.ListPage(c.Request.Context(), accessToken, userID, filter)
	if err != nil {
		response.Error(c, http.StatusBadGateway, response.CodeInternalError, err.Error())
		return
	}
	response.SuccessList(c, response.FromTransactions(items), page.Page, page.Size, total)
}

// listLegacy Flutter 二手车列表实际使用的接口。
func (ctrl *TransactionController) listLegacy(c *gin.Context) {
	accessToken, userID, ok := ctrl.authContext(c)
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

	items, err := ctrl.transactionUC.List(c.Request.Context(), accessToken, userID, filter)
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
func (ctrl *TransactionController) Get(c *gin.Context) {
	accessToken, userID, ok := ctrl.authContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}

	id, err := parseTransactionID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "无效的 id")
		return
	}

	item, err := ctrl.transactionUC.Get(c.Request.Context(), accessToken, userID, id)
	if err != nil {
		response.Error(c, http.StatusNotFound, response.CodeNotFound, err.Error())
		return
	}
	response.Success(c, response.FromTransaction(item))
}

// GetLegacy Flutter 兼容单条查询。
func (ctrl *TransactionController) GetLegacy(c *gin.Context) {
	accessToken, userID, ok := ctrl.authContext(c)
	if !ok {
		response.BackendError(c, http.StatusUnauthorized, "未授权")
		return
	}

	id, err := parseTransactionID(c)
	if err != nil {
		response.BackendError(c, http.StatusBadRequest, "无效的 id")
		return
	}

	item, err := ctrl.transactionUC.Get(c.Request.Context(), accessToken, userID, id)
	if err != nil {
		response.BackendError(c, http.StatusNotFound, err.Error())
		return
	}
	response.BackendJSON(c, http.StatusOK, item)
}

// Create 创建记录（统一响应格式）。
// POST /api/v1/transactions/manage
func (ctrl *TransactionController) Create(c *gin.Context) {
	accessToken, userID, ok := ctrl.authContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}

	var req request.CreateTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}

	item, err := ctrl.transactionUC.Create(c.Request.Context(), accessToken, userID, entity.CreateTransactionInput{
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
func (ctrl *TransactionController) CreateLegacy(c *gin.Context) {
	accessToken, userID, ok := ctrl.authContext(c)
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

	item, err := ctrl.transactionUC.Create(c.Request.Context(), accessToken, userID, input)
	if err != nil {
		response.BackendError(c, http.StatusBadGateway, err.Error())
		return
	}
	response.BackendJSON(c, http.StatusCreated, item)
}

// Update 更新记录（统一响应格式）。
// PUT /api/v1/transactions/manage/:id
func (ctrl *TransactionController) Update(c *gin.Context) {
	accessToken, userID, ok := ctrl.authContext(c)
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

	item, err := ctrl.transactionUC.Update(c.Request.Context(), accessToken, userID, id, entity.UpdateTransactionInput{
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
func (ctrl *TransactionController) UpdateLegacy(c *gin.Context) {
	accessToken, userID, ok := ctrl.authContext(c)
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

	item, err := ctrl.transactionUC.Update(c.Request.Context(), accessToken, userID, id, input)
	if err != nil {
		response.BackendError(c, http.StatusBadGateway, err.Error())
		return
	}
	response.BackendJSON(c, http.StatusOK, item)
}

// Delete 删除记录（统一响应格式）。
// DELETE /api/v1/transactions/manage/:id
func (ctrl *TransactionController) Delete(c *gin.Context) {
	accessToken, userID, ok := ctrl.authContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}

	id, err := parseTransactionID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "无效的 id")
		return
	}

	if err := ctrl.transactionUC.Delete(c.Request.Context(), accessToken, userID, id); err != nil {
		response.Error(c, http.StatusBadGateway, response.CodeInternalError, err.Error())
		return
	}
	response.SuccessWithMessage(c, "删除成功", gin.H{})
}

// DeleteLegacy Flutter 兼容删除（204）。
func (ctrl *TransactionController) DeleteLegacy(c *gin.Context) {
	accessToken, userID, ok := ctrl.authContext(c)
	if !ok {
		response.BackendError(c, http.StatusUnauthorized, "未授权")
		return
	}

	id, err := parseTransactionID(c)
	if err != nil {
		response.BackendError(c, http.StatusBadRequest, "无效的 id")
		return
	}

	if err := ctrl.transactionUC.Delete(c.Request.Context(), accessToken, userID, id); err != nil {
		response.BackendError(c, http.StatusBadGateway, err.Error())
		return
	}
	response.BackendNoContent(c)
}

func parseTransactionID(c *gin.Context) (int64, error) {
	return strconv.ParseInt(c.Param("id"), 10, 64)
}
