package csagent

import (
	"context"
	"strings"
	"testing"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

type stubSearcher struct {
	rows []entity.Transaction
}

func (s stubSearcher) Search(ctx context.Context, userID, q string, limit int) ([]entity.Transaction, error) {
	return s.rows, nil
}

func TestTransactionSearchToolInvokableRun(t *testing.T) {
	note := "地铁"
	tool, err := NewTransactionSearchTool(stubSearcher{
		rows: []entity.Transaction{{ID: 9, Type: "expense", Category: "交通", Amount: 5, Date: "2026-09-01", Note: &note}},
	}, "u1")
	if err != nil {
		t.Fatal(err)
	}
	info, err := tool.Info(context.Background())
	if err != nil || info.Name != "transaction_search" {
		t.Fatalf("info=%+v err=%v", info, err)
	}
	out, err := tool.InvokableRun(context.Background(), `{"query":"交通","limit":3}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "交通") || !strings.Contains(out, `"id":9`) {
		t.Fatalf("out=%s", out)
	}
}
