package usecase

import (
	"context"
	"testing"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

type mockAccessRepo struct {
	platform map[string][]string
	store    map[string]map[int][]string
	members  map[string]map[int]bool
	current  map[string]*int
	roles    map[string]entity.Role
	admins   int
}

func (m *mockAccessRepo) CreateStore(context.Context, entity.WysStore) error { return nil }
func (m *mockAccessRepo) GetStore(context.Context, int) (*entity.WysStore, error) {
	return &entity.WysStore{}, nil
}
func (m *mockAccessRepo) UpsertMember(context.Context, entity.WysStoreMember) error { return nil }
func (m *mockAccessRepo) RemoveMember(context.Context, string, int) error           { return nil }
func (m *mockAccessRepo) GetMember(_ context.Context, userID string, storeID int) (*entity.WysStoreMember, error) {
	if m.members[userID][storeID] {
		return &entity.WysStoreMember{UserID: userID, StoreID: storeID}, nil
	}
	return nil, repository.ErrAccessMemberNotFound
}
func (m *mockAccessRepo) GetRole(_ context.Context, code string) (*entity.Role, error) {
	if r, ok := m.roles[code]; ok {
		return &r, nil
	}
	return nil, repository.ErrAccessRoleNotFound
}
func (m *mockAccessRepo) UserExists(context.Context, string) (bool, error) { return true, nil }
func (m *mockAccessRepo) CurrentStoreID(_ context.Context, userID string) (*int, error) {
	return m.current[userID], nil
}
func (m *mockAccessRepo) ClearCurrentStoreIf(context.Context, string, int) error { return nil }
func (m *mockAccessRepo) PlatformPermissionCodes(_ context.Context, userID string) ([]string, error) {
	return m.platform[userID], nil
}
func (m *mockAccessRepo) StorePermissionCodes(_ context.Context, userID string, storeID int) ([]string, error) {
	return m.store[userID][storeID], nil
}
func (m *mockAccessRepo) AssignRole(context.Context, entity.UserRole) error { return nil }
func (m *mockAccessRepo) RevokeRole(context.Context, string, string, *int) error {
	return nil
}
func (m *mockAccessRepo) HasRoleAssignment(context.Context, string, string, *int) (bool, error) {
	return true, nil
}
func (m *mockAccessRepo) CountPlatformAdmins(context.Context) (int, error) { return m.admins, nil }

func TestAccessUsecase_PlatformIgnoresCurrentStore(t *testing.T) {
	repo := &mockAccessRepo{
		platform: map[string][]string{"admin": {entity.PermMemberWrite}},
		members:  map[string]map[int]bool{},
		current:  map[string]*int{},
	}
	uc := NewAccessUsecase(repo)
	storeID := 9
	if err := uc.require(context.Background(), "admin", entity.PermMemberWrite, &storeID); err != nil {
		t.Fatalf("platform should manage any store: %v", err)
	}
}

func TestAccessUsecase_StoreRoleNeedsCurrentStore(t *testing.T) {
	cur := 2
	repo := &mockAccessRepo{
		platform: map[string][]string{},
		store: map[string]map[int][]string{
			"mgr": {1: {entity.PermMemberWrite}, 2: {entity.PermMemberWrite}},
		},
		members: map[string]map[int]bool{"mgr": {1: true, 2: true}},
		current: map[string]*int{"mgr": &cur},
	}
	uc := NewAccessUsecase(repo)
	wrong := 1
	if err := uc.require(context.Background(), "mgr", entity.PermMemberWrite, &wrong); err == nil {
		t.Fatal("expected forbid when target != current store")
	}
	if err := uc.require(context.Background(), "mgr", entity.PermMemberWrite, &cur); err != nil {
		t.Fatalf("current store should allow: %v", err)
	}
}

func TestAccessUsecase_RevokeLastPlatformAdmin(t *testing.T) {
	repo := &mockAccessRepo{
		platform: map[string][]string{"admin": {entity.PermRoleAssignPlatform}},
		roles: map[string]entity.Role{
			entity.RolePlatformAdmin: {Code: entity.RolePlatformAdmin, Scope: entity.ScopePlatform},
		},
		admins: 1,
	}
	uc := NewAccessUsecase(repo)
	err := uc.RevokeRole(context.Background(), "admin", "admin", entity.RolePlatformAdmin, nil)
	if err != ErrAccessLastPlatformAdmin {
		t.Fatalf("got %v", err)
	}
}
