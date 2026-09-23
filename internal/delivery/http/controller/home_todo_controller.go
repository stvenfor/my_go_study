// home_todo_controller.go 首页待办与入店申请 HTTP。
package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

// HomeTodoController 首页待办。
type HomeTodoController struct {
	uc *usecase.HomeTodoUsecase
}

// NewHomeTodoController 创建。
func NewHomeTodoController(uc *usecase.HomeTodoUsecase) *HomeTodoController {
	return &HomeTodoController{uc: uc}
}

// ListTodoCards GET /api/v1/home/todo-cards
func (ctrl *HomeTodoController) ListTodoCards(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	list, err := ctrl.uc.ListTodoCards(c.Request.Context(), user.ID)
	if err != nil {
		writeHomeTodoError(c, err)
		return
	}
	response.Success(c, gin.H{"items": list})
}

// ApplyToStore POST /api/v1/stores/:store_id/join-applications
func (ctrl *HomeTodoController) ApplyToStore(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	storeID, err := strconv.Atoi(c.Param("store_id"))
	if err != nil || storeID <= 0 {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "store_id 无效")
		return
	}
	app, err := ctrl.uc.ApplyToStore(c.Request.Context(), user.ID, storeID)
	if err != nil {
		writeHomeTodoError(c, err)
		return
	}
	response.Success(c, app)
}

// ListPendingJoinApplications GET /api/v1/home/join-applications
func (ctrl *HomeTodoController) ListPendingJoinApplications(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	list, err := ctrl.uc.ListPendingJoinApplications(c.Request.Context(), user.ID)
	if err != nil {
		writeHomeTodoError(c, err)
		return
	}
	response.Success(c, gin.H{"items": list})
}

// ApproveJoinApplication POST /api/v1/home/join-applications/:application_id/approve
func (ctrl *HomeTodoController) ApproveJoinApplication(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	id, err := strconv.ParseInt(c.Param("application_id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "application_id 无效")
		return
	}
	if err := ctrl.uc.ApproveJoinApplication(c.Request.Context(), user.ID, id); err != nil {
		writeHomeTodoError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// RejectJoinApplication POST /api/v1/home/join-applications/:application_id/reject
func (ctrl *HomeTodoController) RejectJoinApplication(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	id, err := strconv.ParseInt(c.Param("application_id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "application_id 无效")
		return
	}
	if err := ctrl.uc.RejectJoinApplication(c.Request.Context(), user.ID, id); err != nil {
		writeHomeTodoError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// ListOverdueCustomers GET /api/v1/home/follow-up-customers
func (ctrl *HomeTodoController) ListOverdueCustomers(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	list, err := ctrl.uc.ListOverdueCustomers(c.Request.Context(), user.ID)
	if err != nil {
		writeHomeTodoError(c, err)
		return
	}
	response.Success(c, gin.H{"items": list})
}

// ListPendingAppointments GET /api/v1/home/after-sales-appointments
func (ctrl *HomeTodoController) ListPendingAppointments(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	list, err := ctrl.uc.ListPendingAppointments(c.Request.Context(), user.ID)
	if err != nil {
		writeHomeTodoError(c, err)
		return
	}
	response.Success(c, gin.H{"items": list})
}

// ListPendingReviewOrders GET /api/v1/home/store-review-orders
func (ctrl *HomeTodoController) ListPendingReviewOrders(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	list, err := ctrl.uc.ListPendingReviewOrders(c.Request.Context(), user.ID)
	if err != nil {
		writeHomeTodoError(c, err)
		return
	}
	response.Success(c, gin.H{"items": list})
}

func writeHomeTodoError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrAccessForbidden), errors.Is(err, usecase.ErrAccessNotMember):
		response.Error(c, http.StatusForbidden, response.CodeForbidden, "仅门店管理员可查看")
	case errors.Is(err, usecase.ErrJoinApplicationNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "入店申请不存在")
	case errors.Is(err, usecase.ErrJoinApplicationDuplicate):
		response.Error(c, http.StatusConflict, response.CodeInvalidParams, "已有待审申请")
	case errors.Is(err, usecase.ErrJoinAlreadyMember):
		response.Error(c, http.StatusConflict, response.CodeInvalidParams, "已是门店成员")
	case errors.Is(err, usecase.ErrJoinApplicationNotPending),
		errors.Is(err, usecase.ErrJoinApplicationInvalid):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
	default:
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "服务异常")
	}
}
