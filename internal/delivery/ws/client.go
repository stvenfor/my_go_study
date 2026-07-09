// =============================================================================
// 文件：client.go
// 层级：Delivery/WS —— 代表「一条 WebSocket 连接」
//
// 【初学者】为什么 readPump 和 writePump 分开两个 goroutine？
//   WebSocket 全双工：读和写可以同时进行。
//   若在同一线程里读+写，写阻塞会导致无法及时处理 ping/pong。
//
// 【为什么 Send 用 channel 而不是直接 conn.Write？】
//   多个 goroutine（Hub 广播、handler 回 pong）可能同时写；
//   统一进 send 通道，由 writePump 单线程写出，避免并发写冲突。
// =============================================================================
package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"go.uber.org/zap"
)

// 心跳与超时常量（gorilla/websocket 官方推荐模式）
const (
	writeWait  = 10 * time.Second  // 单次写操作最长等待
	pongWait   = 60 * time.Second  // 多久没收到协议层 Pong 就认为断线
	pingPeriod = (pongWait * 9) / 10 // 略小于 pongWait，提前发 Ping
	maxMsgSize = 64 * 1024         // 单条 JSON 最大 64KB，防恶意大包
)

// Client 表示一个已建立的 WebSocket 连接及其会话状态。
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	log  *zap.Logger

	// send：待发送消息队列，缓冲 32 条，避免瞬时广播阻塞
	send chan entity.RealtimeEnvelope

	// auth 成功后由 handler 填充
	userID    string
	sessionID string // 本次 WS 会话 ID，便于日志追踪
	connID    string // 与换票时一致，区分同用户多连接

	// topics：该连接订阅的主题集合
	topics   map[string]struct{}
	topicsMu sync.RWMutex // 订阅读写可能并发，需要锁
}

// NewClient 在 HTTP Upgrade 成功后创建，此时尚未 auth。
func NewClient(hub *Hub, conn *websocket.Conn, log *zap.Logger) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		log:    log,
		send:   make(chan entity.RealtimeEnvelope, 32),
		topics: make(map[string]struct{}),
	}
}

// IsSubscribed 广播前检查：只有订阅了 topic 的连接才收 event。
func (c *Client) IsSubscribed(topic string) bool {
	c.topicsMu.RLock()
	defer c.topicsMu.RUnlock()
	_, ok := c.topics[topic]
	return ok
}

// Subscribe 添加订阅，返回当前完整列表（用于 ack 回给客户端）。
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

// Send 非阻塞入队。队列满则丢弃并打日志，避免拖死 Hub。
func (c *Client) Send(envelope entity.RealtimeEnvelope) {
	select {
	case c.send <- envelope:
	default:
		c.log.Warn("ws send buffer full", zap.String("userId", c.userID))
	}
}

// readPump 循环读客户端 JSON，交给 handler 处理。
func (c *Client) readPump(handle func(*Client, entity.RealtimeEnvelope)) {
	defer func() {
		c.hub.Unregister(c) // 读循环结束 = 连接断开，必须从 Hub 移除
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMsgSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))

	// 收到 WebSocket 协议层 Pong 时刷新读超时（与 writePump 的 Ping 配合）
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return // 正常关闭或网络错误，defer 会 Unregister
		}

		var envelope entity.RealtimeEnvelope
		if err := json.Unmarshal(data, &envelope); err != nil {
			c.log.Warn("ws invalid json", zap.Error(err))
			continue // 单条坏 JSON 不断开连接
		}
		handle(c, envelope)
	}
}

// writePump 从 send 通道取消息写出，并定时发协议层 Ping。
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
				// send 通道被 Hub 关闭，发 Close 帧优雅退出
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
			// 协议层 Ping（不是 JSON type:ping），防止 NAT 静默断 TCP
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// CloseWithCode auth 失败等场景，带自定义关闭码断开。
func (c *Client) CloseWithCode(code int, reason string) {
	msg := websocket.FormatCloseMessage(code, reason)
	_ = c.conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(writeWait))
	_ = c.conn.Close()
}
