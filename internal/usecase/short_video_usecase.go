package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

var (
	ErrShortVideoInvalid   = errors.New("short_video: invalid params")
	ErrShortVideoNotFound  = repository.ErrShortVideoNotFound
	ErrShortVideoForbidden = errors.New("short_video: forbidden")
)

var (
	defaultShortVideoURLs = []string{
		"https://flutter.github.io/assets-for-api-docs/assets/videos/bee.mp4",
		"https://vjs.zencdn.net/v/oceans.mp4",
		"https://www.w3school.com.cn/example/html5/mov_bbb.mp4",
	}
	defaultShortCoverURLs = []string{
		"https://picsum.photos/seed/sv_play_1/400/640",
		"https://picsum.photos/seed/sv_play_2/400/500",
		"https://picsum.photos/seed/sv_play_3/400/700",
	}
	defaultShortDuration    = "0:15"
	defaultShortAspectRatio = 1.25
)

// ShortVideoUsecase 小视频。
type ShortVideoUsecase struct {
	repo         repository.ShortVideoRepository
	reviewDelay  time.Duration
	now          func() time.Time
}

func NewShortVideoUsecase(repo repository.ShortVideoRepository, reviewDelaySeconds int) *ShortVideoUsecase {
	if reviewDelaySeconds <= 0 {
		reviewDelaySeconds = 30
	}
	return &ShortVideoUsecase{
		repo:        repo,
		reviewDelay: time.Duration(reviewDelaySeconds) * time.Second,
		now:         time.Now,
	}
}

// CreateShortVideoInput 发布写模型。
type CreateShortVideoInput struct {
	Title       string
	VideoURL    *string
	CoverURL    *string
	Duration    *string
	AspectRatio *float64
	TopicID     *string
}

// ShortVideoTopicDTO 关联话题摘要。
type ShortVideoTopicDTO struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ShortVideoDTO 读模型。
type ShortVideoDTO struct {
	ID           string               `json:"id"`
	UserID       string               `json:"user_id"`
	Nickname     string               `json:"nickname"`
	Avatar       string               `json:"avatar"`
	Title        string               `json:"title"`
	CoverURL     string               `json:"cover_url"`
	VideoURL     string               `json:"video_url"`
	ViewCount    int64                `json:"view_count"`
	LikeCount    int                  `json:"like_count"`
	Duration     string               `json:"duration"`
	AspectRatio  float64              `json:"aspect_ratio"`
	Status       string               `json:"status"`
	IsLiked      bool                 `json:"is_liked"`
	IsMine       bool                 `json:"is_mine"`
	Topic        *ShortVideoTopicDTO  `json:"topic"`
	PublishTime  string               `json:"publish_time"`
	ApprovedAt   *string              `json:"approved_at"`
}

// ShortVideoProfileDTO 个人主页。
type ShortVideoProfileDTO struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	RoleBadge   string `json:"role_badge"`
	StoreName   string `json:"store_name"`
	IsMe        bool   `json:"is_me"`
	Stats       struct {
		VideoCount int64 `json:"video_count"`
		ViewCount  int64 `json:"view_count"`
		LikeCount  int64 `json:"like_count"`
	} `json:"stats"`
}

func (u *ShortVideoUsecase) Create(ctx context.Context, userID string, in CreateShortVideoInput) (*ShortVideoDTO, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" || utf8.RuneCountInString(title) > 50 {
		return nil, fmt.Errorf("%w: title", ErrShortVideoInvalid)
	}

	id := uuid.New()
	videoURL, coverURL := pickDefaultMedia(id, in.VideoURL, in.CoverURL)
	duration := defaultShortDuration
	if in.Duration != nil && strings.TrimSpace(*in.Duration) != "" {
		duration = strings.TrimSpace(*in.Duration)
	}
	aspect := defaultShortAspectRatio
	if in.AspectRatio != nil && *in.AspectRatio > 0 {
		aspect = *in.AspectRatio
	}

	var topicID *uuid.UUID
	var topic *entity.WysTopic
	if in.TopicID != nil && strings.TrimSpace(*in.TopicID) != "" {
		tid, err := uuid.Parse(strings.TrimSpace(*in.TopicID))
		if err != nil {
			return nil, fmt.Errorf("%w: topic_id", ErrShortVideoInvalid)
		}
		t, err := u.repo.GetTopic(ctx, tid)
		if err != nil {
			if errors.Is(err, repository.ErrShortVideoNotFound) || errors.Is(err, repository.ErrCommunityNotFound) {
				return nil, ErrShortVideoNotFound
			}
			return nil, err
		}
		topic = t
		topicID = &tid
	}

	now := u.now()
	row := &entity.WysShortVideo{
		ID:          id,
		UserID:      userID,
		Title:       title,
		VideoURL:    videoURL,
		CoverURL:    coverURL,
		Duration:    duration,
		AspectRatio: aspect,
		TopicID:     topicID,
		Status:      entity.ShortVideoStatusReviewing,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := u.repo.Create(ctx, row); err != nil {
		return nil, err
	}
	return u.toDTO(ctx, userID, row, topic, false)
}

