// cs_agent_usecase.go 客服 Agent：支持 RAG（本地 KB 检索 + LLM 生成）与模板回落。
package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"app/internal/repository/csagent"
	"app/internal/shared"
	"common/config"
	"common/metrics"
	"github.com/google/uuid"
	"model/entity"
	"model/provider"
)

var (
	ErrCsAgentDisabled = errors.New("cs agent disabled")
	ErrCsAgentPrompt   = errors.New("prompt is required")
)

// CsAgentRequest 客服对话入参。
type CsAgentRequest struct {
	Prompt          string
	ClientRequestID string
}

// CsAgentUsecase 客服 Agent 用例。
type CsAgentUsecase struct {
	searcher csagent.TransactionSearcher // optional
	cfg      config.CsAgentConfig
	retriever provider.Retriever   // optional：RAG 检索
	llm      provider.StreamProvider // optional：RAG 生成
}

// CsAgentOption 可选依赖注入（保持向后兼容的 2 参构造）。
type CsAgentOption func(*CsAgentUsecase)

// WithRetriever 注入 RAG 检索器。
func WithRetriever(r provider.Retriever) CsAgentOption {
	return func(u *CsAgentUsecase) { u.retriever = r }
}

// WithLLM 注入 LLM 流式生成器（RAG 生成阶段使用）。
func WithLLM(p provider.StreamProvider) CsAgentOption {
	return func(u *CsAgentUsecase) { u.llm = p }
}

// NewCsAgentUsecase 创建用例。变参保持兼容：不传 opts 时与旧 2 参签名行为一致（模板回落）。
func NewCsAgentUsecase(searcher csagent.TransactionSearcher, cfg config.CsAgentConfig, opts ...CsAgentOption) *CsAgentUsecase {
	u := &CsAgentUsecase{searcher: searcher, cfg: cfg}
	for _, o := range opts {
		o(u)
	}
	return u
}

// Stream 返回 meta → delta* → done|error。
func (u *CsAgentUsecase) Stream(ctx context.Context, userID string, req CsAgentRequest) (<-chan entity.SSEFrame, error) {
	if !u.cfg.Enabled {
		return nil, ErrCsAgentDisabled
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, ErrCsAgentPrompt
	}
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("userId required")
	}

	requestID := uuid.NewString()
	clientReqID := strings.TrimSpace(req.ClientRequestID)
	model := u.cfg.ModelName()

	ch := make(chan entity.SSEFrame, 16)
	go u.run(ctx, ch, userID, prompt, requestID, clientReqID, model)
	return ch, nil
}

func (u *CsAgentUsecase) run(
	ctx context.Context,
	ch chan<- entity.SSEFrame,
	userID, prompt, requestID, clientReqID, model string,
) {
	defer close(ch)
	seq := 0
	send := func(event string, data entity.SSEEventData) {
		seq++
		data.RequestID = requestID
		data.Seq = seq
		data.Type = event
		if clientReqID != "" {
			data.ClientRequestID = clientReqID
		}
		select {
		case <-ctx.Done():
		case ch <- entity.SSEFrame{Event: event, Data: data}:
		}
	}

	send(entity.SSEEventMeta, entity.SSEEventData{
		Model: model,
		Text:  "cs-agent",
	})

	// 优先走 RAG（检索 + LLM 生成）；任何环节失败回落模板，保证可用。
	if u.generateRAG(ctx, ch, send, userID, prompt, model) {
		return
	}

	toolJSON, toolErr := u.runSearchTool(ctx, userID, prompt)
	if toolErr != nil && !errors.Is(toolErr, shared.ErrSearchUnavailable) {
		send(entity.SSEEventError, entity.SSEEventData{
			Error: &entity.SSEErrorBody{Code: entity.SSEErrorInternal, Message: toolErr.Error()},
		})
		return
	}

	answer := buildCsAnswer(prompt, toolJSON, toolErr)
	// 模板拼接 + 伪流式：未调用任何 LLM，但对外是 200/SSE 成功语义，故计数可见（parity-p0-5）。
	metrics.DegradedPathsTotal.WithLabelValues("cs_agent_template").Inc()
	u.streamText(ch, send, answer, model)
}

