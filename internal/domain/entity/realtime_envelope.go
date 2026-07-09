// =============================================================================
// 文件：realtime_envelope.go
// 层级：Domain（领域层）—— 只定义「消息长什么样」，不依赖 HTTP/Redis/WebSocket
//
// 【初学者】为什么单独放一个文件？
//   Go 后端和 Flutter 客户端通过 JSON 通信，两边字段必须一致。
//   把结构体放在 domain/entity，是全项目共用的「契约」。
// =============================================================================
package entity

// RealtimeEnvelope 是 WebSocket 上每一条 JSON 消息的外壳（包络）。
//
// 类比：信封上写「收件人(topic)」「信件类型(type)」，信纸内容是 payload。
type RealtimeEnvelope struct {
	// ID：客户端生成的消息编号。
	// 用途：ping 和 pong 必须用同一个 id 配对；ack 通过 refId 指回原始 id。
	ID string `json:"id,omitempty"`

	// Type：消息类型，决定服务端/客户端如何处理。
	// 常见值：auth / auth_ok / ping / pong / sub / unsub / ack / event / error
	Type string `json:"type"`

	// Topic：发布订阅主题，仅 event 类型需要。
	// 例：sys.notify（系统通知）、presence.bulk（在线状态广播）
	Topic string `json:"topic,omitempty"`

	// TS：毫秒时间戳，便于日志与调试排序。
	TS int64 `json:"ts,omitempty"`

	// Seq：按用户单调递增的序号，用于断线 sync 后去重。
	// 为什么按用户而不是全局？每个用户有自己的事件流，互不干扰。
	Seq int64 `json:"seq,omitempty"`

	// Payload：具体业务字段，不同 type/event 内容不同。
	// 为什么用 map 而不是固定 struct？扩展新事件时不用改结构体。
	Payload map[string]any `json:"payload,omitempty"`
}

// Topic 常量：与 Flutter RealtimeTopics 保持一致。
// 用常量而不是手写字符串，避免拼写错误导致订阅失败。
const (
	TopicSysNotify    = "sys.notify"     // 系统通知
	TopicPresenceBulk = "presence.bulk"  // 在线状态批量频道
)

// EventName 常量：放在 payload.name 里，用于区分同一 topic 下的不同事件。
const (
	EventSysNotifyShow  = "sys.notify.show"  // 展示 Banner 通知
	EventPresenceReport = "presence.report"  // 客户端 → 服务端：上报在线
	EventPresenceUpdate = "presence.update"  // 服务端 → 其他客户端：某用户状态变更
)

// WebSocket 关闭码：与 Flutter RealtimeConfig 对齐。
// 为什么自定义 4001/4003？客户端可根据 code 决定是重换票还是提示用户。
const (
	WSCloseAuthFailed    = 4001 // auth 缺少 ticket
	WSCloseKicked        = 4002 // 预留：被踢下线
	WSCloseTicketExpired = 4003 // ticket 无效、过期或已使用
)
