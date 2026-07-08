// user_repo.go 实现 UserRepository 接口，基于 GORM 访问 PostgreSQL。
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	domainrepo "github.com/stvenfor/my_go_study/internal/domain/repository"
	"gorm.io/gorm"
)

// userRepository PostgreSQL 用户仓储实现。
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓储实例。
func NewUserRepository(db *gorm.DB) domainrepo.UserRepository {
	return &userRepository{db: db}
}

// Create 写入新用户。
func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("创建用户失败: %w", err)
	}
	return nil
}

// FindByID 按主键查询用户。
func (r *userRepository) FindByID(ctx context.Context, id uint) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("按 ID 查询用户失败: %w", err)
	}
	return &user, nil
}

// FindByUsername 按用户名查询用户。
func (r *userRepository) FindByUsername(ctx context.Context, username string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("按用户名查询用户失败: %w", err)
	}
	return &user, nil
}

// FindByEmail 按邮箱查询用户。
func (r *userRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("按邮箱查询用户失败: %w", err)
	}
	return &user, nil
}
