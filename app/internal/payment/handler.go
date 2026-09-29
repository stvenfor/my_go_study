// handler.go 微信 / 支付宝预支付 HTTP 适配层。
//
// 由 app/internal/delivery/http/controller/payment_controller.go 迁入。
// 类型改小写 handler；取身份改走 authctx.SupabaseAuthContext。
package payment

import (
	"errors"
	"net/http"
	"strings"

	"app/internal/delivery/http/authctx"
	"app/internal/delivery/http/dto/response"
	"github.com/gin-gonic/gin"
)

// handler 微信 / 支付宝预支付。
type handler struct {
	svc *Service
}

// newHandler 创建支付 handler。
func newHandler(svc *Service) *handler {
	return &handler{svc: svc}
}

type prepayRequest struct {
	Channel   string `json:"channel" binding:"required"`
	AmountFen int64  `json:"amount_fen" binding:"required"`
	Subject   string `json:"subject" binding:"required"`
}

// Prepay POST /api/v1/payments/prepay
func (h *handler) Prepay(c *gin.Context) {
	if h.svc == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "支付服务未启用")
		return
	}
	_, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录")
		return
	}

	var req prepayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}

	result, err := h.svc.Prepay(c.Request.Context(), PrepayInput{
		Channel:   req.Channel,
		AmountFen: req.AmountFen,
		Subject:   req.Subject,
		ClientIP:  c.ClientIP(),
	})
	if err != nil {
		h.mapError(c, err)
		return
	}
	response.Success(c, result)
}

func (h *handler) mapError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidParams):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
	case errors.Is(err, ErrPayChannelInvalid):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
	case errors.Is(err, ErrPayNotConfigured):
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, err.Error())
	case errors.Is(err, ErrPayFailed):
		response.Error(c, http.StatusBadGateway, response.CodeInternalError, err.Error())
	default:
		msg := err.Error()
		if strings.Contains(msg, "支付未配置") {
			response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, msg)
			return
		}
		if strings.Contains(msg, "预支付失败") {
			response.Error(c, http.StatusBadGateway, response.CodeInternalError, msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "服务器内部错误")
	}
}
