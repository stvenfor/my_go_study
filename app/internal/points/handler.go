package points

import (
	"errors"
	"net/http"

	"app/internal/delivery/http/authctx"
	"app/internal/delivery/http/dto/response"
	"github.com/gin-gonic/gin"
)

// handler 签到与积分。
type handler struct {
	svc *Service
}

// newHandler 创建。
func newHandler(svc *Service) *handler {
	return &handler{svc: svc}
}

// GetStatus GET /api/v1/points/status
func (ctrl *handler) GetStatus(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	st, err := ctrl.svc.GetStatus(c.Request.Context(), user.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "查询失败")
		return
	}
	response.Success(c, st)
}

// CheckIn POST /api/v1/points/check-in
func (ctrl *handler) CheckIn(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	res, err := ctrl.svc.CheckIn(c.Request.Context(), user.ID)
	if err != nil {
		writePointsError(c, err)
		return
	}
	response.Success(c, res)
}

// ListTasks GET /api/v1/points/tasks
func (ctrl *handler) ListTasks(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	tasks, err := ctrl.svc.ListTasks(c.Request.Context(), user.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "查询失败")
		return
	}
	response.Success(c, gin.H{"items": tasks})
}

// ClaimTask POST /api/v1/points/tasks/:code/claim
func (ctrl *handler) ClaimTask(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	code := c.Param("code")
	res, err := ctrl.svc.ClaimTask(c.Request.Context(), user.ID, code)
	if err != nil {
		writePointsError(c, err)
		return
	}
	response.Success(c, res)
}

func writePointsError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrPointsAlreadyCheckedIn):
		response.Error(c, http.StatusConflict, response.CodeInvalidParams, "今日已签到")
	case errors.Is(err, ErrPointsTaskNotClaimable):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "任务不可领取")
	case errors.Is(err, ErrPointsTaskUnknown):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "未知任务")
	case errors.Is(err, ErrPointsInsufficient), errors.Is(err, ErrPointsInsufficient):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "积分不足")
	default:
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "操作失败")
	}
}
