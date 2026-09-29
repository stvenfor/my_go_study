// sse_event.go SSE 流式帧契约（与 Flutter SseClient / AiStreamEvent 对齐）。
package entity

// SSE 事件名（event: 行）。
const (
	SSEEventMeta  = "meta"
	SSEEventDelta = "delta"
	SSEEventDone  = "done"
	SSEEventError = "error"
)

// SSE 帧 type 字段（与 event 名一致）。
const (
	SSETypeMeta  = "meta"
	SSETypeDelta = "delta"
	SSETypeDone  = "done"
	SSETypeError = "error"
)

// FinishReason 正常结束原因。
const (
	FinishReasonStop   = "stop"
	FinishReasonLength = "length"
)

// 稳定错误码（流中 error.code 或映射）。
const (
	SSEErrorUpstreamTimeout     = "upstream_timeout"
	SSEErrorUpstream4xx         = "upstream_4xx"
	SSEErrorProviderUnavailable = "provider_unavailable"
	SSEErrorCanceled            = "canceled"
	SSEErrorInternal            = "internal"
)

// SSEErrorBody 流中错误载荷。
type SSEErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// SSEEventData 单帧 data JSON（单行）。
type SSEEventData struct {
	RequestID       string        `json:"requestId"`
	Seq             int           `json:"seq"`
	Type            string        `json:"type"`
	Text            string        `json:"text,omitempty"`
	ConversationID  string        `json:"conversationId,omitempty"`
	Model           string        `json:"model,omitempty"`
	ClientRequestID string        `json:"clientRequestId,omitempty"`
	FinishReason    *string       `json:"finishReason,omitempty"`
	Error           *SSEErrorBody `json:"error,omitempty"`
}

// SSEFrame 控制器写出的一帧。
type SSEFrame struct {
	Event string
	Data  SSEEventData
}

// ChatRole 对话角色。
const (
	ChatRoleUser      = "user"
	ChatRoleAssistant = "assistant"
)

// ChatTurn 停留会话中的一轮消息。
type ChatTurn struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
