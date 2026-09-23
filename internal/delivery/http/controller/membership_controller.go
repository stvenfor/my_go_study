package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

// MembershipController 会员订阅 HTTP。
type MembershipController struct {
	uc *usecase.MembershipUsecase
}

func NewMembershipController(uc *usecase.MembershipUsecase) *MembershipController {
	return &MembershipController{uc: uc}
}

// Me GET /api/v1/membership/me
func (ctrl *MembershipController) Me(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录")
		return
	}
	me, err := ctrl.uc.Me(c.Request.Context(), user.ID)
	if err != nil {
		writeMembershipError(c, err)
		return
	}
	response.Success(c, me)
}

// Buyout POST /api/v1/membership/buyout
func (ctrl *MembershipController) Buyout(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
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
	out, err := ctrl.uc.Buyout(c.Request.Context(), user.ID, body.PlanID, body.Channel)
	if err != nil {
		writeMembershipError(c, err)
		return
	}
	response.Success(c, out)
}

// ConfirmBuyout POST /api/v1/membership/buyout/confirm
func (ctrl *MembershipController) ConfirmBuyout(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
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
	out, err := ctrl.uc.ConfirmBuyout(c.Request.Context(), user.ID, body.OrderID)
	if err != nil {
		writeMembershipError(c, err)
		return
	}
	response.Success(c, out)
}

// VerifyHuawei POST /api/v1/membership/huawei/verify
func (ctrl *MembershipController) VerifyHuawei(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
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
	ent, err := ctrl.uc.VerifyHuawei(c.Request.Context(), user.ID, usecase.HuaweiVerifyInput{
		ProductID:        body.ProductID,
		PurchaseToken:    body.PurchaseToken,
		PurchaseOrderID:  body.PurchaseOrderID,
		SubscriptionID:   body.SubscriptionID,
		JWSPurchaseOrder: body.JWSPurchaseOrder,
	})
	if err != nil {
		writeMembershipError(c, err)
		return
	}
	response.Success(c, gin.H{
		"entitlement":     ent,
		"finish_purchase": true,
	})
}

// VerifyApple POST /api/v1/membership/apple/verify
func (ctrl *MembershipController) VerifyApple(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
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
	ent, err := ctrl.uc.VerifyApple(c.Request.Context(), user.ID, usecase.AppleVerifyInput{
		ProductID:             body.ProductID,
		TransactionID:         body.TransactionID,
		OriginalTransactionID: body.OriginalTransactionID,
		ReceiptData:           body.ReceiptData,
	})
	if err != nil {
		writeMembershipError(c, err)
		return
	}
	response.Success(c, gin.H{
		"entitlement":       ent,
		"complete_purchase": true,
	})
}

// HuaweiNotifications POST /api/v1/membership/huawei/notifications （stub）
func (ctrl *MembershipController) HuaweiNotifications(c *gin.Context) {
	response.Success(c, gin.H{"accepted": true})
}

func writeMembershipError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrMembershipPlanInvalid),
		errors.Is(err, usecase.ErrMembershipChannelInvalid),
		errors.Is(err, usecase.ErrMembershipTierMismatch),
		errors.Is(err, usecase.ErrMembershipHuaweiMismatch),
		errors.Is(err, usecase.ErrMembershipHuaweiInvalid),
		errors.Is(err, usecase.ErrMembershipAppleMismatch),
		errors.Is(err, usecase.ErrMembershipAppleInvalid),
		errors.Is(err, usecase.ErrInvalidParams):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
	case errors.Is(err, usecase.ErrCashInsufficient):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "余额不足")
	case errors.Is(err, repository.ErrMembershipOrderNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "订单不存在")
	case errors.Is(err, repository.ErrMembershipOrderOwner):
		response.Error(c, http.StatusForbidden, response.CodeForbidden, "无权操作该订单")
	case errors.Is(err, usecase.ErrPayNotConfigured),
		errors.Is(err, usecase.ErrMembershipHuaweiNotConfig),
		errors.Is(err, usecase.ErrMembershipAppleNotConfig):
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, err.Error())
	case errors.Is(err, usecase.ErrPayChannelInvalid),
		errors.Is(err, usecase.ErrPayFailed):
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
