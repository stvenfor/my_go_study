package membership

import (
	"errors"
	"net/http"
	"strings"

	"app/internal/delivery/http/authctx"
	"app/internal/delivery/http/dto/response"
	"app/internal/payment"
	"app/internal/wallet"
	"github.com/gin-gonic/gin"
)

// handler 会员订阅 HTTP。
type handler struct {
	svc *Service
}

func newHandler(svc *Service) *handler {
	return &handler{svc: svc}
}

// Me GET /api/v1/membership/me
func (h *handler) Me(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录")
		return
	}
	me, err := h.svc.Me(c.Request.Context(), user.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, me)
}

// Buyout POST /api/v1/membership/buyout
func (h *handler) Buyout(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录")
		return
	}
	var body struct {
		PlanID  string `json:"plan_id" binding:"required"`
		Channel string `json:"channel" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	out, err := h.svc.Buyout(c.Request.Context(), user.ID, body.PlanID, body.Channel)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, out)
}

// ConfirmBuyout POST /api/v1/membership/buyout/confirm
func (h *handler) ConfirmBuyout(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录")
		return
	}
	var body struct {
		OrderID int64 `json:"order_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	out, err := h.svc.ConfirmBuyout(c.Request.Context(), user.ID, body.OrderID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, out)
}

// VerifyHuawei POST /api/v1/membership/huawei/verify
func (h *handler) VerifyHuawei(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录")
		return
	}
	var body struct {
		ProductID        string `json:"product_id" binding:"required"`
		PurchaseToken    string `json:"purchase_token" binding:"required"`
		PurchaseOrderID  string `json:"purchase_order_id"`
		SubscriptionID   string `json:"subscription_id"`
		JWSPurchaseOrder string `json:"jws_purchase_order"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	ent, err := h.svc.VerifyHuawei(c.Request.Context(), user.ID, HuaweiVerifyInput{
		ProductID:        body.ProductID,
		PurchaseToken:    body.PurchaseToken,
		PurchaseOrderID:  body.PurchaseOrderID,
		SubscriptionID:   body.SubscriptionID,
		JWSPurchaseOrder: body.JWSPurchaseOrder,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, gin.H{
		"entitlement":     ent,
		"finish_purchase": true,
	})
}

// VerifyApple POST /api/v1/membership/apple/verify
func (h *handler) VerifyApple(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录")
		return
	}
	var body struct {
		ProductID             string `json:"product_id" binding:"required"`
		TransactionID         string `json:"transaction_id" binding:"required"`
		OriginalTransactionID string `json:"original_transaction_id"`
		ReceiptData           string `json:"receipt_data"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	ent, err := h.svc.VerifyApple(c.Request.Context(), user.ID, AppleVerifyInput{
		ProductID:             body.ProductID,
		TransactionID:         body.TransactionID,
		OriginalTransactionID: body.OriginalTransactionID,
		ReceiptData:           body.ReceiptData,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, gin.H{
		"entitlement":       ent,
		"complete_purchase": true,
	})
}

// HuaweiNotifications POST /api/v1/membership/huawei/notifications
//
// 华为 IAP 服务端回调尚未接入（无验签、无流水落库），因此不能再返回
// accepted:true —— 那会让华为侧认为投递成功而不再重试，静默丢单。
// 显式 501 保留「未接入」语义；接入见 parity-p0-2（登记 docs/placeholder-inventory.md）。
func (h *handler) HuaweiNotifications(c *gin.Context) {
	response.Error(c, http.StatusNotImplemented, response.CodeNotImplemented,
		"华为 IAP 回调未接入：当前不会处理该通知，请勿按成功记账")
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrPlanInvalid),
		errors.Is(err, ErrChannelInvalid),
		errors.Is(err, ErrTierMismatch),
		errors.Is(err, ErrHuaweiMismatch),
		errors.Is(err, ErrHuaweiInvalid),
		errors.Is(err, ErrAppleMismatch),
		errors.Is(err, ErrAppleInvalid),
		errors.Is(err, payment.ErrInvalidParams):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
	case errors.Is(err, wallet.ErrCashInsufficient):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "余额不足")
	case errors.Is(err, ErrOrderNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "订单不存在")
	case errors.Is(err, ErrOrderOwner):
		response.Error(c, http.StatusForbidden, response.CodeForbidden, "无权操作该订单")
	case errors.Is(err, payment.ErrPayNotConfigured),
		errors.Is(err, ErrHuaweiNotConfig),
		errors.Is(err, ErrAppleNotConfig):
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, err.Error())
	case errors.Is(err, payment.ErrPayChannelInvalid),
		errors.Is(err, payment.ErrPayFailed):
		response.Error(c, http.StatusBadGateway, response.CodeInternalError, err.Error())
	default:
		msg := err.Error()
		if strings.Contains(msg, "支付未配置") ||
			strings.Contains(msg, "华为 IAP 未配置") ||
			strings.Contains(msg, "苹果 IAP 未配置") {
			response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, msg)
			return
		}
		if strings.Contains(msg, "预支付失败") ||
			strings.Contains(msg, "华为内购凭证无效") ||
			strings.Contains(msg, "苹果内购凭证无效") {
			response.Error(c, http.StatusBadGateway, response.CodeInternalError, msg)
			return
		}
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "服务器内部错误")
	}
}
