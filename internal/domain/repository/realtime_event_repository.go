package repository

import (
	"context"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

// RealtimeEventRepository 用户增量事件仓储（重连 sync）。
type RealtimeEventRepository interface {
	NextSeq(ctx context.Context, userID string) (int64, error)
	Append(ctx context.Context, userID string, event entity.RealtimeEnvelope, retention int) error
	ListSince(ctx context.Context, userID string, sinceSeq int64, topics []string, retention int) ([]entity.RealtimeEnvelope, int64, error)
}
