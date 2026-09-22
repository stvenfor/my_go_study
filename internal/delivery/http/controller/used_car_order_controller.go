package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

type UsedCarOrderController struct {
	uc *usecase.UsedCarOrderUsecase
}

func NewUsedCarOrderController(uc *usecase.UsedCarOrderUsecase) *UsedCarOrderController {
	return &UsedCarOrderController{uc: uc}
}

func (ctrl *UsedCarOrderController) Summary(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	dto, err := ctrl.uc.Summary(c.Request.Context(), user.ID)
	if err != nil {
		writeUsedCarOrderError(c, err)
		return
	}
	response.Success(c, dto)
}

func (ctrl *UsedCarOrderController) List(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	pq := response.ParsePageQuery(c, 10)
	list, total, err := ctrl.uc.List(
		c.Request.Context(),
		user.ID,
		strings.TrimSpace(c.Query("status")),
		strings.TrimSpace(c.Query("kind")),
		pq.Page, pq.Size,
	)
	if err != nil {
		writeUsedCarOrderError(c, err)
		return
	}
	response.SuccessList(c, list, pq.Page, pq.Size, total)
}

func (ctrl *UsedCarOrderController) Get(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	id, err := usecase.ParseUsedCarOrderID(c.Param("order_id"))
	if err != nil {
		writeUsedCarOrderError(c, err)
		return
	}
	dto, err := ctrl.uc.Get(c.Request.Context(), user.ID, id)
	if err != nil {
		writeUsedCarOrderError(c, err)
		return
	}
	response.Success(c, dto)
}

func (ctrl *UsedCarOrderController) Create(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	var body struct {
		Kind         string  `json:"kind"`
		CustomerID   int64   `json:"customer_id"`
		VehicleModel string  `json:"vehicle_model"`
		PlateNo      string  `json:"plate_no"`
		VIN          string  `json:"vin"`
		MileageKm    int     `json:"mileage_km"`
		ModelYear    int     `json:"model_year"`
		Amount       float64 `json:"amount"`
		ImageURL     *string `json:"image_url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.CustomerID <= 0 {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	dto, err := ctrl.uc.Create(c.Request.Context(), user.ID, usecase.CreateUsedCarOrderInput{
		Kind: body.Kind, CustomerID: body.CustomerID,
		VehicleModel: body.VehicleModel, PlateNo: body.PlateNo, VIN: body.VIN,
		MileageKm: body.MileageKm, ModelYear: body.ModelYear, Amount: body.Amount,
		ImageURL: body.ImageURL,
	})
	if err != nil {
		writeUsedCarOrderError(c, err)
		return
	}
	response.SuccessCreated(c, dto)
}

func (ctrl *UsedCarOrderController) ListCustomers(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	pq := response.ParsePageQuery(c, 20)
	list, total, err := ctrl.uc.ListCustomers(
		c.Request.Context(),
		user.ID,
		strings.TrimSpace(c.Query("q")),
		pq.Page, pq.Size,
	)
	if err != nil {
		writeUsedCarOrderError(c, err)
		return
	}
	response.SuccessList(c, list, pq.Page, pq.Size, total)
}

func writeUsedCarOrderError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrUsedCarOrderNoStore):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "未选择当前门店")
	case errors.Is(err, usecase.ErrUsedCarOrderNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "二手车业务单不存在")
	case errors.Is(err, usecase.ErrUsedCarOrderBadCustomer):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "客户无效")
	case errors.Is(err, usecase.ErrUsedCarOrderBadFilter):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "筛选参数无效")
	case errors.Is(err, usecase.ErrUsedCarOrderBadInput):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "业务单参数无效")
	default:
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "服务异常")
	}
}
