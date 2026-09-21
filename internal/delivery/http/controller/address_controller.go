// address_controller.go 用户收货地址 HTTP。
package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

// AddressController 地址簿。
type AddressController struct {
	uc *usecase.AddressUsecase
}

// NewAddressController 创建。
func NewAddressController(uc *usecase.AddressUsecase) *AddressController {
	return &AddressController{uc: uc}
}

type addressBody struct {
	ReceiverName  string `json:"receiver_name"`
	ReceiverPhone string `json:"receiver_phone"`
	Province      string `json:"province"`
	City          string `json:"city"`
	District      string `json:"district"`
	DetailAddress string `json:"detail_address"`
	PostalCode    string `json:"postal_code"`
	IsDefault     bool   `json:"is_default"`
	Label         string `json:"label"`
}

func (b addressBody) toInput() usecase.AddressInput {
	return usecase.AddressInput{
		ReceiverName:  b.ReceiverName,
		ReceiverPhone: b.ReceiverPhone,
		Province:      b.Province,
		City:          b.City,
		District:      b.District,
		DetailAddress: b.DetailAddress,
		PostalCode:    b.PostalCode,
		IsDefault:     b.IsDefault,
		Label:         b.Label,
	}
}

// List GET /api/v1/user/addresses
func (ctrl *AddressController) List(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	list, err := ctrl.uc.List(c.Request.Context(), user.ID)
	if err != nil {
		writeAddressError(c, err)
		return
	}
	response.Success(c, gin.H{"items": list})
}

// Get GET /api/v1/user/addresses/:address_id
func (ctrl *AddressController) Get(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	id, err := parseAddressID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "address_id 无效")
		return
	}
	v, err := ctrl.uc.Get(c.Request.Context(), user.ID, id)
	if err != nil {
		writeAddressError(c, err)
		return
	}
	response.Success(c, v)
}

// Create POST /api/v1/user/addresses
func (ctrl *AddressController) Create(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	var body addressBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	v, err := ctrl.uc.Create(c.Request.Context(), user.ID, body.toInput())
	if err != nil {
		writeAddressError(c, err)
		return
	}
	response.SuccessCreated(c, v)
}

// Update PATCH /api/v1/user/addresses/:address_id
func (ctrl *AddressController) Update(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	id, err := parseAddressID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "address_id 无效")
		return
	}
	var body addressBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	v, err := ctrl.uc.Update(c.Request.Context(), user.ID, id, body.toInput())
	if err != nil {
		writeAddressError(c, err)
		return
	}
	response.Success(c, v)
}

// SetDefault POST /api/v1/user/addresses/:address_id/default
func (ctrl *AddressController) SetDefault(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	id, err := parseAddressID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "address_id 无效")
		return
	}
	v, err := ctrl.uc.SetDefault(c.Request.Context(), user.ID, id)
	if err != nil {
		writeAddressError(c, err)
		return
	}
	response.Success(c, v)
}

// Delete DELETE /api/v1/user/addresses/:address_id
func (ctrl *AddressController) Delete(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	id, err := parseAddressID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "address_id 无效")
		return
	}
	if err := ctrl.uc.Delete(c.Request.Context(), user.ID, id); err != nil {
		writeAddressError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func parseAddressID(c *gin.Context) (int64, error) {
	id, err := strconv.ParseInt(c.Param("address_id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, err
	}
	return id, nil
}

func writeAddressError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrAddressNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, err.Error())
	case errors.Is(err, usecase.ErrAddressInvalid), errors.Is(err, usecase.ErrAddressLimit):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
	default:
		response.Error(c, http.StatusBadGateway, response.CodeInternalError, err.Error())
	}
}
