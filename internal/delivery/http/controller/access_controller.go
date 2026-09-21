// access_controller.go 门店与角色管理。
package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

// AccessController 门店、成员与角色分配。
type AccessController struct {
	accessUC *usecase.AccessUsecase
}

// NewAccessController 创建。
func NewAccessController(accessUC *usecase.AccessUsecase) *AccessController {
	return &AccessController{accessUC: accessUC}
}

// CreateStore POST /api/v1/stores
func (ctrl *AccessController) CreateStore(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	var body struct {
		StoreID int    `json:"store_id"`
		Name    string `json:"name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	store, err := ctrl.accessUC.CreateStore(c.Request.Context(), user.ID, body.StoreID, body.Name)
	if err != nil {
		writeAccessError(c, err)
		return
	}
	response.Success(c, store)
}

// UpsertMember POST /api/v1/stores/:store_id/members
func (ctrl *AccessController) UpsertMember(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	storeID, err := parsePathStoreID(c.Param("store_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		return
	}
	var body struct {
		UserID   string `json:"target_user_id"`
		Position int16  `json:"position"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	member, err := ctrl.accessUC.UpsertMember(c.Request.Context(), user.ID, storeID, body.UserID, body.Position)
	if err != nil {
		writeAccessError(c, err)
		return
	}
	response.Success(c, member)
}

// RemoveMember DELETE /api/v1/stores/:store_id/members/:user_id
func (ctrl *AccessController) RemoveMember(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	storeID, err := parsePathStoreID(c.Param("store_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		return
	}
	if err := ctrl.accessUC.RemoveMember(c.Request.Context(), user.ID, storeID, c.Param("user_id")); err != nil {
		writeAccessError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// AssignRole POST /api/v1/roles/assignments
func (ctrl *AccessController) AssignRole(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	var body roleAssignmentBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	if err := ctrl.accessUC.AssignRole(c.Request.Context(), user.ID, body.UserID, body.RoleCode, body.StoreID); err != nil {
		writeAccessError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// RevokeRole DELETE /api/v1/roles/assignments
func (ctrl *AccessController) RevokeRole(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	var body roleAssignmentBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	if err := ctrl.accessUC.RevokeRole(c.Request.Context(), user.ID, body.UserID, body.RoleCode, body.StoreID); err != nil {
		writeAccessError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// ListMyPermissions GET /api/v1/me/permissions?store_id=
func (ctrl *AccessController) ListMyPermissions(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	storeID := 0
	if raw := c.Query("store_id"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "store_id 必须为正整数")
			return
		}
		storeID = n
	}
	codes, err := ctrl.accessUC.ListEffectivePermissions(c.Request.Context(), user.ID, storeID)
	if err != nil {
		writeAccessError(c, err)
		return
	}
	response.Success(c, gin.H{"permissions": codes})
}

type roleAssignmentBody struct {
	UserID   string `json:"target_user_id"`
	RoleCode string `json:"role_code"`
	StoreID  *int   `json:"store_id"`
}

func parsePathStoreID(raw string) (int, error) {
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, usecase.ErrAccessInvalidStoreID
	}
	return n, nil
}

func writeAccessError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrAccessForbidden):
		response.Error(c, http.StatusForbidden, response.CodeForbidden, err.Error())
	case errors.Is(err, usecase.ErrAccessStoreNotFound),
		errors.Is(err, usecase.ErrAccessUserNotFound),
		errors.Is(err, usecase.ErrAccessMemberNotFound),
		errors.Is(err, usecase.ErrAccessRoleNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, err.Error())
	case errors.Is(err, usecase.ErrAccessStoreExists),
		errors.Is(err, usecase.ErrAccessRoleScope),
		errors.Is(err, usecase.ErrAccessNotMember),
		errors.Is(err, usecase.ErrAccessLastPlatformAdmin),
		errors.Is(err, usecase.ErrAccessInvalidPosition),
		errors.Is(err, usecase.ErrAccessInvalidStoreID),
		errors.Is(err, usecase.ErrAccessInvalidRole),
		errors.Is(err, usecase.ErrAccessInvalidName):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
	default:
		response.Error(c, http.StatusBadGateway, response.CodeInternalError, err.Error())
	}
}
