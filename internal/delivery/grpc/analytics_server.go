package grpcdelivery

import (
	"context"

	analyticsv1 "github.com/stvenfor/my_go_study/api/gen/go/analytics/v1"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	grpcauth "github.com/stvenfor/my_go_study/internal/delivery/grpc/interceptor"
	"github.com/stvenfor/my_go_study/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AnalyticsServer 实现 AnalyticsService。
type AnalyticsServer struct {
	analyticsv1.UnimplementedAnalyticsServiceServer
	uc *usecase.AnalyticsUsecase
}

// NewAnalyticsServer 创建服务。
func NewAnalyticsServer(uc *usecase.AnalyticsUsecase) *AnalyticsServer {
	return &AnalyticsServer{uc: uc}
}

func (s *AnalyticsServer) ListAnalyticsRecords(
	ctx context.Context,
	req *analyticsv1.ListAnalyticsRecordsRequest,
) (*analyticsv1.ListAnalyticsRecordsResponse, error) {
	if _, ok := grpcauth.UserFromContext(ctx); !ok {
		return nil, status.Error(codes.Unauthenticated, "未授权")
	}
	page := int(req.GetPage())
	pageSize := int(req.GetPageSize())
	items, total, err := s.uc.ListPage(ctx, page, pageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "列表查询失败: %v", err)
	}
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 50 {
		pageSize = 50
	}
	out := make([]*analyticsv1.AnalyticsRecord, 0, len(items))
	for i := range items {
		out = append(out, toProto(&items[i]))
	}
	return &analyticsv1.ListAnalyticsRecordsResponse{
		Items:    out,
		Total:    total,
		Page:     int32(page),
		PageSize: int32(pageSize),
	}, nil
}

func (s *AnalyticsServer) GetAnalyticsRecord(
	ctx context.Context,
	req *analyticsv1.GetAnalyticsRecordRequest,
) (*analyticsv1.GetAnalyticsRecordResponse, error) {
	if _, ok := grpcauth.UserFromContext(ctx); !ok {
		return nil, status.Error(codes.Unauthenticated, "未授权")
	}
	item, err := s.uc.Get(ctx, req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "%v", err)
	}
	return &analyticsv1.GetAnalyticsRecordResponse{Item: toProto(item)}, nil
}

func toProto(item *entity.AnalyticsRecord) *analyticsv1.AnalyticsRecord {
	if item == nil {
		return nil
	}
	return &analyticsv1.AnalyticsRecord{
		Id:                   item.ID,
		Code:                 item.Code,
		Title:                item.Title,
		Subtitle:             item.Subtitle,
		Category:             item.Category,
		SubCategory:          item.SubCategory,
		Status:               item.Status,
		Priority:             item.Priority,
		Region:               item.Region,
		Channel:              item.Channel,
		OwnerName:            item.OwnerName,
		OwnerTeam:            item.OwnerTeam,
		SourceSystem:         item.SourceSystem,
		MetricPv:             item.MetricPV,
		MetricUv:             item.MetricUV,
		MetricClick:          item.MetricClick,
		MetricConvert:        item.MetricConvert,
		MetricRevenue:        item.MetricRevenue,
		MetricCost:           item.MetricCost,
		MetricRoi:            item.MetricROI,
		MetricBounceRate:     item.MetricBounceRate,
		MetricAvgDurationSec: item.MetricAvgDurationSec,
		ScoreQuality:         item.ScoreQuality,
		ScoreRisk:            item.ScoreRisk,
		TagPrimary:           item.TagPrimary,
		TagSecondary:         item.TagSecondary,
		FlagFeatured:         item.FlagFeatured,
		FlagAnomaly:          item.FlagAnomaly,
		Notes:                item.Notes,
		ObservedAt:           item.ObservedAt,
		WindowStart:          item.WindowStart,
		WindowEnd:            item.WindowEnd,
		PublishedAt:          item.PublishedAt,
		CreatedAt:            item.CreatedAt,
		UpdatedAt:            item.UpdatedAt,
	}
}
