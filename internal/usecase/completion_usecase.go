// completion_usecase.go SSE completions 编排：限流、停留会话、Provider、事件序。
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/provider"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"github.com/stvenfor/my_go_study/pkg/config"
)

// Completion 预流错误（Controller 映射为 JSON {"error"}）。
var (
	ErrSSEDisabled         = errors.New("sse disabled")
	ErrPromptEmpty         = errors.New("prompt is required")
	ErrPromptTooLarge      = errors.New("prompt too large")
	ErrRateLimited         = errors.New("rate limit exceeded")
	ErrConversationInvalid = errors.New("conversation not found or expired")
	ErrProviderUnavailable = errors.New("provider unavailable")
)

// CompletionRequest 用例入参（与 HTTP DTO 解耦）。
type CompletionRequest struct {
	Prompt          string
	ConversationID  string
	ClientRequestID string
	Model           string
	Temperature     float64
	MaxTokens       int
}

// CompletionUsecase SSE 生成用例。
type CompletionUsecase struct {
	provider provider.StreamProvider
	convs    repository.ConversationRepository
	limiter  RateLimiter
	cfg      config.SSEConfig
}

// RateLimiter 按用户限流。
type RateLimiter interface {
	Allow(userID string) bool
}

// NewCompletionUsecase 创建用例。
func NewCompletionUsecase(
	p provider.StreamProvider,
	convs repository.ConversationRepository,
	limiter RateLimiter,
	cfg config.SSEConfig,
) *CompletionUsecase {
	if limiter == nil {
		limiter = NewMemoryRateLimiter(cfg.RateLimitPerUserPerMinute)
	}
	return &CompletionUsecase{
		provider: p,
		convs:    convs,
		limiter:  limiter,
		cfg:      cfg,
	}
}

// Stream 校验后启动后台生成，返回只读事件通道。
// 预流错误以 error 返回（勿写 SSE 头）；通道以 meta → delta* → done|error 结束并关闭。
func (u *CompletionUsecase) Stream(ctx context.Context, userID string, req CompletionRequest) (<-chan entity.SSEFrame, error) {
	if !u.cfg.Enabled {
		return nil, ErrSSEDisabled
	}
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("userId required")
	}

	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, ErrPromptEmpty
	}
	maxBytes := u.cfg.MaxPromptBytesOrDefault()
	if len(prompt) > maxBytes {
		return nil, ErrPromptTooLarge
	}

	if u.limiter != nil && !u.limiter.Allow(userID) {
		return nil, ErrRateLimited
	}

	ttl := u.cfg.ConversationTTL()
	maxTurns := u.cfg.MaxTurnsOrDefault()

	var (
		conversationID string
		history        []entity.ChatTurn
		err            error
	)

	if strings.TrimSpace(req.ConversationID) == "" {
		conversationID, err = u.convs.Create(ctx, userID, ttl)
		if err != nil {
			return nil, err
		}
		history = nil
	} else {
		conversationID = strings.TrimSpace(req.ConversationID)
		history, err = u.convs.Get(ctx, userID, conversationID)
		if err != nil {
			if errors.Is(err, repository.ErrConversationNotFound) {
				return nil, ErrConversationInvalid
			}
			return nil, err
		}
		_ = u.convs.Touch(ctx, userID, conversationID, ttl)
	}

	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = u.cfg.MaxTokensOrDefault()
	} else if maxTokens > u.cfg.MaxTokensOrDefault() {
		maxTokens = u.cfg.MaxTokensOrDefault()
	}

	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = u.provider.Name()
	}

	requestID := "sse_" + uuid.NewString()
	out := make(chan entity.SSEFrame, 32)

	go u.runStream(ctx, runStreamArgs{
		userID:          userID,
		requestID:       requestID,
		conversationID:  conversationID,
		clientRequestID: req.ClientRequestID,
		model:           model,
		prompt:          prompt,
		history:         history,
		maxTokens:       maxTokens,
		temperature:     req.Temperature,
		maxTurns:        maxTurns,
		ttl:             ttl,
		out:             out,
	})

	return out, nil
}

