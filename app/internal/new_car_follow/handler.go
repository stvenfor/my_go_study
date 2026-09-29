// handler.go 新车跟进档案 HTTP 适配层。
//
// 由 app/internal/delivery/http/controller/new_car_follow_controller.go 迁入。
// 类型改小写 handler；取身份改走 authctx.SupabaseAuthContext。
package new_car_follow

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"app/internal/delivery/http/authctx"
	"app/internal/delivery/http/dto/response"
	"github.com/gin-gonic/gin"
)

// handler 新车跟进档案 HTTP。
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
	overdue := strings.TrimSpace(c.Query("overdue")) == "1"
	list, total, err := h.svc.List(
		c.Request.Context(),
		user.ID,
		strings.TrimSpace(c.Query("follow_level")),
		strings.TrimSpace(c.Query("intent_band")),
		strings.TrimSpace(c.Query("stage")),
		overdue,
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
	id, err := parseFileID(c.Param("file_id"))
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
		CustomerID      int64   `json:"customer_id"`
		DisplayName     string  `json:"display_name"`
		Phone           string  `json:"phone"`
		FollowLevel     string  `json:"follow_level"`
		Stage           string  `json:"stage"`
		VehicleInterest string  `json:"vehicle_interest"`
		BudgetNote      string  `json:"budget_note"`
		Source          string  `json:"source"`
		NextFollowUpAt  *string `json:"next_follow_up_at"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	next, err := parseOptionalRFC3339(body.NextFollowUpAt)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "下次跟进时间无效")
		return
	}
	dto, err := h.svc.Create(c.Request.Context(), user.ID, CreateInput{
		CustomerID:      body.CustomerID,
		DisplayName:     body.DisplayName,
		Phone:           body.Phone,
		FollowLevel:     body.FollowLevel,
		Stage:           body.Stage,
		VehicleInterest: body.VehicleInterest,
		BudgetNote:      body.BudgetNote,
		Source:          body.Source,
		NextFollowUpAt:  next,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessCreated(c, dto)
}

func (h *handler) Patch(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	id, err := parseFileID(c.Param("file_id"))
	if err != nil {
		writeError(c, err)
		return
	}
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil || len(raw) == 0 {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	in := PatchInput{}
	if v, ok := fields["follow_level"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "跟进级别无效")
			return
		}
		in.FollowLevel = &s
	}
	if v, ok := fields["stage"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "阶段无效")
			return
		}
		in.Stage = &s
	}
	if v, ok := fields["vehicle_interest"]; ok {
		var s string
		_ = json.Unmarshal(v, &s)
		in.VehicleInterest = &s
	}
	if v, ok := fields["budget_note"]; ok {
		var s string
		_ = json.Unmarshal(v, &s)
		in.BudgetNote = &s
	}
	if v, ok := fields["source"]; ok {
		var s string
		_ = json.Unmarshal(v, &s)
		in.Source = &s
	}
	if v, ok := fields["closed_reason"]; ok {
		var s string
		_ = json.Unmarshal(v, &s)
		in.ClosedReason = &s
	}
	if v, ok := fields["next_follow_up_at"]; ok {
		in.TouchNextFollow = true
		if string(v) == "null" {
			in.NextFollowUpAt = nil
		} else {
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "下次跟进时间无效")
				return
			}
			t, err := time.Parse(time.RFC3339, strings.TrimSpace(s))
			if err != nil {
				response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "下次跟进时间无效")
				return
			}
			in.NextFollowUpAt = &t
		}
	}
	dto, err := h.svc.Patch(c.Request.Context(), user.ID, id, in)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, dto)
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

func (h *handler) ListLogs(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	id, err := parseFileID(c.Param("file_id"))
	if err != nil {
		writeError(c, err)
		return
	}
	pq := response.ParsePageQuery(c, 20)
	list, total, err := h.svc.ListLogs(c.Request.Context(), user.ID, id, pq.Page, pq.Size)
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessList(c, list, pq.Page, pq.Size, total)
}

func (h *handler) CreateLog(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	id, err := parseFileID(c.Param("file_id"))
	if err != nil {
		writeError(c, err)
		return
	}
	raw, err := io.ReadAll(c.Request.Body)
	if err != nil || len(raw) == 0 {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	in := CreateFollowLogInput{}
	if v, ok := fields["body"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "跟进内容无效")
			return
		}
		in.Body = s
	}
	if v, ok := fields["follow_level"]; ok {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "跟进级别无效")
			return
		}
		in.FollowLevel = s
	}
	if v, ok := fields["next_follow_up_at"]; ok {
		in.TouchNext = true
		if string(v) == "null" {
			in.NextFollowUpAt = nil
		} else {
			var s string
			if err := json.Unmarshal(v, &s); err != nil {
				response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "下次跟进时间无效")
				return
			}
			t, err := time.Parse(time.RFC3339, strings.TrimSpace(s))
			if err != nil {
				response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "下次跟进时间无效")
				return
			}
			in.NextFollowUpAt = &t
		}
	}
	dto, err := h.svc.CreateLog(c.Request.Context(), user.ID, id, in)
	if err != nil {
		writeError(c, err)
		return
	}
	response.SuccessCreated(c, dto)
}

func parseOptionalRFC3339(raw *string) (*time.Time, error) {
	if raw == nil {
		return nil, nil
	}
	s := strings.TrimSpace(*raw)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNoStore):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "未选择当前门店")
	case errors.Is(err, ErrNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "跟进档案不存在")
	case errors.Is(err, ErrBadCustomer):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "客户无效")
	case errors.Is(err, ErrBadLevel):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "跟进级别无效，须为 A/B/E/H")
	case errors.Is(err, ErrBadStage):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "档案阶段无效")
	case errors.Is(err, ErrBadFilter):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "筛选参数无效")
	case errors.Is(err, ErrDuplicate):
		response.Error(c, http.StatusConflict, response.CodeInvalidParams, "该客户已有未关闭跟进档案")
	case errors.Is(err, ErrBadLogBody):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "跟进内容不能为空")
	default:
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "服务异常")
	}
}
