package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

// AfterSalesZoneController 售后专区 HTTP。
type AfterSalesZoneController struct {
	uc *usecase.AfterSalesZoneUsecase
}

func NewAfterSalesZoneController(uc *usecase.AfterSalesZoneUsecase) *AfterSalesZoneController {
	return &AfterSalesZoneController{uc: uc}
}

func (ctrl *AfterSalesZoneController) ListRecords(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	pq := response.ParsePageQuery(c, 10)
	list, total, canCreate, err := ctrl.uc.List(c.Request.Context(), user.ID, pq.Page, pq.Size)
	if err != nil {
		writeAfterSalesError(c, err)
		return
	}
	response.Success(c, gin.H{
		"list":       list,
		"pagination": response.NewPagination(pq.Page, pq.Size, total),
		"can_create": canCreate,
	})
}

func (ctrl *AfterSalesZoneController) GetRecord(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	id, err := usecase.ParseAfterSalesRecordID(c.Param("record_id"))
	if err != nil {
		writeAfterSalesError(c, err)
		return
	}
	dto, err := ctrl.uc.Get(c.Request.Context(), user.ID, id)
	if err != nil {
		writeAfterSalesError(c, err)
		return
	}
	response.Success(c, dto)
}

func (ctrl *AfterSalesZoneController) CreateRecord(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	var body struct {
		AppointmentID  *int64  `json:"appointment_id"`
		CustomerID     *int64  `json:"customer_id"`
		CustomerUserID *string `json:"customer_user_id"`
		CustomerName   string  `json:"customer_name"`
		CustomerPhone  string  `json:"customer_phone"`
		PlateNo        string  `json:"plate_no"`
		Mileage        *int    `json:"mileage"`
		ServiceKind    any     `json:"service_kind"`
		Title          string  `json:"title"`
		Content        string  `json:"content"`
		ServiceDate    string  `json:"service_date"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	dto, err := ctrl.uc.Create(c.Request.Context(), user.ID, usecase.CreateAfterSalesRecordInput{
		AppointmentID:  body.AppointmentID,
		CustomerID:     body.CustomerID,
		CustomerUserID: body.CustomerUserID,
		CustomerName:   body.CustomerName,
		CustomerPhone:  body.CustomerPhone,
		PlateNo:        body.PlateNo,
		Mileage:        body.Mileage,
		ServiceKind:    stringifyServiceKind(body.ServiceKind),
		Title:          body.Title,
		Content:        body.Content,
		ServiceDate:    body.ServiceDate,
	})
	if err != nil {
		writeAfterSalesError(c, err)
		return
	}
	response.SuccessCreated(c, dto)
}

func (ctrl *AfterSalesZoneController) ListPendingAppointments(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	list, err := ctrl.uc.ListPendingAppointments(c.Request.Context(), user.ID)
	if err != nil {
		writeAfterSalesError(c, err)
		return
	}
	response.Success(c, gin.H{"list": list})
}

func stringifyServiceKind(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case float64:
		n := int(x)
		if float64(n) != x {
			return ""
		}
		return fmt.Sprintf("%d", n)
	case int:
		return fmt.Sprintf("%d", x)
	default:
		return ""
	}
}

func writeAfterSalesError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrAfterSalesNoStore):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "未选择当前门店")
	case errors.Is(err, usecase.ErrAfterSalesNotMember), errors.Is(err, usecase.ErrAfterSalesForbidden):
		response.Error(c, http.StatusForbidden, response.CodeForbidden, "无权操作")
	case errors.Is(err, usecase.ErrAfterSalesRecordNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "维修保养记录不存在")
	case errors.Is(err, usecase.ErrAfterSalesBadKind):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "服务类型无效")
	case errors.Is(err, usecase.ErrAfterSalesBadCustomer):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "客户信息无效")
	case errors.Is(err, usecase.ErrAfterSalesBadTitle):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "标题不能为空")
	case errors.Is(err, usecase.ErrAfterSalesBadDate):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "服务日期无效")
	case errors.Is(err, usecase.ErrAfterSalesBadAppointment):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "预约无效")
	case errors.Is(err, usecase.ErrAfterSalesAppointmentTaken):
		response.Error(c, http.StatusConflict, response.CodeInvalidParams, "该预约已有记录")
	default:
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "服务器错误")
	}
}
