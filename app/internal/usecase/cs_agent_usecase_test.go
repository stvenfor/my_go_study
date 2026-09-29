package usecase

import (
	"context"
	"strings"
	"testing"
	"time"

	"model/entity"
	"common/config"
)

type stubTxSearcher struct {
	rows []entity.Transaction
	err  error
}

func (s stubTxSearcher) Search(ctx context.Context, userID, q string, limit int) ([]entity.Transaction, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.rows, nil
}

func TestCsAgentStreamMockWithSearchTool(t *testing.T) {
	note := "午餐"
	uc := NewCsAgentUsecase(stubTxSearcher{
		rows: []entity.Transaction{{ID: 1, Type: "expense", Category: "餐饮", Amount: 32, Date: "2026-09-28", Note: &note}},
	}, config.CsAgentConfig{Enabled: true, Provider: "mock", ChunkDelayMS: 0})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ch, err := uc.Stream(ctx, "user-1", CsAgentRequest{Prompt: "查一下餐饮交易"})
	if err != nil {
		t.Fatal(err)
	}
	var deltas strings.Builder
	var sawMeta, sawDone bool
	for fr := range ch {
		switch fr.Event {
		case entity.SSEEventMeta:
			sawMeta = true
		case entity.SSEEventDelta:
			deltas.WriteString(fr.Data.Text)
		case entity.SSEEventDone:
			sawDone = true
		case entity.SSEEventError:
			t.Fatalf("unexpected error frame: %+v", fr.Data.Error)
		}
	}
	if !sawMeta || !sawDone {
		t.Fatalf("meta=%v done=%v", sawMeta, sawDone)
	}
	out := deltas.String()
	if !strings.Contains(out, "客服") || !strings.Contains(out, "餐饮") {
		t.Fatalf("answer missing expected content: %q", out)
	}
}

func TestCsAgentDisabled(t *testing.T) {
	uc := NewCsAgentUsecase(nil, config.CsAgentConfig{Enabled: false})
	_, err := uc.Stream(context.Background(), "u", CsAgentRequest{Prompt: "hi"})
	if err != ErrCsAgentDisabled {
		t.Fatalf("err=%v", err)
	}
}
