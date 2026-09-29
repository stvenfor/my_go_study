// handler.go 门店与角色管理的 HTTP 适配层。
//
// 由 app/internal/delivery/http/controller/access_controller.go 迁入。
// 类型改小写 handler（域内私有，仅本包 router 使用）；取身份改走
// authctx.SupabaseAuthContext。
package access

import (
	"errors"
	"net/http"
	"strconv"

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

// CreateStore POST /api/v1/stores
func (h *handler) CreateStore(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
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
	store, err := h.svc.CreateStore(c.Request.Context(), user.ID, body.StoreID, body.Name)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, store)
}

// UpsertMember POST /api/v1/stores/:store_id/members
func (h *handler) UpsertMember(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
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
	member, err := h.svc.UpsertMember(c.Request.Context(), user.ID, storeID, body.UserID, body.Position)
	if err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, member)
}

// RemoveMember DELETE /api/v1/stores/:store_id/members/:user_id
func (h *handler) RemoveMember(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	storeID, err := parsePathStoreID(c.Param("store_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		return
	}
	if err := h.svc.RemoveMember(c.Request.Context(), user.ID, storeID, c.Param("user_id")); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// AssignRole POST /api/v1/roles/assignments
func (h *handler) AssignRole(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	var body roleAssignmentBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	if err := h.svc.AssignRole(c.Request.Context(), user.ID, body.UserID, body.RoleCode, body.StoreID); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// RevokeRole DELETE /api/v1/roles/assignments
func (h *handler) RevokeRole(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	var body roleAssignmentBody
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	if err := h.svc.RevokeRole(c.Request.Context(), user.ID, body.UserID, body.RoleCode, body.StoreID); err != nil {
		writeError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// ListMyPermissions GET /api/v1/me/permissions?store_id=
func (h *handler) ListMyPermissions(c *gin.Context) {
	user, _, ok := authctx.SupabaseAuthContext(c)
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
	codes, err := h.svc.ListEffectivePermissions(c.Request.Context(), user.ID, storeID)
	if err != nil {
		writeError(c, err)
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
		return 0, ErrInvalidStoreID
	}
	return n, nil
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		response.Error(c, http.StatusForbidden, response.CodeForbidden, err.Error())
	case errors.Is(err, ErrStoreNotFound),
		errors.Is(err, ErrUserNotFound),
		errors.Is(err, ErrMemberNotFound),
		errors.Is(err, ErrRoleNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, err.Error())
	case errors.Is(err, ErrStoreExists),
		errors.Is(err, ErrRoleScope),
		errors.Is(err, ErrNotMember),
		errors.Is(err, ErrLastPlatformAdmin),
		errors.Is(err, ErrInvalidPosition),
		errors.Is(err, ErrInvalidStoreID),
		errors.Is(err, ErrInvalidRole),
		errors.Is(err, ErrInvalidName):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
	default:
		response.Error(c, http.StatusBadGateway, response.CodeInternalError, err.Error())
	}
}
