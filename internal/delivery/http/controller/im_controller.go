package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

// ImController 融云 IM SessionAuth 接口。
type ImController struct {
	sessionUC *usecase.ImSessionUsecase
	friendUC  *usecase.ImFriendUsecase
	groupUC   *usecase.ImGroupUsecase
	backupUC  *usecase.ImBackupUsecase
}

func NewImController(
	sessionUC *usecase.ImSessionUsecase,
	friendUC *usecase.ImFriendUsecase,
	groupUC *usecase.ImGroupUsecase,
	backupUC *usecase.ImBackupUsecase,
) *ImController {
	return &ImController{
		sessionUC: sessionUC,
		friendUC:  friendUC,
		groupUC:   groupUC,
		backupUC:  backupUC,
	}
}

// CreateSession POST /api/v1/im/session
func (ctrl *ImController) CreateSession(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录")
		return
	}
	if ctrl.sessionUC == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "im 未启用")
		return
	}
	var body struct {
		DisplayName string `json:"display_name"`
	}
	_ = c.ShouldBindJSON(&body)
	out, err := ctrl.sessionUC.IssueSession(c.Request.Context(), user.ID, body.DisplayName)
	if err != nil {
		if errors.Is(err, usecase.ErrImNotConfigured) {
			response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "融云未配置")
			return
		}
		if errors.Is(err, usecase.ErrImUserIDRequired) {
			response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "用户无效")
			return
		}
		response.Error(c, http.StatusBadGateway, response.CodeInternalError, err.Error())
		return
	}
	response.Success(c, out)
}

// SearchUsers GET /api/v1/im/users/search?q=
func (ctrl *ImController) SearchUsers(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录")
		return
	}
	if ctrl.friendUC == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "im 未启用")
		return
	}
	out, err := ctrl.friendUC.Search(c.Request.Context(), user.ID, c.Query("q"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		return
	}
	response.Success(c, gin.H{"items": out})
}

// ListFriends GET /api/v1/im/friends
func (ctrl *ImController) ListFriends(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录")
		return
	}
	if ctrl.friendUC == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "im 未启用")
		return
	}
	out, err := ctrl.friendUC.ListFriends(c.Request.Context(), user.ID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, err.Error())
		return
	}
	response.Success(c, gin.H{"items": out})
}

// RequestFriend POST /api/v1/im/friends/requests
func (ctrl *ImController) RequestFriend(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录")
		return
	}
	if ctrl.friendUC == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "im 未启用")
		return
	}
	var body struct {
		ToUserID string `json:"to_user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	out, err := ctrl.friendUC.Request(c.Request.Context(), user.ID, body.ToUserID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		return
	}
	response.Success(c, out)
}

// RespondFriend POST /api/v1/im/friends/requests/:id/respond
func (ctrl *ImController) RespondFriend(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录")
		return
	}
	if ctrl.friendUC == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "im 未启用")
		return
	}
	var body struct {
		Accept bool `json:"accept"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	out, err := ctrl.friendUC.Respond(c.Request.Context(), user.ID, c.Param("id"), body.Accept)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		return
	}
	response.Success(c, out)
}

// CheckPrivateAdmission GET /api/v1/im/private/admission?peer_user_id=
func (ctrl *ImController) CheckPrivateAdmission(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录")
		return
	}
	if ctrl.friendUC == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "im 未启用")
		return
	}
	okAdmit, err := ctrl.friendUC.CanPrivateChat(c.Request.Context(), user.ID, c.Query("peer_user_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		return
	}
	response.Success(c, gin.H{"allowed": okAdmit})
}

// CreateFreeGroup POST /api/v1/im/groups/free
func (ctrl *ImController) CreateFreeGroup(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录")
		return
	}
	if ctrl.groupUC == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "im 未启用")
		return
	}
	var body struct {
		Name      string   `json:"name" binding:"required"`
		MemberIDs []string `json:"member_ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	out, err := ctrl.groupUC.CreateFreeGroup(c.Request.Context(), user.ID, body.Name, body.MemberIDs)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		return
	}
	response.Success(c, out)
}

// InviteFreeGroup POST /api/v1/im/groups/free/:id/invite
func (ctrl *ImController) InviteFreeGroup(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录")
		return
	}
	if ctrl.groupUC == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "im 未启用")
		return
	}
	var body struct {
		MemberIDs []string `json:"member_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	if err := ctrl.groupUC.InviteFreeGroup(c.Request.Context(), user.ID, c.Param("id"), body.MemberIDs); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// KickFreeGroup POST /api/v1/im/groups/free/:id/kick
func (ctrl *ImController) KickFreeGroup(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录")
		return
	}
	if ctrl.groupUC == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "im 未启用")
		return
	}
	var body struct {
		MemberIDs []string `json:"member_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	if err := ctrl.groupUC.KickFreeGroup(c.Request.Context(), user.ID, c.Param("id"), body.MemberIDs); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// QuitFreeGroup POST /api/v1/im/groups/free/:id/quit
func (ctrl *ImController) QuitFreeGroup(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录")
		return
	}
	if ctrl.groupUC == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "im 未启用")
		return
	}
	if err := ctrl.groupUC.QuitFreeGroup(c.Request.Context(), user.ID, c.Param("id")); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// DismissFreeGroup POST /api/v1/im/groups/free/:id/dismiss
func (ctrl *ImController) DismissFreeGroup(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录")
		return
	}
	if ctrl.groupUC == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "im 未启用")
		return
	}
	if err := ctrl.groupUC.DismissFreeGroup(c.Request.Context(), user.ID, c.Param("id")); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// SyncStoreGroup POST /api/v1/im/groups/store/sync
func (ctrl *ImController) SyncStoreGroup(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录")
		return
	}
	if ctrl.groupUC == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "im 未启用")
		return
	}
	var body struct {
		StoreID string `json:"store_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	out, err := ctrl.groupUC.EnsureStoreMembership(c.Request.Context(), strings.TrimSpace(body.StoreID), user.ID)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		return
	}
	response.Success(c, out)
}

// BackupMessages POST /api/v1/im/messages/backup
func (ctrl *ImController) BackupMessages(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录")
		return
	}
	if ctrl.backupUC == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "im 未启用")
		return
	}
	var body usecase.ImBackupBatchInput
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	body.UserID = user.ID
	n, err := ctrl.backupUC.Ingest(c.Request.Context(), body)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
		return
	}
	response.Success(c, gin.H{"accepted": n})
}
