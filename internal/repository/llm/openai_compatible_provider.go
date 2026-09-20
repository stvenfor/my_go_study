// openai_compatible_provider.go OpenAI 兼容 Chat Completions 流式 Provider。
// 无 API Key 时 Stream 返回 provider_unavailable（装配仍可成功，默认路径用 mock）。
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/provider"
)

// OpenAICompatibleConfig 上游配置。
type OpenAICompatibleConfig struct {
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
}

// OpenAICompatibleProvider OpenAI 兼容 SSE 上游。
type OpenAICompatibleProvider struct {
	cfg    OpenAICompatibleConfig
	client *http.Client
}

// NewOpenAICompatibleProvider 创建 Provider（允许 apiKey 为空，Stream 时明确失败）。
func NewOpenAICompatibleProvider(cfg OpenAICompatibleConfig) *OpenAICompatibleProvider {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = "gpt-4o-mini"
	}
	cfg.BaseURL = base
	cfg.Model = model
	cfg.Timeout = timeout
	return &OpenAICompatibleProvider{
		cfg: cfg,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Name 实现 StreamProvider。
func (p *OpenAICompatibleProvider) Name() string {
	return p.cfg.Model
}

// Stream 调用上游 chat/completions?stream=true。
func (p *OpenAICompatibleProvider) Stream(ctx context.Context, in provider.CompletionInput, out chan<- string) (string, error) {
	if strings.TrimSpace(p.cfg.APIKey) == "" {
		return "", &ProviderError{
			Code:    entity.SSEErrorProviderUnavailable,
			Message: "openai_compatible api key not configured (set SSE_OPENAI_API_KEY)",
		}
	}

	messages := make([]map[string]string, 0, len(in.Messages)+2)
	messages = append(messages, map[string]string{
		"role":    "system",
		"content": "你是 AI小石头，本 App / 4S 场景的业务向导。用简洁纯文本回答如何使用 App 功能，不要输出 Markdown。",
	})
	for _, turn := range in.Messages {
		role := turn.Role
		if role != entity.ChatRoleUser && role != entity.ChatRoleAssistant {
			continue
		}
		messages = append(messages, map[string]string{"role": role, "content": turn.Content})
	}
	messages = append(messages, map[string]string{"role": "user", "content": in.Prompt})

	model := in.Model
	if model == "" {
		model = p.cfg.Model
	}
	body := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   true,
	}
	if in.MaxTokens > 0 {
		body["max_tokens"] = in.MaxTokens
	}
	if in.Temperature > 0 {
		body["temperature"] = in.Temperature
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.BaseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", &ProviderError{Code: entity.SSEErrorUpstreamTimeout, Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		code := entity.SSEErrorUpstream4xx
		if resp.StatusCode >= 500 {
			code = entity.SSEErrorProviderUnavailable
		}
		return "", &ProviderError{
			Code:    code,
			Message: fmt.Sprintf("upstream http %d: %s", resp.StatusCode, strings.TrimSpace(string(b))),
		}
	}

	scanner := bufio.NewScanner(resp.Body)
	// 增大 buffer 以防长行
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	finishReason := entity.FinishReasonStop
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}
		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		for _, choice := range chunk.Choices {
			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}
			delta := choice.Delta.Content
			if delta == "" {
				continue
			}
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case out <- delta:
			}
		}
	}
	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", &ProviderError{Code: entity.SSEErrorUpstreamTimeout, Message: err.Error()}
	}
	return finishReason, nil
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// ProviderError 可映射到 SSE error.code 的上游错误。
type ProviderError struct {
	Code    string
	Message string
}

func (e *ProviderError) Error() string {
	return e.Message
}

// CodeValue 供 usecase 映射稳定 error.code（避免循环依赖）。
func (e *ProviderError) CodeValue() string {
	if e == nil || e.Code == "" {
		return entity.SSEErrorInternal
	}
	return e.Code
}

var _ provider.StreamProvider = (*OpenAICompatibleProvider)(nil)
