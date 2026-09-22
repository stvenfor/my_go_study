package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CommunityRepository struct {
	db *gorm.DB
}

func NewCommunityRepository(db *gorm.DB) *CommunityRepository {
	return &CommunityRepository{db: db}
}

func (r *CommunityRepository) ListTopics(ctx context.Context, q string, offset, limit int) ([]entity.WysTopic, int64, error) {
	tx := r.db.WithContext(ctx).Model(&entity.WysTopic{})
	if q != "" {
		tx = tx.Where("name ILIKE ?", "%"+q+"%")
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []entity.WysTopic
	err := tx.Order("is_ask_everyone DESC, heat DESC, created_at DESC").
		Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

func (r *CommunityRepository) GetTopic(ctx context.Context, id uuid.UUID) (*entity.WysTopic, error) {
	var t entity.WysTopic
	if err := r.db.WithContext(ctx).First(&t, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrCommunityNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r *CommunityRepository) CreatePost(ctx context.Context, post *entity.WysPost) error {
	return r.db.WithContext(ctx).Create(post).Error
}

func (r *CommunityRepository) GetPost(ctx context.Context, id uuid.UUID) (*entity.WysPost, error) {
	var p entity.WysPost
	if err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrCommunityNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *CommunityRepository) SoftDeletePost(ctx context.Context, id uuid.UUID, userID string) (bool, error) {
	now := time.Now()
	res := r.db.WithContext(ctx).Model(&entity.WysPost{}).
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", id, userID).
		Update("deleted_at", now)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *CommunityRepository) ListPosts(ctx context.Context, offset, limit int) ([]entity.WysPost, int64, error) {
	tx := r.db.WithContext(ctx).Model(&entity.WysPost{}).Where("deleted_at IS NULL")
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []entity.WysPost
	err := tx.Order("created_at DESC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

func (r *CommunityRepository) AddLike(ctx context.Context, postID uuid.UUID, userID string) (bool, error) {
	var added bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		like := entity.WysPostLike{PostID: postID, UserID: userID}
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&like)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		upd := tx.Model(&entity.WysPost{}).Where("id = ? AND deleted_at IS NULL", postID).
			UpdateColumn("like_count", gorm.Expr("like_count + 1"))
		if upd.Error != nil {
			return upd.Error
		}
		if upd.RowsAffected == 0 {
			return repository.ErrCommunityNotFound
		}
		added = true
		return nil
	})
	return added, err
}

func (r *CommunityRepository) RemoveLike(ctx context.Context, postID uuid.UUID, userID string) (bool, error) {
	var removed bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("post_id = ? AND user_id = ?", postID, userID).Delete(&entity.WysPostLike{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		upd := tx.Model(&entity.WysPost{}).Where("id = ? AND deleted_at IS NULL AND like_count > 0", postID).
			UpdateColumn("like_count", gorm.Expr("like_count - 1"))
		if upd.Error != nil {
			return upd.Error
		}
		removed = true
		return nil
	})
	return removed, err
}

func (r *CommunityRepository) LikedPostIDs(ctx context.Context, userID string, postIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	out := map[uuid.UUID]bool{}
	if len(postIDs) == 0 {
		return out, nil
	}
	var likes []entity.WysPostLike
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND post_id IN ?", userID, postIDs).
		Find(&likes).Error; err != nil {
		return nil, err
	}
	for _, l := range likes {
		out[l.PostID] = true
	}
	return out, nil
}

func (r *CommunityRepository) CreateComment(ctx context.Context, c *entity.WysPostComment) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var n int64
		if err := tx.Model(&entity.WysPost{}).Where("id = ? AND deleted_at IS NULL", c.PostID).Count(&n).Error; err != nil {
			return err
		}
		if n == 0 {
			return repository.ErrCommunityNotFound
		}
		if err := tx.Create(c).Error; err != nil {
			return err
		}
		return tx.Model(&entity.WysPost{}).Where("id = ? AND deleted_at IS NULL", c.PostID).
			UpdateColumn("comment_count", gorm.Expr("comment_count + 1")).Error
	})
}

func (r *CommunityRepository) ListComments(ctx context.Context, postID uuid.UUID, offset, limit int) ([]entity.WysPostComment, int64, error) {
	tx := r.db.WithContext(ctx).Model(&entity.WysPostComment{}).
		Where("post_id = ? AND deleted_at IS NULL", postID)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []entity.WysPostComment
	err := tx.Order("created_at ASC").Offset(offset).Limit(limit).Find(&list).Error
	return list, total, err
}

