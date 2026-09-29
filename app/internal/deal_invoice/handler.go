// handler.go 新车成交发票 HTTP 适配层。
//
// 由 app/internal/delivery/http/controller/deal_invoice_controller.go 迁入。
// 类型改小写 handler；取身份改走 authctx.SupabaseAuthContext。
package deal_invoice

import (
	"errors"
	"net/http"
	"strings"

	"app/internal/delivery/http/authctx"
	"app/internal/delivery/http/dto/response"
	"github.com/gin-gonic/gin"
)

// handler 新车成交发票 HTTP。
type handler struct {
	svc *Service
}

func newHandler(svc *Service) *handler {
	return &handler{svc: svc}
}

func (h *handler) Summary(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	dto, err := h.svc.Summary(c.Request.Context(), user.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, dto)
}

func (h *handler) List(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	pq := response.ParsePageQuery(c, 10)
	list, total, err := h.svc.List(
		c.Request.Context(),
		user.ID,
		strings.TrimSpace(c.Query("status")),
		pq.Page, pq.Size,
	)
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessList(c, list, pq.Page, pq.Size, total)
}

func (h *handler) Get(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	id, err := parseInvoiceID(c.Param("invoice_id"))
	if err != nil {
		writeError(c, err)
		return
	}
	dto, err := h.svc.Get(c.Request.Context(), user.ID, id)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, dto)
}

func (h *handler) Create(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	var body struct {
		CustomerID int64   `json:"customer_id"`
		ImageURL   *string `json:"image_url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.CustomerID <= 0 {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	dto, err := h.svc.Create(c.Request.Context(), user.ID, CreateInput{
		CustomerID: body.CustomerID,
		ImageURL:   body.ImageURL,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessCreated(c, dto)
}

func (h *handler) Resubmit(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	id, err := parseInvoiceID(c.Param("invoice_id"))
	if err != nil {
		writeError(c, err)
		return
	}
	var body struct {
		ImageURL *string `json:"image_url"`
	}
	_ = c.ShouldBindJSON(&body)
	dto, err := h.svc.Resubmit(c.Request.Context(), user.ID, id, ResubmitInput{
		ImageURL: body.ImageURL,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, dto)
}

func (h *handler) ListCustomers(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	pq := response.ParsePageQuery(c, 20)
	list, total, err := h.svc.ListCustomers(
		c.Request.Context(),
		user.ID,
		strings.TrimSpace(c.Query("q")),
		pq.Page, pq.Size,
	)
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessList(c, list, pq.Page, pq.Size, total)
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNoStore):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "未选择当前门店")
	case errors.Is(err, ErrNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "成交发票不存在")
	case errors.Is(err, ErrBadCustomer):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "购车客户无效")
	case errors.Is(err, ErrBadFilter):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "status 参数无效")
	case errors.Is(err, ErrInvalidStatus):
		response.Error(c, http.StatusConflict, response.CodeInvalidParams, "当前状态不可重提")
	default:
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "服务异常")
	}
}
