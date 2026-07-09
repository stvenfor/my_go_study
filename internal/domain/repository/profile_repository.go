// profile_repository.go 定义 Profile 数据访问接口。
package repository

import (
	"context"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

// ProfileRepository Supabase profiles 仓储接口。
type ProfileRepository interface {
	GetByUserID(ctx context.Context, accessToken, userID string) (*entity.Profile, error)
	UpdateByUserID(ctx context.Context, accessToken, userID string, input entity.UpdateProfileInput) (*entity.Profile, error)
}
