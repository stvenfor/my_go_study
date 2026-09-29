package short_video

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"model/entity"
)

var ErrNotFound = errors.New("short_video: not found")

const (
	ShortVideoScopeDiscovery = "discovery"
	ShortVideoScopeUser      = "user"
)

// ShortVideoListQuery 列表查询。
type ShortVideoListQuery struct {
	Scope            string
	AuthorID         string // scope=user
	IncludeReviewing bool   // 本人看自己
	Now              time.Time
	ReviewDelay      time.Duration
	Offset           int
	Limit            int
}

// Repository 小视频仓储。
type Repository interface {
	Create(ctx context.Context, v *entity.WysShortVideo) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.WysShortVideo, error)
	SoftDelete(ctx context.Context, id uuid.UUID, userID string) (bool, error)
	List(ctx context.Context, q ShortVideoListQuery) ([]entity.WysShortVideo, int64, error)
	ApproveDue(ctx context.Context, ids []uuid.UUID, now time.Time) error
	MarkApproved(ctx context.Context, id uuid.UUID, now time.Time) error

	AddLike(ctx context.Context, videoID uuid.UUID, userID string) (added bool, err error)
	RemoveLike(ctx context.Context, videoID uuid.UUID, userID string) (removed bool, err error)
	LikedIDs(ctx context.Context, userID string, videoIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	IncrementView(ctx context.Context, id uuid.UUID) (int64, error)

	Stats(ctx context.Context, authorID string, includeReviewing bool, now time.Time, delay time.Duration) (entity.ShortVideoStats, error)

	GetTopic(ctx context.Context, id uuid.UUID) (*entity.WysTopic, error)
	TopicsByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*entity.WysTopic, error)
	AuthorsByIDs(ctx context.Context, ids []string) (map[string]entity.AuthorProfile, error)
}
