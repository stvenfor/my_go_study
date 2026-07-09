package ws

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/usecase"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Handler WebSocket 网关。
type Handler struct {
	hub        *Hub
	ticketUC   *usecase.RealtimeTicketUsecase
	presenceUC *usecase.RealtimePresenceUsecase
	log        *zap.Logger
}

// NewHandler 创建 WS Handler。
func NewHandler(
	hub *Hub,
	ticketUC *usecase.RealtimeTicketUsecase,
	presenceUC *usecase.RealtimePresenceUsecase,
	log *zap.Logger,
) *Handler {
	return &Handler{hub: hub, ticketUC: ticketUC, presenceUC: presenceUC, log: log}
}

// Hub 返回 Hub（供 push usecase 注入）。
func (h *Handler) Hub() *Hub {
	return h.hub
}

// ServeWS 升级 HTTP 为 WebSocket。
func (h *Handler) ServeWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.log.Warn("ws upgrade failed", zap.Error(err))
		return
	}

	client := NewClient(h.hub, conn, h.log)
	go client.writePump()
	go client.readPump(h.handleEnvelope)
}

func (h *Handler) handleEnvelope(client *Client, envelope entity.RealtimeEnvelope) {
	switch envelope.Type {
	case "auth":
		h.handleAuth(client, envelope)
	case "ping":
		client.Send(entity.RealtimeEnvelope{
			ID:   envelope.ID,
			Type: "pong",
			TS:   time.Now().UnixMilli(),
		})
	case "sub":
		topics := stringSlice(envelope.Payload["topics"])
		active := client.Subscribe(topics)
		h.sendAck(client, envelope.ID, map[string]any{"topics": active})
	case "unsub":
		topics := stringSlice(envelope.Payload["topics"])
		active := client.Unsubscribe(topics)
		h.sendAck(client, envelope.ID, map[string]any{"topics": active})
	case "event":
		h.handleEvent(client, envelope)
	default:
		h.log.Debug("ws ignore type", zap.String("type", envelope.Type))
	}
}

func (h *Handler) handleAuth(client *Client, envelope entity.RealtimeEnvelope) {
	ticket, _ := envelope.Payload["ticket"].(string)
	if ticket == "" {
		client.Send(entity.RealtimeEnvelope{
			Type: "error",
			TS:   time.Now().UnixMilli(),
			Payload: map[string]any{
				"code": entity.WSCloseAuthFailed,
				"message": "missing ticket",
			},
		})
		client.CloseWithCode(entity.WSCloseAuthFailed, "auth failed")
		return
	}

	data, err := h.ticketUC.Consume(context.Background(), ticket)
	if err != nil {
		client.Send(entity.RealtimeEnvelope{
			Type: "error",
			TS:   time.Now().UnixMilli(),
			Payload: map[string]any{
				"code":    entity.WSCloseTicketExpired,
				"message": err.Error(),
			},
		})
		client.CloseWithCode(entity.WSCloseTicketExpired, "ticket expired")
		return
	}

	client.userID = data.UserID
	client.connID = data.ConnID
	if client.connID == "" {
		if cid, ok := envelope.Payload["connId"].(string); ok {
			client.connID = cid
		}
	}
	client.sessionID = uuid.NewString()
	h.hub.Register(client)

	client.Send(entity.RealtimeEnvelope{
		ID:   fmt.Sprintf("auth_ok_%d", time.Now().UnixMilli()),
		Type: "auth_ok",
		TS:   time.Now().UnixMilli(),
		Payload: map[string]any{
			"userId":     client.userID,
			"sessionId":  client.sessionID,
			"serverTime": time.Now().UnixMilli(),
		},
	})
}

func (h *Handler) handleEvent(client *Client, envelope entity.RealtimeEnvelope) {
	defer h.sendAck(client, envelope.ID, map[string]any{"accepted": true})

	if client.userID == "" {
		h.log.Warn("ws event before auth", zap.String("type", envelope.Type))
		return
	}
	if h.presenceUC == nil {
		return
	}

	name, _ := envelope.Payload["name"].(string)
	if envelope.Topic != entity.TopicPresenceBulk || name != entity.EventPresenceReport {
		return
	}

	online := true
	if v, ok := envelope.Payload["online"].(bool); ok {
		online = v
	}
	device, _ := envelope.Payload["device"].(string)

	out, delivered, err := h.presenceUC.Report(context.Background(), usecase.RealtimePresenceInput{
		UserID: client.userID,
		Online: online,
		Device: device,
	})
	if err != nil {
		h.log.Warn("presence report failed", zap.Error(err), zap.String("userId", client.userID))
		return
	}
	h.log.Info("presence broadcast",
		zap.String("userId", client.userID),
		zap.Bool("online", online),
		zap.Int("delivered", delivered),
		zap.String("eventId", out.ID),
	)
}

func (h *Handler) sendAck(client *Client, refID string, payload map[string]any) {
	if refID != "" {
		payload["refId"] = refID
	}
	client.Send(entity.RealtimeEnvelope{
		ID:      fmt.Sprintf("ack_%d", time.Now().UnixMilli()),
		Type:    "ack",
		TS:      time.Now().UnixMilli(),
		Payload: payload,
	})
}

func stringSlice(v any) []string {
	list, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok && s != "" {
			out = append(out, s)
		}
	}
	return out
}
