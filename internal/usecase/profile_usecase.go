// profile_usecase.go Profile 业务用例。
package usecase

import (
	"context"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

// ProfileUsecase 处理用户资料读写。
type ProfileUsecase struct {
	repo repository.ProfileRepository
}

// NewProfileUsecase 创建 Profile 用例。
func NewProfileUsecase(repo repository.ProfileRepository) *ProfileUsecase {
	return &ProfileUsecase{repo: repo}
}

// GetProfile 获取当前用户资料。
func (u *ProfileUsecase) GetProfile(ctx context.Context, accessToken, userID string) (*entity.Profile, error) {
	return u.repo.GetByUserID(ctx, accessToken, userID)
}

// UpdateProfile 更新当前用户资料。
func (u *ProfileUsecase) UpdateProfile(ctx context.Context, accessToken, userID string, input entity.UpdateProfileInput) (*entity.Profile, error) {
	return u.repo.UpdateByUserID(ctx, accessToken, userID, input)
}
