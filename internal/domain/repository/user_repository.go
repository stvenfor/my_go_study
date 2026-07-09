// user_repository.go 定义用户仓储接口，供 usecase 依赖。
package repository

import (
	"context"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

// UserRepository 用户数据访问抽象。
type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	FindByID(ctx context.Context, id uint) (*entity.User, error)
	FindByUsername(ctx context.Context, username string) (*entity.User, error)
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	List(ctx context.Context, offset, limit int) ([]entity.User, int64, error)
}
