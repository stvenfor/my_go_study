package request

// RealtimeTicketRequest WS 换票请求。
type RealtimeTicketRequest struct {
	Platform string `json:"platform"`
	ConnID   string `json:"connId"`
}

// RealtimeSyncRequest 重连增量同步请求。
type RealtimeSyncRequest struct {
	SinceSeq int64    `json:"sinceSeq"`
	Topics   []string `json:"topics"`
}

// RealtimePushRequest 开发环境推送测试通知。
type RealtimePushRequest struct {
	UserID string         `json:"userId"`
	Topic  string         `json:"topic"`
	Title  string         `json:"title"`
	Body   string         `json:"body"`
	Name   string         `json:"name"`
	Extra  map[string]any `json:"extra"`
}
