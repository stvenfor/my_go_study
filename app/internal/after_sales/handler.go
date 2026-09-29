// handler.go 售后专区 HTTP 适配层。
//
// 由 app/internal/delivery/http/controller/after_sales_zone_controller.go 迁入。
// 类型改小写 handler；取身份改走 authctx.SupabaseAuthContext。
package after_sales

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"app/internal/delivery/http/authctx"
	"app/internal/delivery/http/dto/response"
	"github.com/gin-gonic/gin"
)

// handler 售后专区 HTTP。
type handler struct {
	svc *Service
}

func newHandler(svc *Service) *handler {
	return &handler{svc: svc}
}

func (h *handler) ListRecords(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	pq := response.ParsePageQuery(c, 10)
	list, total, canCreate, err := h.svc.List(c.Request.Context(), user.ID, pq.Page, pq.Size)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, gin.H{
		"list":       list,
		"pagination": response.NewPagination(pq.Page, pq.Size, total),
		"can_create": canCreate,
	})
}

func (h *handler) GetRecord(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	id, err := parseRecordID(c.Param("record_id"))
	if err != nil {
		writeError(c, err)
		return
	}
	dto, err := h.svc.Get(c.Request.Context(), user.ID, id)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, dto)
}

func (h *handler) CreateRecord(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
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
	dto, err := h.svc.Create(c.Request.Context(), user.ID, CreateInput{
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
		writeError(c, err)
		return
	}
	response.SuccessCreated(c, dto)
}

func (h *handler) ListPendingAppointments(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	list, err := h.svc.ListPendingAppointments(c.Request.Context(), user.ID)
	if err != nil {
		writeError(c, err)
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

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNoStore):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "未选择当前门店")
	case errors.Is(err, ErrNotMember), errors.Is(err, ErrForbidden):
		response.Error(c, http.StatusForbidden, response.CodeForbidden, "无权操作")
	case errors.Is(err, ErrRecordNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "维修保养记录不存在")
	case errors.Is(err, ErrBadKind):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "服务类型无效")
	case errors.Is(err, ErrBadCustomer):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "客户信息无效")
	case errors.Is(err, ErrBadTitle):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "标题不能为空")
	case errors.Is(err, ErrBadDate):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "服务日期无效")
	case errors.Is(err, ErrBadAppointment):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "预约无效")
	case errors.Is(err, ErrAppointmentTaken):
		response.Error(c, http.StatusConflict, response.CodeInvalidParams, "该预约已有记录")
	default:
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "服务器错误")
	}
}
