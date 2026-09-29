package wallet

import (
	"errors"
	"net/http"
	"strconv"

	"app/internal/delivery/http/authctx"
	"app/internal/delivery/http/dto/response"
	"github.com/gin-gonic/gin"
)

// handler 人民币钱包。
type handler struct {
	svc *Service
}

func newHandler(svc *Service) *handler {
	return &handler{svc: svc}
}

// GetSummary GET /api/v1/wallet
func (ctrl *handler) GetSummary(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	sum, err := ctrl.svc.GetSummary(c.Request.Context(), user.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "查询失败")
		return
	}
	response.Success(c, sum)
}

// ListLedger GET /api/v1/wallet/ledger
func (ctrl *handler) ListLedger(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	page, err := ctrl.svc.ListLedger(c.Request.Context(), user.ID, limit, offset)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "查询失败")
		return
	}
	response.Success(c, page)
}

// ListCards GET /api/v1/wallet/cards
func (ctrl *handler) ListCards(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	cards, err := ctrl.svc.ListCards(c.Request.Context(), user.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "查询失败")
		return
	}
	response.Success(c, gin.H{"items": cards})
}

type bindCardBody struct {
	BankName   string `json:"bank_name"`
	CardLast4  string `json:"card_last4"`
	HolderName string `json:"holder_name"`
	IsDefault  bool   `json:"is_default"`
}

// BindCard POST /api/v1/wallet/cards
func (ctrl *handler) BindCard(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	var body bindCardBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	card, err := ctrl.svc.BindCard(c.Request.Context(), user.ID, BindCardInput{
		BankName: body.BankName, CardLast4: body.CardLast4,
		HolderName: body.HolderName, IsDefault: body.IsDefault,
	})
	if err != nil {
		writeCashWalletError(c, err)
		return
	}
	response.Success(c, card)
}

// SetDefaultCard POST /api/v1/wallet/cards/:card_id/default
func (ctrl *handler) SetDefaultCard(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	cardID, err := strconv.ParseInt(c.Param("card_id"), 10, 64)
	if err != nil || cardID <= 0 {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	if err := ctrl.svc.SetDefaultCard(c.Request.Context(), user.ID, cardID); err != nil {
		writeCashWalletError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// DeleteCard DELETE /api/v1/wallet/cards/:card_id
func (ctrl *handler) DeleteCard(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	cardID, err := strconv.ParseInt(c.Param("card_id"), 10, 64)
	if err != nil || cardID <= 0 {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	if err := ctrl.svc.DeleteCard(c.Request.Context(), user.ID, cardID); err != nil {
		writeCashWalletError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

type rechargeBody struct {
	Amount  string `json:"amount"`
	Channel int16  `json:"channel"`
	CardID  *int64 `json:"card_id"`
}

// Recharge POST /api/v1/wallet/recharge
func (ctrl *handler) Recharge(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	var body rechargeBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	res, err := ctrl.svc.Recharge(c.Request.Context(), user.ID, RechargeInput{
		Amount: body.Amount, Channel: body.Channel, CardID: body.CardID,
	})
	if err != nil {
		writeCashWalletError(c, err)
		return
	}
	response.Success(c, res)
}

func writeCashWalletError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrCashInvalidAmount), errors.Is(err, ErrCashInvalidAmount):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "金额不合法")
	case errors.Is(err, ErrCashInvalidChannel):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "充值渠道不合法")
	case errors.Is(err, ErrCashCardRequired):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "绑卡充值须指定银行卡")
	case errors.Is(err, ErrCashInvalidCardLast4):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "卡号后四位不合法")
	case errors.Is(err, ErrCashInvalidBankName):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "银行名称不合法")
	case errors.Is(err, ErrCashCardNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "银行卡不存在")
	case errors.Is(err, ErrCashInsufficient):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "余额不足")
	default:
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "操作失败")
	}
}
