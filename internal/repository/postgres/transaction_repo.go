// transaction_repo.go 基于 PostgreSQL 的 transactions 仓储（自建 JWT 用户体系）。
package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	domainrepo "github.com/stvenfor/my_go_study/internal/domain/repository"
	"gorm.io/gorm"
)

type transactionRepository struct {
	db *gorm.DB
}

// NewTransactionRepository 创建 transactions 仓储。
func NewTransactionRepository(db *gorm.DB) domainrepo.TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) List(ctx context.Context, _, userID string, filter entity.TransactionFilter) ([]entity.Transaction, error) {
	items, _, err := r.ListPage(ctx, "", userID, filter)
	return items, err
}

func (r *transactionRepository) ListPage(ctx context.Context, _, userID string, filter entity.TransactionFilter) ([]entity.Transaction, int64, error) {
	uid, err := parseUserID(userID)
	if err != nil {
		return nil, 0, err
	}

	query := r.db.WithContext(ctx).Model(&entity.TransactionRecord{}).Where("user_id = ?", uid)
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计 transactions 失败: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	var records []entity.TransactionRecord
	err = query.Order("date DESC, id DESC").
		Offset(offset).
		Limit(limit).
		Find(&records).Error
	if err != nil {
		return nil, 0, fmt.Errorf("查询 transactions 失败: %w", err)
	}

	return toTransactions(records), total, nil
}

func (r *transactionRepository) GetByID(ctx context.Context, _, userID string, id int64) (*entity.Transaction, error) {
	uid, err := parseUserID(userID)
	if err != nil {
		return nil, err
	}

	var record entity.TransactionRecord
	err = r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, uid).
		First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("交易记录不存在")
		}
		return nil, fmt.Errorf("查询 transaction 失败: %w", err)
	}
	item := record.ToTransaction()
	return &item, nil
}

func (r *transactionRepository) Create(ctx context.Context, _, userID string, input entity.CreateTransactionInput) (*entity.Transaction, error) {
	uid, err := parseUserID(userID)
	if err != nil {
		return nil, err
	}

	record := entity.TransactionRecord{
		UserID:   uid,
		Type:     input.Type,
		Category: input.Category,
		Amount:   input.Amount,
		Date:     input.Date,
		Note:     input.Note,
	}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, fmt.Errorf("创建 transaction 失败: %w", err)
	}
	item := record.ToTransaction()
	return &item, nil
}

func (r *transactionRepository) Update(ctx context.Context, _, userID string, id int64, input entity.UpdateTransactionInput) (*entity.Transaction, error) {
	uid, err := parseUserID(userID)
	if err != nil {
		return nil, err
	}

	var record entity.TransactionRecord
	err = r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, uid).
		First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("交易记录不存在")
		}
		return nil, fmt.Errorf("查询 transaction 失败: %w", err)
	}

	updates := map[string]any{}
	if input.Type != nil {
		updates["type"] = *input.Type
	}
	if input.Category != nil {
		updates["category"] = *input.Category
	}
	if input.Amount != nil {
		updates["amount"] = *input.Amount
	}
	if input.Date != nil {
		updates["date"] = *input.Date
	}
	if input.Note != nil {
		updates["note"] = *input.Note
	}
	if len(updates) == 0 {
		item := record.ToTransaction()
		return &item, nil
	}

	if err := r.db.WithContext(ctx).Model(&record).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新 transaction 失败: %w", err)
	}
	if err := r.db.WithContext(ctx).First(&record, record.ID).Error; err != nil {
		return nil, fmt.Errorf("刷新 transaction 失败: %w", err)
	}
	item := record.ToTransaction()
	return &item, nil
}

func (r *transactionRepository) Delete(ctx context.Context, _, userID string, id int64) error {
	uid, err := parseUserID(userID)
	if err != nil {
		return err
	}

	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, uid).
		Delete(&entity.TransactionRecord{})
	if result.Error != nil {
		return fmt.Errorf("删除 transaction 失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("交易记录不存在")
	}
	return nil
}

func parseUserID(userID string) (uint, error) {
	if userID == "" {
		return 0, fmt.Errorf("无效的用户 ID")
	}
	parsed, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("无效的用户 ID")
	}
	return uint(parsed), nil
}

func toTransactions(records []entity.TransactionRecord) []entity.Transaction {
	items := make([]entity.Transaction, 0, len(records))
	for _, record := range records {
		items = append(items, record.ToTransaction())
	}
	return items
}
