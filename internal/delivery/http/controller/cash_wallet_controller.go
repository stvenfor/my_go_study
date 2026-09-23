package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

// CashWalletController 人民币钱包。
type CashWalletController struct {
	walletUC *usecase.CashWalletUsecase
}

func NewCashWalletController(walletUC *usecase.CashWalletUsecase) *CashWalletController {
	return &CashWalletController{walletUC: walletUC}
}

// GetSummary GET /api/v1/wallet
func (ctrl *CashWalletController) GetSummary(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	sum, err := ctrl.walletUC.GetSummary(c.Request.Context(), user.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "查询失败")
		return
	}
	response.Success(c, sum)
}

// ListLedger GET /api/v1/wallet/ledger
func (ctrl *CashWalletController) ListLedger(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	page, err := ctrl.walletUC.ListLedger(c.Request.Context(), user.ID, limit, offset)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "查询失败")
		return
	}
	response.Success(c, page)
}

// ListCards GET /api/v1/wallet/cards
func (ctrl *CashWalletController) ListCards(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	cards, err := ctrl.walletUC.ListCards(c.Request.Context(), user.ID)
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
func (ctrl *CashWalletController) BindCard(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	var body bindCardBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	card, err := ctrl.walletUC.BindCard(c.Request.Context(), user.ID, usecase.BindCardInput{
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
func (ctrl *CashWalletController) SetDefaultCard(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	cardID, err := strconv.ParseInt(c.Param("card_id"), 10, 64)
	if err != nil || cardID <= 0 {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	if err := ctrl.walletUC.SetDefaultCard(c.Request.Context(), user.ID, cardID); err != nil {
		writeCashWalletError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// DeleteCard DELETE /api/v1/wallet/cards/:card_id
func (ctrl *CashWalletController) DeleteCard(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	cardID, err := strconv.ParseInt(c.Param("card_id"), 10, 64)
	if err != nil || cardID <= 0 {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	if err := ctrl.walletUC.DeleteCard(c.Request.Context(), user.ID, cardID); err != nil {
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
func (ctrl *CashWalletController) Recharge(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	var body rechargeBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	res, err := ctrl.walletUC.Recharge(c.Request.Context(), user.ID, usecase.RechargeInput{
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
	case errors.Is(err, usecase.ErrCashInvalidAmount), errors.Is(err, repository.ErrCashInvalidAmount):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "金额不合法")
	case errors.Is(err, usecase.ErrCashInvalidChannel):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "充值渠道不合法")
	case errors.Is(err, usecase.ErrCashCardRequired):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "绑卡充值须指定银行卡")
	case errors.Is(err, usecase.ErrCashInvalidCardLast4):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "卡号后四位不合法")
	case errors.Is(err, usecase.ErrCashInvalidBankName):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "银行名称不合法")
	case errors.Is(err, usecase.ErrCashCardNotFound), errors.Is(err, repository.ErrCashCardNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "银行卡不存在")
	case errors.Is(err, usecase.ErrCashInsufficient), errors.Is(err, repository.ErrCashInsufficient):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "余额不足")
	default:
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "操作失败")
	}
}
