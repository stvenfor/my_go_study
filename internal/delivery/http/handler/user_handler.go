// =============================================================================
// 文件：user_handler.go
// 层级：Delivery/HTTP —— 用户注册/登录的 HTTP 入口（Flutter 调用的接口）
//
// 【注意】登录返回的是 Supabase token，不是 Go 自建 JWT。
// 遗留路由 /api/v1/user/profile 仍用自建 JWT，与 Flutter 当前链路不兼容。
// =============================================================================
package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/request"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/delivery/http/middleware"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

// UserHandler 用户 API 处理器。
type UserHandler struct {
	userUsecase    *usecase.UserUsecase
	supabaseAuthUC *usecase.SupabaseAuthUsecase
}

// NewUserHandler 创建用户处理器。
func NewUserHandler(
	userUsecase *usecase.UserUsecase,
	supabaseAuthUC *usecase.SupabaseAuthUsecase,
) *UserHandler {
	return &UserHandler{
		userUsecase:    userUsecase,
		supabaseAuthUC: supabaseAuthUC,
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

	if h.supabaseAuthUC == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "认证服务未配置，请联系管理员")
		return
	}

	result, err := h.supabaseAuthUC.Register(c.Request.Context(), usecase.RegisterInput{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
	})
	if err != nil {
		h.handleUsecaseError(c, err)
		return
	}

	user := response.FromSupabaseAuthUser(result.UserID, result.Username, result.Email)
	if result.Token != "" {
		response.Success(c, response.LoginData{
			Token: result.Token,
			User:  user,
		})
		return
	}
	response.Success(c, user)
}

// Login POST /api/v1/user/login
// req.Username = 邮箱；成功返回 { token, user } 包在 ResultModel.data 里。
func (h *UserHandler) Login(c *gin.Context) {
	var req request.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误: "+err.Error())
		return
	}

	if h.supabaseAuthUC == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "认证服务未配置，请联系管理员")
		return
	}

	result, err := h.supabaseAuthUC.Login(c.Request.Context(), usecase.LoginInput{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		h.handleUsecaseError(c, err)
		return
	}

	response.Success(c, response.LoginData{
		Token: result.Token,
		User:  response.FromSupabaseAuthUser(result.UserID, result.Username, result.Email),
	})
}

// Profile 获取当前登录用户信息。
func (h *UserHandler) Profile(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未授权")
		return
	}

	user, err := h.userUsecase.GetProfile(c.Request.Context(), userID)
	if err != nil {
		h.handleUsecaseError(c, err)
		return
	}

	response.Success(c, response.FromUserProfile(user))
}

// List 分页获取用户列表。
func (h *UserHandler) List(c *gin.Context) {
	page := response.ParsePageQuery(c, 20)

	users, total, err := h.userUsecase.ListUsers(c.Request.Context(), page.Page, page.Size)
	if err != nil {
		h.handleUsecaseError(c, err)
		return
	}

	list := make([]response.UserItem, 0, len(users))
	for i := range users {
		list = append(list, response.FromUser(&users[i]))
	}
	response.SuccessList(c, list, page.Page, page.Size, total)
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
	case errors.Is(err, usecase.ErrInvalidCredentials):
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "密码错误")
	case errors.Is(err, usecase.ErrUserNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "用户不存在")
	case errors.Is(err, usecase.ErrEmailConfirmationRequired):
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "注册成功，请查收验证邮件后再登录")
	case errors.Is(err, usecase.ErrSupabaseUnavailable):
		response.Error(c, http.StatusBadGateway, response.CodeInternalError, "认证服务暂时不可用，请检查后端网络或配置")
	default:
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, "服务器内部错误")
	}
}
