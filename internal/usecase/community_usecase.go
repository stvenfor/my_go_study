package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

var (
	ErrCommunityInvalid   = errors.New("community: invalid params")
	ErrCommunityNotFound  = repository.ErrCommunityNotFound
	ErrCommunityForbidden = errors.New("community: forbidden")
)

var (
	defaultImageURLs = []string{
		"https://picsum.photos/seed/wys_post_1/400/400",
		"https://picsum.photos/seed/wys_post_2/400/400",
		"https://picsum.photos/seed/wys_post_3/400/400",
	}
	defaultVideoURL      = "https://flutter.github.io/assets-for-api-docs/assets/videos/bee.mp4"
	defaultVideoCoverURL = "https://picsum.photos/seed/wys_post_video/640/360"
)

// CommunityUsecase 社区动态。
type CommunityUsecase struct {
	repo              repository.CommunityRepository
	push              *RealtimePushUsecase // 可为 nil
	inviteUserIDs     []string
}

func NewCommunityUsecase(repo repository.CommunityRepository, push *RealtimePushUsecase, inviteUserIDs []string) *CommunityUsecase {
	return &CommunityUsecase{repo: repo, push: push, inviteUserIDs: inviteUserIDs}
}

func (u *CommunityUsecase) EnsureSeed(ctx context.Context) error {
	return u.repo.EnsureSeed(ctx)
}

// CreatePostInput 发帖写模型。
type CreatePostInput struct {
	Content         string
	MediaType       string // none|image|video
	ImageURLs       []string
	VideoURL        *string
	VideoCoverURL   *string
	TopicID         *string
	IsAskEveryone   bool
	Source          string
}

// TopicDTO / PostDTO / CommentDTO — 读模型 snake_case。
type TopicDTO struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Heat          int64  `json:"heat"`
	IsAskEveryone bool   `json:"is_ask_everyone"`
}

type TopicBriefDTO struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	IsAskEveryone bool   `json:"is_ask_everyone"`
}

type CommentDTO struct {
	ID              string  `json:"id"`
	PostID          string  `json:"post_id"`
	Nickname        string  `json:"nickname"`
	Avatar          string  `json:"avatar"`
	Content         string  `json:"content"`
	CreateTime      string  `json:"create_time"`
	ReplyToNickname *string `json:"reply_to_nickname"`
}

type PostDTO struct {
	ID              string         `json:"id"`
	UserID          string         `json:"user_id"`
	Nickname        string         `json:"nickname"`
	Avatar          string         `json:"avatar"`
	Content         string         `json:"content"`
	PublishTime     string         `json:"publish_time"`
	Source          string         `json:"source"`
	MediaType       string         `json:"media_type"`
	Images          []string       `json:"images"`
	VideoURL        *string        `json:"video_url"`
	VideoCoverURL   *string        `json:"video_cover_url"`
	LikeCount       int            `json:"like_count"`
	CommentCount    int            `json:"comment_count"`
	IsLiked         bool           `json:"is_liked"`
	IsMine          bool           `json:"is_mine"`
	IsAskEveryone   bool           `json:"is_ask_everyone"`
	Topic           *TopicBriefDTO `json:"topic"`
	PreviewComments []CommentDTO   `json:"preview_comments"`
}

func (u *CommunityUsecase) ListTopics(ctx context.Context, q string, page, size int) ([]TopicDTO, int64, error) {
	offset := (page - 1) * size
	list, total, err := u.repo.ListTopics(ctx, strings.TrimSpace(q), offset, size)
	if err != nil {
		return nil, 0, err
	}
	out := make([]TopicDTO, 0, len(list))
	for _, t := range list {
		out = append(out, TopicDTO{
			ID: t.ID.String(), Name: t.Name, Heat: t.Heat, IsAskEveryone: t.IsAskEveryone,
		})
	}
	return out, total, nil
}