func (r *CommunityRepository) PreviewComments(ctx context.Context, postID uuid.UUID, limit int) ([]entity.WysPostComment, error) {
	var list []entity.WysPostComment
	err := r.db.WithContext(ctx).
		Where("post_id = ? AND deleted_at IS NULL", postID).
		Order("created_at DESC").Limit(limit).Find(&list).Error
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	return list, err
}

func (r *CommunityRepository) AuthorsByIDs(ctx context.Context, ids []string) (map[string]repository.AuthorProfile, error) {
	out := map[string]repository.AuthorProfile{}
	if len(ids) == 0 {
		return out, nil
	}
	var users []entity.User
	if err := r.db.WithContext(ctx).Where("user_id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, u := range users {
		out[u.UserID] = repository.AuthorProfile{
			Nickname: u.UserName,
			Avatar:   u.AvatarURL,
		}
	}
	return out, nil
}

func (r *CommunityRepository) EnsureSeed(ctx context.Context) error {
	var topicCount int64
	if err := r.db.WithContext(ctx).Model(&entity.WysTopic{}).Count(&topicCount).Error; err != nil {
		return err
	}
	if topicCount == 0 {
		topics := []entity.WysTopic{
			{ID: uuid.New(), Name: "问大家", Heat: 0, IsAskEveryone: true},
			{ID: uuid.New(), Name: "纳指大涨超2%再创新高", Heat: 1015000},
			{ID: uuid.New(), Name: "Meta智能体引爆AI交易", Heat: 21500},
			{ID: uuid.New(), Name: "Flutter开发", Heat: 8800},
			{ID: uuid.New(), Name: "周末打卡", Heat: 3200},
		}
		if err := r.db.WithContext(ctx).Create(&topics).Error; err != nil {
			return fmt.Errorf("seed topics: %w", err)
		}
	}

	var postCount int64
	if err := r.db.WithContext(ctx).Model(&entity.WysPost{}).Where("deleted_at IS NULL").Count(&postCount).Error; err != nil {
		return err
	}
	if postCount > 0 {
		return nil
	}

	var user entity.User
	if err := r.db.WithContext(ctx).Order("user_id ASC").First(&user).Error; err != nil {
		return nil
	}

	var askTopic entity.WysTopic
	_ = r.db.WithContext(ctx).Where("is_ask_everyone = ?", true).First(&askTopic).Error
	var flutterTopic entity.WysTopic
	_ = r.db.WithContext(ctx).Where("name = ?", "Flutter开发").First(&flutterTopic).Error

	imgJSON, _ := json.Marshal([]string{
		"https://picsum.photos/seed/wys_seed_1/400/400",
		"https://picsum.photos/seed/wys_seed_2/400/400",
	})
	vid := "https://flutter.github.io/assets-for-api-docs/assets/videos/bee.mp4"
	cover := "https://picsum.photos/seed/wys_seed_video/640/360"

	posts := []entity.WysPost{
		{
			ID: uuid.New(), UserID: user.UserID, Content: "社区种子：欢迎来到盘友圈", MediaType: entity.MediaNone,
			ImageURLs: []byte("[]"), Source: "来自 iPhone",
		},
		{
			ID: uuid.New(), UserID: user.UserID, Content: "分享几张图\n#Flutter开发", MediaType: entity.MediaImage,
			ImageURLs: imgJSON, Source: "来自 Android", TopicID: ptrUUID(flutterTopic.ID),
		},
		{
			ID: uuid.New(), UserID: user.UserID, Content: "视频打卡\n#问大家", MediaType: entity.MediaVideo,
			ImageURLs: []byte("[]"), VideoURL: &vid, VideoCoverURL: &cover,
			Source: "来自 iPhone", IsAskEveryone: true, TopicID: ptrUUID(askTopic.ID),
		},
	}
	for i := range posts {
		if err := r.db.WithContext(ctx).Create(&posts[i]).Error; err != nil {
			return fmt.Errorf("seed post: %w", err)
		}
	}
	return nil
}

func ptrUUID(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}

var _ repository.CommunityRepository = (*CommunityRepository)(nil)
