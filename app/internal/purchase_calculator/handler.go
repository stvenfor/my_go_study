// handler.go 购车计算器公开 HTTP 适配层。
//
// 由 app/internal/delivery/http/controller/purchase_calculator_controller.go 迁入。
// 类型改小写 handler；本域不取身份，故不引 authctx。
package purchase_calculator

import (
	"errors"
	"net/http"

	"app/internal/delivery/http/dto/response"
	"github.com/gin-gonic/gin"
)

// handler 购车计算器公开 HTTP。
type handler struct {
	svc *Service
}

func newHandler(svc *Service) *handler {
	return &handler{svc: svc}
}

func (h *handler) ListProducts(c *gin.Context) {
	list, err := h.svc.ListProducts(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "查询失败")
		return
	}
	response.Success(c, gin.H{"items": list})
}

func (h *handler) Quote(c *gin.Context) {
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
	dto, err := h.svc.Quote(c.Request.Context(), QuoteInput{
		Mode:              body.Mode,
		BarePrice:         body.BarePrice,
		TaxablePrice:      body.TaxablePrice,
		ProductID:         body.ProductID,
		DownPayment:       body.DownPayment,
		TermMonths:        body.TermMonths,
		DisableCommercial: body.DisableCommercial,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, dto)
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidMode),
		errors.Is(err, ErrInvalidPrice),
		errors.Is(err, ErrProductRequired),
		errors.Is(err, ErrTermNotAllowed),
		errors.Is(err, ErrDownPaymentTooLow):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
	case errors.Is(err, ErrProductNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "报价失败")
	}
}
