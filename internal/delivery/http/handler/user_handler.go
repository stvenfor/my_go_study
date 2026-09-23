// =============================================================================
// 文件：user_handler.go
// 层级：Delivery/HTTP —— 用户注册/登录的 HTTP 入口（Flutter 调用的接口）
//
// 【注意】登录返回的是 Supabase token，不是 Go 自建 JWT。
// 遗留路由 /api/v1/user/profile 仍用自建 JWT，与 Flutter 当前链路不兼容。
// =============================================================================
package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/request"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/delivery/http/middleware"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

// UserHandler 用户 API 处理器。
type UserHandler struct {
	sessionAuthUC   usecase.SessionAuth
	deviceSessionUC *usecase.DeviceSessionUsecase
	phoneOTPUC      usecase.PhoneOTPAuth
	wechatLoginUC   usecase.WeChatLoginAuth
	huaweiLoginUC   usecase.HuaweiLoginAuth
}

// NewUserHandler 创建用户处理器。
func NewUserHandler(
	sessionAuthUC usecase.SessionAuth,
	deviceSessionUC *usecase.DeviceSessionUsecase,
	phoneOTPUC usecase.PhoneOTPAuth,
	wechatLoginUC usecase.WeChatLoginAuth,
	huaweiLoginUC usecase.HuaweiLoginAuth,
) *UserHandler {
	return &UserHandler{
		sessionAuthUC:   sessionAuthUC,
		deviceSessionUC: deviceSessionUC,
		phoneOTPUC:      phoneOTPUC,
		wechatLoginUC:   wechatLoginUC,
		huaweiLoginUC:   huaweiLoginUC,
	}
}

// Register POST /api/v1/user/register
// Flutter 注册页提交 username + email + password。
func (h *UserHandler) Register(c *gin.Context) {
	var req request.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}

	if h.sessionAuthUC == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "认证服务未配置，请联系管理员")
		return
	}

	result, err := h.sessionAuthUC.Register(c.Request.Context(), usecase.RegisterInput{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
	})
	if err != nil {
		h.handleUsecaseError(c, err)
		return
	}

	user := response.FromSupabaseAuthUser(result.UserID, result.Username, result.Email)
	user.Status = result.Status
	user.Phone = result.Phone
	user.AvatarURL = result.AvatarURL
	if result.Token != "" {
		sessionID, err := h.issueDeviceSession(c, result.UserID, result.Email, req.DeviceID, req.Platform)
		if err != nil {
			h.handleUsecaseError(c, err)
			return
		}
		response.Success(c, loginDataFrom(result, sessionID))
		return
	}
	response.Success(c, user)
}

// Refresh POST /api/v1/user/refresh
func (h *UserHandler) Refresh(c *gin.Context) {
	var req request.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}
	if h.sessionAuthUC == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "认证服务未配置，请联系管理员")
		return
	}

	result, err := h.sessionAuthUC.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		h.handleUsecaseError(c, err)
		return
	}

	data := response.RefreshTokenData{
		Token:        result.Token,
		RefreshToken: result.RefreshToken,
	}
	if h.deviceSessionUC != nil && strings.TrimSpace(req.DeviceID) != "" {
		sessionID, renewErr := h.deviceSessionUC.RenewOnRefresh(c.Request.Context(), usecase.RenewSessionInput{
			UserID:    result.UserID,
			Email:     result.Email,
			DeviceID:  req.DeviceID,
			Platform:  req.Platform,
			SessionID: req.SessionID,
		})
		if renewErr != nil {
			h.handleSessionError(c, renewErr)
			return
		}
		data.SessionID = sessionID
	}

	response.Success(c, data)
}

// Logout POST /api/v1/user/logout
func (h *UserHandler) Logout(c *gin.Context) {
	user, ok := middleware.GetSupabaseUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}

	sessionID := strings.TrimSpace(c.GetHeader(middleware.HeaderSessionID))
	deviceID := strings.TrimSpace(c.GetHeader(middleware.HeaderDeviceID))
	if h.deviceSessionUC != nil {
		if err := h.deviceSessionUC.RevokeOnLogout(c.Request.Context(), user.ID, user.Email, sessionID, deviceID); err != nil {
			h.handleSessionError(c, err)
			return
		}
	}

	if accessToken, ok := middleware.GetAccessToken(c); ok && accessToken != "" && h.sessionAuthUC != nil {
		_ = h.sessionAuthUC.Logout(c.Request.Context(), accessToken)
	}

	response.Success(c, gin.H{"ok": true})
}

