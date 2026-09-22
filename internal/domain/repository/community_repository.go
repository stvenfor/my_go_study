package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

var ErrCommunityNotFound = errors.New("community: not found")

// CommunityRepository 社区动态本地仓储。
type CommunityRepository interface {
	ListTopics(ctx context.Context, q string, offset, limit int) ([]entity.WysTopic, int64, error)
	GetTopic(ctx context.Context, id uuid.UUID) (*entity.WysTopic, error)

	CreatePost(ctx context.Context, post *entity.WysPost) error
	GetPost(ctx context.Context, id uuid.UUID) (*entity.WysPost, error)
	SoftDeletePost(ctx context.Context, id uuid.UUID, userID string) (bool, error)
	ListPosts(ctx context.Context, offset, limit int) ([]entity.WysPost, int64, error)

	AddLike(ctx context.Context, postID uuid.UUID, userID string) (added bool, err error)
	RemoveLike(ctx context.Context, postID uuid.UUID, userID string) (removed bool, err error)
	LikedPostIDs(ctx context.Context, userID string, postIDs []uuid.UUID) (map[uuid.UUID]bool, error)

	CreateComment(ctx context.Context, c *entity.WysPostComment) error
	ListComments(ctx context.Context, postID uuid.UUID, offset, limit int) ([]entity.WysPostComment, int64, error)
	PreviewComments(ctx context.Context, postID uuid.UUID, limit int) ([]entity.WysPostComment, error)

	AuthorsByIDs(ctx context.Context, ids []string) (map[string]AuthorProfile, error)
	EnsureSeed(ctx context.Context) error
}

// AuthorProfile 列表展示用作者信息。
type AuthorProfile struct {
	Nickname string
	Avatar   string
}
