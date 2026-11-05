package short_video

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"app/internal/testsupport"
	"github.com/google/uuid"
	"model/entity"
)

type memShortVideoRepo struct {
	mu     sync.Mutex
	videos map[uuid.UUID]*entity.WysShortVideo
	likes  map[string]struct{} // videoID|userID
	topics map[uuid.UUID]*entity.WysTopic
	users  map[string]entity.AuthorProfile
}

func newMemShortVideoRepo() *memShortVideoRepo {
	return &memShortVideoRepo{
		videos: map[uuid.UUID]*entity.WysShortVideo{},
		likes:  map[string]struct{}{},
		topics: map[uuid.UUID]*entity.WysTopic{},
		users:  map[string]entity.AuthorProfile{},
	}
}

func likeKey(vid uuid.UUID, uid string) string { return vid.String() + "|" + uid }

func (m *memShortVideoRepo) Create(_ context.Context, v *entity.WysShortVideo) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *v
	m.videos[v.ID] = &cp
	return nil
}

func (m *memShortVideoRepo) GetByID(_ context.Context, id uuid.UUID) (*entity.WysShortVideo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.videos[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *v
	return &cp, nil
}

func (m *memShortVideoRepo) SoftDelete(_ context.Context, id uuid.UUID, userID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.videos[id]
	if !ok || v.DeletedAt != nil || v.UserID != userID {
		return false, nil
	}
	now := time.Now()
	v.DeletedAt = &now
	return true, nil
}

func publicOrApproved(v *entity.WysShortVideo) bool {
	return v.Status == entity.ShortVideoStatusNormal
}

func (m *memShortVideoRepo) List(_ context.Context, q ShortVideoListQuery) ([]entity.WysShortVideo, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var all []entity.WysShortVideo
	for _, v := range m.videos {
		if v.DeletedAt != nil {
			continue
		}
		switch q.Scope {
		case ShortVideoScopeDiscovery:
			if !q.IncludeReviewing && !publicOrApproved(v) {
				continue
			}
		case ShortVideoScopeUser:
			if v.UserID != q.AuthorID {
				continue
			}
			if !q.IncludeReviewing && !publicOrApproved(v) {
				continue
			}
		default:
			continue
		}
		all = append(all, *v)
	}
	// newest first
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].CreatedAt.After(all[i].CreatedAt) {
				all[i], all[j] = all[j], all[i]
			}
		}
	}
	total := int64(len(all))
	if q.Offset >= len(all) {
		return []entity.WysShortVideo{}, total, nil
	}
	end := q.Offset + q.Limit
	if end > len(all) {
		end = len(all)
	}
	return all[q.Offset:end], total, nil
}

func (m *memShortVideoRepo) ApproveDue(_ context.Context, ids []uuid.UUID, now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range ids {
		if v, ok := m.videos[id]; ok {
			v.Status = entity.ShortVideoStatusNormal
			t := now
			v.ApprovedAt = &t
		}
	}
	return nil
}

func (m *memShortVideoRepo) MarkApproved(_ context.Context, id uuid.UUID, now time.Time) error {
	return m.ApproveDue(context.Background(), []uuid.UUID{id}, now)
}

func (m *memShortVideoRepo) AddLike(_ context.Context, videoID uuid.UUID, userID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := likeKey(videoID, userID)
	if _, ok := m.likes[k]; ok {
		return false, nil
	}
	m.likes[k] = struct{}{}
	if v, ok := m.videos[videoID]; ok {
		v.LikeCount++
	}
	return true, nil
}

func (m *memShortVideoRepo) RemoveLike(_ context.Context, videoID uuid.UUID, userID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := likeKey(videoID, userID)
	if _, ok := m.likes[k]; !ok {
		return false, nil
	}
	delete(m.likes, k)
	if v, ok := m.videos[videoID]; ok && v.LikeCount > 0 {
		v.LikeCount--
	}
	return true, nil
}

