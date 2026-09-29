// cs_agent_controller.go 客服 Agent SSE HTTP。
package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"app/internal/delivery/http/dto/request"
	"app/internal/delivery/http/dto/response"
	"model/entity"
	"app/internal/usecase"
	"common/config"
)

// CsAgentController 客服 Agent 控制器。
type CsAgentController struct {
	uc  *usecase.CsAgentUsecase
	cfg config.CsAgentConfig
}

// NewCsAgentController 创建控制器。
func NewCsAgentController(uc *usecase.CsAgentUsecase, cfg config.CsAgentConfig) *CsAgentController {
	return &CsAgentController{uc: uc, cfg: cfg}
}

// Chat POST /api/v1/cs-agent/chat
func (ctrl *CsAgentController) Chat(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.BackendError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req request.CsAgentChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BackendError(c, http.StatusBadRequest, "invalid request")
		return
	}

	streamCtx := c.Request.Context()
	if timeout := ctrl.cfg.RequestTimeout(); timeout > 0 {
		var cancel context.CancelFunc
		streamCtx, cancel = context.WithTimeout(streamCtx, timeout)
		defer cancel()
	}

	ch, err := ctrl.uc.Stream(streamCtx, user.ID, usecase.CsAgentRequest{
		Prompt:          req.Prompt,
		ClientRequestID: req.ClientRequestID,
	})
	if err != nil {
		status, msg := mapCsAgentPreStreamError(err)
		response.BackendError(c, status, msg)
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache, no-transform")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return
	}

	keepalive := ctrl.cfg.KeepaliveOrDefault()
	ticker := time.NewTicker(keepalive)
	defer ticker.Stop()

	for {
		select {
		case <-streamCtx.Done():
			return
		case <-ticker.C:
			_, _ = c.Writer.Write([]byte(": keepalive\n\n"))
			flusher.Flush()
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if err := writeSSEFrame(c, ev); err != nil {
				return
			}
			flusher.Flush()
			if ev.Event == entity.SSEEventDone || ev.Event == entity.SSEEventError {
				return
			}
		}
	}
}

func mapCsAgentPreStreamError(err error) (int, string) {
	switch {
	case err == usecase.ErrCsAgentDisabled:
		return http.StatusServiceUnavailable, "cs agent disabled"
	case err == usecase.ErrCsAgentPrompt:
		return http.StatusBadRequest, "prompt is required"
	default:
		return http.StatusBadRequest, err.Error()
	}
}
