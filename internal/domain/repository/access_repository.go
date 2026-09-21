// access_repository.go 门店组织与权限分配。
package repository

import (
	"context"
	"errors"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

var (
	ErrAccessStoreExists      = errors.New("门店已存在")
	ErrAccessStoreNotFound    = errors.New("门店不存在")
	ErrAccessUserNotFound     = errors.New("用户不存在")
	ErrAccessMemberNotFound   = errors.New("不是该店成员")
	ErrAccessRoleNotFound     = errors.New("角色不存在")
	ErrAccessRoleScope        = errors.New("角色范围与门店不匹配")
	ErrAccessNotMember        = errors.New("授店内角色前必须先是成员")
	ErrAccessForbidden        = errors.New("没有权限")
	ErrAccessLastPlatformAdmin = errors.New("不能撤销最后一个平台管理员")
	ErrAccessInvalidPosition  = errors.New("职务无效")
	ErrAccessInvalidStoreID   = errors.New("store_id 必须为正整数")
	ErrAccessInvalidRole      = errors.New("角色无效")
)

// AccessRepository 门店、成员与角色分配。
type AccessRepository interface {
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
