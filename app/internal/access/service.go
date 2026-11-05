// service.go 门店、成员与角色分配 —— 域内业务逻辑。
//
// 由 app/internal/usecase/access_usecase.go 迁入。三处改动：
//  1. AccessUsecase → Service（跨域调用方以 access.Service 引用）；
//  2. 错误哨兵去 Access 前缀（ErrAccessForbidden → ErrForbidden），域内即命名空间；
//  3. 新增公开读取 API（CurrentStoreID / IsMember / StoreLabels / RequireCurrentStore），
//     替代原先 home / after_sales / deal_invoice / new_car_follow / used_car 直接
//     访问 u.access.repo 私有字段的 9 处层级泄漏。
package access

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"app/internal/shared"
	"core/authz"
	"model/entity"
)

// Service 管理门店、成员与内置角色分配。
type Service struct {
	repo           Repository
	storeGroupSync shared.ImStoreGroupSyncer
	enforcer       *authz.Enforcer // 可选：Casbin 授权器，nil 时行为与原 SQL 逻辑一致
}

// NewService 创建。
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// SetEnforcer 注入 Casbin Enforcer（可选，nil 表示不启用 Casbin，权限判定保持原有 SQL 逻辑不变）。
func (s *Service) SetEnforcer(enforcer *authz.Enforcer) {
	if s != nil {
		s.enforcer = enforcer
	}
}

// SetStoreGroupSync 接线门店群：入店拉人 / 离店踢人。
func (s *Service) SetStoreGroupSync(syncer shared.ImStoreGroupSyncer) {
	if s != nil {
		s.storeGroupSync = syncer
	}
}

// CreateStore 建店。不自动让操作者成为成员。
func (s *Service) CreateStore(ctx context.Context, actorID string, storeID int, name string) (*entity.WysStore, error) {
	if storeID <= 0 {
		return nil, ErrInvalidStoreID
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidName
	}
	if err := s.require(ctx, actorID, entity.PermStoreCreate, nil); err != nil {
		return nil, err
	}
	store := entity.WysStore{StoreID: storeID, Name: name}
	if err := s.repo.CreateStore(ctx, store); err != nil {
		return nil, err
	}
	return &store, nil
}

// UpsertMember 添加或改职务。目标必须是已有用户。
func (s *Service) UpsertMember(ctx context.Context, actorID string, storeID int, targetUserID string, position int16) (*entity.WysStoreMember, error) {
	targetUserID = strings.TrimSpace(targetUserID)
	if targetUserID == "" {
		return nil, ErrUserNotFound
	}
	if storeID <= 0 {
		return nil, ErrInvalidStoreID
	}
	if position < 0 || position > 2 {
		return nil, ErrInvalidPosition
	}
	if err := s.require(ctx, actorID, entity.PermMemberWrite, &storeID); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetStore(ctx, storeID); err != nil {
		return nil, err
	}
	ok, err := s.repo.UserExists(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrUserNotFound
	}
	member := entity.WysStoreMember{
		UserID:   targetUserID,
		StoreID:  storeID,
		Position: position,
	}
	if err := s.repo.UpsertMember(ctx, member); err != nil {
		return nil, err
	}
	if s.storeGroupSync != nil {
		_, _ = s.storeGroupSync.EnsureStoreMembership(ctx, strconv.Itoa(storeID), targetUserID)
	}
	return &member, nil
}

// RemoveMember 离开门店：成员与该店店内角色一并撤销。
func (s *Service) RemoveMember(ctx context.Context, actorID string, storeID int, targetUserID string) error {
	targetUserID = strings.TrimSpace(targetUserID)
	if targetUserID == "" {
		return ErrUserNotFound
	}
	if storeID <= 0 {
		return ErrInvalidStoreID
	}
	if err := s.require(ctx, actorID, entity.PermMemberWrite, &storeID); err != nil {
		return err
	}
	if err := s.repo.RemoveMember(ctx, targetUserID, storeID); err != nil {
		return err
	}
	if s.storeGroupSync != nil {
		_ = s.storeGroupSync.RemoveStoreMembership(ctx, strconv.Itoa(storeID), targetUserID)
	}
	return nil
}

