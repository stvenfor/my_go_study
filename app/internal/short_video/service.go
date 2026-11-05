// service.go 小视频（发布 / 列表 / 审核 / 点赞 / 播放量 / 作者主页）—— 域内业务逻辑。
//
// 由 app/internal/usecase/short_video_usecase.go 迁入。四点改动：
//  1. ShortVideoUsecase → Service，NewShortVideoUsecase → NewService；
//  2. 错误哨兵 ErrShortVideo* → Err*（ErrNotFound 下沉 repository.go）；
//  3. repository.ShortVideoRepository → 域内 Repository；
//  4. repository.AuthorProfile → entity.AuthorProfile（跨域展示型作者信息，
//     community 与 short_video 双域消费，已上提跨域实体层）；
//     原 `errors.Is(err, repository.ErrCommunityNotFound)` 分支随之删除 ——
//     GetTopic 的 Postgres 实现本就返回本域 ErrNotFound，该分支是历史冗余。
//
// 审核语义（parity-p0-4）：发布 → `reviewing`，此后状态只由人工放行
// （Approve）改变；「等待若干秒即自动过审」的假审核路径已删除。
// 仅当 config 显式开启演示模式（short_video.review_mode=auto_allow_local_only
// 且 APP_ENV 非正式）时，读路径才就地放行，保证 Flutter 联调不阻塞。
package short_video

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"model/entity"

	"app/internal/access"
	"common/config"
)

