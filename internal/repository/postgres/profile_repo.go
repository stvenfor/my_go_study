// profile_repo.go 基于 PostgreSQL 的 profiles 仓储。
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
)

type profileRepository struct {
	db *gorm.DB
}

// NewProfileRepository 创建 profiles 仓储。
func NewProfileRepository(db *gorm.DB) domainrepo.ProfileRepository {
	return &profileRepository{db: db}
}

func (r *profileRepository) GetByUserID(ctx context.Context, _, userID string) (*entity.Profile, error) {
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return nil, fmt.Errorf("无效的用户 ID")
	}
	var profile entity.Profile
	err := r.db.WithContext(ctx).Where("id = ?", uid).First(&profile).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("资料不存在")
		}
		return nil, fmt.Errorf("查询 profile 失败: %w", err)
	}
	return &profile, nil
}

func (r *profileRepository) UpdateByUserID(ctx context.Context, _, userID string, input entity.UpdateProfileInput) (*entity.Profile, error) {
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return nil, fmt.Errorf("无效的用户 ID")
	}

	var profile entity.Profile
	err := r.db.WithContext(ctx).Where("id = ?", uid).First(&profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		now := time.Now().UTC()
		profile = entity.Profile{
			ID:          uid,
			DisplayName: input.DisplayName,
			AvatarURL:   input.AvatarURL,
			CreatedAt:   &now,
			UpdatedAt:   &now,
		}
		if err := r.db.WithContext(ctx).Create(&profile).Error; err != nil {
			return nil, fmt.Errorf("创建 profile 失败: %w", err)
		}
		return &profile, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询 profile 失败: %w", err)
	}

	updates := map[string]any{}
	if input.DisplayName != nil {
		updates["display_name"] = *input.DisplayName
	}
	if input.AvatarURL != nil {
		updates["avatar_url"] = *input.AvatarURL
	}
	if len(updates) == 0 {
		return &profile, nil
	}
	now := time.Now().UTC()
	updates["updated_at"] = now
	if err := r.db.WithContext(ctx).Model(&profile).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新 profile 失败: %w", err)
	}
	if err := r.db.WithContext(ctx).First(&profile, "id = ?", uid).Error; err != nil {
		return nil, fmt.Errorf("刷新 profile 失败: %w", err)
	}
	return &profile, nil
}