// Deactivate POST /api/v1/user/deactivate
// 只把当前账号 deleted_at 写上，并撤销 refresh token。不提供把 status 设为停用的接口。
func (h *UserHandler) Deactivate(c *gin.Context) {
	user, ok := middleware.GetSupabaseUser(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}
	closer, ok := h.sessionAuthUC.(interface {
		Deactivate(ctx context.Context, userID string) error
	})
	if !ok || closer == nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "当前认证方式不支持注销")
		return
	}
	if err := closer.Deactivate(c.Request.Context(), user.ID); err != nil {
		h.handleUsecaseError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// Login POST /api/v1/user/login
// req.Username = 邮箱；成功返回 { token, user } 包在 ResultModel.data 里。
func (h *UserHandler) Login(c *gin.Context) {
	var req request.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}

	if h.sessionAuthUC == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "认证服务未配置，请联系管理员")
		return
	}

	result, err := h.sessionAuthUC.Login(c.Request.Context(), usecase.LoginInput{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		h.handleUsecaseError(c, err)
		return
	}

	sessionID, err := h.issueDeviceSession(c, result.UserID, result.Email, req.DeviceID, req.Platform)
	if err != nil {
		h.handleUsecaseError(c, err)
		return
	}

	response.Success(c, loginDataFrom(result, sessionID))
}

// SendPhoneOTP POST /api/v1/user/phone/otp/send
func (h *UserHandler) SendPhoneOTP(c *gin.Context) {
	var req request.SendPhoneOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}
	if h.phoneOTPUC == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "认证服务未配置，请联系管理员")
		return
	}
	if err := h.phoneOTPUC.SendPhoneOTP(c.Request.Context(), req.Phone); err != nil {
		h.handleUsecaseError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// VerifyPhoneOTP POST /api/v1/user/phone/otp/verify
func (h *UserHandler) VerifyPhoneOTP(c *gin.Context) {
	var req request.VerifyPhoneOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}
	if h.phoneOTPUC == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "认证服务未配置，请联系管理员")
		return
	}

	result, err := h.phoneOTPUC.VerifyPhoneOTP(c.Request.Context(), req.Phone, req.OTP)
	if err != nil {
		h.handleUsecaseError(c, err)
		return
	}

	sessionID, err := h.issueDeviceSession(c, result.UserID, result.Email, req.DeviceID, req.Platform)
	if err != nil {
		h.handleUsecaseError(c, err)
		return
	}

	response.Success(c, loginDataFrom(result, sessionID))
}

// LoginWithWechat POST /api/v1/user/wechat/login
func (h *UserHandler) LoginWithWechat(c *gin.Context) {
	var req request.WeChatLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}
	if h.wechatLoginUC == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "微信登录未启用（需 auth.provider=local）")
		return
	}

	result, err := h.wechatLoginUC.LoginWithWechatCode(c.Request.Context(), req.Code)
	if err != nil {
		h.handleUsecaseError(c, err)
		return
	}

	sessionID, err := h.issueDeviceSession(c, result.UserID, result.Email, req.DeviceID, req.Platform)
	if err != nil {
		h.handleUsecaseError(c, err)
		return
	}

	response.Success(c, loginDataFrom(result, sessionID))
}

// LoginWithHuawei POST /api/v1/user/huawei/login
func (h *UserHandler) LoginWithHuawei(c *gin.Context) {
	var req request.HuaweiLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}
	if h.huaweiLoginUC == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "华为登录未启用（需 auth.provider=local）")
		return
	}

	result, err := h.huaweiLoginUC.LoginWithHuaweiCode(c.Request.Context(), req.Code)
	if err != nil {
		h.handleUsecaseError(c, err)
		return
	}

	sessionID, err := h.issueDeviceSession(c, result.UserID, result.Email, req.DeviceID, req.Platform)
	if err != nil {
		h.handleUsecaseError(c, err)
		return
	}

	response.Success(c, loginDataFrom(result, sessionID))
}

