// =============================================================================
// 文件：hub.go
// 层级：Delivery/WS —— 内存中的「连接管理中心」
//
// 【初学者】Hub 解决什么问题？
//   WebSocket 连接散落在各个 goroutine，Hub 统一登记：
//   - 谁在线（哪个 userID 有哪些连接）
//   - 推消息时找得到目标连接
//
// 【为什么用 map[*Client]struct{} 而不是 []Client？】
//   用指针当 key，Register/Unregister O(1)；struct{} 不占额外内存。
// =============================================================================
package ws

import (
	"sync"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

// Hub 管理所有已建立的 WebSocket 连接。
type Hub struct {
	mu sync.RWMutex // 读写锁：读多写少（广播时读，注册/注销时写）

	// clients：全部连接集合，用于「广播给所有人（除某人）」
	clients map[*Client]struct{}

	// byUser：userID → 该用户的所有连接（一人多设备时会多条）
	byUser map[string]map[*Client]struct{}
}

// NewHub 创建空 Hub。在 main.go 启动时 new 一次，全局共享。
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Client]struct{}),
		byUser:  make(map[string]map[*Client]struct{}),
	}
}

// Register 在 auth 成功后调用，把连接纳入管理。
func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client] = struct{}{}

	// auth 完成前 userID 为空，只进 clients 不进 byUser
	if client.userID == "" {
		return
	}
	if h.byUser[client.userID] == nil {
		h.byUser[client.userID] = make(map[*Client]struct{})
	}
	h.byUser[client.userID][client] = struct{}{}
}

// Unregister 连接断开时调用，防止向已关闭连接写数据。
func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.clients, client)

	if client.userID != "" {
		if set := h.byUser[client.userID]; set != nil {
			delete(set, client)
			if len(set) == 0 {
				delete(h.byUser, client.userID) // 该用户无在线连接了
			}
		}
	}

	// 关闭 send 通道 → writePump 退出 → TCP 连接释放
	close(client.send)
}

// BroadcastToUser 向**指定用户**的所有连接推送（需已 sub 对应 topic）。
// 返回 delivered：实际送达的连接数（用于 push 接口调试）。
func (h *Hub) BroadcastToUser(userID, topic string, envelope entity.RealtimeEnvelope) int {
	// 先复制 client 列表再解锁，避免 Send 时长时间持锁
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

// BroadcastToTopicExcept 向**除 excludeUserID 外**、且订阅了 topic 的所有连接推送。
// 用于 presence：A 上报在线，B/C 应收到，A 不需要收到自己的广播。
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
