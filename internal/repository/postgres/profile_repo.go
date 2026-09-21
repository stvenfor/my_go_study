// profile_repo.go 本地资料读写统一 users 表。
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

// NewProfileRepository 创建基于 users 表的资料仓储。
func NewProfileRepository(db *gorm.DB) domainrepo.ProfileRepository {
	return &profileRepository{db: db}
}

func (r *profileRepository) GetByUserID(ctx context.Context, _, userID string) (*entity.Profile, error) {
	user, err := r.findUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return profileFromUser(user), nil
}

func (r *profileRepository) UpdateByUserID(ctx context.Context, _, userID string, input entity.UpdateProfileInput) (*entity.Profile, error) {
	user, err := r.findUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	updates := userProfileUpdates(input, time.Now().UTC())
	if len(updates) == 0 {
		return profileFromUser(user), nil
	}
	if err := r.db.WithContext(ctx).Model(&entity.User{}).Where("user_id = ?", user.UserID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新 profile 失败: %w", err)
	}
	updated, err := r.findUser(ctx, user.UserID)
	if err != nil {
		return nil, err
	}
	return profileFromUser(updated), nil
}

func (r *profileRepository) findUser(ctx context.Context, userID string) (*entity.User, error) {
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return nil, fmt.Errorf("无效的用户 ID")
	}
	var user entity.User
	err := r.db.WithContext(ctx).Where("user_id = ? AND deleted_at IS NULL", uid).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("资料不存在")
		}
		return nil, fmt.Errorf("查询 profile 失败: %w", err)
	}
	return &user, nil
}

func profileFromUser(user *entity.User) *entity.Profile {
	name := user.UserName
	profile := &entity.Profile{
		ID:        user.UserID,
		UserID:    user.UserID,
		UserName:  name,
		Email:     user.Email,
		Status:    user.Status,
		CreatedAt: timePtr(user.CreatedAt),
		UpdatedAt: timePtr(user.UpdatedAt),
	}
	if name != "" {
		profile.DisplayName = &name
	}
	if user.AvatarURL != "" {
		avatar := user.AvatarURL
		profile.AvatarURL = &avatar
	}
	if user.Phone != "" {
		phone := user.Phone
		profile.Phone = &phone
	}
	if user.CurrentStoreID != nil {
		storeID := *user.CurrentStoreID
		profile.CurrentStoreID = &storeID
	}
	return profile
}

// userProfileUpdates 只写资料列。status 等安全列不进入更新。
func userProfileUpdates(input entity.UpdateProfileInput, now time.Time) map[string]any {
	updates := map[string]any{}
	if name := input.ResolvedUserName(); name != nil {
		updates["user_name"] = strings.TrimSpace(*name)
	}
	if input.AvatarURL != nil {
		updates["avatar_url"] = *input.AvatarURL
	}
	if input.Phone != nil {
		updates["phone"] = strings.TrimSpace(*input.Phone)
	}
	if len(updates) == 0 {
		return updates
	}
	updates["updated_at"] = now
	return updates
}

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	v := t
	return &v
}
