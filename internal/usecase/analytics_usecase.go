package usecase

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	domainrepo "github.com/stvenfor/my_go_study/internal/domain/repository"
)

// AnalyticsUsecase 数据分析业务。
type AnalyticsUsecase struct {
	repo domainrepo.AnalyticsRepository
}

// NewAnalyticsUsecase 创建用例。
func NewAnalyticsUsecase(repo domainrepo.AnalyticsRepository) *AnalyticsUsecase {
	return &AnalyticsUsecase{repo: repo}
}

// ListPage 分页列表（page 从 1 起）。
func (u *AnalyticsUsecase) ListPage(ctx context.Context, page, pageSize int) ([]entity.AnalyticsRecord, int64, error) {
	return u.repo.ListPage(ctx, entity.AnalyticsListFilter{Page: page, PageSize: pageSize})
}

// Get 详情。
func (u *AnalyticsUsecase) Get(ctx context.Context, id int64) (*entity.AnalyticsRecord, error) {
	if id <= 0 {
		return nil, fmt.Errorf("无效的记录 ID")
	}
	return u.repo.GetByID(ctx, id)
}

// EnsureSeedData 若表为空则写入演示数据。
func (u *AnalyticsUsecase) EnsureSeedData(ctx context.Context) error {
	count, err := u.repo.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	now := time.Now().UTC().Unix()
	items := make([]entity.AnalyticsRecord, 0, 36)
	categories := []string{"获客", "转化", "留存", "营收", "投放"}
	regions := []string{"华东", "华南", "华北", "西南", "西北"}
	channels := []string{"App", "小程序", "H5", "线下", "合作方"}
	statuses := []string{"active", "paused", "draft", "archived"}

	for i := 1; i <= 36; i++ {
		rev := 1000.0 + float64(i)*37.5
		cost := 400.0 + float64(i)*12.3
		roi := 0.0
		if cost > 0 {
			roi = math.Round((rev/cost)*100) / 100
		}
		observed := now - int64(i*3600)
		items = append(items, entity.AnalyticsRecord{
			Code:                 fmt.Sprintf("AN-%04d", i),
			Title:                fmt.Sprintf("分析样本 #%d", i),
			Subtitle:             fmt.Sprintf("%s · %s 渠道周报", categories[(i-1)%len(categories)], channels[(i-1)%len(channels)]),
			Category:             categories[(i-1)%len(categories)],
			SubCategory:          fmt.Sprintf("子类-%d", (i%5)+1),
			Status:               statuses[(i-1)%len(statuses)],
			Priority:             int32((i % 5) + 1),
			Region:               regions[(i-1)%len(regions)],
			Channel:              channels[(i-1)%len(channels)],
			OwnerName:            fmt.Sprintf("分析师%02d", (i%8)+1),
			OwnerTeam:            fmt.Sprintf("增长组-%d", (i%3)+1),
			SourceSystem:         "seed",
			MetricPV:             int64(10000 + i*137),
			MetricUV:             int64(3000 + i*41),
			MetricClick:          int64(800 + i*17),
			MetricConvert:        int64(40 + i*2),
			MetricRevenue:        rev,
			MetricCost:           cost,
			MetricROI:            roi,
			MetricBounceRate:     math.Round((0.2+float64(i%10)*0.03)*100) / 100,
			MetricAvgDurationSec: int32(30 + (i % 90)),
			ScoreQuality:         math.Round((60+float64(i%40))*10) / 10,
			ScoreRisk:            math.Round((10+float64(i%25))*10) / 10,
			TagPrimary:           fmt.Sprintf("tag-a-%d", i%4),
			TagSecondary:         fmt.Sprintf("tag-b-%d", i%6),
			FlagFeatured:         i%7 == 0,
			FlagAnomaly:          i%11 == 0,
			Notes:                fmt.Sprintf("自动种子数据，用于 gRPC 列表/详情联调（#%d）。", i),
			ObservedAt:           observed,
			WindowStart:          observed - 7*24*3600,
			WindowEnd:            observed,
			PublishedAt:          observed - 3600,
			CreatedAt:            now - int64(i*60),
			UpdatedAt:            now - int64(i*30),
		})
	}
	return u.repo.CreateBatch(ctx, items)
}
