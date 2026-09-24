package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

// PaymentController 微信 / 支付宝预支付。
type PaymentController struct {
	uc *usecase.PaymentUsecase
}

// NewPaymentController 创建支付控制器。
func NewPaymentController(uc *usecase.PaymentUsecase) *PaymentController {
	return &PaymentController{uc: uc}
}

type prepayRequest struct {
	Channel   string `json:"channel" binding:"required"`
	AmountFen int64  `json:"amount_fen" binding:"required"`
	Subject   string `json:"subject" binding:"required"`
}

// Prepay POST /api/v1/payments/prepay
func (ctrl *PaymentController) Prepay(c *gin.Context) {
	if ctrl.uc == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "支付服务未启用")
		return
	}
	_, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录")
		return
	}

	var req prepayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}

	result, err := ctrl.uc.Prepay(c.Request.Context(), usecase.PrepayInput{
		Channel:   req.Channel,
		AmountFen: req.AmountFen,
		Subject:   req.Subject,
		ClientIP:  c.ClientIP(),
	})
	if err != nil {
		ctrl.mapError(c, err)
		return
	}
	response.Success(c, result)
}

func (ctrl *PaymentController) mapError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidParams):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
	case errors.Is(err, usecase.ErrPayChannelInvalid):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
	case errors.Is(err, usecase.ErrPayNotConfigured):
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, err.Error())
	case errors.Is(err, usecase.ErrPayFailed):
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
