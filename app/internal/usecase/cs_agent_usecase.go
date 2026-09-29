// cs_agent_usecase.go 客服 Agent：Eino Tool + SSE 帧（独立于小石头 completions）。
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
	"github.com/google/uuid"
	"model/entity"
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
}

// NewCsAgentUsecase 创建用例。
func NewCsAgentUsecase(searcher csagent.TransactionSearcher, cfg config.CsAgentConfig) *CsAgentUsecase {
	return &CsAgentUsecase{searcher: searcher, cfg: cfg}
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

	toolJSON, toolErr := u.runSearchTool(ctx, userID, prompt)
	if toolErr != nil && !errors.Is(toolErr, shared.ErrSearchUnavailable) {
		send(entity.SSEEventError, entity.SSEEventData{
			Error: &entity.SSEErrorBody{Code: entity.SSEErrorInternal, Message: toolErr.Error()},
		})
		return
	}

	answer := buildCsAnswer(prompt, toolJSON, toolErr)
	delay := u.cfg.ChunkDelay()
	var built strings.Builder
	for _, r := range answer {
		if ctx.Err() != nil {
			send(entity.SSEEventError, entity.SSEEventData{
				Error: &entity.SSEErrorBody{Code: entity.SSEErrorCanceled, Message: "canceled"},
			})
			return
		}
		piece := string(r)
		built.WriteString(piece)
		send(entity.SSEEventDelta, entity.SSEEventData{Text: piece, Model: model})
		if delay > 0 {
			select {
			case <-ctx.Done():
				send(entity.SSEEventError, entity.SSEEventData{
					Error: &entity.SSEErrorBody{Code: entity.SSEErrorCanceled, Message: "canceled"},
				})
				return
			case <-time.After(delay):
			}
		}
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