func (u *ShortVideoUsecase) List(ctx context.Context, viewerID, scope, userID string, page, size int) ([]ShortVideoDTO, int64, error) {
	scope = strings.TrimSpace(scope)
	if scope != repository.ShortVideoScopeDiscovery && scope != repository.ShortVideoScopeUser {
		return nil, 0, fmt.Errorf("%w: scope", ErrShortVideoInvalid)
	}
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 50 {
		size = 50
	}

	authorID := strings.TrimSpace(userID)
	includeReviewing := false
	if scope == repository.ShortVideoScopeUser {
		if authorID == "" {
			authorID = viewerID
		}
		includeReviewing = authorID == viewerID
	}

	now := u.now()
	q := repository.ShortVideoListQuery{
		Scope:            scope,
		AuthorID:         authorID,
		IncludeReviewing: includeReviewing,
		Now:              now,
		ReviewDelay:      u.reviewDelay,
		Offset:           (page - 1) * size,
		Limit:            size,
	}
	list, total, err := u.repo.List(ctx, q)
	if err != nil {
		return nil, 0, err
	}
	if err := u.lazyApproveBatch(ctx, list, now); err != nil {
		return nil, 0, err
	}
	out, err := u.toDTOList(ctx, viewerID, list)
	return out, total, err
}

func (u *ShortVideoUsecase) Get(ctx context.Context, viewerID, idStr string) (*ShortVideoDTO, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("%w: id", ErrShortVideoInvalid)
	}
	row, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if row.DeletedAt != nil {
		return nil, ErrShortVideoNotFound
	}
	now := u.now()
	if err := u.lazyApproveOne(ctx, row, now); err != nil {
		return nil, err
	}
	if !u.visibleTo(viewerID, row, now) {
		return nil, ErrShortVideoNotFound
	}
	return u.toDTO(ctx, viewerID, row, nil, true)
}

func (u *ShortVideoUsecase) SoftDelete(ctx context.Context, userID, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("%w: id", ErrShortVideoInvalid)
	}
	row, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if row.DeletedAt != nil {
		return ErrShortVideoNotFound
	}
	if row.UserID != userID {
		return ErrShortVideoForbidden
	}
	ok, err := u.repo.SoftDelete(ctx, id, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrShortVideoNotFound
	}
	return nil
}

func (u *ShortVideoUsecase) SetLike(ctx context.Context, userID, idStr string, liked bool) (*ShortVideoDTO, error) {
	dto, err := u.Get(ctx, userID, idStr)
	if err != nil {
		return nil, err
	}
	id := uuid.MustParse(dto.ID)
	if liked {
		if _, err := u.repo.AddLike(ctx, id, userID); err != nil {
			return nil, err
		}
	} else {
		if _, err := u.repo.RemoveLike(ctx, id, userID); err != nil {
			return nil, err
		}
	}
	return u.Get(ctx, userID, idStr)
}

func (u *ShortVideoUsecase) AddView(ctx context.Context, viewerID, idStr string) (int64, error) {
	if _, err := u.Get(ctx, viewerID, idStr); err != nil {
		return 0, err
	}
	id := uuid.MustParse(idStr)
	return u.repo.IncrementView(ctx, id)
}

func (u *ShortVideoUsecase) Profile(ctx context.Context, viewerID, userID string) (*ShortVideoProfileDTO, error) {
	target := strings.TrimSpace(userID)
	if target == "" {
		target = viewerID
	}
	includeReviewing := target == viewerID
	now := u.now()
	stats, err := u.repo.Stats(ctx, target, includeReviewing, now, u.reviewDelay)
	if err != nil {
		return nil, err
	}
	authors, err := u.repo.AuthorsByIDs(ctx, []string{target})
	if err != nil {
		return nil, err
	}
	ap := authors[target]
	out := &ShortVideoProfileDTO{
		UserID:      target,
		DisplayName: ap.Nickname,
		AvatarURL:   ap.Avatar,
		RoleBadge:   "",
		StoreName:   "",
		IsMe:        target == viewerID,
	}
	out.Stats.VideoCount = stats.VideoCount
	out.Stats.ViewCount = stats.ViewCount
	out.Stats.LikeCount = stats.LikeCount
	return out, nil
}

func pickDefaultMedia(id uuid.UUID, videoURL, coverURL *string) (string, string) {
	idx := int(id[0]) % len(defaultShortVideoURLs)
	v := defaultShortVideoURLs[idx]
	c := defaultShortCoverURLs[idx]
	if videoURL != nil && strings.TrimSpace(*videoURL) != "" {
		v = strings.TrimSpace(*videoURL)
	}
	if coverURL != nil && strings.TrimSpace(*coverURL) != "" {
		c = strings.TrimSpace(*coverURL)
	}
	return v, c
}

func statusString(s int16) string {
	if s == entity.ShortVideoStatusNormal {
		return "normal"
	}
	return "reviewing"
}