func (m *memShortVideoRepo) LikedIDs(_ context.Context, userID string, videoIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[uuid.UUID]bool{}
	for _, id := range videoIDs {
		if _, ok := m.likes[likeKey(id, userID)]; ok {
			out[id] = true
		}
	}
	return out, nil
}

func (m *memShortVideoRepo) IncrementView(_ context.Context, id uuid.UUID) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.videos[id]
	if !ok {
		return 0, ErrNotFound
	}
	v.ViewCount++
	return v.ViewCount, nil
}

func (m *memShortVideoRepo) Stats(_ context.Context, authorID string, includeReviewing bool) (entity.ShortVideoStats, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var s entity.ShortVideoStats
	for _, v := range m.videos {
		if v.DeletedAt != nil || v.UserID != authorID {
			continue
		}
		if !includeReviewing && !publicOrApproved(v) {
			continue
		}
		s.VideoCount++
		s.ViewCount += v.ViewCount
		s.LikeCount += int64(v.LikeCount)
	}
	return s, nil
}

func (m *memShortVideoRepo) GetTopic(_ context.Context, id uuid.UUID) (*entity.WysTopic, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.topics[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *t
	return &cp, nil
}

func (m *memShortVideoRepo) TopicsByIDs(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]*entity.WysTopic, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[uuid.UUID]*entity.WysTopic{}
	for _, id := range ids {
		if t, ok := m.topics[id]; ok {
			cp := *t
			out[id] = &cp
		}
	}
	return out, nil
}

func (m *memShortVideoRepo) AuthorsByIDs(_ context.Context, ids []string) (map[string]entity.AuthorProfile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]entity.AuthorProfile{}
	for _, id := range ids {
		if p, ok := m.users[id]; ok {
			out[id] = p
		}
	}
	return out, nil
}

func TestShortVideo_CreateDefaultsAndReviewing(t *testing.T) {
	repo := newMemShortVideoRepo()
	repo.users["u1"] = entity.AuthorProfile{Nickname: "开发者", Avatar: "a.png"}
	uc := NewService(repo, nil, false, "dev")
	fixed := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	uc.now = func() time.Time { return fixed }

	dto, err := uc.Create(context.Background(), "u1", CreateShortVideoInput{Title: "试驾"})
	if err != nil {
		t.Fatal(err)
	}
	if dto.Status != "reviewing" {
		t.Fatalf("status=%s", dto.Status)
	}
	if dto.VideoURL == "" || dto.CoverURL == "" {
		t.Fatal("expected default media")
	}
	if dto.Duration != "0:15" || dto.AspectRatio != 1.25 {
		t.Fatalf("defaults duration/aspect: %s %v", dto.Duration, dto.AspectRatio)
	}
	if dto.Title != "试驾" || !dto.IsMine {
		t.Fatalf("dto=%+v", dto)
	}
}

func TestShortVideo_CreateRejectsEmptyTitle(t *testing.T) {
	uc := NewService(newMemShortVideoRepo(), nil, false, "dev")
	_, err := uc.Create(context.Background(), "u1", CreateShortVideoInput{Title: "  "})
	if err == nil || !errors.Is(err, ErrInvalid) {
		t.Fatalf("err=%v", err)
	}
}

// parity-p0-5：正式环境对象存储未接入，缺真实 URL 必须拒绝而非填样例地址。
func TestShortVideo_ProdEnvRejectsMockMedia(t *testing.T) {
	repo := newMemShortVideoRepo()
	repo.users["u1"] = entity.AuthorProfile{Nickname: "A"}
	uc := NewService(repo, nil, false, "production")

	if _, err := uc.Create(context.Background(), "u1", CreateShortVideoInput{Title: "no-media"}); err == nil || !errors.Is(err, ErrInvalid) {
		t.Fatalf("prod 缺 URL 应拒绝，err=%v", err)
	}

	video, cover := "https://cdn.example.com/a.mp4", "https://cdn.example.com/a.jpg"
	dto, err := uc.Create(context.Background(), "u1", CreateShortVideoInput{Title: "real", VideoURL: &video, CoverURL: &cover})
	if err != nil {
		t.Fatalf("prod 带真实 URL 应放行，err=%v", err)
	}
	if dto.VideoURL != video || dto.CoverURL != cover {
		t.Fatalf("dto=%+v", dto)
	}
}

