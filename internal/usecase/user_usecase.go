// user_usecase.go 实现用户注册、登录、查询等业务用例。
package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	domainrepo "github.com/stvenfor/my_go_study/internal/domain/repository"
	jwtmanager "github.com/stvenfor/my_go_study/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrInvalidParams 参数不合法。
	ErrInvalidParams = errors.New("invalid params")
	// ErrUserExists 用户已存在。
	ErrUserExists = errors.New("user already exists")
	// ErrInvalidCredentials 用户名或密码错误。
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrUserNotFound 用户不存在。
	ErrUserNotFound = errors.New("user not found")
)

// RegisterInput 注册入参。
type RegisterInput struct {
	Username string
	Password string
	Email    string
}

// LoginInput 登录入参。
type LoginInput struct {
	Username string
	Password string
}

// AuthOutput 登录成功返回 token 与用户信息。
type AuthOutput struct {
	Token string
	User  *entity.User
}

// UserUsecase 用户业务用例。
type UserUsecase struct {
	repo       domainrepo.UserRepository
	jwtManager *jwtmanager.Manager
}

// NewUserUsecase 创建用户用例实例。
func NewUserUsecase(repo domainrepo.UserRepository, jwtManager *jwtmanager.Manager) *UserUsecase {
	return &UserUsecase{repo: repo, jwtManager: jwtManager}
}

// Register 注册新用户。
func (u *UserUsecase) Register(ctx context.Context, input RegisterInput) (*entity.User, error) {
	if input.Username == "" || input.Password == "" || input.Email == "" {
		return nil, ErrInvalidParams
	}
	if len(input.Password) < 6 {
		return nil, ErrInvalidParams
	}

	exists, err := u.repo.FindByUsername(ctx, input.Username)
	if err != nil {
		return nil, fmt.Errorf("检查用户名失败: %w", err)
	}
	if exists != nil {
		return nil, ErrUserExists
	}

	existsEmail, err := u.repo.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("检查邮箱失败: %w", err)
	}
	if existsEmail != nil {
		return nil, ErrUserExists
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	user := &entity.User{
		Username: input.Username,
		Email:    input.Email,
		Password: string(hashed),
	}
	if err := u.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// Login 用户登录并签发 JWT。
func (u *UserUsecase) Login(ctx context.Context, input LoginInput) (*AuthOutput, error) {
	if input.Username == "" || input.Password == "" {
		return nil, ErrInvalidParams
	}

	user, err := u.repo.FindByUsername(ctx, input.Username)
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := u.jwtManager.Generate(user.ID, user.Username)
	if err != nil {
		return nil, err
	}

	return &AuthOutput{Token: token, User: user}, nil
}

// GetProfile 获取用户个人信息。
func (u *UserUsecase) GetProfile(ctx context.Context, userID uint) (*entity.User, error) {
	user, err := u.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}