func (u *ShortVideoUsecase) dueForApprove(row *entity.WysShortVideo, now time.Time) bool {
	return row.Status == entity.ShortVideoStatusReviewing &&
		!row.CreatedAt.After(now.Add(-u.reviewDelay))
}

func (u *ShortVideoUsecase) visibleTo(viewerID string, row *entity.WysShortVideo, now time.Time) bool {
	if row.DeletedAt != nil {
		return false
	}
	if row.UserID == viewerID {
		return true
	}
	if row.Status == entity.ShortVideoStatusNormal {
		return true
	}
	return u.dueForApprove(row, now)
}

func (u *ShortVideoUsecase) lazyApproveOne(ctx context.Context, row *entity.WysShortVideo, now time.Time) error {
	if !u.dueForApprove(row, now) {
		return nil
	}
	if err := u.repo.MarkApproved(ctx, row.ID, now); err != nil {
		return err
	}
	row.Status = entity.ShortVideoStatusNormal
	t := now
	row.ApprovedAt = &t
	return nil
}

func (u *ShortVideoUsecase) lazyApproveBatch(ctx context.Context, list []entity.WysShortVideo, now time.Time) error {
	ids := make([]uuid.UUID, 0)
	for i := range list {
		if u.dueForApprove(&list[i], now) {
			ids = append(ids, list[i].ID)
			list[i].Status = entity.ShortVideoStatusNormal
			t := now
			list[i].ApprovedAt = &t
		}
	}
	if len(ids) == 0 {
		return nil
	}
	return u.repo.ApproveDue(ctx, ids, now)
}

func (u *ShortVideoUsecase) toDTOList(ctx context.Context, viewerID string, list []entity.WysShortVideo) ([]ShortVideoDTO, error) {
	if len(list) == 0 {
		return []ShortVideoDTO{}, nil
	}
	ids := make([]uuid.UUID, len(list))
	authorIDs := make([]string, 0, len(list))
	topicIDs := map[uuid.UUID]struct{}{}
	seenAuthor := map[string]struct{}{}
	for i, v := range list {
		ids[i] = v.ID
		if _, ok := seenAuthor[v.UserID]; !ok {
			seenAuthor[v.UserID] = struct{}{}
			authorIDs = append(authorIDs, v.UserID)
		}
		if v.TopicID != nil {
			topicIDs[*v.TopicID] = struct{}{}
		}
	}
	liked, err := u.repo.LikedIDs(ctx, viewerID, ids)
	if err != nil {
		return nil, err
	}
	authors, err := u.repo.AuthorsByIDs(ctx, authorIDs)
	if err != nil {
		return nil, err
	}
	topicIDList := make([]uuid.UUID, 0, len(topicIDs))
	for tid := range topicIDs {
		topicIDList = append(topicIDList, tid)
	}
	topics, err := u.repo.TopicsByIDs(ctx, topicIDList)
	if err != nil {
		return nil, err
	}
	out := make([]ShortVideoDTO, 0, len(list))
	for _, v := range list {
		var topic *entity.WysTopic
		if v.TopicID != nil {
			topic = topics[*v.TopicID]
		}
		ap := authors[v.UserID]
		dto := ShortVideoDTO{
			ID:          v.ID.String(),
			UserID:      v.UserID,
			Nickname:    ap.Nickname,
			Avatar:      ap.Avatar,
			Title:       v.Title,
			CoverURL:    v.CoverURL,
			VideoURL:    v.VideoURL,
			ViewCount:   v.ViewCount,
			LikeCount:   v.LikeCount,
			Duration:    v.Duration,
			AspectRatio: v.AspectRatio,
			Status:      statusString(v.Status),
			IsLiked:     liked[v.ID],
			IsMine:      v.UserID == viewerID,
			PublishTime: v.CreatedAt.UTC().Format(time.RFC3339),
		}
		if topic != nil {
			dto.Topic = &ShortVideoTopicDTO{ID: topic.ID.String(), Name: topic.Name}
		}
		if v.ApprovedAt != nil {
			s := v.ApprovedAt.UTC().Format(time.RFC3339)
			dto.ApprovedAt = &s
		}
		out = append(out, dto)
	}
	return out, nil
}

func (u *ShortVideoUsecase) toDTO(ctx context.Context, viewerID string, row *entity.WysShortVideo, topic *entity.WysTopic, loadTopic bool) (*ShortVideoDTO, error) {
	list, err := u.toDTOList(ctx, viewerID, []entity.WysShortVideo{*row})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, ErrShortVideoNotFound
	}
	dto := list[0]
	if topic != nil {
		dto.Topic = &ShortVideoTopicDTO{ID: topic.ID.String(), Name: topic.Name}
	} else if loadTopic && row.TopicID != nil && dto.Topic == nil {
		t, err := u.repo.GetTopic(ctx, *row.TopicID)
		if err == nil {
			dto.Topic = &ShortVideoTopicDTO{ID: t.ID.String(), Name: t.Name}
		}
	}
	return &dto, nil
}