func (u *CommunityUsecase) CreatePost(ctx context.Context, userID string, in CreatePostInput) (*PostDTO, error) {
	content := strings.TrimSpace(in.Content)
	mt, images, videoURL, coverURL, err := normalizeMedia(in.MediaType, in.ImageURLs, in.VideoURL, in.VideoCoverURL)
	if err != nil {
		return nil, err
	}
	if content == "" && mt == entity.MediaNone {
		return nil, fmt.Errorf("%w: 正文与媒体不能都为空", ErrCommunityInvalid)
	}

	var topicID *uuid.UUID
	var topic *entity.WysTopic
	isAsk := in.IsAskEveryone
	if in.TopicID != nil && strings.TrimSpace(*in.TopicID) != "" {
		id, err := uuid.Parse(strings.TrimSpace(*in.TopicID))
		if err != nil {
			return nil, fmt.Errorf("%w: topic_id", ErrCommunityInvalid)
		}
		t, err := u.repo.GetTopic(ctx, id)
		if err != nil {
			if errors.Is(err, repository.ErrCommunityNotFound) {
				return nil, ErrCommunityNotFound
			}
			return nil, err
		}
		topic = t
		topicID = &id
		if t.IsAskEveryone {
			isAsk = true
		}
		tag := "#" + t.Name
		if !strings.Contains(content, tag) {
			if content == "" {
				content = tag
			} else {
				content = content + "\n" + tag
			}
		}
	}

	imgBytes, _ := json.Marshal(images)
	source := strings.TrimSpace(in.Source)
	if source == "" {
		source = "来自 iPhone"
	}
	post := &entity.WysPost{
		ID:            uuid.New(),
		UserID:        userID,
		Content:       content,
		MediaType:     mt,
		ImageURLs:     imgBytes,
		VideoURL:      videoURL,
		VideoCoverURL: coverURL,
		TopicID:       topicID,
		IsAskEveryone: isAsk,
		Source:        source,
	}
	if err := u.repo.CreatePost(ctx, post); err != nil {
		return nil, err
	}

	if isAsk && u.push != nil {
		u.notifyAskEveryone(ctx, userID, post.ID.String())
	}

	_ = topic
	return u.buildPostDTO(ctx, post, userID, true)
}

func (u *CommunityUsecase) notifyAskEveryone(ctx context.Context, authorID, postID string) {
	for _, uid := range u.inviteUserIDs {
		uid = strings.TrimSpace(uid)
		if uid == "" || uid == authorID {
			continue
		}
		_, _ = u.push.PushToUser(ctx, RealtimePushInput{
			UserID: uid,
			Topic:  entity.TopicSysNotify,
			Title:  "有人邀请你回答",
			Body:   "来自盘友圈·问大家",
			Name:   entity.EventSysNotifyShow,
			Extra: map[string]any{
				"type":      "community.ask_everyone",
				"post_id":   postID,
				"deep_link": "/community?post_id=" + postID,
			},
		})
	}
}

func (u *CommunityUsecase) ListPosts(ctx context.Context, viewerID string, page, size int) ([]PostDTO, int64, error) {
	offset := (page - 1) * size
	list, total, err := u.repo.ListPosts(ctx, offset, size)
	if err != nil {
		return nil, 0, err
	}
	out, err := u.mapPosts(ctx, list, viewerID)
	return out, total, err
}

func (u *CommunityUsecase) SoftDelete(ctx context.Context, viewerID, postID string) error {
	id, err := uuid.Parse(postID)
	if err != nil {
		return ErrCommunityInvalid
	}
	post, err := u.repo.GetPost(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			return ErrCommunityNotFound
		}
		return err
	}
	if post.UserID != viewerID {
		return ErrCommunityForbidden
	}
	ok, err := u.repo.SoftDeletePost(ctx, id, viewerID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrCommunityNotFound
	}
	return nil
}

func (u *CommunityUsecase) SetLike(ctx context.Context, viewerID, postID string, liked bool) (*PostDTO, error) {
	id, err := uuid.Parse(postID)
	if err != nil {
		return nil, ErrCommunityInvalid
	}
	if _, err := u.repo.GetPost(ctx, id); err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			return nil, ErrCommunityNotFound
		}
		return nil, err
	}
	if liked {
		_, err = u.repo.AddLike(ctx, id, viewerID)
	} else {
		_, err = u.repo.RemoveLike(ctx, id, viewerID)
	}
	if err != nil {
		return nil, err
	}
	post, err := u.repo.GetPost(ctx, id)
	if err != nil {
		return nil, err
	}
	return u.buildPostDTO(ctx, post, viewerID, true)
}