var (
	ErrInvalid   = errors.New("short_video: invalid params")
	ErrForbidden = errors.New("short_video: forbidden")
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

// Service 小视频。
type Service struct {
	repo Repository
	// accessSvc 为 nil 时人工放行一律拒绝（缺权限体系 = 不放行）。
	accessSvc *access.Service
	// demoMedia 为 true 时允许用内置公网样例 URL 填充未上传的媒体（仅联调/演示）；
	// 正式环境对象存储未接入，缺 URL 必须报错而不是伪造媒体。
	demoMedia   bool
	autoApprove bool
	now         func() time.Time
}

// NewService 创建。autoApprove 仅应来自 config.ShortVideoAutoAllowUnreviewed
// （演示模式：读取即放行）；正式环境必须为 false，未过审内容只有人工放行后可见。
// appEnv 只用于 IsLabAppEnv 判定演示媒体填充，正式环境（prod/production）关闭。
func NewService(repo Repository, accessSvc *access.Service, autoApprove bool, appEnv string) *Service {
	return &Service{
		repo:        repo,
		accessSvc:   accessSvc,
		demoMedia:   config.IsLabAppEnv(appEnv),
		autoApprove: autoApprove,
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
	ID          string              `json:"id"`
	UserID      string              `json:"user_id"`
	Nickname    string              `json:"nickname"`
	Avatar      string              `json:"avatar"`
	Title       string              `json:"title"`
	CoverURL    string              `json:"cover_url"`
	VideoURL    string              `json:"video_url"`
	ViewCount   int64               `json:"view_count"`
	LikeCount   int                 `json:"like_count"`
	Duration    string              `json:"duration"`
	AspectRatio float64             `json:"aspect_ratio"`
	Status      string              `json:"status"`
	IsLiked     bool                `json:"is_liked"`
	IsMine      bool                `json:"is_mine"`
	Topic       *ShortVideoTopicDTO `json:"topic"`
	PublishTime string              `json:"publish_time"`
	ApprovedAt  *string             `json:"approved_at"`
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

func (u *Service) Create(ctx context.Context, userID string, in CreateShortVideoInput) (*ShortVideoDTO, error) {
	title := strings.TrimSpace(in.Title)
	if title == "" || utf8.RuneCountInString(title) > 50 {
		return nil, fmt.Errorf("%w: title", ErrInvalid)
	}

	id := uuid.New()
	// 对象存储未接入：正式环境不再用内置样例 URL 冒充已上传的视频/封面。
	if !u.demoMedia && (blankMediaURL(in.VideoURL) || blankMediaURL(in.CoverURL)) {
		return nil, fmt.Errorf("%w: 缺少 video_url / cover_url，上传通道未接入（见 docs/placeholder-inventory.md）", ErrInvalid)
	}
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
			return nil, fmt.Errorf("%w: topic_id", ErrInvalid)
		}
		t, err := u.repo.GetTopic(ctx, tid)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, ErrNotFound
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

func (u *Service) List(ctx context.Context, viewerID, scope, userID string, page, size int) ([]ShortVideoDTO, int64, error) {
	scope = strings.TrimSpace(scope)
	if scope != ShortVideoScopeDiscovery && scope != ShortVideoScopeUser {
		return nil, 0, fmt.Errorf("%w: scope", ErrInvalid)
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
	// 本人看自己带 reviewing；演示模式（autoApprove）下读取即放行，同样放行查询
	includeReviewing := u.autoApprove
	if scope == ShortVideoScopeUser {
		if authorID == "" {
			authorID = viewerID
		}
		includeReviewing = includeReviewing || authorID == viewerID
	}

	now := u.now()
	q := ShortVideoListQuery{
		Scope:            scope,
		AuthorID:         authorID,
		IncludeReviewing: includeReviewing,
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

func (u *Service) Get(ctx context.Context, viewerID, idStr string) (*ShortVideoDTO, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("%w: id", ErrInvalid)
	}
	row, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if row.DeletedAt != nil {
		return nil, ErrNotFound
	}
	now := u.now()
	if err := u.lazyApproveOne(ctx, row, now); err != nil {
		return nil, err
	}
	if !u.visibleTo(viewerID, row) {
		return nil, ErrNotFound
	}
	return u.toDTO(ctx, viewerID, row, nil, true)
}

func (u *Service) SoftDelete(ctx context.Context, userID, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("%w: id", ErrInvalid)
	}
	row, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if row.DeletedAt != nil {
		return ErrNotFound
	}
	if row.UserID != userID {
		return ErrForbidden
	}
	ok, err := u.repo.SoftDelete(ctx, id, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

func (u *Service) SetLike(ctx context.Context, userID, idStr string, liked bool) (*ShortVideoDTO, error) {
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

func (u *Service) AddView(ctx context.Context, viewerID, idStr string) (int64, error) {
	if _, err := u.Get(ctx, viewerID, idStr); err != nil {
		return 0, err
	}
	id := uuid.MustParse(idStr)
	return u.repo.IncrementView(ctx, id)
}

func (u *Service) Profile(ctx context.Context, viewerID, userID string) (*ShortVideoProfileDTO, error) {
	target := strings.TrimSpace(userID)
	if target == "" {
		target = viewerID
	}
	includeReviewing := u.autoApprove || target == viewerID
	stats, err := u.repo.Stats(ctx, target, includeReviewing)
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

func blankMediaURL(u *string) bool {
	return u == nil || strings.TrimSpace(*u) == ""
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

// visibleTo 未过审内容只有作者本人可见；不存在「等一会儿就对外可见」的路径。
func (u *Service) visibleTo(viewerID string, row *entity.WysShortVideo) bool {
	if row.DeletedAt != nil {
		return false
	}
	if row.UserID == viewerID {
		return true
	}
	return row.Status == entity.ShortVideoStatusNormal
}

// Approve 人工放行（POST /api/v1/short-videos/:id/review）。
// 权限对齐现成的平台能力码 role.assign_platform（平台管理员专属），不新建权限码以免动迁移。
func (u *Service) Approve(ctx context.Context, actorID, idStr string) (*ShortVideoDTO, error) {
	if u.accessSvc == nil {
		return nil, ErrForbidden
	}
	if err := u.accessSvc.RequirePermission(ctx, actorID, entity.PermRoleAssignPlatform, nil); err != nil {
		if errors.Is(err, access.ErrForbidden) {
			return nil, ErrForbidden
		}
		return nil, err
	}
	id, err := uuid.Parse(strings.TrimSpace(idStr))
	if err != nil {
		return nil, fmt.Errorf("%w: id", ErrInvalid)
	}
	row, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if row.DeletedAt != nil {
		return nil, ErrNotFound
	}
	if row.Status != entity.ShortVideoStatusReviewing {
		// 已是 normal 时幂等返回，避免重复放行把 approved_at 冲掉
		return u.toDTO(ctx, actorID, row, nil, true)
	}
	now := u.now()
	if err := u.repo.MarkApproved(ctx, id, now); err != nil {
		return nil, err
	}
	row.Status = entity.ShortVideoStatusNormal
	row.ApprovedAt = &now
	return u.toDTO(ctx, actorID, row, nil, true)
}

// lazyApproveOne 演示模式下把待审内容就地放行；正式模式（autoApprove=false）什么都不做。
func (u *Service) lazyApproveOne(ctx context.Context, row *entity.WysShortVideo, now time.Time) error {
	if !u.autoApprove || row.Status != entity.ShortVideoStatusReviewing {
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

func (u *Service) lazyApproveBatch(ctx context.Context, list []entity.WysShortVideo, now time.Time) error {
	if !u.autoApprove {
		return nil
	}
	ids := make([]uuid.UUID, 0)
	for i := range list {
		if list[i].Status == entity.ShortVideoStatusReviewing {
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

func (u *Service) toDTOList(ctx context.Context, viewerID string, list []entity.WysShortVideo) ([]ShortVideoDTO, error) {
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

func (u *Service) toDTO(ctx context.Context, viewerID string, row *entity.WysShortVideo, topic *entity.WysTopic, loadTopic bool) (*ShortVideoDTO, error) {
	list, err := u.toDTOList(ctx, viewerID, []entity.WysShortVideo{*row})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, ErrNotFound
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