func (h *UserHandler) issueDeviceSession(c *gin.Context, userID, email, deviceID, platform string) (string, error) {
	if h.deviceSessionUC == nil {
		return "", usecase.ErrSupabaseUnavailable
	}
	return h.deviceSessionUC.IssueOnLogin(c.Request.Context(), usecase.IssueSessionInput{
		UserID:   userID,
		Email:    email,
		DeviceID: deviceID,
		Platform: platform,
	})
}

// handleUsecaseError 业务错误 → HTTP 状态码 + 中文提示（Flutter _mapFailure 依赖这些文案）。
func (h *UserHandler) handleUsecaseError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrInvalidParams):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error())
	case errors.Is(err, usecase.ErrUserExists):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "用户已存在")
	case errors.Is(err, usecase.ErrAccountNotRegistered):
		response.Error(c, http.StatusNotFound, response.CodeForbidden, "账号未注册，请先注册")
	case errors.Is(err, usecase.ErrAccountDisabled):
		response.Error(c, http.StatusForbidden, response.CodeForbidden, "账号已停用")
	case errors.Is(err, usecase.ErrAccountLocked):
		response.Error(c, http.StatusForbidden, response.CodeForbidden, "账号已锁定，请稍后再试")
	case errors.Is(err, usecase.ErrInvalidCredentials):
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "密码错误")
	case errors.Is(err, usecase.ErrUserNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "用户不存在")
	case errors.Is(err, usecase.ErrEmailConfirmationRequired):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "注册成功，请查收验证邮件后再登录")
	case errors.Is(err, usecase.ErrSupabaseUnavailable):
		response.Error(c, http.StatusBadGateway, response.CodeInternalError, "认证服务暂时不可用，请检查后端网络或配置")
	case errors.Is(err, usecase.ErrInvalidPlatform):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "platform 必须为 android 或 ios")
	case errors.Is(err, usecase.ErrInvalidDeviceID):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "device_id 不能为空")
	case errors.Is(err, usecase.ErrInvalidOTP):
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "验证码错误或已失效")
	case errors.Is(err, usecase.ErrPhoneLoginNotAvailable):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "短信登录暂未开放，请使用邮箱登录")
	case errors.Is(err, usecase.ErrWechatNotConfigured):
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, err.Error())
	case errors.Is(err, usecase.ErrWechatLocalOnly):
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, err.Error())
	case errors.Is(err, usecase.ErrWechatAuthFailed):
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, err.Error())
	case errors.Is(err, usecase.ErrHuaweiNotConfigured):
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, err.Error())
	case errors.Is(err, usecase.ErrHuaweiLocalOnly):
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, err.Error())
	case errors.Is(err, usecase.ErrHuaweiAuthFailed):
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, err.Error())
	default:
		// 带微信 / 华为 errmsg 后缀
		if strings.Contains(err.Error(), "微信授权失败") || strings.Contains(err.Error(), "华为授权失败") {
			response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, err.Error())
			return
		}
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "服务器内部错误")
	}
}

func (h *UserHandler) handleSessionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, usecase.ErrSessionReplaced):
		response.Error(c, http.StatusUnauthorized, response.CodeSessionReplaced, usecase.MsgSessionReplaced)
	case errors.Is(err, usecase.ErrSessionInvalid):
		response.Error(c, http.StatusUnauthorized, response.CodeSessionInvalid, usecase.MsgSessionInvalid)
	default:
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "服务器内部错误")
	}
}

func loginDataFrom(result *usecase.SupabaseAuthOutput, sessionID string) response.LoginData {
	return response.LoginData{
		Token:        result.Token,
		RefreshToken: result.RefreshToken,
		SessionID:    sessionID,
		User:         response.AuthUserFromOutput(result.UserID, result.Username, result.Email, result.Phone, result.AvatarURL, result.Status),
	}
}
