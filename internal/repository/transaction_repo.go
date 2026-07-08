package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	postgrest "github.com/supabase-community/postgrest-go"

	"github.com/stvenfor/my_go_study/internal/domain"
	"github.com/stvenfor/my_go_study/internal/supabase"
)

type TransactionRepository struct {
	supabase *supabase.Client
}

func NewTransactionRepository(client *supabase.Client) *TransactionRepository {
	return &TransactionRepository{supabase: client}
}

func (r *TransactionRepository) List(ctx context.Context, filter domain.TransactionFilter) ([]domain.Transaction, error) {
	query := r.supabase.Admin.From(domain.TransactionsTable).
		Select("*", "", false).
		Order("date", &postgrest.OrderOpts{Ascending: false})

	if filter.Type != "" {
		query = query.Eq("type", filter.Type)
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	query = query.Limit(limit, "")
	if filter.Offset > 0 {
		query = query.Range(filter.Offset, filter.Offset+limit-1, "")
	}

	data, _, err := query.Execute()
	if err != nil {
		return nil, fmt.Errorf("查询 transactions 失败: %w", err)
	}

	var items []domain.Transaction
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("解析 transactions 失败: %w", err)
	}
	return items, nil
}

func (r *TransactionRepository) GetByID(ctx context.Context, id int64) (*domain.Transaction, error) {
	data, _, err := r.supabase.Admin.From(domain.TransactionsTable).
		Select("*", "", false).
		Eq("id", strconv.FormatInt(id, 10)).
		Single().
		Execute()
	if err != nil {
		return nil, fmt.Errorf("查询 transaction 失败: %w", err)
	}

	var item domain.Transaction
	if err := json.Unmarshal(data, &item); err != nil {
		var rows []domain.Transaction
		if err2 := json.Unmarshal(data, &rows); err2 == nil && len(rows) > 0 {
			return &rows[0], nil
		}
		return nil, fmt.Errorf("解析 transaction 失败: %w", err)
	}
	return &item, nil
}

func (r *TransactionRepository) Create(ctx context.Context, input domain.CreateTransactionInput) (*domain.Transaction, error) {
	payload := map[string]any{
		"type":     input.Type,
		"category": input.Category,
		"amount":   input.Amount,
		"date":     input.Date,
	}
	if input.Note != nil {
		payload["note"] = *input.Note
	}
	now := time.Now().UTC().Format(time.RFC3339)
	payload["created_at"] = now
	payload["updated_at"] = now

	data, _, err := r.supabase.Admin.From(domain.TransactionsTable).
		Insert(payload, false, "", "representation", "").
		Single().
		Execute()
	if err != nil {
		return nil, fmt.Errorf("创建 transaction 失败: %w", err)
	}

	var item domain.Transaction
	if err := json.Unmarshal(data, &item); err != nil {
		var rows []domain.Transaction
		if err2 := json.Unmarshal(data, &rows); err2 == nil && len(rows) > 0 {
			return &rows[0], nil
		}
		return nil, fmt.Errorf("解析 transaction 失败: %w", err)
	}
	return &item, nil
}

func (r *TransactionRepository) Update(ctx context.Context, id int64, input domain.UpdateTransactionInput) (*domain.Transaction, error) {
	payload := map[string]any{
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	}
	if input.Type != nil {
		payload["type"] = *input.Type
	}
	if input.Category != nil {
		payload["category"] = *input.Category
	}
	if input.Amount != nil {
		payload["amount"] = *input.Amount
	}
	if input.Date != nil {
		payload["date"] = *input.Date
	}
	if input.Note != nil {
		payload["note"] = *input.Note
	}

	data, _, err := r.supabase.Admin.From(domain.TransactionsTable).
		Update(payload, "representation", "").
		Eq("id", strconv.FormatInt(id, 10)).
		Single().
		Execute()
	if err != nil {
		return nil, fmt.Errorf("更新 transaction 失败: %w", err)
	}

	var item domain.Transaction
	if err := json.Unmarshal(data, &item); err != nil {
		var rows []domain.Transaction
		if err2 := json.Unmarshal(data, &rows); err2 == nil && len(rows) > 0 {
			return &rows[0], nil
		}
		return nil, fmt.Errorf("解析 transaction 失败: %w", err)
	}
	return &item, nil
}

func (r *TransactionRepository) Delete(ctx context.Context, id int64) error {
	_, _, err := r.supabase.Admin.From(domain.TransactionsTable).
		Delete("", "").
		Eq("id", strconv.FormatInt(id, 10)).
		Execute()
	if err != nil {
		return fmt.Errorf("删除 transaction 失败: %w", err)
	}
	return nil
}