func (u *CommunityUsecase) ListComments(ctx context.Context, postID string, page, size int) ([]CommentDTO, int64, error) {
	id, err := uuid.Parse(postID)
	if err != nil {
		return nil, 0, ErrCommunityInvalid
	}
	if _, err := u.repo.GetPost(ctx, id); err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			return nil, 0, ErrCommunityNotFound
		}
		return nil, 0, err
	}
	offset := (page - 1) * size
	list, total, err := u.repo.ListComments(ctx, id, offset, size)
	if err != nil {
		return nil, 0, err
	}
	dtos, err := u.mapComments(ctx, list)
	return dtos, total, err
}

func (u *CommunityUsecase) AddComment(ctx context.Context, viewerID, postID, content string, replyTo *string) (*CommentDTO, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("%w: content", ErrCommunityInvalid)
	}
	id, err := uuid.Parse(postID)
	if err != nil {
		return nil, ErrCommunityInvalid
	}
	if _, err := u.repo.GetPost(ctx, id); err != nil {
		if errors.Is(err, repository.ErrCommunityNotFound) {
			return nil, ErrCommunityNotFound
		}
		return nil, err
	}
	var reply *string
	if replyTo != nil {
		s := strings.TrimSpace(*replyTo)
		if s != "" {
			reply = &s
		}
	}
	c := &entity.WysPostComment{
		ID: uuid.New(), PostID: id, UserID: viewerID, Content: content, ReplyToNickname: reply,
	}
	if err := u.repo.CreateComment(ctx, c); err != nil {
		return nil, err
	}
	dtos, err := u.mapComments(ctx, []entity.WysPostComment{*c})
	if err != nil || len(dtos) == 0 {
		return nil, err
	}
	return &dtos[0], nil
}

func normalizeMedia(mediaType string, images []string, videoURL, coverURL *string) (int16, []string, *string, *string, error) {
	mt := strings.ToLower(strings.TrimSpace(mediaType))
	if mt == "" {
		mt = "none"
	}
	switch mt {
	case "none", "0":
		return entity.MediaNone, []string{}, nil, nil, nil
	case "image", "1":
		imgs := images
		if len(imgs) == 0 {
			imgs = append([]string{}, defaultImageURLs...)
		}
		if len(imgs) > 9 {
			return 0, nil, nil, nil, fmt.Errorf("%w: 图片最多 9 张", ErrCommunityInvalid)
		}
		return entity.MediaImage, imgs, nil, nil, nil
	case "video", "2":
		v := videoURL
		c := coverURL
		if v == nil || strings.TrimSpace(*v) == "" {
			s := defaultVideoURL
			v = &s
		}
		if c == nil || strings.TrimSpace(*c) == "" {
			s := defaultVideoCoverURL
			c = &s
		}
		return entity.MediaVideo, []string{}, v, c, nil
	default:
		return 0, nil, nil, nil, fmt.Errorf("%w: media_type", ErrCommunityInvalid)
	}
}

func mediaTypeString(mt int16) string {
	switch mt {
	case entity.MediaImage:
		return "image"
	case entity.MediaVideo:
		return "video"
	default:
		return "none"
	}
}

func (u *CommunityUsecase) mapPosts(ctx context.Context, list []entity.WysPost, viewerID string) ([]PostDTO, error) {
	if len(list) == 0 {
		return []PostDTO{}, nil
	}
	ids := make([]uuid.UUID, 0, len(list))
	authorIDs := make([]string, 0, len(list))
	topicIDs := map[uuid.UUID]struct{}{}
	for _, p := range list {
		ids = append(ids, p.ID)
		authorIDs = append(authorIDs, p.UserID)
		if p.TopicID != nil {
			topicIDs[*p.TopicID] = struct{}{}
		}
	}
	liked, err := u.repo.LikedPostIDs(ctx, viewerID, ids)
	if err != nil {
		return nil, err
	}
	authors, err := u.repo.AuthorsByIDs(ctx, authorIDs)
	if err != nil {
		return nil, err
	}
	topics := map[uuid.UUID]entity.WysTopic{}
	for tid := range topicIDs {
		t, err := u.repo.GetTopic(ctx, tid)
		if err == nil {
			topics[tid] = *t
		}
	}
	out := make([]PostDTO, 0, len(list))
	for _, p := range list {
		dto, err := u.projectPost(ctx, &p, viewerID, authors, liked, topics)
		if err != nil {
			return nil, err
		}
		out = append(out, *dto)
	}
	return out, nil
}