type runStreamArgs struct {
	userID          string
	requestID       string
	conversationID  string
	clientRequestID string
	model           string
	prompt          string
	history         []entity.ChatTurn
	maxTokens       int
	temperature     float64
	maxTurns        int
	ttl             time.Duration
	out             chan<- entity.SSEFrame
}

func (u *CompletionUsecase) runStream(ctx context.Context, a runStreamArgs) {
	defer close(a.out)

	seq := 0
	emit := func(event string, data entity.SSEEventData) {
		data.RequestID = a.requestID
		data.Seq = seq
		seq++
		select {
		case <-ctx.Done():
		case a.out <- entity.SSEFrame{Event: event, Data: data}:
		}
	}

	emit(entity.SSEEventMeta, entity.SSEEventData{
		Type:            entity.SSETypeMeta,
		ConversationID:  a.conversationID,
		Model:           a.model,
		ClientRequestID: a.clientRequestID,
	})

	deltaCh := make(chan string, 16)
	var (
		finishReason string
		streamErr    error
		wg           sync.WaitGroup
	)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(deltaCh)
		finishReason, streamErr = u.provider.Stream(ctx, provider.CompletionInput{
			Prompt:      a.prompt,
			Messages:    a.history,
			Model:       a.model,
			MaxTokens:   a.maxTokens,
			Temperature: a.temperature,
		}, deltaCh)
	}()

	var assistant strings.Builder
	canceled := false

loop:
	for {
		select {
		case <-ctx.Done():
			canceled = true
			break loop
		case chunk, ok := <-deltaCh:
			if !ok {
				break loop
			}
			assistant.WriteString(chunk)
			emit(entity.SSEEventDelta, entity.SSEEventData{
				Type: entity.SSETypeDelta,
				Text: chunk,
			})
		}
	}

	wg.Wait()

	assistantText := assistant.String()
	// 取消或上游取消：截断助手回复写入上下文，不再发 done
	if canceled || errors.Is(ctx.Err(), context.Canceled) || errors.Is(streamErr, context.Canceled) {
		u.persistTurn(context.WithoutCancel(ctx), a, assistantText)
		return
	}

	if streamErr != nil {
		code, msg := mapProviderError(streamErr)
		emit(entity.SSEEventError, entity.SSEEventData{
			Type: entity.SSETypeError,
			Error: &entity.SSEErrorBody{
				Code:    code,
				Message: msg,
			},
		})
		// 流中失败仍保存已生成片段，避免下一轮丢上下文
		if assistantText != "" {
			u.persistTurn(context.WithoutCancel(ctx), a, assistantText)
		} else {
			// 仅用户句入库，便于多轮
			u.persistTurn(context.WithoutCancel(ctx), a, "")
		}
		return
	}

	if finishReason == "" {
		finishReason = entity.FinishReasonStop
	}
	fr := finishReason
	emit(entity.SSEEventDone, entity.SSEEventData{
		Type:         entity.SSETypeDone,
		Text:         "",
		FinishReason: &fr,
	})
	u.persistTurn(context.WithoutCancel(ctx), a, assistantText)
}

func (u *CompletionUsecase) persistTurn(ctx context.Context, a runStreamArgs, assistantText string) {
	turns := append([]entity.ChatTurn{}, a.history...)
	turns = append(turns, entity.ChatTurn{Role: entity.ChatRoleUser, Content: a.prompt})
	if strings.TrimSpace(assistantText) != "" {
		turns = append(turns, entity.ChatTurn{Role: entity.ChatRoleAssistant, Content: assistantText})
	}
	turns = trimToMaxTurns(turns, a.maxTurns)
	_ = u.convs.SaveTurns(ctx, a.userID, a.conversationID, turns, a.ttl)
}

// trimToMaxTurns 保留最近 maxTurns 轮（一轮 = user + 紧随 assistant）。
func trimToMaxTurns(turns []entity.ChatTurn, maxTurns int) []entity.ChatTurn {
	if maxTurns <= 0 || len(turns) == 0 {
		return turns
	}
	// 按消息对裁剪：从尾部数最多 maxTurns*2 条
	maxMsgs := maxTurns * 2
	if len(turns) <= maxMsgs {
		return turns
	}
	return turns[len(turns)-maxMsgs:]
}

type codedProviderError interface {
	CodeValue() string
	Error() string
}