// 关键反证（parity-p0-4）：待审内容不会「等一会儿」自动对外可见。
func TestShortVideo_ReviewingNeverPassesByTimeAlone(t *testing.T) {
	repo := newMemShortVideoRepo()
	repo.users["u1"] = entity.AuthorProfile{Nickname: "A"}
	uc := NewService(repo, nil, false, "dev")
	t0 := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	uc.now = func() time.Time { return t0 }

	dto, err := uc.Create(context.Background(), "u1", CreateShortVideoInput{Title: "hidden"})
	if err != nil {
		t.Fatal(err)
	}

	list, total, err := uc.List(context.Background(), "u2", "discovery", "", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || len(list) != 0 {
		t.Fatalf("discovery should hide reviewing: total=%d len=%d", total, len(list))
	}

	mine, total, err := uc.List(context.Background(), "u1", "user", "u1", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || mine[0].ID != dto.ID || mine[0].Status != "reviewing" {
		t.Fatalf("author list: %+v total=%d", mine, total)
	}

	// 远超原 30s 假审核阈值：状态仍是 reviewing，对外依旧不可见
	uc.now = func() time.Time { return t0.Add(72 * time.Hour) }
	list, total, err = uc.List(context.Background(), "u2", "discovery", "", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || len(list) != 0 {
		t.Fatalf("time must not approve content: total=%d len=%d", total, len(list))
	}
	if _, err := uc.Get(context.Background(), "u2", dto.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("non-author get: %v", err)
	}
	row, err := repo.GetByID(context.Background(), uuid.MustParse(dto.ID))
	if err != nil {
		t.Fatal(err)
	}
	if row.Status != entity.ShortVideoStatusReviewing {
		t.Fatalf("status leaked to %d without manual review", row.Status)
	}
	// 作者侧统计含自己的待审内容，他人侧不含
	if p, err := uc.Profile(context.Background(), "u1", "u1"); err != nil || p.Stats.VideoCount != 1 {
		t.Fatalf("author profile: %+v err=%v", p, err)
	}
	if p, err := uc.Profile(context.Background(), "u2", "u1"); err != nil || p.Stats.VideoCount != 0 {
		t.Fatalf("visitor profile: %+v err=%v", p, err)
	}
}

func TestShortVideo_ManualApproveRequiresPlatformAdmin(t *testing.T) {
	repo := newMemShortVideoRepo()
	repo.users["u1"] = entity.AuthorProfile{Nickname: "A"}
	accessStub := &testsupport.AccessStub{
		PlatformPermsByUser: map[string][]string{"admin": {entity.PermRoleAssignPlatform}},
	}
	uc := NewService(repo, accessStub.Service(), false, "dev")
	uc.now = func() time.Time { return time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC) }

	dto, err := uc.Create(context.Background(), "u1", CreateShortVideoInput{Title: "v"})
	if err != nil {
		t.Fatal(err)
	}

	// 无权限：作者本人与陌生人都放行不了
	if _, err := uc.Approve(context.Background(), "u1", dto.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("author approve: %v", err)
	}
	if _, err := uc.Approve(context.Background(), "u2", dto.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("stranger approve: %v", err)
	}

	out, err := uc.Approve(context.Background(), "admin", dto.ID)
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "normal" || out.ApprovedAt == nil {
		t.Fatalf("approved dto=%+v", out)
	}
	list, total, err := uc.List(context.Background(), "u2", "discovery", "", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || list[0].Status != "normal" {
		t.Fatalf("after manual approve: total=%d list=%+v", total, list)
	}
	// 重复放行幂等，不冲掉 approved_at
	again, err := uc.Approve(context.Background(), "admin", dto.ID)
	if err != nil {
		t.Fatal(err)
	}
	if again.ApprovedAt == nil || *again.ApprovedAt != *out.ApprovedAt {
		t.Fatalf("re-approve changed approved_at: %v -> %v", *out.ApprovedAt, *again.ApprovedAt)
	}
}

// 演示模式（APP_ENV 非正式 + review_mode=auto_allow_local_only）：读取即放行，联调不阻塞。
func TestShortVideo_AutoAllowDemoMode(t *testing.T) {
	repo := newMemShortVideoRepo()
	repo.users["u1"] = entity.AuthorProfile{Nickname: "A"}
	uc := NewService(repo, nil, true, "dev")
	uc.now = func() time.Time { return time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC) }

	if _, err := uc.Create(context.Background(), "u1", CreateShortVideoInput{Title: "demo"}); err != nil {
		t.Fatal(err)
	}
	list, total, err := uc.List(context.Background(), "u2", "discovery", "", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || list[0].Status != "normal" {
		t.Fatalf("demo mode should allow: total=%d list=%+v", total, list)
	}
}

func TestShortVideo_SoftDeleteLikeView(t *testing.T) {
	repo := newMemShortVideoRepo()
	repo.users["u1"] = entity.AuthorProfile{Nickname: "A"}
	accessStub := &testsupport.AccessStub{
		PlatformPermsByUser: map[string][]string{"admin": {entity.PermRoleAssignPlatform}},
	}
	uc := NewService(repo, accessStub.Service(), false, "dev")
	uc.now = func() time.Time { return time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC) }

	dto, err := uc.Create(context.Background(), "u1", CreateShortVideoInput{Title: "v"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.Approve(context.Background(), "admin", dto.ID); err != nil {
		t.Fatal(err)
	}

	n, err := uc.AddView(context.Background(), "u2", dto.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("view=%d", n)
	}
	n, err = uc.AddView(context.Background(), "u2", dto.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("view should stack: %d", n)
	}

	liked, err := uc.SetLike(context.Background(), "u2", dto.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if !liked.IsLiked || liked.LikeCount != 1 {
		t.Fatalf("liked=%+v", liked)
	}

	if err := uc.SoftDelete(context.Background(), "u2", dto.ID); err == nil || !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-author delete: %v", err)
	}
	if err := uc.SoftDelete(context.Background(), "u1", dto.ID); err != nil {
		t.Fatal(err)
	}
	_, err = uc.Get(context.Background(), "u1", dto.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted get: %v", err)
	}
}

func TestShortVideo_ProfileStats(t *testing.T) {
	repo := newMemShortVideoRepo()
	repo.users["u1"] = entity.AuthorProfile{Nickname: "开发者", Avatar: "x"}
	uc := NewService(repo, nil, false, "dev")
	uc.now = func() time.Time { return time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC) }

	dto, _ := uc.Create(context.Background(), "u1", CreateShortVideoInput{Title: "a"})
	_, _ = uc.AddView(context.Background(), "u1", dto.ID)
	_, _ = uc.SetLike(context.Background(), "u1", dto.ID, true)

	p, err := uc.Profile(context.Background(), "u1", "")
	if err != nil {
		t.Fatal(err)
	}
	if !p.IsMe || p.Stats.VideoCount != 1 || p.Stats.ViewCount != 1 || p.Stats.LikeCount != 1 {
		t.Fatalf("profile=%+v", p)
	}
}

func TestShortVideo_TopicNotFound(t *testing.T) {
	uc := NewService(newMemShortVideoRepo(), nil, false, "dev")
	tid := uuid.New().String()
	_, err := uc.Create(context.Background(), "u1", CreateShortVideoInput{
		Title: "x", TopicID: &tid,
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
}
