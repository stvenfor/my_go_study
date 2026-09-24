// access_usecase.go 门店组织与角色分配。现有业务接口不走这里。
package usecase

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

var (
	ErrAccessForbidden         = repository.ErrAccessForbidden
	ErrAccessStoreExists       = repository.ErrAccessStoreExists
	ErrAccessStoreNotFound     = repository.ErrAccessStoreNotFound
	ErrAccessUserNotFound      = repository.ErrAccessUserNotFound
	ErrAccessMemberNotFound    = repository.ErrAccessMemberNotFound
	ErrAccessRoleNotFound      = repository.ErrAccessRoleNotFound
	ErrAccessRoleScope         = repository.ErrAccessRoleScope
	ErrAccessNotMember         = repository.ErrAccessNotMember
	ErrAccessLastPlatformAdmin = repository.ErrAccessLastPlatformAdmin
	ErrAccessInvalidPosition   = repository.ErrAccessInvalidPosition
	ErrAccessInvalidStoreID    = repository.ErrAccessInvalidStoreID
	ErrAccessInvalidRole       = repository.ErrAccessInvalidRole
	ErrAccessInvalidName       = errors.New("门店名称不能为空")
)

// AccessUsecase 管理门店、成员与内置角色分配。
type AccessUsecase struct {
	repo           repository.AccessRepository
	storeGroupSync ImStoreGroupSyncer
}

// NewAccessUsecase 创建。
func NewAccessUsecase(repo repository.AccessRepository) *AccessUsecase {
	return &AccessUsecase{repo: repo}
}

// SetStoreGroupSync 接线门店群：入店拉人 / 离店踢人。
func (u *AccessUsecase) SetStoreGroupSync(s ImStoreGroupSyncer) {
	if u != nil {
		u.storeGroupSync = s
	}
}

// CreateStore 建店。不自动让操作者成为成员。
func (u *AccessUsecase) CreateStore(ctx context.Context, actorID string, storeID int, name string) (*entity.WysStore, error) {
	if storeID <= 0 {
		return nil, ErrAccessInvalidStoreID
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrAccessInvalidName
	}
	if err := u.require(ctx, actorID, entity.PermStoreCreate, nil); err != nil {
		return nil, err
	}
	store := entity.WysStore{StoreID: storeID, Name: name}
	if err := u.repo.CreateStore(ctx, store); err != nil {
		return nil, err
	}
	return &store, nil
}

// UpsertMember 添加或改职务。目标必须是已有用户。
func (u *AccessUsecase) UpsertMember(ctx context.Context, actorID string, storeID int, targetUserID string, position int16) (*entity.WysStoreMember, error) {
	targetUserID = strings.TrimSpace(targetUserID)
	if targetUserID == "" {
		return nil, ErrAccessUserNotFound
	}
	if storeID <= 0 {
		return nil, ErrAccessInvalidStoreID
	}
	if position < 0 || position > 2 {
		return nil, ErrAccessInvalidPosition
	}
	if err := u.require(ctx, actorID, entity.PermMemberWrite, &storeID); err != nil {
		return nil, err
	}
	if _, err := u.repo.GetStore(ctx, storeID); err != nil {
		return nil, err
	}
	ok, err := u.repo.UserExists(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrAccessUserNotFound
	}
	member := entity.WysStoreMember{
		UserID:   targetUserID,
		StoreID:  storeID,
		Position: position,
	}
	if err := u.repo.UpsertMember(ctx, member); err != nil {
		return nil, err
	}
	if u.storeGroupSync != nil {
		_, _ = u.storeGroupSync.EnsureStoreMembership(ctx, strconv.Itoa(storeID), targetUserID)
	}
	return &member, nil
}

// RemoveMember 离开门店：成员与该店店内角色一并撤销。
func (u *AccessUsecase) RemoveMember(ctx context.Context, actorID string, storeID int, targetUserID string) error {
	targetUserID = strings.TrimSpace(targetUserID)
	if targetUserID == "" {
		return ErrAccessUserNotFound
	}
	if storeID <= 0 {
		return ErrAccessInvalidStoreID
	}
	if err := u.require(ctx, actorID, entity.PermMemberWrite, &storeID); err != nil {
		return err
	}
	if err := u.repo.RemoveMember(ctx, targetUserID, storeID); err != nil {
		return err
	}
	if u.storeGroupSync != nil {
		_ = u.storeGroupSync.RemoveStoreMembership(ctx, strconv.Itoa(storeID), targetUserID)
	}
	return nil
}

