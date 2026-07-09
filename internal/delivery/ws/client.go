package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"go.uber.org/zap"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	maxMsgSize = 64 * 1024
)

// Client WebSocket 客户端连接。
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	log       *zap.Logger
	send      chan entity.RealtimeEnvelope
	userID    string
	sessionID string
	connID    string
	topics    map[string]struct{}
	topicsMu  sync.RWMutex
}

// NewClient 创建客户端。
func NewClient(hub *Hub, conn *websocket.Conn, log *zap.Logger) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		log:    log,
		send:   make(chan entity.RealtimeEnvelope, 32),
		topics: make(map[string]struct{}),
	}
}

// IsSubscribed 是否订阅 topic。
func (c *Client) IsSubscribed(topic string) bool {
	c.topicsMu.RLock()
	defer c.topicsMu.RUnlock()
	_, ok := c.topics[topic]
	return ok
}

// Subscribe 订阅 topics。
func (c *Client) Subscribe(topics []string) []string {
	c.topicsMu.Lock()
	defer c.topicsMu.Unlock()
	for _, t := range topics {
		c.topics[t] = struct{}{}
	}
	out := make([]string, 0, len(c.topics))
	for t := range c.topics {
		out = append(out, t)
	}
	return out
}

// Unsubscribe 取消订阅。
func (c *Client) Unsubscribe(topics []string) []string {
	c.topicsMu.Lock()
	defer c.topicsMu.Unlock()
	for _, t := range topics {
		delete(c.topics, t)
	}
	out := make([]string, 0, len(c.topics))
	for t := range c.topics {
		out = append(out, t)
	}
	return out
}

// Send 非阻塞发送 envelope。
func (c *Client) Send(envelope entity.RealtimeEnvelope) {
	select {
	case c.send <- envelope:
	default:
		c.log.Warn("ws send buffer full", zap.String("userId", c.userID))
	}
}

// readPump 读取客户端消息。
func (c *Client) readPump(handle func(*Client, entity.RealtimeEnvelope)) {
	defer func() {
		c.hub.Unregister(c)
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMsgSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.log.Debug("ws read closed", zap.Error(err))
			}
			return
		}
		var envelope entity.RealtimeEnvelope
		if err := json.Unmarshal(data, &envelope); err != nil {
			c.log.Warn("ws invalid json", zap.Error(err))
			continue
		}
		handle(c, envelope)
	}
}

// writePump 写入服务端消息。
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case envelope, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			data, err := json.Marshal(envelope)
			if err != nil {
				continue
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// CloseWithCode 带 code 关闭连接。
func (c *Client) CloseWithCode(code int, reason string) {
	msg := websocket.FormatCloseMessage(code, reason)
	_ = c.conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(writeWait))
	_ = c.conn.Close()
}
