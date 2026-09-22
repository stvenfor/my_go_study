// store_stats_repo.go 读取 wys_user_store_stats 单表数字字段。
package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	domainrepo "github.com/stvenfor/my_go_study/internal/domain/repository"
	"gorm.io/gorm"
)

type storeStatsRepository struct {
	db *gorm.DB
}

// NewStoreStatsRepository 创建门店统计仓储。
func NewStoreStatsRepository(db *gorm.DB) domainrepo.StoreStatsRepository {
	return &storeStatsRepository{db: db}
}

func (r *storeStatsRepository) Load(ctx context.Context, userID string, storeID int) (entity.UserStoreStats, error) {
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return entity.ZeroUserStoreStats(), nil
	}
	sid, err := r.resolveStoreID(ctx, uid, storeID)
	if err != nil {
		return entity.ZeroUserStoreStats(), err
	}
	if sid <= 0 {
		return entity.ZeroUserStoreStats(), nil
	}
	stats, err := r.loadNumbers(ctx, uid, sid)
	if err != nil {
		return entity.ZeroUserStoreStats(), err
	}
	position, err := r.memberPosition(ctx, uid, sid)
	if err != nil {
		return entity.ZeroUserStoreStats(), err
	}
	return stats.WithPosition(position), nil
}

func (r *storeStatsRepository) Switch(ctx context.Context, userID string, storeID int) (entity.UserStoreStats, error) {
	uid := strings.TrimSpace(userID)
	if uid == "" || storeID <= 0 {
		return entity.ZeroUserStoreStats(), domainrepo.ErrStoreNotFound
	}
	position, err := r.memberPosition(ctx, uid, storeID)
	if err != nil {
		return entity.ZeroUserStoreStats(), err
	}
	if position == nil {
		return entity.ZeroUserStoreStats(), domainrepo.ErrStoreNotFound
	}
	if err := r.db.WithContext(ctx).
		Table(entity.UsersTable).
		Where("user_id = ?", uid).
		Update("current_store_id", storeID).Error; err != nil {
		return entity.ZeroUserStoreStats(), fmt.Errorf("保存当前店铺失败: %w", err)
	}
	stats, err := r.loadNumbers(ctx, uid, storeID)
	if err != nil {
		return entity.ZeroUserStoreStats(), err
	}
	return stats.WithPosition(position), nil
}

func (r *storeStatsRepository) ListMyStores(ctx context.Context, userID string) ([]entity.UserStoreListItem, error) {
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return nil, nil
	}
	type row struct {
		StoreID   int
		StoreName string
		Position  int16
		IsCurrent bool
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT m.store_id, s.name AS store_name, m.position,
		       (u.current_store_id IS NOT NULL AND u.current_store_id = m.store_id) AS is_current
		FROM wys_store_member m
		JOIN wys_store s ON s.store_id = m.store_id
		JOIN users u ON u.user_id = m.user_id
		WHERE m.user_id = ?
		ORDER BY CASE WHEN u.current_store_id = m.store_id THEN 0 ELSE 1 END, m.store_id
	`, uid).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("查询经销商列表失败: %w", err)
	}
	out := make([]entity.UserStoreListItem, 0, len(rows))
	for _, row := range rows {
		p := row.Position
		out = append(out, entity.UserStoreListItem{
			StoreID:   row.StoreID,
			StoreName: row.StoreName,
			Role:      &p,
			RoleLabel: entity.StoreRoleLabel(p),
			IsCurrent: row.IsCurrent,
		})
	}
	return out, nil
}

func (r *storeStatsRepository) resolveStoreID(ctx context.Context, userID string, storeID int) (int, error) {
	if storeID > 0 {
		return storeID, nil
	}
	var first int
	err := r.db.WithContext(ctx).Raw(`
		SELECT m.store_id
		FROM wys_store_member m
		JOIN users u ON u.user_id = m.user_id
		WHERE m.user_id = ?
		ORDER BY CASE WHEN u.current_store_id = m.store_id THEN 0 ELSE 1 END, m.store_id
		LIMIT 1
	`, userID).Scan(&first).Error
	if err != nil {
		return 0, fmt.Errorf("查询当前店铺失败: %w", err)
	}
	return first, nil
}

func (r *storeStatsRepository) loadNumbers(ctx context.Context, userID string, storeID int) (entity.UserStoreStats, error) {
	row, err := r.findStats(ctx, userID, storeID)
	if err == nil {
		return row.ToAPIStats(), nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return entity.ZeroUserStoreStats(), err
	}
	var names []string
	err = r.db.WithContext(ctx).Raw(
		`SELECT name FROM wys_store WHERE store_id = ?`, storeID,
	).Scan(&names).Error
	if err != nil {
		return entity.ZeroUserStoreStats(), fmt.Errorf("查询门店失败: %w", err)
	}
	if len(names) == 0 {
		return entity.ZeroUserStoreStats(), nil
	}
	return entity.UserStoreStats{StoreID: storeID, StoreName: names[0]}, nil
}

func (r *storeStatsRepository) memberPosition(ctx context.Context, userID string, storeID int) (*int16, error) {
	var positions []int16
	err := r.db.WithContext(ctx).Raw(
		`SELECT position FROM wys_store_member WHERE user_id = ? AND store_id = ?`,
		userID, storeID,
	).Scan(&positions).Error
	if err != nil {
		return nil, fmt.Errorf("查询门店职务失败: %w", err)
	}
	if len(positions) == 0 {
		return nil, nil
	}
	p := positions[0]
	return &p, nil
}

func (r *storeStatsRepository) findStats(ctx context.Context, userID string, storeID int) (*entity.WysUserStoreStats, error) {
	var row entity.WysUserStoreStats
	err := r.db.WithContext(ctx).
		Where("store_id = ? AND user_id = ?", storeID, userID).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, fmt.Errorf("查询门店统计失败: %w", err)
	}
	return &row, nil
}