// AssignRole 分配内置角色。店内角色要求对方已是成员。
func (u *AccessUsecase) AssignRole(ctx context.Context, actorID string, targetUserID, roleCode string, storeID *int) error {
	targetUserID = strings.TrimSpace(targetUserID)
	roleCode = strings.TrimSpace(roleCode)
	if targetUserID == "" {
		return ErrAccessUserNotFound
	}
	role, err := u.repo.GetRole(ctx, roleCode)
	if err != nil {
		return err
	}
	ok, err := u.repo.UserExists(ctx, targetUserID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrAccessUserNotFound
	}

	switch role.Scope {
	case entity.ScopePlatform:
		if storeID != nil {
			return ErrAccessRoleScope
		}
		if err := u.require(ctx, actorID, entity.PermRoleAssignPlatform, nil); err != nil {
			return err
		}
	case entity.ScopeStore:
		if storeID == nil || *storeID <= 0 {
			return ErrAccessInvalidStoreID
		}
		if err := u.require(ctx, actorID, entity.PermRoleAssignStore, storeID); err != nil {
			return err
		}
		if _, err := u.repo.GetMember(ctx, targetUserID, *storeID); err != nil {
			if errors.Is(err, repository.ErrAccessMemberNotFound) {
				return ErrAccessNotMember
			}
			return err
		}
	default:
		return ErrAccessInvalidRole
	}

	grantedBy := actorID
	assignment := entity.UserRole{
		UserID:    targetUserID,
		RoleCode:  roleCode,
		StoreID:   storeID,
		GrantedBy: &grantedBy,
	}
	return u.repo.AssignRole(ctx, assignment)
}

// RevokeRole 撤销角色。不能撤掉最后一个平台管理员。
func (u *AccessUsecase) RevokeRole(ctx context.Context, actorID string, targetUserID, roleCode string, storeID *int) error {
	targetUserID = strings.TrimSpace(targetUserID)
	roleCode = strings.TrimSpace(roleCode)
	if targetUserID == "" {
		return ErrAccessUserNotFound
	}
	role, err := u.repo.GetRole(ctx, roleCode)
	if err != nil {
		return err
	}

	switch role.Scope {
	case entity.ScopePlatform:
		if storeID != nil {
			return ErrAccessRoleScope
		}
		if err := u.require(ctx, actorID, entity.PermRoleAssignPlatform, nil); err != nil {
			return err
		}
		if roleCode == entity.RolePlatformAdmin {
			has, err := u.repo.HasRoleAssignment(ctx, targetUserID, roleCode, nil)
			if err != nil {
				return err
			}
			if has {
				n, err := u.repo.CountPlatformAdmins(ctx)
				if err != nil {
					return err
				}
				if n <= 1 {
					return ErrAccessLastPlatformAdmin
				}
			}
		}
	case entity.ScopeStore:
		if storeID == nil || *storeID <= 0 {
			return ErrAccessInvalidStoreID
		}
		if err := u.require(ctx, actorID, entity.PermRoleAssignStore, storeID); err != nil {
			return err
		}
	default:
		return ErrAccessInvalidRole
	}

	return u.repo.RevokeRole(ctx, targetUserID, roleCode, storeID)
}

// ListEffectivePermissions 调试/客户端用：平台权限 ∪（当前店若是成员则含其店内权限）。
func (u *AccessUsecase) ListEffectivePermissions(ctx context.Context, userID string, storeID int) ([]string, error) {
	platform, err := u.repo.PlatformPermissionCodes(ctx, userID)
	if err != nil {
		return nil, err
	}
	set := map[string]struct{}{}
	for _, c := range platform {
		set[c] = struct{}{}
	}
	sid := storeID
	if sid <= 0 {
		cur, err := u.repo.CurrentStoreID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if cur != nil {
			sid = *cur
		}
	}
	if sid > 0 {
		if _, err := u.repo.GetMember(ctx, userID, sid); err == nil {
			storeCodes, err := u.repo.StorePermissionCodes(ctx, userID, sid)
			if err != nil {
				return nil, err
			}
			for _, c := range storeCodes {
				set[c] = struct{}{}
			}
		} else if !errors.Is(err, repository.ErrAccessMemberNotFound) {
			return nil, err
		}
	}
	out := make([]string, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	return out, nil
}

// RequirePermission 对外暴露权限校验（商城写目录等）。
func (u *AccessUsecase) RequirePermission(ctx context.Context, actorID, permission string, targetStoreID *int) error {
	return u.require(ctx, actorID, permission, targetStoreID)
}

func (u *AccessUsecase) require(ctx context.Context, actorID, permission string, targetStoreID *int) error {
	ok, err := u.can(ctx, actorID, permission, targetStoreID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrAccessForbidden
	}
	return nil
}

func (u *AccessUsecase) can(ctx context.Context, actorID, permission string, targetStoreID *int) (bool, error) {
	platform, err := u.repo.PlatformPermissionCodes(ctx, actorID)
	if err != nil {
		return false, err
	}
	if contains(platform, permission) {
		return true, nil
	}
	if !entity.PermissionNeedsStore(permission) {
		return false, nil
	}
	if targetStoreID == nil || *targetStoreID <= 0 {
		return false, nil
	}
	current, err := u.repo.CurrentStoreID(ctx, actorID)
	if err != nil {
		return false, err
	}
	if current == nil || *current != *targetStoreID {
		return false, nil
	}
	if _, err := u.repo.GetMember(ctx, actorID, *targetStoreID); err != nil {
		if errors.Is(err, repository.ErrAccessMemberNotFound) {
			return false, nil
		}
		return false, err
	}
	storeCodes, err := u.repo.StorePermissionCodes(ctx, actorID, *targetStoreID)
	if err != nil {
		return false, err
	}
	return contains(storeCodes, permission), nil
}

func contains(codes []string, want string) bool {
	for _, c := range codes {
		if c == want {
			return true
		}
	}
	return false
}
