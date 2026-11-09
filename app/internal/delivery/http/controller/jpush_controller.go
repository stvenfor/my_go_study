package controller

import (
	"net/http"
	"strings"

	"app/internal/delivery/http/dto/response"
	"app/internal/delivery/http/middleware"
	"app/internal/usecase"
	"common/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// JPushController 极光设备登记与调试下发。
type JPushController struct {
	uc *usecase.JPushUsecase
}

func NewJPushController(uc *usecase.JPushUsecase) *JPushController {
	return &JPushController{uc: uc}
}

// RegisterDevice POST /api/v1/push/devices
func (ctrl *JPushController) RegisterDevice(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录")
		return
	}
	if ctrl.uc == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "push 未启用")
		return
	}
	var body struct {
		DeviceID       string `json:"device_id"`
		Platform       string `json:"platform"`
		RegistrationID string `json:"registration_id" binding:"required"`
		Alias          string `json:"alias"`
		Mock           bool   `json:"mock"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	deviceID := strings.TrimSpace(body.DeviceID)
	if deviceID == "" {
		deviceID = strings.TrimSpace(c.GetHeader(middleware.HeaderDeviceID))
	}
	if deviceID == "" {
		deviceID = "default"
	}
	// 越权防护：忽略客户端 alias，强制绑定会话用户（h4 验收口径）。
	out, err := ctrl.uc.RegisterDevice(c.Request.Context(), usecase.RegisterDeviceInput{
		UserID:         user.ID,
		DeviceID:       deviceID,
		Platform:       body.Platform,
		RegistrationID: body.RegistrationID,
		Alias:          user.ID,
		Mock:           body.Mock,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, err.Error())
		return
	}
	response.Success(c, out)
}

// Send POST /api/v1/push/send（调试：仅允许向会话用户本人下发）
func (ctrl *JPushController) Send(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录")
		return
	}
	if ctrl.uc == nil {
		response.Error(c, http.StatusServiceUnavailable, response.CodeInternalError, "push 未启用")
		return
	}
	var body struct {
		UserID          string         `json:"user_id"`
		Alias           string         `json:"alias"`
		RegistrationIDs []string       `json:"registration_ids"`
		Title           string         `json:"title"`
		Body            string         `json:"body" binding:"required"`
		Deeplink        string         `json:"deeplink"`
		Extras          map[string]any `json:"extras"`
		Platform        string         `json:"platform"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeInvalidParams, "参数错误")
		return
	}
	// 越权防护：客户端只允许向本人下发。user_id 留空/填自己 → 本人；填他人 → 403。
	requestedUser := strings.TrimSpace(body.UserID)
	if requestedUser != "" && requestedUser != user.ID {
		logger.L().Warn("push/send 越权拦截: 拒绝跨用户下发",
			zap.String("session_user", user.ID),
			zap.String("requested_user", requestedUser))
		response.Error(c, http.StatusForbidden, response.CodeForbidden, "只能向本人下发通知")
		return
	}
	// 忽略客户端指定的 alias / registration_ids：调试端点不得成为任意目标下发通道。
	// 不解析「是否本人登记」需注入 device repo（要改 main.go，列为 Deferred），
	// 故一律丢弃并留 warn，目标锁定会话用户。
	if strings.TrimSpace(body.Alias) != "" || len(body.RegistrationIDs) > 0 {
		logger.L().Warn("push/send 忽略客户端指定的 alias/registration_ids，强制以会话用户为目标",
			zap.String("session_user", user.ID),
			zap.String("ignored_alias", body.Alias),
			zap.Int("ignored_registration_ids", len(body.RegistrationIDs)))
	}
	// 目标锁定会话用户：alias 取 user.ID（与 RegisterDevice 一致），不下发 registration_ids。
	out, err := ctrl.uc.Send(c.Request.Context(), usecase.SendInput{
		UserID:          user.ID,
		Alias:           user.ID,
		RegistrationIDs: nil,
		Title:           body.Title,
		Body:            body.Body,
		Deeplink:        body.Deeplink,
		Extras:          body.Extras,
		Platform:        body.Platform,
	})
	if err != nil {
		response.Error(c, http.StatusBadGateway, response.CodeInternalError, err.Error())
		return
	}
	response.Success(c, out)
}
