// handler.go 二手车业务单 HTTP 适配层。
//
// 由 app/internal/delivery/http/controller/used_car_order_controller.go 迁入。
// 类型改小写 handler（域内私有，仅本包 router 使用）；取身份改走
// authctx.SupabaseAuthContext；路径 id 解析随域内解析函数。
package used_car

import (
	"errors"
	"net/http"
	"strings"

	"app/internal/delivery/http/authctx"
	"app/internal/delivery/http/dto/response"
	"github.com/gin-gonic/gin"
)

type handler struct {
	svc *Service
}

func newHandler(svc *Service) *handler {
	return &handler{svc: svc}
}

func (h *handler) Summary(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	dto, err := h.svc.Summary(c.Request.Context(), user.ID)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, dto)
}

func (h *handler) List(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	pq := response.ParsePageQuery(c, 10)
	list, total, err := h.svc.List(
		c.Request.Context(),
		user.ID,
		strings.TrimSpace(c.Query("status")),
		strings.TrimSpace(c.Query("kind")),
		pq.Page, pq.Size,
	)
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessList(c, list, pq.Page, pq.Size, total)
}

func (h *handler) Get(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	id, err := parseOrderID(c.Param("order_id"))
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

func (h *handler) Create(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
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
	dto, err := h.svc.Create(c.Request.Context(), user.ID, CreateInput{
		Kind: body.Kind, CustomerID: body.CustomerID,
		VehicleModel: body.VehicleModel, PlateNo: body.PlateNo, VIN: body.VIN,
		MileageKm: body.MileageKm, ModelYear: body.ModelYear, Amount: body.Amount,
		ImageURL: body.ImageURL,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessCreated(c, dto)
}

func (h *handler) ListCustomers(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	pq := response.ParsePageQuery(c, 20)
	list, total, err := h.svc.ListCustomers(
		c.Request.Context(),
		user.ID,
		strings.TrimSpace(c.Query("q")),
		pq.Page, pq.Size,
	)
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessList(c, list, pq.Page, pq.Size, total)
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNoStore):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "未选择当前门店")
	case errors.Is(err, ErrOrderNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "二手车业务单不存在")
	case errors.Is(err, ErrBadCustomer):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "客户无效")
	case errors.Is(err, ErrBadFilter):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "筛选参数无效")
	case errors.Is(err, ErrBadInput):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "业务单参数无效")
	default:
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "服务异常")
	}
}
