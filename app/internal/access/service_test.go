package access

import (
	"context"
	"testing"

	"model/entity"
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
	return nil, ErrMemberNotFound
}
func (m *mockAccessRepo) GetRole(_ context.Context, code string) (*entity.Role, error) {
	if r, ok := m.roles[code]; ok {
		return &r, nil
	}
	return nil, ErrRoleNotFound
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

func TestService_PlatformIgnoresCurrentStore(t *testing.T) {
	repo := &mockAccessRepo{
		platform: map[string][]string{"admin": {entity.PermMemberWrite}},
		members:  map[string]map[int]bool{},
		current:  map[string]*int{},
	}
	svc := NewService(repo)
	storeID := 9
	if err := svc.require(context.Background(), "admin", entity.PermMemberWrite, &storeID); err != nil {
		t.Fatalf("platform should manage any store: %v", err)
	}
}

func TestService_StoreRoleNeedsCurrentStore(t *testing.T) {
	cur := 2
	repo := &mockAccessRepo{
		platform: map[string][]string{},
		store: map[string]map[int][]string{
			"mgr": {1: {entity.PermMemberWrite}, 2: {entity.PermMemberWrite}},
		},
		members: map[string]map[int]bool{"mgr": {1: true, 2: true}},
		current: map[string]*int{"mgr": &cur},
	}
	svc := NewService(repo)
	wrong := 1
	if err := svc.require(context.Background(), "mgr", entity.PermMemberWrite, &wrong); err == nil {
		t.Fatal("expected forbid when target != current store")
	}
	if err := svc.require(context.Background(), "mgr", entity.PermMemberWrite, &cur); err != nil {
		t.Fatalf("current store should allow: %v", err)
	}
}

func TestService_RevokeLastPlatformAdmin(t *testing.T) {
	repo := &mockAccessRepo{
		platform: map[string][]string{"admin": {entity.PermRoleAssignPlatform}},
		roles: map[string]entity.Role{
			entity.RolePlatformAdmin: {Code: entity.RolePlatformAdmin, Scope: entity.ScopePlatform},
		},
		admins: 1,
	}
	svc := NewService(repo)
	err := svc.RevokeRole(context.Background(), "admin", "admin", entity.RolePlatformAdmin, nil)
	if err != ErrLastPlatformAdmin {
		t.Fatalf("got %v", err)
	}
}

// TestService_NilReceiverSafe 覆盖切后新增的跨域读取 API 的 nil 接收者保护：
// access 未接线（配置关闭）时各调用方不应 panic。
func TestService_NilReceiverSafe(t *testing.T) {
	var svc *Service
	ctx := context.Background()
	if cur, err := svc.CurrentStoreID(ctx, "u1"); cur != nil || err != nil {
		t.Fatalf("CurrentStoreID: got (%v, %v)", cur, err)
	}
	noStore := ErrStoreNotFound
	if _, err := svc.RequireCurrentStore(ctx, "u1", noStore); err != noStore {
		t.Fatalf("RequireCurrentStore: want noStore, got %v", err)
	}
	if ok, err := svc.IsMember(ctx, "u1", 1); ok || err != nil {
		t.Fatalf("IsMember: got (%v, %v)", ok, err)
	}
	if name, label := svc.StoreLabels(ctx, "u1", 1); name != "" || label != "" {
		t.Fatalf("StoreLabels: got (%q, %q)", name, label)
	}
}
