// stream_provider.go LLM 流式生成 Provider 接口。
package provider

import (
	"context"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

// CompletionInput Provider 输入（含停留会话上下文）。
type CompletionInput struct {
	Prompt      string
	Messages    []entity.ChatTurn // 不含本轮 user；本轮在 Prompt
	Model       string
	MaxTokens   int
	Temperature float64
}

// StreamProvider 向 out 发送纯文本增量；ctx 取消时立即返回。
type StreamProvider interface {
	// Name 返回配置侧模型/提供者标识（写入 meta.model）。
	Name() string
	// Stream 流式生成；finishReason 为 stop/length；err 非空时 usecase 发 event:error。
	Stream(ctx context.Context, in CompletionInput, out chan<- string) (finishReason string, err error)
}