func (u *CommunityUsecase) buildPostDTO(ctx context.Context, p *entity.WysPost, viewerID string, withPreview bool) (*PostDTO, error) {
	authors, err := u.repo.AuthorsByIDs(ctx, []string{p.UserID})
	if err != nil {
		return nil, err
	}
	liked, err := u.repo.LikedPostIDs(ctx, viewerID, []uuid.UUID{p.ID})
	if err != nil {
		return nil, err
	}
	topics := map[uuid.UUID]entity.WysTopic{}
	if p.TopicID != nil {
		if t, err := u.repo.GetTopic(ctx, *p.TopicID); err == nil {
			topics[*p.TopicID] = *t
		}
	}
	dto, err := u.projectPost(ctx, p, viewerID, authors, liked, topics)
	if err != nil {
		return nil, err
	}
	if !withPreview {
		dto.PreviewComments = []CommentDTO{}
	}
	return dto, nil
}

func (u *CommunityUsecase) projectPost(
	ctx context.Context,
	p *entity.WysPost,
	viewerID string,
	authors map[string]repository.AuthorProfile,
	liked map[uuid.UUID]bool,
	topics map[uuid.UUID]entity.WysTopic,
) (*PostDTO, error) {
	images := []string{}
	_ = json.Unmarshal(p.ImageURLs, &images)
	if images == nil {
		images = []string{}
	}
	a := authors[p.UserID]
	nickname := a.Nickname
	if nickname == "" {
		nickname = "用户"
	}
	var topic *TopicBriefDTO
	if p.TopicID != nil {
		if t, ok := topics[*p.TopicID]; ok {
			topic = &TopicBriefDTO{ID: t.ID.String(), Name: t.Name, IsAskEveryone: t.IsAskEveryone}
		}
	}
	previews, err := u.repo.PreviewComments(ctx, p.ID, 2)
	if err != nil {
		return nil, err
	}
	previewDTOs, err := u.mapComments(ctx, previews)
	if err != nil {
		return nil, err
	}
	return &PostDTO{
		ID: p.ID.String(), UserID: p.UserID, Nickname: nickname, Avatar: a.Avatar,
		Content: p.Content, PublishTime: p.CreatedAt.UTC().Format(time.RFC3339),
		Source: p.Source, MediaType: mediaTypeString(p.MediaType), Images: images,
		VideoURL: p.VideoURL, VideoCoverURL: p.VideoCoverURL,
		LikeCount: p.LikeCount, CommentCount: p.CommentCount,
		IsLiked: liked[p.ID], IsMine: p.UserID == viewerID,
		IsAskEveryone: p.IsAskEveryone, Topic: topic, PreviewComments: previewDTOs,
	}, nil
}

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"

func (u *CommunityUsecase) mapComments(ctx context.Context, list []entity.WysPostComment) ([]CommentDTO, error) {
	if len(list) == 0 {
		return []CommentDTO{}, nil
	}
	ids := make([]string, 0, len(list))
	for _, c := range list {
		ids = append(ids, c.UserID)
	}
	authors, err := u.repo.AuthorsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]CommentDTO, 0, len(list))
	for _, c := range list {
		a := authors[c.UserID]
		nick := a.Nickname
		if nick == "" {
			nick = "用户"
		}
		out = append(out, CommentDTO{
			ID: c.ID.String(), PostID: c.PostID.String(), Nickname: nick, Avatar: a.Avatar,
			Content: c.Content, CreateTime: c.CreatedAt.UTC().Format(time.RFC3339),
			ReplyToNickname: c.ReplyToNickname,
		})
	}
	return out, nil
}