func mapProviderError(err error) (code, message string) {
	if err == nil {
		return entity.SSEErrorInternal, "unknown error"
	}
	message = err.Error()
	if ce, ok := err.(codedProviderError); ok {
		return ce.CodeValue(), ce.Error()
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return entity.SSEErrorUpstreamTimeout, message
	}
	return entity.SSEErrorInternal, message
}

// MemoryRateLimiter 进程内滑动窗口限流（每用户每分钟）。
type MemoryRateLimiter struct {
	limit  int
	mu     sync.Mutex
	events map[string][]time.Time
}

// NewMemoryRateLimiter 创建限流器；limit<=0 表示不限制。
func NewMemoryRateLimiter(limit int) *MemoryRateLimiter {
	return &MemoryRateLimiter{
		limit:  limit,
		events: make(map[string][]time.Time),
	}
}

// Allow 实现 RateLimiter。
func (l *MemoryRateLimiter) Allow(userID string) bool {
	if l == nil || l.limit <= 0 {
		return true
	}
	now := time.Now()
	windowStart := now.Add(-time.Minute)
	l.mu.Lock()
	defer l.mu.Unlock()
	times := l.events[userID]
	kept := times[:0]
	for _, t := range times {
		if t.After(windowStart) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.limit {
		l.events[userID] = kept
		return false
	}
	kept = append(kept, now)
	l.events[userID] = kept
	return true
}

// MemoryConversationRepository 测试用内存会话仓储。
type MemoryConversationRepository struct {
	mu    sync.Mutex
	store map[string]memoryConv
	ttl   time.Duration
	now   func() time.Time
}

type memoryConv struct {
	turns   []entity.ChatTurn
	expires time.Time
}

// NewMemoryConversationRepository 创建内存仓储。
func NewMemoryConversationRepository() *MemoryConversationRepository {
	return &MemoryConversationRepository{
		store: make(map[string]memoryConv),
		now:   time.Now,
	}
}

func (m *MemoryConversationRepository) key(userID, id string) string {
	return userID + ":" + id
}

// Create 实现 ConversationRepository。
func (m *MemoryConversationRepository) Create(ctx context.Context, userID string, ttl time.Duration) (string, error) {
	_ = ctx
	id := "conv_" + uuid.NewString()
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[m.key(userID, id)] = memoryConv{turns: nil, expires: m.now().Add(ttl)}
	return id, nil
}

// Get 实现 ConversationRepository。
func (m *MemoryConversationRepository) Get(ctx context.Context, userID, conversationID string) ([]entity.ChatTurn, error) {
	_ = ctx
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.store[m.key(userID, conversationID)]
	if !ok || m.now().After(c.expires) {
		if ok {
			delete(m.store, m.key(userID, conversationID))
		}
		return nil, repository.ErrConversationNotFound
	}
	out := append([]entity.ChatTurn{}, c.turns...)
	return out, nil
}

// SaveTurns 实现 ConversationRepository。
func (m *MemoryConversationRepository) SaveTurns(ctx context.Context, userID, conversationID string, turns []entity.ChatTurn, ttl time.Duration) error {
	_ = ctx
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := m.key(userID, conversationID)
	if _, ok := m.store[key]; !ok {
		return repository.ErrConversationNotFound
	}
	cp := append([]entity.ChatTurn{}, turns...)
	m.store[key] = memoryConv{turns: cp, expires: m.now().Add(ttl)}
	return nil
}

// Touch 实现 ConversationRepository。
func (m *MemoryConversationRepository) Touch(ctx context.Context, userID, conversationID string, ttl time.Duration) error {
	_ = ctx
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := m.key(userID, conversationID)
	c, ok := m.store[key]
	if !ok || m.now().After(c.expires) {
		return repository.ErrConversationNotFound
	}
	c.expires = m.now().Add(ttl)
	m.store[key] = c
	return nil
}

// ExpireNow 测试辅助：立即过期会话。
func (m *MemoryConversationRepository) ExpireNow(userID, conversationID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := m.key(userID, conversationID)
	c, ok := m.store[key]
	if !ok {
		return
	}
	c.expires = m.now().Add(-time.Second)
	m.store[key] = c
}

var _ repository.ConversationRepository = (*MemoryConversationRepository)(nil)