// AssignRole 分配内置角色。店内角色要求对方已是成员。
func (s *Service) AssignRole(ctx context.Context, actorID string, targetUserID, roleCode string, storeID *int) error {
	targetUserID = strings.TrimSpace(targetUserID)
	roleCode = strings.TrimSpace(roleCode)
	if targetUserID == "" {
		return ErrUserNotFound
	}
	role, err := s.repo.GetRole(ctx, roleCode)
	if err != nil {
		return err
	}
	ok, err := s.repo.UserExists(ctx, targetUserID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrUserNotFound
	}

	switch role.Scope {
	case entity.ScopePlatform:
		if storeID != nil {
			return ErrRoleScope
		}
		if err := s.require(ctx, actorID, entity.PermRoleAssignPlatform, nil); err != nil {
			return err
		}
	case entity.ScopeStore:
		if storeID == nil || *storeID <= 0 {
			return ErrInvalidStoreID
		}
		if err := s.require(ctx, actorID, entity.PermRoleAssignStore, storeID); err != nil {
			return err
		}
		if _, err := s.repo.GetMember(ctx, targetUserID, *storeID); err != nil {
			if errors.Is(err, ErrMemberNotFound) {
				return ErrNotMember
			}
			return err
		}
	default:
		return ErrInvalidRole
	}

	grantedBy := actorID
	assignment := entity.UserRole{
		UserID:    targetUserID,
		RoleCode:  roleCode,
		StoreID:   storeID,
		GrantedBy: &grantedBy,
	}
	return s.repo.AssignRole(ctx, assignment)
}

// RevokeRole 撤销角色。不能撤掉最后一个平台管理员。
func (s *Service) RevokeRole(ctx context.Context, actorID string, targetUserID, roleCode string, storeID *int) error {
	targetUserID = strings.TrimSpace(targetUserID)
	roleCode = strings.TrimSpace(roleCode)
	if targetUserID == "" {
		return ErrUserNotFound
	}
	role, err := s.repo.GetRole(ctx, roleCode)
	if err != nil {
		return err
	}

	switch role.Scope {
	case entity.ScopePlatform:
		if storeID != nil {
			return ErrRoleScope
		}
		if err := s.require(ctx, actorID, entity.PermRoleAssignPlatform, nil); err != nil {
			return err
		}
		if roleCode == entity.RolePlatformAdmin {
			has, err := s.repo.HasRoleAssignment(ctx, targetUserID, roleCode, nil)
			if err != nil {
				return err
			}
			if has {
				n, err := s.repo.CountPlatformAdmins(ctx)
				if err != nil {
					return err
				}
				if n <= 1 {
					return ErrLastPlatformAdmin
				}
			}
		}
	case entity.ScopeStore:
		if storeID == nil || *storeID <= 0 {
			return ErrInvalidStoreID
		}
		if err := s.require(ctx, actorID, entity.PermRoleAssignStore, storeID); err != nil {
			return err
		}
	default:
		return ErrInvalidRole
	}

	return s.repo.RevokeRole(ctx, targetUserID, roleCode, storeID)
}

