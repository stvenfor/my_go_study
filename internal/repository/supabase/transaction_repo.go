// transaction_repo.go 通过 Supabase PostgREST 访问 transactions 表（用户 token + user_id 过滤）。
package supabase

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	postgrest "github.com/supabase-community/postgrest-go"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	pkgsb "github.com/stvenfor/my_go_study/pkg/supabase"
)

// TransactionRepository transactions 仓储实现。
type TransactionRepository struct {
	client *pkgsb.Client
}

// NewTransactionRepository 创建 transactions 仓储。
func NewTransactionRepository(client *pkgsb.Client) *TransactionRepository {
	return &TransactionRepository{client: client}
}

func (r *TransactionRepository) List(ctx context.Context, accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, error) {
	client, err := r.client.WithUserToken(accessToken)
	if err != nil {
		return nil, err
	}

	query := client.From(entity.TransactionsTable).
		Select("*", "", false).
		Eq("user_id", userID).
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

	var items []entity.Transaction
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("解析 transactions 失败: %w", err)
	}
	return items, nil
}

func (r *TransactionRepository) ListPage(ctx context.Context, accessToken, userID string, filter entity.TransactionFilter) ([]entity.Transaction, int64, error) {
	client, err := r.client.WithUserToken(accessToken)
	if err != nil {
		return nil, 0, err
	}

	query := client.From(entity.TransactionsTable).
		Select("*", "exact", false).
		Eq("user_id", userID).
		Order("date", &postgrest.OrderOpts{Ascending: false})

	if filter.Type != "" {
		query = query.Eq("type", filter.Type)
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	query = query.Limit(limit, "")
	if filter.Offset > 0 {
		query = query.Range(filter.Offset, filter.Offset+limit-1, "")
	}

	data, total, err := query.Execute()
	if err != nil {
		return nil, 0, fmt.Errorf("查询 transactions 失败: %w", err)
	}

	var items []entity.Transaction
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, 0, fmt.Errorf("解析 transactions 失败: %w", err)
	}
	return items, total, nil
}

func (r *TransactionRepository) GetByID(ctx context.Context, accessToken, userID string, id int64) (*entity.Transaction, error) {
	client, err := r.client.WithUserToken(accessToken)
	if err != nil {
		return nil, err
	}

	data, _, err := client.From(entity.TransactionsTable).
		Select("*", "", false).
		Eq("id", strconv.FormatInt(id, 10)).
		Eq("user_id", userID).
		Single().
		Execute()
	if err != nil {
		return nil, fmt.Errorf("查询 transaction 失败: %w", err)
	}

	return unmarshalSingle[entity.Transaction](data)
}

func (r *TransactionRepository) Create(ctx context.Context, accessToken, userID string, input entity.CreateTransactionInput) (*entity.Transaction, error) {
	client, err := r.client.WithUserToken(accessToken)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"user_id":  userID,
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

	data, _, err := client.From(entity.TransactionsTable).
		Insert(payload, false, "", "representation", "").
		Single().
		Execute()
	if err != nil {
		return nil, fmt.Errorf("创建 transaction 失败: %w", err)
	}

	return unmarshalSingle[entity.Transaction](data)
}

func (r *TransactionRepository) Update(ctx context.Context, accessToken, userID string, id int64, input entity.UpdateTransactionInput) (*entity.Transaction, error) {
	client, err := r.client.WithUserToken(accessToken)
	if err != nil {
		return nil, err
	}

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

	data, _, err := client.From(entity.TransactionsTable).
		Update(payload, "representation", "").
		Eq("id", strconv.FormatInt(id, 10)).
		Eq("user_id", userID).
		Single().
		Execute()
	if err != nil {
		return nil, fmt.Errorf("更新 transaction 失败: %w", err)
	}

	return unmarshalSingle[entity.Transaction](data)
}

func (r *TransactionRepository) Delete(ctx context.Context, accessToken, userID string, id int64) error {
	client, err := r.client.WithUserToken(accessToken)
	if err != nil {
		return err
	}

	_, _, err = client.From(entity.TransactionsTable).
		Delete("", "").
		Eq("id", strconv.FormatInt(id, 10)).
		Eq("user_id", userID).
		Execute()
	if err != nil {
		return fmt.Errorf("删除 transaction 失败: %w", err)
	}
	return nil
}
