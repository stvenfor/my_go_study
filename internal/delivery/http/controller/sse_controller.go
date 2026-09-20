// sse_controller.go SSE completions HTTP 控制器。
package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/request"
	"github.com/stvenfor/my_go_study/internal/delivery/http/dto/response"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/usecase"
	"github.com/stvenfor/my_go_study/pkg/config"
)

// SseController SSE 生成流控制器。
type SseController struct {
	uc  *usecase.CompletionUsecase
	cfg config.SSEConfig
}

// NewSseController 创建控制器。
func NewSseController(uc *usecase.CompletionUsecase, cfg config.SSEConfig) *SseController {
	return &SseController{uc: uc, cfg: cfg}
}

// Completions POST /api/v1/sse/completions
func (ctrl *SseController) Completions(c *gin.Context) {
	user, _, ok := supabaseAuthContext(c)
	if !ok {
		response.BackendError(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req request.CompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BackendError(c, http.StatusBadRequest, "invalid request")
		return
	}

	ucReq := usecase.CompletionRequest{
		Prompt:          req.Prompt,
		ConversationID:  req.ConversationIDValue(),
		ClientRequestID: req.ClientRequestID,
	}
	if req.Options != nil {
		ucReq.Model = req.Options.Model
		if req.Options.Temperature != nil {
			ucReq.Temperature = *req.Options.Temperature
		}
		if req.Options.MaxTokens != nil {
			ucReq.MaxTokens = *req.Options.MaxTokens
		}
	}

	streamCtx := c.Request.Context()
	if timeout := ctrl.cfg.RequestTimeout(); timeout > 0 {
		var cancel context.CancelFunc
		streamCtx, cancel = context.WithTimeout(streamCtx, timeout)
		defer cancel()
	}

	ch, err := ctrl.uc.Stream(streamCtx, user.ID, ucReq)
	if err != nil {
		status, msg := mapCompletionPreStreamError(err)
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

func writeSSEFrame(c *gin.Context, frame entity.SSEFrame) error {
	raw, err := json.Marshal(frame.Data)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", frame.Event, raw)
	return err
}

func mapCompletionPreStreamError(err error) (int, string) {
	switch {
	case errors.Is(err, usecase.ErrSSEDisabled):
		return http.StatusServiceUnavailable, "sse disabled"
	case errors.Is(err, usecase.ErrPromptEmpty):
		return http.StatusBadRequest, "prompt is required"
	case errors.Is(err, usecase.ErrPromptTooLarge):
		return http.StatusBadRequest, "prompt too large"
	case errors.Is(err, usecase.ErrRateLimited):
		return http.StatusTooManyRequests, "rate limit exceeded"
	case errors.Is(err, usecase.ErrConversationInvalid):
		return http.StatusBadRequest, "conversation not found or expired"
	case errors.Is(err, usecase.ErrProviderUnavailable):
		return http.StatusServiceUnavailable, "provider unavailable"
	default:
		return http.StatusInternalServerError, err.Error()
	}
}