// ListEffectivePermissions 调试/客户端用：平台权限 ∪（当前店若是成员则含其店内权限）。
func (s *Service) ListEffectivePermissions(ctx context.Context, userID string, storeID int) ([]string, error) {
	platform, err := s.repo.PlatformPermissionCodes(ctx, userID)
	if err != nil {
		return nil, err
	}
	set := map[string]struct{}{}
	for _, c := range platform {
		set[c] = struct{}{}
	}
	sid := storeID
	if sid <= 0 {
		cur, err := s.repo.CurrentStoreID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if cur != nil {
			sid = *cur
		}
	}
	if sid > 0 {
		if _, err := s.repo.GetMember(ctx, userID, sid); err == nil {
			storeCodes, err := s.repo.StorePermissionCodes(ctx, userID, sid)
			if err != nil {
				return nil, err
			}
			for _, c := range storeCodes {
				set[c] = struct{}{}
			}
		} else if !errors.Is(err, ErrMemberNotFound) {
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
func (s *Service) RequirePermission(ctx context.Context, actorID, permission string, targetStoreID *int) error {
	return s.require(ctx, actorID, permission, targetStoreID)
}

func (s *Service) require(ctx context.Context, actorID, permission string, targetStoreID *int) error {
	ok, err := s.can(ctx, actorID, permission, targetStoreID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

func (s *Service) can(ctx context.Context, actorID, permission string, targetStoreID *int) (bool, error) {
	// 优先走 Casbin（若已注入）。Casbin 策略存储于 casbin_rule 表，
	// 匹配形如 sub=actorID, obj=permission, act=access 的规则。
	if s.enforcer != nil {
		ok, err := s.enforcer.Enforce(actorID, permission, "access")
		if err != nil {
			return false, fmt.Errorf("casbin enforce 失败: %w", err)
		}
		if ok {
			return true, nil
		}
		// 若指定了目标门店，尝试带门店维度的策略（obj 形如 "member.write:store:5"）
		if targetStoreID != nil && *targetStoreID > 0 {
			storeObj := permission + ":store:" + strconv.Itoa(*targetStoreID)
			ok, err = s.enforcer.Enforce(actorID, storeObj, "access")
			if err != nil {
				return false, fmt.Errorf("casbin enforce 失败: %w", err)
			}
			if ok {
				return true, nil
			}
		}
	}

	// 回退到原有 SQL 权限码检查（平台角色 ∪ 店内角色）。
	platform, err := s.repo.PlatformPermissionCodes(ctx, actorID)
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
	current, err := s.repo.CurrentStoreID(ctx, actorID)
	if err != nil {
		return false, err
	}
	if current == nil || *current != *targetStoreID {
		return false, nil
	}
	if _, err := s.repo.GetMember(ctx, actorID, *targetStoreID); err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			return false, nil
		}
		return false, err
	}
	storeCodes, err := s.repo.StorePermissionCodes(ctx, actorID, *targetStoreID)
	if err != nil {
		return false, err
	}
	return contains(storeCodes, permission), nil
}

// ---------------------------------------------------------------------------
// 跨域读取 API
//
// 原先 home / after_sales / deal_invoice / new_car_follow / used_car 这些域
// 直接访问 u.access.repo 的私有字段（共 9 处），切片后必须走公开方法。
// 注意这些方法都做了 nil 接收者保护：access 未接线时（配置关闭）返回零值。
// ---------------------------------------------------------------------------

// CurrentStoreID 该用户当前选中的门店；未选中返回 nil。
func (s *Service) CurrentStoreID(ctx context.Context, userID string) (*int, error) {
	if s == nil {
		return nil, nil
	}
	return s.repo.CurrentStoreID(ctx, userID)
}

// RequireCurrentStore 读当前门店；未选中时返回调用方给定的 noStore 错误。
func (s *Service) RequireCurrentStore(ctx context.Context, actorID string, noStore error) (int, error) {
	if s == nil {
		return 0, noStore
	}
	cur, err := s.repo.CurrentStoreID(ctx, actorID)
	if err != nil {
		return 0, err
	}
	if cur == nil || *cur <= 0 {
		return 0, noStore
	}
	return *cur, nil
}

// IsMember 该用户是否为目标门店成员。非成员返回 (false, nil)。
func (s *Service) IsMember(ctx context.Context, userID string, storeID int) (bool, error) {
	if s == nil {
		return false, nil
	}
	if _, err := s.repo.GetMember(ctx, userID, storeID); err != nil {
		if errors.Is(err, ErrMemberNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Can 权限判定但不返回 ErrForbidden（调用方要自己决定语义，如"降级为只读"）。
// 原先 new_car_follow 直接调私有方法 u.access.can，切片后需走公开入口。
func (s *Service) Can(ctx context.Context, actorID, permission string, targetStoreID *int) (bool, error) {
	if s == nil {
		return false, nil
	}
	return s.can(ctx, actorID, permission, targetStoreID)
}

// StoreLabels 摘要用门店名与岗位文案；access 未接线或读不到时返回空串。
func (s *Service) StoreLabels(ctx context.Context, actorID string, storeID int) (storeName, positionLabel string) {
	if s == nil {
		return "", ""
	}
	if store, err := s.repo.GetStore(ctx, storeID); err == nil && store != nil {
		storeName = store.Name
	}
	if member, err := s.repo.GetMember(ctx, actorID, storeID); err == nil && member != nil {
		positionLabel = entity.StoreRoleLabel(member.Position)
	}
	return storeName, positionLabel
}

func contains(codes []string, want string) bool {
	for _, c := range codes {
		if c == want {
			return true
		}
	}
	return false
}
