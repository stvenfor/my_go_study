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
	files          map[int64]*entity.WysNewCarFollowFile
	customers      map[int64]*entity.WysStoreCustomer
	userName       string
	avatarURL      string
}

func newMemFollowRepo() *memNewCarFollowRepo {
	return &memNewCarFollowRepo{
		nextFileID:     1,
		nextCustomerID: 1,
		files:          map[int64]*entity.WysNewCarFollowFile{},
		customers:      map[int64]*entity.WysStoreCustomer{},
		userName:       "东东枪",
		avatarURL:      "https://example.com/a.png",
	}
}

func (m *memNewCarFollowRepo) CountStats(_ context.Context, storeID int, owner string, now time.Time) (entity.NewCarFollowStats, error) {
	var s entity.NewCarFollowStats
	for _, row := range m.files {
		if row.StoreID != storeID || row.OwnerUserID != owner {
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
		if row.StoreID != storeID || row.OwnerUserID != owner {
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

func (m *memNewCarFollowRepo) GetUserBrief(_ context.Context, _ string) (string, string, error) {
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
