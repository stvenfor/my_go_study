package short_video

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"model/entity"
)

// PostgresRepository 小视频 Postgres 实现。
type PostgresRepository struct {
	db *gorm.DB
}

func NewPostgresRepository(db *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, v *entity.WysShortVideo) error {
	return r.db.WithContext(ctx).Create(v).Error
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*entity.WysShortVideo, error) {
	var row entity.WysShortVideo
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *PostgresRepository) SoftDelete(ctx context.Context, id uuid.UUID, userID string) (bool, error) {
	now := time.Now()
	res := r.db.WithContext(ctx).Model(&entity.WysShortVideo{}).
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", id, userID).
		Update("deleted_at", now)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *PostgresRepository) List(ctx context.Context, q ShortVideoListQuery) ([]entity.WysShortVideo, int64, error) {
	tx := r.db.WithContext(ctx).Model(&entity.WysShortVideo{}).Where("deleted_at IS NULL")

	switch q.Scope {
	case ShortVideoScopeDiscovery:
		if !q.IncludeReviewing {
			tx = tx.Where("status = ?", entity.ShortVideoStatusNormal)
		}
	case ShortVideoScopeUser:
		tx = tx.Where("user_id = ?", q.AuthorID)
		if !q.IncludeReviewing {
			tx = tx.Where("status = ?", entity.ShortVideoStatusNormal)
		}
	default:
		return nil, 0, nil
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []entity.WysShortVideo
	err := tx.Order("created_at DESC").Offset(q.Offset).Limit(q.Limit).Find(&list).Error
	return list, total, err
}

func (r *PostgresRepository) ApproveDue(ctx context.Context, ids []uuid.UUID, now time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&entity.WysShortVideo{}).
		Where("id IN ? AND status = ?", ids, entity.ShortVideoStatusReviewing).
		Updates(map[string]interface{}{
			"status":      entity.ShortVideoStatusNormal,
			"approved_at": now,
			"updated_at":  now,
		}).Error
}

func (r *PostgresRepository) MarkApproved(ctx context.Context, id uuid.UUID, now time.Time) error {
	return r.ApproveDue(ctx, []uuid.UUID{id}, now)
}

func (r *PostgresRepository) AddLike(ctx context.Context, videoID uuid.UUID, userID string) (bool, error) {
	var added bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row := entity.WysShortVideoLike{ShortVideoID: videoID, UserID: userID}
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
		if res.Error != nil {
			return res.Error
		}
		added = res.RowsAffected > 0
		if added {
			return tx.Model(&entity.WysShortVideo{}).Where("id = ?", videoID).
				Update("like_count", gorm.Expr("like_count + 1")).Error
		}
		return nil
	})
	return added, err
}

func (r *PostgresRepository) RemoveLike(ctx context.Context, videoID uuid.UUID, userID string) (bool, error) {
	var removed bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("short_video_id = ? AND user_id = ?", videoID, userID).
			Delete(&entity.WysShortVideoLike{})
		if res.Error != nil {
			return res.Error
		}
		removed = res.RowsAffected > 0
		if removed {
			return tx.Model(&entity.WysShortVideo{}).
				Where("id = ? AND like_count > 0", videoID).
				Update("like_count", gorm.Expr("like_count - 1")).Error
		}
		return nil
	})
	return removed, err
}

func (r *PostgresRepository) LikedIDs(ctx context.Context, userID string, videoIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	out := map[uuid.UUID]bool{}
	if len(videoIDs) == 0 {
		return out, nil
	}
	var rows []entity.WysShortVideoLike
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND short_video_id IN ?", userID, videoIDs).
		Find(&rows).Error
	for _, row := range rows {
		out[row.ShortVideoID] = true
	}
	return out, err
}

func (r *PostgresRepository) IncrementView(ctx context.Context, id uuid.UUID) (int64, error) {
	var viewCount int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&entity.WysShortVideo{}).Where("id = ? AND deleted_at IS NULL", id).
			Update("view_count", gorm.Expr("view_count + 1"))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		return tx.Model(&entity.WysShortVideo{}).Where("id = ?", id).
			Select("view_count").Scan(&viewCount).Error
	})
	return viewCount, err
}

func (r *PostgresRepository) Stats(ctx context.Context, authorID string, includeReviewing bool) (entity.ShortVideoStats, error) {
	tx := r.db.WithContext(ctx).Model(&entity.WysShortVideo{}).
		Where("user_id = ? AND deleted_at IS NULL", authorID)
	if !includeReviewing {
		tx = tx.Where("status = ?", entity.ShortVideoStatusNormal)
	}
	var stats entity.ShortVideoStats
	err := tx.Select("COUNT(*) AS video_count, COALESCE(SUM(view_count),0) AS view_count, COALESCE(SUM(like_count),0) AS like_count").
		Scan(&stats).Error
	return stats, err
}

func (r *PostgresRepository) GetTopic(ctx context.Context, id uuid.UUID) (*entity.WysTopic, error) {
	var t entity.WysTopic
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *PostgresRepository) TopicsByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*entity.WysTopic, error) {
	out := map[uuid.UUID]*entity.WysTopic{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []entity.WysTopic
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		t := rows[i]
		out[t.ID] = &t
	}
	return out, nil
}

func (r *PostgresRepository) AuthorsByIDs(ctx context.Context, ids []string) (map[string]entity.AuthorProfile, error) {
	out := map[string]entity.AuthorProfile{}
	if len(ids) == 0 {
		return out, nil
	}
	var users []entity.User
	if err := r.db.WithContext(ctx).Where("user_id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, u := range users {
		out[u.UserID] = entity.AuthorProfile{
			Nickname: u.UserName,
			Avatar:   u.AvatarURL,
		}
	}
	return out, nil
}
