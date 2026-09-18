package repository

import (
	"context"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

// AnalyticsRepository 数据分析记录仓储。
type AnalyticsRepository interface {
	ListPage(ctx context.Context, filter entity.AnalyticsListFilter) ([]entity.AnalyticsRecord, int64, error)
	GetByID(ctx context.Context, id int64) (*entity.AnalyticsRecord, error)
	Count(ctx context.Context) (int64, error)
	CountNotesContaining(ctx context.Context, marker string) (int64, error)
	DeleteBySourceSystem(ctx context.Context, sourceSystem string) error
	CreateBatch(ctx context.Context, items []entity.AnalyticsRecord) error
}
