package usecase

import (
	"context"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"github.com/stvenfor/my_go_study/pkg/config"
)

// RealtimeSyncInput 增量同步请求。
type RealtimeSyncInput struct {
	UserID   string
	SinceSeq int64
	Topics   []string
}

// RealtimeSyncOutput 增量同步响应。
type RealtimeSyncOutput struct {
	Events    []entity.RealtimeEnvelope
	LatestSeq int64
}

// RealtimeSyncUsecase 重连 sync 用例。
type RealtimeSyncUsecase struct {
	events repository.RealtimeEventRepository
	cfg    config.Config
}

// NewRealtimeSyncUsecase 创建 sync 用例。
func NewRealtimeSyncUsecase(events repository.RealtimeEventRepository, cfg config.Config) *RealtimeSyncUsecase {
	return &RealtimeSyncUsecase{events: events, cfg: cfg}
}

// Sync 按 seq 补发事件。
func (u *RealtimeSyncUsecase) Sync(ctx context.Context, input RealtimeSyncInput) (*RealtimeSyncOutput, error) {
	events, latestSeq, err := u.events.ListSince(
		ctx,
		input.UserID,
		input.SinceSeq,
		input.Topics,
		u.cfg.Realtime.EventRetention,
	)
	if err != nil {
		return nil, err
	}
	if events == nil {
		events = []entity.RealtimeEnvelope{}
	}
	return &RealtimeSyncOutput{
		Events:    events,
		LatestSeq: latestSeq,
	}, nil
}
