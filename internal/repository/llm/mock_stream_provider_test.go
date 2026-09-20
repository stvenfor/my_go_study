package llm_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/provider"
	"github.com/stvenfor/my_go_study/internal/repository/llm"
)

func TestMockStreamKeywordAndCancel(t *testing.T) {
	p := llm.NewMockStreamProvider()
	p.ChunkDelay = 0

	out := make(chan string, 256)
	go func() {
		_, _ = p.Stream(context.Background(), provider.CompletionInput{Prompt: "二手车入口"}, out)
		close(out)
	}()
	var buf strings.Builder
	for c := range out {
		buf.WriteString(c)
	}
	if !strings.Contains(buf.String(), "二手车") {
		t.Fatalf("unexpected answer: %s", buf.String())
	}

	p.ChunkDelay = 50 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	out2 := make(chan string, 8)
	errCh := make(chan error, 1)
	go func() {
		_, err := p.Stream(ctx, provider.CompletionInput{Prompt: "你好，做个自我介绍"}, out2)
		errCh <- err
		close(out2)
	}()
	// 读到一点后取消
	<-out2
	cancel()
	for range out2 {
	}
	err := <-errCh
	if err == nil {
		t.Fatal("expected cancel error")
	}
}

func TestOpenAICompatibleNoKey(t *testing.T) {
	p := llm.NewOpenAICompatibleProvider(llm.OpenAICompatibleConfig{})
	out := make(chan string, 1)
	_, err := p.Stream(context.Background(), provider.CompletionInput{Prompt: "hi"}, out)
	if err == nil {
		t.Fatal("expected error without api key")
	}
	pe, ok := err.(*llm.ProviderError)
	if !ok || pe.Code == "" {
		t.Fatalf("want ProviderError, got %T %v", err, err)
	}
}
