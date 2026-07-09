// realtime_controller.go Realtime Ticket/Sync/Push HTTP 控制器。
package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/request"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

// RealtimeController Realtime HTTP 控制器。
type RealtimeController struct {
	ticketUC *usecase.RealtimeTicketUsecase
	syncUC   *usecase.RealtimeSyncUsecase
	pushUC   *usecase.RealtimePushUsecase
}

// NewRealtimeController 创建 Realtime 控制器。
func NewRealtimeController(
	ticketUC *usecase.RealtimeTicketUsecase,
	syncUC *usecase.RealtimeSyncUsecase,
	pushUC *usecase.RealtimePushUsecase,
) *RealtimeController {
	return &RealtimeController{
		ticketUC: ticketUC,
		syncUC:   syncUC,
		pushUC:   pushUC,
	}
}

// WSTicket 签发 WebSocket 连接票据。
// POST /api/v1/realtime/ws-ticket
func (ctrl *RealtimeController) WSTicket(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.BackendError(c, http.StatusUnauthorized, "未授权")
		return
	}

	var req request.RealtimeTicketRequest
	_ = c.ShouldBindJSON(&req)

	result, err := ctrl.ticketUC.Issue(c.Request.Context(), usecase.RealtimeTicketInput{
		UserID:   user.ID,
		Platform: req.Platform,
		ConnID:   req.ConnID,
	})
	if err != nil {
		response.BackendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.BackendJSON(c, http.StatusOK, response.RealtimeTicketData{
		Ticket:           result.Ticket,
		WSURL:            result.WSURL,
		ExpiresInSeconds: result.ExpiresInSeconds,
		ConnID:           result.ConnID,
	})
}

// Sync 重连后增量同步。
// POST /api/v1/realtime/sync
func (ctrl *RealtimeController) Sync(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.BackendError(c, http.StatusUnauthorized, "未授权")
		return
	}

	var req request.RealtimeSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BackendError(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	result, err := ctrl.syncUC.Sync(c.Request.Context(), usecase.RealtimeSyncInput{
		UserID:   user.ID,
		SinceSeq: req.SinceSeq,
		Topics:   req.Topics,
	})
	if err != nil {
		response.BackendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.BackendJSON(c, http.StatusOK, response.RealtimeSyncData{
		Events:    result.Events,
		LatestSeq: result.LatestSeq,
	})
}

// Push 开发环境推送测试通知。
// POST /api/v1/realtime/push
func (ctrl *RealtimeController) Push(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.BackendError(c, http.StatusUnauthorized, "未授权")
		return
	}

	var req request.RealtimePushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BackendError(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	targetUserID := req.UserID
	if targetUserID == "" {
		targetUserID = user.ID
	}

	envelope, delivered, err := ctrl.pushUC.PushToUser(c.Request.Context(), usecase.RealtimePushInput{
		UserID: targetUserID,
		Topic:  req.Topic,
		Title:  req.Title,
		Body:   req.Body,
		Name:   req.Name,
		Extra:  req.Extra,
	})
	if err != nil {
		response.BackendError(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.BackendJSON(c, http.StatusOK, response.RealtimePushData{
		Envelope:  envelope,
		Delivered: delivered,
	})
}
