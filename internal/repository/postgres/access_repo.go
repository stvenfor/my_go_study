// access_repo.go 门店组织与权限分配。
package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	domainrepo "github.com/stvenfor/my_go_study/internal/domain/repository"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type accessRepository struct {
	db *gorm.DB
}

// NewAccessRepository 创建门店与权限仓储。
func NewAccessRepository(db *gorm.DB) domainrepo.AccessRepository {
	return &accessRepository{db: db}
}

func (r *accessRepository) CreateStore(ctx context.Context, store entity.WysStore) error {
	now := time.Now().UTC()
	store.CreatedAt = now
	store.UpdatedAt = now
	err := r.db.WithContext(ctx).Create(&store).Error
	if err != nil {
		if isUniqueViolation(err) {
			return domainrepo.ErrAccessStoreExists
		}
		return fmt.Errorf("创建门店失败: %w", err)
	}
	return nil
}

func (r *accessRepository) GetStore(ctx context.Context, storeID int) (*entity.WysStore, error) {
	var store entity.WysStore
	err := r.db.WithContext(ctx).Where("store_id = ?", storeID).First(&store).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domainrepo.ErrAccessStoreNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询门店失败: %w", err)
	}
	return &store, nil
}

func (r *accessRepository) UpsertMember(ctx context.Context, member entity.WysStoreMember) error {
	now := time.Now().UTC()
	member.CreatedAt = now
	member.UpdatedAt = now
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "store_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"position", "updated_at"}),
	}).Create(&member).Error
	if err != nil {
		return fmt.Errorf("写入成员失败: %w", err)
	}
	return nil
}

func (r *accessRepository) RemoveMember(ctx context.Context, userID string, storeID int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("user_id = ? AND store_id = ?", userID, storeID).
			Delete(&entity.WysStoreMember{})
		if res.Error != nil {
			return fmt.Errorf("删除成员失败: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return domainrepo.ErrAccessMemberNotFound
		}
		if err := tx.Exec(
			`DELETE FROM user_role WHERE user_id = ? AND store_id = ?`,
			userID, storeID,
		).Error; err != nil {
			return fmt.Errorf("撤销店内角色失败: %w", err)
		}
		if err := tx.Table(entity.UsersTable).
			Where("user_id = ? AND current_store_id = ?", userID, storeID).
			Update("current_store_id", nil).Error; err != nil {
			return fmt.Errorf("清空当前店失败: %w", err)
		}
		return nil
	})
}

func (r *accessRepository) GetMember(ctx context.Context, userID string, storeID int) (*entity.WysStoreMember, error) {
	var member entity.WysStoreMember
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND store_id = ?", userID, storeID).
		First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domainrepo.ErrAccessMemberNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询成员失败: %w", err)
	}
	return &member, nil
}

func (r *accessRepository) GetRole(ctx context.Context, code string) (*entity.Role, error) {
	var role entity.Role
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domainrepo.ErrAccessRoleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询角色失败: %w", err)
	}
	return &role, nil
}

func (r *accessRepository) UserExists(ctx context.Context, userID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Table(entity.UsersTable).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("查询用户失败: %w", err)
	}
	return count > 0, nil
}

func (r *accessRepository) CurrentStoreID(ctx context.Context, userID string) (*int, error) {
	var ids []int
	err := r.db.WithContext(ctx).Raw(
		`SELECT current_store_id FROM users WHERE user_id = ? AND current_store_id IS NOT NULL`,
		userID,
	).Scan(&ids).Error
	if err != nil {
		return nil, fmt.Errorf("查询当前店失败: %w", err)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	id := ids[0]
	return &id, nil
}

func (r *accessRepository) ClearCurrentStoreIf(ctx context.Context, userID string, storeID int) error {
	return r.db.WithContext(ctx).Table(entity.UsersTable).
		Where("user_id = ? AND current_store_id = ?", userID, storeID).
		Update("current_store_id", nil).Error
}

func (r *accessRepository) PlatformPermissionCodes(ctx context.Context, userID string) ([]string, error) {
	var codes []string
	err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT rp.permission_code
		FROM user_role ur
		JOIN role_permission rp ON rp.role_code = ur.role_code
		JOIN role ro ON ro.code = ur.role_code
		WHERE ur.user_id = ?
		  AND ur.store_id IS NULL
		  AND ro.scope = 'platform'
	`, userID).Scan(&codes).Error
	if err != nil {
		return nil, fmt.Errorf("查询平台权限失败: %w", err)
	}
	return codes, nil
}

func (r *accessRepository) StorePermissionCodes(ctx context.Context, userID string, storeID int) ([]string, error) {
	var codes []string
	err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT rp.permission_code
		FROM user_role ur
		JOIN role_permission rp ON rp.role_code = ur.role_code
		JOIN role ro ON ro.code = ur.role_code
		JOIN wys_store_member m
		  ON m.user_id = ur.user_id AND m.store_id = ur.store_id
		WHERE ur.user_id = ?
		  AND ur.store_id = ?
		  AND ro.scope = 'store'
	`, userID, storeID).Scan(&codes).Error
	if err != nil {
		return nil, fmt.Errorf("查询店内权限失败: %w", err)
	}
	return codes, nil
}

func (r *accessRepository) AssignRole(ctx context.Context, assignment entity.UserRole) error {
	assignment.CreatedAt = time.Now().UTC()
	err := r.db.WithContext(ctx).Exec(`
		INSERT INTO user_role (user_id, role_code, store_id, granted_by, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT DO NOTHING
	`, assignment.UserID, assignment.RoleCode, assignment.StoreID, assignment.GrantedBy, assignment.CreatedAt).Error
	if err != nil {
		return fmt.Errorf("分配角色失败: %w", err)
	}
	return nil
}

func (r *accessRepository) RevokeRole(ctx context.Context, userID, roleCode string, storeID *int) error {
	var res *gorm.DB
	if storeID == nil {
		res = r.db.WithContext(ctx).Exec(
			`DELETE FROM user_role WHERE user_id = ? AND role_code = ? AND store_id IS NULL`,
			userID, roleCode,
		)
	} else {
		res = r.db.WithContext(ctx).Exec(
			`DELETE FROM user_role WHERE user_id = ? AND role_code = ? AND store_id = ?`,
			userID, roleCode, *storeID,
		)
	}
	if res.Error != nil {
		return fmt.Errorf("撤销角色失败: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return domainrepo.ErrAccessRoleNotFound
	}
	return nil
}

func (r *accessRepository) HasRoleAssignment(ctx context.Context, userID, roleCode string, storeID *int) (bool, error) {
	q := r.db.WithContext(ctx).Model(&entity.UserRole{}).
		Where("user_id = ? AND role_code = ?", userID, roleCode)
	if storeID == nil {
		q = q.Where("store_id IS NULL")
	} else {
		q = q.Where("store_id = ?", *storeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return false, fmt.Errorf("查询角色分配失败: %w", err)
	}
	return count > 0, nil
}

func (r *accessRepository) CountPlatformAdmins(ctx context.Context) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.UserRole{}).
		Where("role_code = ? AND store_id IS NULL", entity.RolePlatformAdmin).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("统计平台管理员失败: %w", err)
	}
	return int(count), nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique constraint")
}
