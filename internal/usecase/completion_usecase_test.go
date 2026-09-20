package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/provider"
	llmrepo "github.com/stvenfor/my_go_study/internal/repository/llm"
	"github.com/stvenfor/my_go_study/internal/usecase"
	"github.com/stvenfor/my_go_study/pkg/config"
)

func testSSEConfig() config.SSEConfig {
	return config.SSEConfig{
		Enabled:                   true,
		Provider:                  "mock",
		MaxPromptBytes:            8192,
		MaxTokens:                 2048,
		KeepaliveSeconds:          15,
		RequestTimeoutSeconds:     30,
		RateLimitPerUserPerMinute: 100,
		ConversationTTLSeconds:    1800,
		MaxTurns:                  10,
	}
}

func collectFrames(t *testing.T, ch <-chan entity.SSEFrame, timeout time.Duration) []entity.SSEFrame {
	t.Helper()
	deadline := time.After(timeout)
	var frames []entity.SSEFrame
	for {
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for stream close; got %d frames", len(frames))
		case ev, ok := <-ch:
			if !ok {
				return frames
			}
			frames = append(frames, ev)
		}
	}
}

func TestCompletionEventOrderMetaDeltaDone(t *testing.T) {
	mock := llmrepo.NewMockStreamProvider()
	mock.ChunkDelay = 0
	convs := usecase.NewMemoryConversationRepository()
	uc := usecase.NewCompletionUsecase(mock, convs, usecase.NewMemoryRateLimiter(100), testSSEConfig())

	ch, err := uc.Stream(context.Background(), "user-1", usecase.CompletionRequest{
		Prompt:          "你好，做个自我介绍",
		ClientRequestID: "req_test_1",
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	frames := collectFrames(t, ch, 5*time.Second)
	if len(frames) < 3 {
		t.Fatalf("expected meta+delta+done, got %d frames", len(frames))
	}
	if frames[0].Event != entity.SSEEventMeta {
		t.Fatalf("first event=%s want meta", frames[0].Event)
	}
	if frames[0].Data.ConversationID == "" {
		t.Fatal("meta missing conversationId")
	}
	if frames[0].Data.ClientRequestID != "req_test_1" {
		t.Fatalf("clientRequestId=%s", frames[0].Data.ClientRequestID)
	}
	if frames[0].Data.Model == "" {
		t.Fatal("meta missing model")
	}

	var sawDelta bool
	for i := 1; i < len(frames)-1; i++ {
		if frames[i].Event != entity.SSEEventDelta {
			t.Fatalf("frame[%d] event=%s want delta", i, frames[i].Event)
		}
		sawDelta = true
	}
	if !sawDelta {
		t.Fatal("expected at least one delta")
	}
	last := frames[len(frames)-1]
	if last.Event != entity.SSEEventDone {
		t.Fatalf("last event=%s want done", last.Event)
	}
	if last.Data.FinishReason == nil || *last.Data.FinishReason != entity.FinishReasonStop {
		t.Fatalf("finishReason=%v", last.Data.FinishReason)
	}
}

func TestCompletionCancelStopsAndTruncatesContext(t *testing.T) {
	mock := llmrepo.NewMockStreamProvider()
	mock.ChunkDelay = 30 * time.Millisecond
	convs := usecase.NewMemoryConversationRepository()
	uc := usecase.NewCompletionUsecase(mock, convs, nil, testSSEConfig())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := uc.Stream(ctx, "user-cancel", usecase.CompletionRequest{Prompt: "二手车入口在哪里？"})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	var conversationID string
	var deltaCount int
	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for stream")
		case ev, ok := <-ch:
			if !ok {
				goto done
			}
			switch ev.Event {
			case entity.SSEEventMeta:
				conversationID = ev.Data.ConversationID
			case entity.SSEEventDelta:
				deltaCount++
				if deltaCount >= 2 {
					cancel()
				}
			}
		}
	}
done:
	if conversationID == "" {
		t.Fatal("missing conversationId")
	}
	time.Sleep(80 * time.Millisecond)

	turns, err := convs.Get(context.Background(), "user-cancel", conversationID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(turns) < 1 || turns[0].Role != entity.ChatRoleUser {
		t.Fatalf("expected user turn persisted, got %+v", turns)
	}
	if deltaCount > 0 {
		if len(turns) < 2 || turns[1].Role != entity.ChatRoleAssistant {
			t.Fatalf("expected truncated assistant turn, got %+v", turns)
		}
		if turns[1].Content == "" {
			t.Fatal("assistant content empty after cancel with deltas")
		}
	}
}

func TestCompletionBadConversationID(t *testing.T) {
	mock := llmrepo.NewMockStreamProvider()
	mock.ChunkDelay = 0
	convs := usecase.NewMemoryConversationRepository()
	uc := usecase.NewCompletionUsecase(mock, convs, nil, testSSEConfig())

	_, err := uc.Stream(context.Background(), "user-2", usecase.CompletionRequest{
		Prompt:         "你好",
		ConversationID: "conv_does_not_exist",
	})
	if !errors.Is(err, usecase.ErrConversationInvalid) {
		t.Fatalf("err=%v want ErrConversationInvalid", err)
	}

	// 过期 id：先创建再 ExpireNow
	ch, err := uc.Stream(context.Background(), "user-2", usecase.CompletionRequest{Prompt: "你好"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	frames := collectFrames(t, ch, 3*time.Second)
	id := frames[0].Data.ConversationID
	convs.ExpireNow("user-2", id)
	_, err = uc.Stream(context.Background(), "user-2", usecase.CompletionRequest{
		Prompt:         "继续",
		ConversationID: id,
	})
	if !errors.Is(err, usecase.ErrConversationInvalid) {
		t.Fatalf("expired err=%v", err)
	}
}

func TestCompletionMultiTurnContext(t *testing.T) {
	fake := &recordingProvider{answer: "续答内容"}
	convs := usecase.NewMemoryConversationRepository()
	uc := usecase.NewCompletionUsecase(fake, convs, nil, testSSEConfig())

	ch1, err := uc.Stream(context.Background(), "user-mt", usecase.CompletionRequest{Prompt: "二手车入口在哪里？"})
	if err != nil {
		t.Fatalf("turn1: %v", err)
	}
	frames1 := collectFrames(t, ch1, 3*time.Second)
	convID := frames1[0].Data.ConversationID
	if convID == "" {
		t.Fatal("no conversationId")
	}

	ch2, err := uc.Stream(context.Background(), "user-mt", usecase.CompletionRequest{
		Prompt:         "然后呢",
		ConversationID: convID,
	})
	if err != nil {
		t.Fatalf("turn2: %v", err)
	}
	_ = collectFrames(t, ch2, 3*time.Second)

	if len(fake.lastMessages) < 2 {
		t.Fatalf("expected prior turns in provider input, got %d messages", len(fake.lastMessages))
	}
	joined := ""
	for _, m := range fake.lastMessages {
		joined += m.Role + ":" + m.Content + ";"
	}
	if !strings.Contains(joined, entity.ChatRoleUser) || !strings.Contains(joined, "二手车") {
		t.Fatalf("history missing first turn: %s", joined)
	}
}

func TestCompletionPromptTooLargeAndRateLimit(t *testing.T) {
	mock := llmrepo.NewMockStreamProvider()
	mock.ChunkDelay = 0
	convs := usecase.NewMemoryConversationRepository()
	cfg := testSSEConfig()
	cfg.MaxPromptBytes = 8
	uc := usecase.NewCompletionUsecase(mock, convs, nil, cfg)

	_, err := uc.Stream(context.Background(), "u", usecase.CompletionRequest{Prompt: "0123456789"})
	if !errors.Is(err, usecase.ErrPromptTooLarge) {
		t.Fatalf("err=%v", err)
	}

	cfg2 := testSSEConfig()
	limiter := usecase.NewMemoryRateLimiter(1)
	uc2 := usecase.NewCompletionUsecase(mock, convs, limiter, cfg2)
	ch, err := uc2.Stream(context.Background(), "rate-user", usecase.CompletionRequest{Prompt: "hi"})
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	_ = collectFrames(t, ch, 3*time.Second)
	_, err = uc2.Stream(context.Background(), "rate-user", usecase.CompletionRequest{Prompt: "hi again"})
	if !errors.Is(err, usecase.ErrRateLimited) {
		t.Fatalf("err=%v want rate limited", err)
	}
}

func TestCompletionEmptyPrompt(t *testing.T) {
	mock := llmrepo.NewMockStreamProvider()
	convs := usecase.NewMemoryConversationRepository()
	uc := usecase.NewCompletionUsecase(mock, convs, nil, testSSEConfig())
	_, err := uc.Stream(context.Background(), "u", usecase.CompletionRequest{Prompt: "  "})
	if !errors.Is(err, usecase.ErrPromptEmpty) {
		t.Fatalf("err=%v", err)
	}
}

// recordingProvider 记录传入的历史消息，便于断言多轮上下文。
type recordingProvider struct {
	answer       string
	lastMessages []entity.ChatTurn
	lastPrompt   string
}

func (p *recordingProvider) Name() string { return "fake-v1" }

func (p *recordingProvider) Stream(ctx context.Context, in provider.CompletionInput, out chan<- string) (string, error) {
	p.lastMessages = append([]entity.ChatTurn{}, in.Messages...)
	p.lastPrompt = in.Prompt
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case out <- p.answer:
		return entity.FinishReasonStop, nil
	}
}
