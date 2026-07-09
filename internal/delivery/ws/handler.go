// =============================================================================
// 文件：handler.go
// 层级：Delivery/WS —— WebSocket 的「交通警察」
//
// 【初学者阅读路径】
//   ServeWS → readPump → handleEnvelope → handleAuth / handleEvent / ...
//
// 【为什么 WS 升级时不校验 JWT？】
//   鉴权放在首帧 auth + ticket，ticket 由已登录的 HTTP 接口签发，同样安全。
// =============================================================================
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

// upgrader 把 HTTP 请求升级为 WebSocket。
// CheckOrigin 返回 true：开发环境允许任意来源；生产应限制域名。
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Handler 依赖注入：Hub 管连接，Usecase 管业务，Logger 打日志。
type Handler struct {
	hub        *Hub
	ticketUC   *usecase.RealtimeTicketUsecase
	presenceUC *usecase.RealtimePresenceUsecase
	log        *zap.Logger
}

func NewHandler(
	hub *Hub,
	ticketUC *usecase.RealtimeTicketUsecase,
	presenceUC *usecase.RealtimePresenceUsecase,
	log *zap.Logger,
) *Handler {
	return &Handler{hub: hub, ticketUC: ticketUC, presenceUC: presenceUC, log: log}
}

// Hub 暴露给 main，供 PushUsecase 注入广播能力。
func (h *Handler) Hub() *Hub {
	return h.hub
}

// ServeWS 挂载在 GET /realtime/v1/connect，Gin 收到请求后调用。
func (h *Handler) ServeWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.log.Warn("ws upgrade failed", zap.Error(err))
		return
	}

	client := NewClient(h.hub, conn, h.log)
	// 两个 goroutine：读循环 + 写循环，这是 gorilla/websocket 标准写法
	go client.writePump()
	go client.readPump(h.handleEnvelope)
}

// handleEnvelope 根据 type 分发，类似 HTTP 路由。
func (h *Handler) handleEnvelope(client *Client, envelope entity.RealtimeEnvelope) {
	switch envelope.Type {
	case "auth":
		h.handleAuth(client, envelope)
	case "ping":
		// 应用层心跳：Flutter 每 25s 发 ping，必须回同 id 的 pong
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

// handleAuth 消费 Redis 中的一次性 ticket，完成 WS 鉴权。
func (h *Handler) handleAuth(client *Client, envelope entity.RealtimeEnvelope) {
	ticket, _ := envelope.Payload["ticket"].(string)
	if ticket == "" {
		client.Send(entity.RealtimeEnvelope{
			Type: "error",
			TS:   time.Now().UnixMilli(),
			Payload: map[string]any{
				"code":    entity.WSCloseAuthFailed,
				"message": "missing ticket",
			},
		})
		client.CloseWithCode(entity.WSCloseAuthFailed, "auth failed")
		return
	}

	// Consume = GETDEL，ticket 只能用一次，防重放
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
	h.hub.Register(client) // 此时才有 userID，进入 byUser 索引

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

// handleEvent 处理客户端上行的 event（当前实现 presence.report）。
func (h *Handler) handleEvent(client *Client, envelope entity.RealtimeEnvelope) {
	// defer：无论是否处理业务，都回 ack，让 Flutter 知道消息已收到
	defer h.sendAck(client, envelope.ID, map[string]any{"accepted": true})

	if client.userID == "" {
		h.log.Warn("ws event before auth", zap.String("type", envelope.Type))
		return
	}
	if h.presenceUC == nil {
		return
	}

	name, _ := envelope.Payload["name"].(string)
	// 只处理 presence.bulk + presence.report，其他 event 仅 ack
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

// sendAck 统一构造 ack 消息，refId 对应客户端原始消息的 id。
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

// stringSlice 把 JSON 数组 []any 转成 []string（JSON 解码后数字/字符串类型需断言）。
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
