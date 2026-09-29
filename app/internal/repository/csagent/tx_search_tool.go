// Package csagent 客服 Agent 适配（Eino Tool）。
package csagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"model/entity"
)

// TransactionSearcher 交易全文搜索（由 TransactionUsecase 实现）。
type TransactionSearcher interface {
	Search(ctx context.Context, userID, q string, limit int) ([]entity.Transaction, error)
}

type searchHit struct {
	ID       int64   `json:"id"`
	Type     string  `json:"type"`
	Category string  `json:"category"`
	Amount   float64 `json:"amount"`
	Date     string  `json:"date"`
	Note     string  `json:"note,omitempty"`
}

type txSearchTool struct {
	searcher TransactionSearcher
	userID   string
}

// NewTransactionSearchTool 构建 Eino InvokableTool；userID 由闭包注入（会话用户）。
func NewTransactionSearchTool(searcher TransactionSearcher, userID string) (tool.InvokableTool, error) {
	return &txSearchTool{searcher: searcher, userID: userID}, nil
}

func (t *txSearchTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "transaction_search",
		Desc: "搜索当前用户的收支交易记录（全文：备注/分类/类型）",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query": {Type: schema.String, Desc: "搜索关键词", Required: true},
			"limit": {Type: schema.Integer, Desc: "返回条数上限，默认5"},
		}),
	}, nil
}

func (t *txSearchTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var in struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &in); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}
	q := strings.TrimSpace(in.Query)
	if q == "" {
		return `{"items":[],"message":"empty query"}`, nil
	}
	limit := in.Limit
	if limit <= 0 {
		limit = 5
	}
	if t.searcher == nil {
		return `{"items":[],"message":"search unavailable"}`, nil
	}
	rows, err := t.searcher.Search(ctx, t.userID, q, limit)
	if err != nil {
		return "", fmt.Errorf("transaction search: %w", err)
	}
	hits := make([]searchHit, 0, len(rows))
	for _, row := range rows {
		h := searchHit{
			ID: row.ID, Type: row.Type, Category: row.Category,
			Amount: row.Amount, Date: row.Date,
		}
		if row.Note != nil {
			h.Note = *row.Note
		}
		hits = append(hits, h)
	}
	b, err := json.Marshal(map[string]any{"items": hits, "count": len(hits)})
	if err != nil {
		return "", err
	}
	return string(b), nil
}
