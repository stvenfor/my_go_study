package request

// CompletionRequest POST /api/v1/sse/completions 请求体。
type CompletionRequest struct {
	Prompt          string             `json:"prompt"`
	ConversationID  *string            `json:"conversationId"`
	ClientRequestID string             `json:"clientRequestId"`
	Options         *CompletionOptions `json:"options"`
}

// CompletionOptions 可选生成参数。
type CompletionOptions struct {
	Model       string   `json:"model"`
	Temperature *float64 `json:"temperature"`
	MaxTokens   *int     `json:"maxTokens"`
}

// ConversationIDValue 返回 conversationId（空表示首句）。
func (r CompletionRequest) ConversationIDValue() string {
	if r.ConversationID == nil {
		return ""
	}
	return *r.ConversationID
}
