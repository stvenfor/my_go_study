package ws

import (
	"sync"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

// Hub 管理 WebSocket 连接与 topic 订阅。
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
	byUser  map[string]map[*Client]struct{}
}

// NewHub 创建 Hub。
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Client]struct{}),
		byUser:  make(map[string]map[*Client]struct{}),
	}
}

// Register 注册已认证客户端。
func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client] = struct{}{}
	if client.userID == "" {
		return
	}
	if h.byUser[client.userID] == nil {
		h.byUser[client.userID] = make(map[*Client]struct{})
	}
	h.byUser[client.userID][client] = struct{}{}
}

// Unregister 移除客户端。
func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, client)
	if client.userID != "" {
		if set := h.byUser[client.userID]; set != nil {
			delete(set, client)
			if len(set) == 0 {
				delete(h.byUser, client.userID)
			}
		}
	}
	close(client.send)
}

// BroadcastToUser 向用户所有连接推送 topic 事件。
func (h *Hub) BroadcastToUser(userID, topic string, envelope entity.RealtimeEnvelope) int {
	h.mu.RLock()
	set := h.byUser[userID]
	clients := make([]*Client, 0, len(set))
	for c := range set {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	delivered := 0
	for _, c := range clients {
		if c.IsSubscribed(topic) {
			c.Send(envelope)
			delivered++
		}
	}
	return delivered
}

// BroadcastToTopicExcept 向订阅 topic 的所有连接推送，排除指定用户（用于 presence 等广播）。
func (h *Hub) BroadcastToTopicExcept(excludeUserID, topic string, envelope entity.RealtimeEnvelope) int {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for c := range h.clients {
		if c.userID != "" && c.userID != excludeUserID {
			clients = append(clients, c)
		}
	}
	h.mu.RUnlock()

	delivered := 0
	for _, c := range clients {
		if c.IsSubscribed(topic) {
			c.Send(envelope)
			delivered++
		}
	}
	return delivered
}
