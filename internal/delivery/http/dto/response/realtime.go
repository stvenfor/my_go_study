package response

import "github.com/stvenfor/my_go_study/internal/domain/entity"

// RealtimeTicketData WS 换票响应（Flutter WsTicketResult 对齐）。
type RealtimeTicketData struct {
	Ticket           string `json:"ticket"`
	WSURL            string `json:"wsUrl"`
	ExpiresInSeconds int    `json:"expiresInSeconds"`
	ConnID           string `json:"connId"`
}

// RealtimeSyncData 增量同步响应（Flutter WsSyncResult 对齐）。
type RealtimeSyncData struct {
	Events    []entity.RealtimeEnvelope `json:"events"`
	LatestSeq int64                     `json:"latestSeq"`
}

// RealtimePushData 推送调试响应。
type RealtimePushData struct {
	Envelope  entity.RealtimeEnvelope `json:"envelope"`
	Delivered int                     `json:"delivered"`
}
