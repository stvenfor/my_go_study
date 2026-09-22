package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

// PurchaseCalculatorController 购车计算器公开 HTTP。
type PurchaseCalculatorController struct {
	uc *usecase.PurchaseQuoteUsecase
}

func NewPurchaseCalculatorController(uc *usecase.PurchaseQuoteUsecase) *PurchaseCalculatorController {
	return &PurchaseCalculatorController{uc: uc}
}

func (ctrl *PurchaseCalculatorController) ListProducts(c *gin.Context) {
	list, err := ctrl.uc.ListProducts(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "查询失败")
		return
	}
	response.Success(c, gin.H{"items": list})
}

func (ctrl *PurchaseCalculatorController) Quote(c *gin.Context) {
	var body struct {
		Mode              string   `json:"mode"`
		BarePrice         float64  `json:"bare_price"`
		TaxablePrice      *float64 `json:"taxable_price"`
		ProductID         int64    `json:"product_id"`
		DownPayment       float64  `json:"down_payment"`
		TermMonths        int      `json:"term_months"`
		DisableCommercial bool     `json:"disable_commercial"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	dto, err := ctrl.uc.Quote(c.Request.Context(), usecase.QuoteInput{
		Mode:              body.Mode,
		BarePrice:         body.BarePrice,
		TaxablePrice:      body.TaxablePrice,
		ProductID:         body.ProductID,
		DownPayment:       body.DownPayment,
		TermMonths:        body.TermMonths,
		DisableCommercial: body.DisableCommercial,
	})
	if err != nil {
		writePurchaseCalcError(c, err)
		return
	}
	response.Success(c, dto)
}

func writePurchaseCalcError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrFinanceInvalidMode),
		errors.Is(err, usecase.ErrFinanceInvalidPrice),
		errors.Is(err, usecase.ErrFinanceProductRequired),
		errors.Is(err, usecase.ErrFinanceTermNotAllowed),
		errors.Is(err, usecase.ErrFinanceDownPaymentTooLow):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
	case errors.Is(err, usecase.ErrFinanceProductNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "报价失败")
	}
}
