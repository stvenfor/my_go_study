// realtime_envelope.go 与 Flutter RealtimeEnvelope 对齐的 WS 消息结构。
package entity

// RealtimeEnvelope WebSocket JSON 消息包络。
type RealtimeEnvelope struct {
	ID      string         `json:"id,omitempty"`
	Type    string         `json:"type"`
	Topic   string         `json:"topic,omitempty"`
	TS      int64          `json:"ts,omitempty"`
	Seq     int64          `json:"seq,omitempty"`
	Payload map[string]any `json:"payload,omitempty"`
}

// Realtime topics（与 Flutter RealtimeTopics 一致）。
const (
	TopicSysNotify    = "sys.notify"
	TopicPresenceBulk = "presence.bulk"
)

// WS close codes（与 Flutter RealtimeConfig 一致）。
const (
	WSCloseAuthFailed     = 4001
	WSCloseKicked         = 4002
	WSCloseTicketExpired  = 4003
)
