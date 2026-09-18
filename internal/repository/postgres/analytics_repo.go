package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	domainrepo "github.com/stvenfor/my_go_study/internal/domain/repository"
	"gorm.io/gorm"
)

type analyticsRepository struct {
	db *gorm.DB
}

// NewAnalyticsRepository 创建数据分析仓储。
func NewAnalyticsRepository(db *gorm.DB) domainrepo.AnalyticsRepository {
	return &analyticsRepository{db: db}
}

func (r *analyticsRepository) ListPage(ctx context.Context, filter entity.AnalyticsListFilter) ([]entity.AnalyticsRecord, int64, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	var total int64
	if err := r.db.WithContext(ctx).Model(&entity.AnalyticsRecord{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计 analytics_records 失败: %w", err)
	}

	var items []entity.AnalyticsRecord
	err := r.db.WithContext(ctx).
		Order("observed_at DESC, id DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&items).Error
	if err != nil {
		return nil, 0, fmt.Errorf("查询 analytics_records 失败: %w", err)
	}
	return items, total, nil
}

func (r *analyticsRepository) GetByID(ctx context.Context, id int64) (*entity.AnalyticsRecord, error) {
	var item entity.AnalyticsRecord
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("数据分析记录不存在")
		}
		return nil, fmt.Errorf("查询 analytics_record 失败: %w", err)
	}
	return &item, nil
}

func (r *analyticsRepository) Count(ctx context.Context) (int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&entity.AnalyticsRecord{}).Count(&total).Error; err != nil {
		return 0, fmt.Errorf("统计 analytics_records 失败: %w", err)
	}
	return total, nil
}

func (r *analyticsRepository) CreateBatch(ctx context.Context, items []entity.AnalyticsRecord) error {
	if len(items) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).Create(&items).Error; err != nil {
		return fmt.Errorf("批量写入 analytics_records 失败: %w", err)
	}
	return nil
}
