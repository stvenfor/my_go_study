package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

type memNewCarFollowRepo struct {
	nextFileID     int64
	nextCustomerID int64
	nextLogID      int64
	files          map[int64]*entity.WysNewCarFollowFile
	customers      map[int64]*entity.WysStoreCustomer
	logs           map[int64]*entity.WysNewCarFollowLog
	userName       string
	avatarURL      string
	namesByUser    map[string]string
}

func newMemFollowRepo() *memNewCarFollowRepo {
	return &memNewCarFollowRepo{
		nextFileID:     1,
		nextCustomerID: 1,
		nextLogID:      1,
		files:          map[int64]*entity.WysNewCarFollowFile{},
		customers:      map[int64]*entity.WysStoreCustomer{},
		logs:           map[int64]*entity.WysNewCarFollowLog{},
		userName:       "东东枪",
		avatarURL:      "https://example.com/a.png",
	}
}

func (m *memNewCarFollowRepo) CountStats(_ context.Context, storeID int, owner string, now time.Time) (entity.NewCarFollowStats, error) {
	var s entity.NewCarFollowStats
	for _, row := range m.files {
		if row.StoreID != storeID {
			continue
		}
		if owner != "" && row.OwnerUserID != owner {
			continue
		}
		open := entity.FollowFileIsOpen(row.Stage)
		if open {
			s.Active++
			if row.NextFollowUpAt != nil && !row.NextFollowUpAt.After(now) {
				s.Overdue++
			}
			if row.FollowLevel == entity.FollowLevelH || row.FollowLevel == entity.FollowLevelA {
				s.HighIntent++
			}
		}
		if row.Stage == entity.FollowStageLost {
			s.Lost++
		}
	}
	return s, nil
}

