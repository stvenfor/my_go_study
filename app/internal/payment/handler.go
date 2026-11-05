// handler.go 微信 / 支付宝预支付 HTTP 适配层。
//
// 由 app/internal/delivery/http/controller/payment_controller.go 迁入。
// 类型改小写 handler；取身份改走 authctx.SupabaseAuthContext。
package payment

import (
	"errors"
	"io"
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
	RefType   string `json:"ref_type" binding:"required"`
	RefID     string `json:"ref_id" binding:"required"`
}

// Prepay POST /api/v1/payments/prepay
func (h *handler) Prepay(c *gin.Context) {
	if h.svc == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "支付服务未启用")
		return
	}
	user, _, ok := authctx.SupabaseAuthContext(c)
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
		UserID:    user.ID,
		RefType:   req.RefType,
		RefID:     req.RefID,
	})
	if err != nil {
		h.mapError(c, err)
		return
	}
	response.Success(c, result)
}

// WechatNotify POST /api/v1/payments/notify/wechat —— 公网回调，无 SessionAuth，靠验签。
// 必须回微信要求的 XML：处理失败回 FAIL 让渠道重投，已落定回 SUCCESS。
func (h *handler) WechatNotify(c *gin.Context) {
	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, 1<<20))
	if err != nil {
		wechatNotifyReply(c, false)
		return
	}
	res, err := h.svc.HandleWechatNotify(c.Request.Context(), raw)
	wechatNotifyReply(c, acceptForReply(res, err))
}

// AlipayNotify POST /api/v1/payments/notify/alipay —— 同理，正文只认 success / failure。
func (h *handler) AlipayNotify(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		alipayNotifyReply(c, false)
		return
	}
	res, err := h.svc.HandleAlipayNotify(c.Request.Context(), c.Request.PostForm)
	alipayNotifyReply(c, acceptForReply(res, err))
}

// acceptForReply 决定要不要让渠道重投：
// 验签/商户/金额/无流水/核销未成 → 让它重投（本地没做成事实）；
// 已核销、重复投递、渠道明确未付款 → 落定，不再重投。
func acceptForReply(res *NotifyResult, err error) bool {
	if err == nil {
		return true
	}
	switch {
	case errors.Is(err, ErrRecordNotFound),
		errors.Is(err, ErrAmountMismatch),
		errors.Is(err, ErrSettleNotCommitted):
		return false
	default:
		return false
	}
}

func wechatNotifyReply(c *gin.Context, ok bool) {
	code := "FAIL"
	if ok {
		code = "SUCCESS"
	}
	c.XML(http.StatusOK, gin.H{
		"return_code": code,
		"return_msg":  "OK",
	})
}

func alipayNotifyReply(c *gin.Context, ok bool) {
	body := "failure"
	if ok {
		body = "success"
	}
	c.String(http.StatusOK, body)
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
