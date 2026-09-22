package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

// PointsController 签到与积分。
type PointsController struct {
	pointsUC *usecase.PointsUsecase
}

// NewPointsController 创建。
func NewPointsController(pointsUC *usecase.PointsUsecase) *PointsController {
	return &PointsController{pointsUC: pointsUC}
}

// GetStatus GET /api/v1/points/status
func (ctrl *PointsController) GetStatus(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	st, err := ctrl.pointsUC.GetStatus(c.Request.Context(), user.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "查询失败")
		return
	}
	response.Success(c, st)
}

// CheckIn POST /api/v1/points/check-in
func (ctrl *PointsController) CheckIn(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	res, err := ctrl.pointsUC.CheckIn(c.Request.Context(), user.ID)
	if err != nil {
		writePointsError(c, err)
		return
	}
	response.Success(c, res)
}

// ListTasks GET /api/v1/points/tasks
func (ctrl *PointsController) ListTasks(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	tasks, err := ctrl.pointsUC.ListTasks(c.Request.Context(), user.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "查询失败")
		return
	}
	response.Success(c, gin.H{"items": tasks})
}

// ClaimTask POST /api/v1/points/tasks/:code/claim
func (ctrl *PointsController) ClaimTask(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	code := c.Param("code")
	res, err := ctrl.pointsUC.ClaimTask(c.Request.Context(), user.ID, code)
	if err != nil {
		writePointsError(c, err)
		return
	}
	response.Success(c, res)
}

func writePointsError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrPointsAlreadyCheckedIn):
		response.Error(c, http.StatusConflict, response.CodeInvalidParams, "今日已签到")
	case errors.Is(err, usecase.ErrPointsTaskNotClaimable):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "任务不可领取")
	case errors.Is(err, usecase.ErrPointsTaskUnknown):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "未知任务")
	case errors.Is(err, repository.ErrPointsInsufficient), errors.Is(err, usecase.ErrPointsInsufficient):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "积分不足")
	default:
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "操作失败")
	}
}
