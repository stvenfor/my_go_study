package controller

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/delivery/http/middleware"
	"github.com/stvenfor/my_go_study/internal/usecase"
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
	out, err := ctrl.uc.RegisterDevice(c.Request.Context(), usecase.RegisterDeviceInput{
		UserID:         user.ID,
		DeviceID:       deviceID,
		Platform:       body.Platform,
		RegistrationID: body.RegistrationID,
		Alias:          body.Alias,
		Mock:           body.Mock,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeInternalError, err.Error())
		return
	}
	response.Success(c, out)
}

// Send POST /api/v1/push/send（调试：给指定用户/别名发极光通知）
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
	targetUser := strings.TrimSpace(body.UserID)
	if targetUser == "" {
		targetUser = user.ID
	}
	alias := strings.TrimSpace(body.Alias)
	if alias == "" {
		alias = targetUser
	}
	out, err := ctrl.uc.Send(c.Request.Context(), usecase.SendInput{
		UserID:          targetUser,
		Alias:           alias,
		RegistrationIDs: body.RegistrationIDs,
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
