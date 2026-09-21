// profile_controller.go Supabase Profile 用户资料管理控制器。
package controller

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

// ProfileController Profile 管理控制器。
type ProfileController struct {
	profileUC *usecase.ProfileUsecase
}

// NewProfileController 创建 Profile 控制器。
func NewProfileController(profileUC *usecase.ProfileUsecase) *ProfileController {
	return &ProfileController{profileUC: profileUC}
}

// GetMe 获取当前用户资料（统一响应格式）。
// GET /api/v1/profiles/me?store_id=
func (ctrl *ProfileController) GetMe(c *gin.Context) {
	user, token, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}

	profile, stats, err := ctrl.profileUC.GetProfile(c.Request.Context(), token, user.ID, c.Query("store_id"))
	if err != nil {
		writeProfileError(c, err)
		return
	}
	response.Success(c, response.FromProfile(profile, stats))
}

// SwitchStore 切换当前店铺并返回该店铺资料。
// POST /api/v1/profiles/me/store  body: {"store_id":1}
func (ctrl *ProfileController) SwitchStore(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	var body struct {
		StoreID int `json:"store_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	stats, err := ctrl.profileUC.SwitchStore(c.Request.Context(), user.ID, body.StoreID)
	if err != nil {
		writeProfileError(c, err)
		return
	}
	response.Success(c, response.FromUserStoreStats(stats))
}

func writeProfileError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidStoreID):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
	case errors.Is(err, usecase.ErrStoreNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, err.Error())
	default:
		response.Error(c, http.StatusBadGateway, response.CodeInternalError, err.Error())
	}
}

// UpdateMe 更新当前用户资料（统一响应格式）。
// PATCH /api/v1/profiles/me
func (ctrl *ProfileController) UpdateMe(c *gin.Context) {
	user, token, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}

	var input entity.UpdateProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}

	profile, err := ctrl.profileUC.UpdateProfile(c.Request.Context(), token, user.ID, input)
	if err != nil {
		response.Error(c, http.StatusBadGateway, response.CodeInternalError, err.Error())
		return
	}
	_, stats, _ := ctrl.profileUC.GetProfile(c.Request.Context(), token, user.ID, c.Query("store_id"))
	response.Success(c, response.FromProfile(profile, stats))
}

// GetMeLegacy 获取当前用户资料（Flutter BackendApiClient 兼容，snake_case 直出）。
// GET /api/v1/me/profile?store_id=
func (ctrl *ProfileController) GetMeLegacy(c *gin.Context) {
	user, token, ok := supabaseAuthContext(c)
	if !ok {
		response.BackendError(c, http.StatusUnauthorized, "未授权")
		return
	}

	profile, stats, err := ctrl.profileUC.GetProfile(c.Request.Context(), token, user.ID, c.Query("store_id"))
	if err != nil {
		response.BackendError(c, http.StatusBadGateway, err.Error())
		return
	}
	response.BackendJSON(c, http.StatusOK, response.FromProfileLegacy(profile, stats))
}

// UpdateMeLegacy 更新当前用户资料（Flutter 兼容）。
// PATCH /api/v1/me/profile
func (ctrl *ProfileController) UpdateMeLegacy(c *gin.Context) {
	user, token, ok := supabaseAuthContext(c)
	if !ok {
		response.BackendError(c, http.StatusUnauthorized, "未授权")
		return
	}

	var input entity.UpdateProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BackendError(c, http.StatusBadRequest, "请求体格式错误")
		return
	}

	profile, err := ctrl.profileUC.UpdateProfile(c.Request.Context(), token, user.ID, input)
	if err != nil {
		response.BackendError(c, http.StatusBadGateway, err.Error())
		return
	}
	_, stats, _ := ctrl.profileUC.GetProfile(c.Request.Context(), token, user.ID, c.Query("store_id"))
	response.BackendJSON(c, http.StatusOK, response.FromProfileLegacy(profile, stats))
}
