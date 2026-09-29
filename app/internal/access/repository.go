// repository.go 门店组织与权限分配 —— 域内仓储契约与领域错误。
//
// 本文件由 model/repository/access_repository.go 演化而来：接口随域下沉，
// 错误哨兵一并搬入（原先 usecase 包里的 ErrAccessXxx 别名层同时去掉）。
package access

import (
	"context"
	"errors"

	"model/entity"
)

// 领域错误。handler 按这些哨兵映射 HTTP 状态；跨域调用方（home / mall /
// 汽车业务各域）据此判断权限、成员关系与门店是否存在。
var (
	ErrForbidden         = errors.New("没有权限")
	ErrStoreExists       = errors.New("门店已存在")
	ErrStoreNotFound     = errors.New("门店不存在")
	ErrUserNotFound      = errors.New("用户不存在")
	ErrMemberNotFound    = errors.New("不是该店成员")
	ErrRoleNotFound      = errors.New("角色不存在")
	ErrRoleScope         = errors.New("角色范围与门店不匹配")
	ErrNotMember         = errors.New("授店内角色前必须先是成员")
	ErrLastPlatformAdmin = errors.New("不能撤销最后一个平台管理员")
	ErrInvalidPosition   = errors.New("职务无效")
	ErrInvalidStoreID    = errors.New("store_id 必须为正整数")
	ErrInvalidRole       = errors.New("角色无效")
	ErrInvalidName       = errors.New("门店名称不能为空")
)

// Repository 门店、成员与角色分配。
type Repository interface {
	CreateStore(ctx context.Context, store entity.WysStore) error
	GetStore(ctx context.Context, storeID int) (*entity.WysStore, error)

	UpsertMember(ctx context.Context, member entity.WysStoreMember) error
	RemoveMember(ctx context.Context, userID string, storeID int) error
	GetMember(ctx context.Context, userID string, storeID int) (*entity.WysStoreMember, error)

	GetRole(ctx context.Context, code string) (*entity.Role, error)
	UserExists(ctx context.Context, userID string) (bool, error)
	CurrentStoreID(ctx context.Context, userID string) (*int, error)
	ClearCurrentStoreIf(ctx context.Context, userID string, storeID int) error

	// PlatformPermissionCodes 该用户全部平台角色带来的能力码。
	PlatformPermissionCodes(ctx context.Context, userID string) ([]string, error)
	// StorePermissionCodes 该用户在指定店的店内角色带来的能力码。
	StorePermissionCodes(ctx context.Context, userID string, storeID int) ([]string, error)

	AssignRole(ctx context.Context, assignment entity.UserRole) error
	RevokeRole(ctx context.Context, userID, roleCode string, storeID *int) error
	HasRoleAssignment(ctx context.Context, userID, roleCode string, storeID *int) (bool, error)
	CountPlatformAdmins(ctx context.Context) (int, error)
}
