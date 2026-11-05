// access_stub.go 域测试共用的 access.Repository 内存桩。
//
// 背景：切片前汽车业务各域的测试都定义过同一个 stubAccessForDeal（58 行、
// 实现 access.Repository 全量方法）。切片后四个域各自成包，同一份桩再复制
// 4 遍会随 access.Repository 的演进一起漂移，故上提到本测试支撑包。
//
// 本包只被各域的 *_test.go 引用，不会链进 api / worker 等生产二进制。
package testsupport

import (
	"context"

	"app/internal/access"
	"model/entity"
)

// AccessStub 实现 access.Repository，只覆盖测试用到的读取路径。
type AccessStub struct {
	// StoreID 当前选中门店；<=0 表示"未选门店"。
	StoreID int
	// Member 命中该用户 + 门店时返回的成员记录。
	Member *entity.WysStoreMember
	// Store 命中 StoreID 时返回的门店记录。
	Store *entity.WysStore
	// StorePermsByUser 按用户给出的店内能力码（配合 StoreID 判定成员）。
	StorePermsByUser map[string][]string
	// PlatformPermsByUser 按用户给出的平台能力码（平台级权限判定用）。
	PlatformPermsByUser map[string][]string
}

// Service 用桩构造真实的 access.Service（走公开方法，不走私有字段）。
func (s *AccessStub) Service() *access.Service {
	return access.NewService(s)
}

func (s *AccessStub) CreateStore(context.Context, entity.WysStore) error { return nil }

func (s *AccessStub) GetStore(_ context.Context, storeID int) (*entity.WysStore, error) {
	if s.Store != nil && s.Store.StoreID == storeID {
		return s.Store, nil
	}
	return nil, access.ErrStoreNotFound
}

func (s *AccessStub) UpsertMember(context.Context, entity.WysStoreMember) error { return nil }

func (s *AccessStub) RemoveMember(context.Context, string, int) error { return nil }

func (s *AccessStub) GetMember(_ context.Context, userID string, storeID int) (*entity.WysStoreMember, error) {
	if s.Member != nil && s.Member.UserID == userID && s.Member.StoreID == storeID {
		return s.Member, nil
	}
	if s.StorePermsByUser != nil {
		if _, ok := s.StorePermsByUser[userID]; ok && storeID == s.StoreID {
			return &entity.WysStoreMember{UserID: userID, StoreID: storeID}, nil
		}
	}
	return nil, access.ErrMemberNotFound
}

func (s *AccessStub) GetRole(context.Context, string) (*entity.Role, error) { return nil, nil }

func (s *AccessStub) UserExists(context.Context, string) (bool, error) { return true, nil }

func (s *AccessStub) CurrentStoreID(context.Context, string) (*int, error) {
	if s.StoreID <= 0 {
		return nil, nil
	}
	id := s.StoreID
	return &id, nil
}

func (s *AccessStub) ClearCurrentStoreIf(context.Context, string, int) error { return nil }

func (s *AccessStub) PlatformPermissionCodes(_ context.Context, userID string) ([]string, error) {
	if s.PlatformPermsByUser != nil {
		return s.PlatformPermsByUser[userID], nil
	}
	return nil, nil
}

func (s *AccessStub) StorePermissionCodes(_ context.Context, userID string, _ int) ([]string, error) {
	if s.StorePermsByUser != nil {
		return s.StorePermsByUser[userID], nil
	}
	return nil, nil
}

func (s *AccessStub) AssignRole(context.Context, entity.UserRole) error { return nil }

func (s *AccessStub) RevokeRole(context.Context, string, string, *int) error { return nil }

func (s *AccessStub) HasRoleAssignment(context.Context, string, string, *int) (bool, error) {
	return false, nil
}

func (s *AccessStub) CountPlatformAdmins(context.Context) (int, error) { return 0, nil }