// generateRAG 检索 KB 并通过 LLM 流式生成；返回 true 表示已处理（含发送 done）。
// 检索或 LLM 失败（含 provider_unavailable）时返回 false，交由调用方回落模板。
func (u *CsAgentUsecase) generateRAG(
	ctx context.Context,
	ch chan<- entity.SSEFrame,
	send func(string, entity.SSEEventData),
	userID, prompt, model string,
) bool {
	if u.retriever == nil || u.llm == nil {
		return false
	}
	chunks, rErr := u.retriever.Retrieve(ctx, prompt, 0)
	if rErr != nil || ctx.Err() != nil {
		return false
	}
	toolJSON, toolErr := u.runSearchTool(ctx, userID, prompt)
	userMsg := buildRAGUserPrompt(prompt, chunks, toolJSON, toolErr)

	out := make(chan string, 64)
	var streamErr error
	go func() {
		_, streamErr = u.llm.Stream(ctx, provider.CompletionInput{
			Prompt: userMsg,
			Model:  model,
		}, out)
		close(out)
	}()

	emitted := false
	for piece := range out {
		if ctx.Err() != nil {
			send(entity.SSEEventError, entity.SSEEventData{
				Error: &entity.SSEErrorBody{Code: entity.SSEErrorCanceled, Message: "canceled"},
			})
			return true
		}
		if piece == "" {
			continue
		}
		emitted = true
		send(entity.SSEEventDelta, entity.SSEEventData{Text: piece, Model: model})
	}
	if streamErr != nil {
		return false
	}
	if !emitted {
		return false
	}
	reason := entity.FinishReasonStop
	send(entity.SSEEventDone, entity.SSEEventData{
		Model:        model,
		FinishReason: &reason,
	})
	return true
}

// streamText 逐字符伪流式（模板回落路径）。
func (u *CsAgentUsecase) streamText(ch chan<- entity.SSEFrame, send func(string, entity.SSEEventData), answer, model string) {
	delay := u.cfg.ChunkDelay()
	var built strings.Builder
	for _, r := range answer {
		if delay > 0 {
			select {
			case <-time.After(delay):
			}
		}
		piece := string(r)
		built.WriteString(piece)
		send(entity.SSEEventDelta, entity.SSEEventData{Text: piece, Model: model})
	}
	_ = utf8.RuneCountInString(built.String())
	reason := entity.FinishReasonStop
	send(entity.SSEEventDone, entity.SSEEventData{
		Model:        model,
		FinishReason: &reason,
	})
}

func (u *CsAgentUsecase) runSearchTool(ctx context.Context, userID, prompt string) (string, error) {
	if !shouldSearchTransactions(prompt) {
		return "", nil
	}
	t, err := csagent.NewTransactionSearchTool(u.searcher, userID)
	if err != nil {
		return "", err
	}
	args, _ := json.Marshal(map[string]any{"query": prompt, "limit": 5})
	return t.InvokableRun(ctx, string(args))
}

func shouldSearchTransactions(prompt string) bool {
	p := strings.ToLower(prompt)
	keys := []string{"交易", "收支", "账单", "消费", "收入", "支出", "search", "查", "找"}
	for _, k := range keys {
		if strings.Contains(p, k) {
			return true
		}
	}
	return false
}

// buildRAGUserPrompt 把检索到的知识块与（可选）交易结果拼成 LLM 用户消息。
func buildRAGUserPrompt(prompt string, chunks []provider.RetrievedChunk, toolJSON string, toolErr error) string {
	var b strings.Builder
	b.WriteString("请基于以下「知识库」内容回答用户问题。若知识库未覆盖，请基于常识礼貌作答，不要编造。\n\n")
	b.WriteString("【知识库】\n")
	if len(chunks) == 0 {
		b.WriteString("（无相关检索结果）\n")
	}
	for i, c := range chunks {
		title := c.Title
		if title == "" {
			title = c.SourceFile
		}
		b.WriteString(fmt.Sprintf("[%d] (%s) %s\n", i+1, title, c.Content))
	}
	if strings.TrimSpace(toolJSON) != "" {
		b.WriteString("\n【用户交易检索】\n")
		b.WriteString(toolJSON)
		b.WriteString("\n")
	} else if toolErr != nil {
		b.WriteString("\n【用户交易检索】当前不可用。\n")
	}
	b.WriteString("\n【用户问题】\n")
	b.WriteString(prompt)
	return b.String()
}

func buildCsAnswer(prompt, toolJSON string, toolErr error) string {
	var b strings.Builder
	b.WriteString("【客服助手】您好，我是业务客服 Agent（Eino Tool）。\n")
	if toolErr != nil {
		b.WriteString("交易搜索当前不可用，请稍后再试或改用交易列表页。\n")
		b.WriteString("您的问题：")
		b.WriteString(prompt)
		return b.String()
	}
	if strings.TrimSpace(toolJSON) != "" {
		b.WriteString("已根据您的问题检索交易：\n")
		b.WriteString(toolJSON)
		b.WriteString("\n如需更多帮助，请补充时间范围或备注关键词。")
		return b.String()
	}
	b.WriteString("我可以帮您查询个人收支交易（试着说「查一下餐饮支出」）。\n")
	b.WriteString("您的问题：")
	b.WriteString(prompt)
	return b.String()
}