func (m *memNewCarFollowRepo) ListFiles(
	_ context.Context, storeID int, owner string, f entity.NewCarFollowListFilter, now time.Time, offset, limit int,
) ([]entity.WysNewCarFollowFile, int64, error) {
	var all []entity.WysNewCarFollowFile
	for _, row := range m.files {
		if row.StoreID != storeID {
			continue
		}
		if owner != "" && row.OwnerUserID != owner {
			continue
		}
		if f.FollowLevel != "" && row.FollowLevel != f.FollowLevel {
			continue
		}
		if levels := entity.FollowLevelsForIntentBand(f.IntentBand); len(levels) > 0 {
			ok := false
			for _, lv := range levels {
				if row.FollowLevel == lv {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		if f.Stage != nil && row.Stage != *f.Stage {
			continue
		}
		if f.OpenOnly && !entity.FollowFileIsOpen(row.Stage) {
			continue
		}
		if f.OverdueOnly {
			if !entity.FollowFileIsOpen(row.Stage) || row.NextFollowUpAt == nil || row.NextFollowUpAt.After(now) {
				continue
			}
		}
		all = append(all, *row)
	}
	total := int64(len(all))
	if offset >= len(all) {
		return nil, total, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}

func (m *memNewCarFollowRepo) GetFile(_ context.Context, id int64) (*entity.WysNewCarFollowFile, error) {
	row, ok := m.files[id]
	if !ok {
		return nil, repository.ErrNewCarFollowNotFound
	}
	cp := *row
	return &cp, nil
}

func (m *memNewCarFollowRepo) CreateFile(_ context.Context, row *entity.WysNewCarFollowFile) error {
	row.FileID = m.nextFileID
	m.nextFileID++
	cp := *row
	m.files[row.FileID] = &cp
	if row.NextFollowUpAt != nil {
		if c, ok := m.customers[row.CustomerID]; ok {
			c.NextFollowUpAt = row.NextFollowUpAt
		}
	}
	return nil
}

func (m *memNewCarFollowRepo) UpdateFileAndCustomerFollow(_ context.Context, row *entity.WysNewCarFollowFile, sync bool) error {
	if _, ok := m.files[row.FileID]; !ok {
		return repository.ErrNewCarFollowNotFound
	}
	cp := *row
	m.files[row.FileID] = &cp
	if sync {
		if c, ok := m.customers[row.CustomerID]; ok {
			c.NextFollowUpAt = row.NextFollowUpAt
		}
	}
	return nil
}

func (m *memNewCarFollowRepo) FindOpenFileByCustomer(_ context.Context, storeID int, customerID int64) (*entity.WysNewCarFollowFile, error) {
	for _, row := range m.files {
		if row.StoreID == storeID && row.CustomerID == customerID && entity.FollowFileIsOpen(row.Stage) {
			cp := *row
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *memNewCarFollowRepo) ListLogs(_ context.Context, fileID int64, offset, limit int) ([]entity.WysNewCarFollowLog, int64, error) {
	var all []entity.WysNewCarFollowLog
	for _, row := range m.logs {
		if row.FileID == fileID {
			all = append(all, *row)
		}
	}
	// newest first
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].CreatedAt.After(all[i].CreatedAt) ||
				(all[j].CreatedAt.Equal(all[i].CreatedAt) && all[j].LogID > all[i].LogID) {
				all[i], all[j] = all[j], all[i]
			}
		}
	}
	total := int64(len(all))
	if offset >= len(all) {
		return nil, total, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}

func (m *memNewCarFollowRepo) CreateLogAndTouchFile(_ context.Context, log *entity.WysNewCarFollowLog, file *entity.WysNewCarFollowFile, sync bool) error {
	log.LogID = m.nextLogID
	m.nextLogID++
	cpLog := *log
	m.logs[log.LogID] = &cpLog
	cpFile := *file
	m.files[file.FileID] = &cpFile
	if sync {
		if c, ok := m.customers[file.CustomerID]; ok {
			c.NextFollowUpAt = file.NextFollowUpAt
		}
	}
	return nil
}

func (m *memNewCarFollowRepo) GetCustomer(_ context.Context, id int64) (*entity.WysStoreCustomer, error) {
	c, ok := m.customers[id]
	if !ok {
		return nil, repository.ErrNewCarFollowBadCustomer
	}
	cp := *c
	return &cp, nil
}

func (m *memNewCarFollowRepo) CreateCustomer(_ context.Context, row *entity.WysStoreCustomer) error {
	row.CustomerID = m.nextCustomerID
	m.nextCustomerID++
	cp := *row
	m.customers[row.CustomerID] = &cp
	return nil
}

func (m *memNewCarFollowRepo) ListCustomers(_ context.Context, storeID int, _ string, offset, limit int) ([]entity.WysStoreCustomer, int64, error) {
	var all []entity.WysStoreCustomer
	for _, c := range m.customers {
		if c.StoreID == storeID {
			all = append(all, *c)
		}
	}
	total := int64(len(all))
	if offset >= len(all) {
		return nil, total, nil
	}
	end := offset + limit
	if end > len(all) {
		end = len(all)
	}
	return all[offset:end], total, nil
}

func (m *memNewCarFollowRepo) GetUserBrief(_ context.Context, userID string) (string, string, error) {
	if m.namesByUser != nil {
		if n, ok := m.namesByUser[userID]; ok {
			return n, m.avatarURL, nil
		}
	}
	return m.userName, m.avatarURL, nil
}

func TestFollowLevelIntentBandMapping(t *testing.T) {
	cases := []struct {
		level string
		band  string
	}{
		{"H", "高"}, {"A", "高"}, {"B", "中"}, {"E", "低"}, {"h", "高"},
	}
	for _, tc := range cases {
		lv, ok := entity.NormalizeFollowLevel(tc.level)
		if !ok {
			t.Fatalf("normalize %s", tc.level)
		}
		if got := entity.IntentBandFromFollowLevel(lv); got != tc.band {
			t.Fatalf("%s → %s want %s", tc.level, got, tc.band)
		}
	}
	if _, ok := entity.NormalizeFollowLevel("C"); ok {
		t.Fatal("C should be invalid")
	}
}

func TestNewCarFollowCreateRejectsBadLevel(t *testing.T) {
	repo := newMemFollowRepo()
	access := &stubAccessForDeal{storeID: 1, store: &entity.WysStore{StoreID: 1, Name: "店"}}
	uc := NewNewCarFollowUsecase(repo, access.asUsecase())
	_, err := uc.Create(context.Background(), "u1", CreateNewCarFollowInput{
		DisplayName: "张三", Phone: "13800000000", FollowLevel: "C",
	})
	if err != ErrNewCarFollowBadLevel {
		t.Fatalf("got %v", err)
	}
}

func TestNewCarFollowCreateAndIntentFilter(t *testing.T) {
	repo := newMemFollowRepo()
	access := &stubAccessForDeal{storeID: 1, store: &entity.WysStore{StoreID: 1, Name: "店"}}
	uc := NewNewCarFollowUsecase(repo, access.asUsecase())
	fixed := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	uc.now = func() time.Time { return fixed }
	past := fixed.Add(-time.Hour)

	h, err := uc.Create(context.Background(), "u1", CreateNewCarFollowInput{
		DisplayName: "高意向", Phone: "13800000001", FollowLevel: "H", NextFollowUpAt: &past,
	})
	if err != nil {
		t.Fatal(err)
	}
	if h.IntentBand != "高" || h.FollowLevel != "H" {
		t.Fatalf("dto=%+v", h)
	}
	if repo.customers[1].NextFollowUpAt == nil || !repo.customers[1].NextFollowUpAt.Equal(past) {
		t.Fatalf("customer next_follow not synced: %+v", repo.customers[1])
	}

	_, err = uc.Create(context.Background(), "u1", CreateNewCarFollowInput{
		DisplayName: "中意向", Phone: "13800000002", FollowLevel: "B",
	})
	if err != nil {
		t.Fatal(err)
	}

	high, total, err := uc.List(context.Background(), "u1", "", "高", "", false, 1, 10)
	if err != nil || total != 1 || len(high) != 1 || high[0].FollowLevel != "H" {
		t.Fatalf("intent high total=%d list=%+v err=%v", total, high, err)
	}
	onlyB, totalB, err := uc.List(context.Background(), "u1", "B", "", "", false, 1, 10)
	if err != nil || totalB != 1 || onlyB[0].FollowLevel != "B" {
		t.Fatalf("level B total=%d list=%+v err=%v", totalB, onlyB, err)
	}
	overdue, totalO, err := uc.List(context.Background(), "u1", "", "", "", true, 1, 10)
	if err != nil || totalO != 1 || overdue[0].FollowLevel != "H" {
		t.Fatalf("overdue total=%d list=%+v err=%v", totalO, overdue, err)
	}
}

func TestNewCarFollowGetForbiddenOtherOwner(t *testing.T) {
	repo := newMemFollowRepo()
	repo.customers[1] = &entity.WysStoreCustomer{CustomerID: 1, StoreID: 1, DisplayName: "x", Phone: "1"}
	repo.files[9] = &entity.WysNewCarFollowFile{
		FileID: 9, StoreID: 1, OwnerUserID: "other", CustomerID: 1,
		FollowLevel: "A", Stage: entity.FollowStageFollowing,
	}
	access := &stubAccessForDeal{storeID: 1}
	uc := NewNewCarFollowUsecase(repo, access.asUsecase())
	if _, err := uc.Get(context.Background(), "u1", 9); err != ErrNewCarFollowNotFound {
		t.Fatalf("got %v", err)
	}
}

func TestNewCarFollowPatchSyncNextFollow(t *testing.T) {
	repo := newMemFollowRepo()
	repo.customers[1] = &entity.WysStoreCustomer{CustomerID: 1, StoreID: 1, DisplayName: "x", Phone: "138"}
	repo.files[1] = &entity.WysNewCarFollowFile{
		FileID: 1, StoreID: 1, OwnerUserID: "u1", CustomerID: 1,
		FollowLevel: "E", Stage: entity.FollowStageFollowing, CustomerPhone: "138",
	}
	access := &stubAccessForDeal{storeID: 1}
	uc := NewNewCarFollowUsecase(repo, access.asUsecase())
	fixed := time.Date(2026, 9, 22, 18, 0, 0, 0, time.UTC)
	uc.now = func() time.Time { return fixed }
	next := fixed.Add(-time.Minute)
	lv := "A"
	dto, err := uc.Patch(context.Background(), "u1", 1, PatchNewCarFollowInput{
		FollowLevel: &lv, NextFollowUpAt: &next, TouchNextFollow: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if dto.FollowLevel != "A" || dto.IntentBand != "高" {
		t.Fatalf("dto=%+v", dto)
	}
	if repo.customers[1].NextFollowUpAt == nil || !repo.customers[1].NextFollowUpAt.Equal(next) {
		t.Fatalf("customer not synced")
	}
}

func TestNewCarFollowNoStore(t *testing.T) {
	repo := newMemFollowRepo()
	uc := NewNewCarFollowUsecase(repo, (&stubAccessForDeal{storeID: 0}).asUsecase())
	if _, err := uc.Summary(context.Background(), "u1"); err != ErrNewCarFollowNoStore {
		t.Fatalf("got %v", err)
	}
}

func TestNewCarFollowCreateLogAndOrder(t *testing.T) {
	repo := newMemFollowRepo()
	repo.customers[1] = &entity.WysStoreCustomer{CustomerID: 1, StoreID: 1, DisplayName: "x", Phone: "138"}
	repo.files[1] = &entity.WysNewCarFollowFile{
		FileID: 1, StoreID: 1, OwnerUserID: "u1", CustomerID: 1,
		FollowLevel: "E", Stage: entity.FollowStageNew, CustomerPhone: "138",
	}
	access := &stubAccessForDeal{storeID: 1}
	uc := NewNewCarFollowUsecase(repo, access.asUsecase())
	fixed := time.Date(2026, 9, 23, 10, 0, 0, 0, time.UTC)
	uc.now = func() time.Time { return fixed }
	next := fixed.Add(24 * time.Hour)

	log1, err := uc.CreateLog(context.Background(), "u1", 1, CreateFollowLogInput{
		Body: "第一次拜访", FollowLevel: "B", NextFollowUpAt: &next, TouchNext: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if log1.Body != "第一次拜访" || log1.FollowLevel != "B" {
		t.Fatalf("log=%+v", log1)
	}
	file := repo.files[1]
	if file.LastFollowAt == nil || !file.LastFollowAt.Equal(fixed) {
		t.Fatalf("last_follow=%v", file.LastFollowAt)
	}
	if file.FollowLevel != "B" || file.Stage != entity.FollowStageFollowing {
		t.Fatalf("file=%+v", file)
	}
	if repo.customers[1].NextFollowUpAt == nil || !repo.customers[1].NextFollowUpAt.Equal(next) {
		t.Fatal("customer next not synced")
	}

	uc.now = func() time.Time { return fixed.Add(time.Hour) }
	if _, err := uc.CreateLog(context.Background(), "u1", 1, CreateFollowLogInput{Body: "第二次"}); err != nil {
		t.Fatal(err)
	}
	logs, total, err := uc.ListLogs(context.Background(), "u1", 1, 1, 10)
	if err != nil || total != 2 || len(logs) != 2 {
		t.Fatalf("total=%d logs=%+v err=%v", total, logs, err)
	}
	if logs[0].Body != "第二次" || logs[1].Body != "第一次拜访" {
		t.Fatalf("order=%+v", logs)
	}
	if _, err := uc.CreateLog(context.Background(), "u1", 1, CreateFollowLogInput{Body: "  "}); err != ErrNewCarFollowBadLogBody {
		t.Fatalf("empty body got %v", err)
	}
	if _, err := uc.CreateLog(context.Background(), "other", 1, CreateFollowLogInput{Body: "x"}); err != ErrNewCarFollowNotFound {
		t.Fatalf("forbidden got %v", err)
	}
}

func TestNewCarFollowStoreAdminSeesOthers(t *testing.T) {
	repo := newMemFollowRepo()
	repo.namesByUser = map[string]string{"u1": "销售甲", "admin": "店管"}
	repo.customers[1] = &entity.WysStoreCustomer{CustomerID: 1, StoreID: 1, DisplayName: "客", Phone: "1"}
	repo.files[9] = &entity.WysNewCarFollowFile{
		FileID: 9, StoreID: 1, OwnerUserID: "u1", CustomerID: 1,
		FollowLevel: "A", Stage: entity.FollowStageFollowing, CustomerName: "客", CustomerPhone: "1",
	}
	access := &stubAccessForDeal{
		storeID: 1,
		store:   &entity.WysStore{StoreID: 1, Name: "店"},
		storePermsByUser: map[string][]string{
			"admin": {entity.PermRoleAssignStore},
		},
	}
	uc := NewNewCarFollowUsecase(repo, access.asUsecase())

	if _, err := uc.Get(context.Background(), "stranger", 9); err != ErrNewCarFollowNotFound {
		t.Fatalf("stranger got %v", err)
	}
	dto, err := uc.Get(context.Background(), "admin", 9)
	if err != nil {
		t.Fatal(err)
	}
	if dto.OwnerUserID != "u1" || dto.OwnerDisplayName != "销售甲" {
		t.Fatalf("dto=%+v", dto)
	}
	list, total, err := uc.List(context.Background(), "admin", "", "", "", false, 1, 10)
	if err != nil || total != 1 || list[0].OwnerDisplayName != "销售甲" {
		t.Fatalf("list total=%d %+v err=%v", total, list, err)
	}
	sum, err := uc.Summary(context.Background(), "admin")
	if err != nil || sum.Stats.Active != 1 {
		t.Fatalf("summary=%+v err=%v", sum, err)
	}
	// sales still only self
	listSelf, totalSelf, err := uc.List(context.Background(), "u1", "", "", "", false, 1, 10)
	if err != nil || totalSelf != 1 || listSelf[0].FileID != "9" {
		t.Fatalf("sales list total=%d %+v err=%v", totalSelf, listSelf, err)
	}
	repo.files[10] = &entity.WysNewCarFollowFile{
		FileID: 10, StoreID: 1, OwnerUserID: "other", CustomerID: 1,
		FollowLevel: "B", Stage: entity.FollowStageFollowing,
	}
	listSelf2, totalSelf2, _ := uc.List(context.Background(), "u1", "", "", "", false, 1, 10)
	if totalSelf2 != 1 || listSelf2[0].FileID != "9" {
		t.Fatalf("sales should not see other: total=%d %+v", totalSelf2, listSelf2)
	}
	listAdmin, totalAdmin, _ := uc.List(context.Background(), "admin", "", "", "", false, 1, 10)
	if totalAdmin != 2 {
		t.Fatalf("admin total=%d list=%+v", totalAdmin, listAdmin)
	}
}

