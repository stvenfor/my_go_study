package request

// CsAgentChatRequest POST /api/v1/cs-agent/chat
type CsAgentChatRequest struct {
	Prompt          string `json:"prompt"`
	ClientRequestID string `json:"clientRequestId"`
}
