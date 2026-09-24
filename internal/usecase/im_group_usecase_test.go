package usecase

import (
	"context"
	"testing"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

type stubRongGroup struct {
	created map[string][]string
	joined  map[string][]string
	quit    map[string][]string
	dismiss []string
}

func newStubRongGroup() *stubRongGroup {
	return &stubRongGroup{
		created: map[string][]string{},
		joined:  map[string][]string{},
		quit:    map[string][]string{},
	}
}

func (s *stubRongGroup) CreateGroup(_ context.Context, groupID, _ string, memberIDs []string) error {
	s.created[groupID] = append([]string{}, memberIDs...)
	return nil
}
func (s *stubRongGroup) JoinGroup(_ context.Context, groupID string, memberIDs []string) error {
	s.joined[groupID] = append(s.joined[groupID], memberIDs...)
	return nil
}
func (s *stubRongGroup) QuitGroup(_ context.Context, groupID string, memberIDs []string) error {
	s.quit[groupID] = append(s.quit[groupID], memberIDs...)
	return nil
}
func (s *stubRongGroup) DismissGroup(_ context.Context, groupID, _ string) error {
	s.dismiss = append(s.dismiss, groupID)
	return nil
}

type memGroupRepo struct {
	free   map[string]*entity.WysImFreeGroup
	member map[string]map[string]bool
	store  map[string]*entity.WysImStoreGroup
}

func newMemGroupRepo() *memGroupRepo {
	return &memGroupRepo{
		free: map[string]*entity.WysImFreeGroup{},
		member: map[string]map[string]bool{},
		store: map[string]*entity.WysImStoreGroup{},
	}
}

func (m *memGroupRepo) CreateFreeGroup(_ context.Context, g *entity.WysImFreeGroup, memberIDs []string) error {
	cp := *g
	m.free[g.GroupID] = &cp
	m.member[g.GroupID] = map[string]bool{}
	for _, id := range memberIDs {
		m.member[g.GroupID][id] = true
	}
	return nil
}
func (m *memGroupRepo) GetFreeGroup(_ context.Context, groupID string) (*entity.WysImFreeGroup, error) {
	g, ok := m.free[groupID]
	if !ok {
		return nil, repository.ErrImFreeGroupNotFound
	}
	cp := *g
	return &cp, nil
}
func (m *memGroupRepo) AddFreeMembers(_ context.Context, groupID string, memberIDs []string) error {
	if m.member[groupID] == nil {
		m.member[groupID] = map[string]bool{}
	}
	for _, id := range memberIDs {
		m.member[groupID][id] = true
	}
	return nil
}
func (m *memGroupRepo) RemoveFreeMembers(_ context.Context, groupID string, memberIDs []string) error {
	for _, id := range memberIDs {
		delete(m.member[groupID], id)
	}
	return nil
}
func (m *memGroupRepo) IsFreeMember(_ context.Context, groupID, userID string) (bool, error) {
	return m.member[groupID][userID], nil
}
func (m *memGroupRepo) DismissFreeGroup(_ context.Context, groupID string) error {
	g := m.free[groupID]
	if g == nil {
		return repository.ErrImFreeGroupNotFound
	}
	now := g.CreatedAt
	g.DismissedAt = &now
	return nil
}
func (m *memGroupRepo) GetStoreGroup(_ context.Context, storeID string) (*entity.WysImStoreGroup, error) {
	g, ok := m.store[storeID]
	if !ok {
		return nil, repository.ErrImFreeGroupNotFound
	}
	cp := *g
	return &cp, nil
}
func (m *memGroupRepo) UpsertStoreGroup(_ context.Context, g *entity.WysImStoreGroup) error {
	cp := *g
	m.store[g.StoreID] = &cp
	return nil
}

func TestImGroupUsecase_FreeGroupOwnerRules(t *testing.T) {
	rong := newStubRongGroup()
	repo := newMemGroupRepo()
	uc := NewImGroupUsecase(repo, rong)

	out, err := uc.CreateFreeGroup(context.Background(), "owner", "测试群", []string{"m1"})
	if err != nil {
		t.Fatal(err)
	}
	if err := uc.InviteFreeGroup(context.Background(), "m1", out.GroupID, []string{"m2"}); err == nil {
		t.Fatal("non-owner invite should fail")
	}
	if err := uc.InviteFreeGroup(context.Background(), "owner", out.GroupID, []string{"m2"}); err != nil {
		t.Fatal(err)
	}
	if err := uc.QuitFreeGroup(context.Background(), "m1", out.GroupID); err != nil {
		t.Fatal(err)
	}
	if err := uc.DismissFreeGroup(context.Background(), "owner", out.GroupID); err != nil {
		t.Fatal(err)
	}
}

func TestImGroupUsecase_StoreEnsure(t *testing.T) {
	rong := newStubRongGroup()
	uc := NewImGroupUsecase(newMemGroupRepo(), rong)
	out, err := uc.EnsureStoreMembership(context.Background(), "42", "u1")
	if err != nil {
		t.Fatal(err)
	}
	if out.GroupID != "store_42" {
		t.Fatalf("%+v", out)
	}
	// second call idempotent join
	if _, err := uc.EnsureStoreMembership(context.Background(), "42", "u2"); err != nil {
		t.Fatal(err)
	}
}
