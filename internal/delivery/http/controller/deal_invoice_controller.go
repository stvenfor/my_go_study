package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

// DealInvoiceController 新车成交发票 HTTP。
type DealInvoiceController struct {
	uc *usecase.DealInvoiceUsecase
}

func NewDealInvoiceController(uc *usecase.DealInvoiceUsecase) *DealInvoiceController {
	return &DealInvoiceController{uc: uc}
}

func (ctrl *DealInvoiceController) Summary(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	dto, err := ctrl.uc.Summary(c.Request.Context(), user.ID)
	if err != nil {
		writeDealInvoiceError(c, err)
		return
	}
	response.Success(c, dto)
}

func (ctrl *DealInvoiceController) List(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	pq := response.ParsePageQuery(c, 10)
	list, total, err := ctrl.uc.List(
		c.Request.Context(),
		user.ID,
		strings.TrimSpace(c.Query("status")),
		pq.Page, pq.Size,
	)
	if err != nil {
		writeDealInvoiceError(c, err)
		return
	}
	response.SuccessList(c, list, pq.Page, pq.Size, total)
}

func (ctrl *DealInvoiceController) Get(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	id, err := usecase.ParseDealInvoiceID(c.Param("invoice_id"))
	if err != nil {
		writeDealInvoiceError(c, err)
		return
	}
	dto, err := ctrl.uc.Get(c.Request.Context(), user.ID, id)
	if err != nil {
		writeDealInvoiceError(c, err)
		return
	}
	response.Success(c, dto)
}

func (ctrl *DealInvoiceController) Create(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
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
	dto, err := ctrl.uc.Create(c.Request.Context(), user.ID, usecase.CreateDealInvoiceInput{
		CustomerID: body.CustomerID,
		ImageURL:   body.ImageURL,
	})
	if err != nil {
		writeDealInvoiceError(c, err)
		return
	}
	response.SuccessCreated(c, dto)
}

func (ctrl *DealInvoiceController) Resubmit(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	id, err := usecase.ParseDealInvoiceID(c.Param("invoice_id"))
	if err != nil {
		writeDealInvoiceError(c, err)
		return
	}
	var body struct {
		ImageURL *string `json:"image_url"`
	}
	_ = c.ShouldBindJSON(&body)
	dto, err := ctrl.uc.Resubmit(c.Request.Context(), user.ID, id, usecase.ResubmitDealInvoiceInput{
		ImageURL: body.ImageURL,
	})
	if err != nil {
		writeDealInvoiceError(c, err)
		return
	}
	response.Success(c, dto)
}

func (ctrl *DealInvoiceController) ListCustomers(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	pq := response.ParsePageQuery(c, 20)
	list, total, err := ctrl.uc.ListCustomers(
		c.Request.Context(),
		user.ID,
		strings.TrimSpace(c.Query("q")),
		pq.Page, pq.Size,
	)
	if err != nil {
		writeDealInvoiceError(c, err)
		return
	}
	response.SuccessList(c, list, pq.Page, pq.Size, total)
}

func writeDealInvoiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrDealInvoiceNoStore):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "未选择当前门店")
	case errors.Is(err, usecase.ErrDealInvoiceNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "成交发票不存在")
	case errors.Is(err, usecase.ErrDealInvoiceBadCustomer):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "购车客户无效")
	case errors.Is(err, usecase.ErrDealInvoiceBadFilter):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "status 参数无效")
	case errors.Is(err, usecase.ErrDealInvoiceInvalidStatus):
		response.Error(c, http.StatusConflict, response.CodeInvalidParams, "当前状态不可重提")
	default:
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "服务异常")
	}
}
