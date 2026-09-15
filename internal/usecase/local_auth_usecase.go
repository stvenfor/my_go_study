package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/pkg/config"
	jwtmanager "github.com/stvenfor/my_go_study/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const refreshTokenTTL = 30 * 24 * time.Hour

// SessionAuth 登录注册契约（local / supabase 共用，响应形状对齐 Flutter）。
type SessionAuth interface {
	Register(ctx context.Context, input RegisterInput) (*SupabaseAuthOutput, error)
	Login(ctx context.Context, input LoginInput) (*SupabaseAuthOutput, error)
	RefreshToken(ctx context.Context, refreshToken string) (*SupabaseAuthOutput, error)
	Logout(ctx context.Context, accessToken string) error
}

// PhoneOTPAuth 手机号 OTP（local / supabase）。
type PhoneOTPAuth interface {
	SendPhoneOTP(ctx context.Context, phone string) error
	VerifyPhoneOTP(ctx context.Context, phone, otp string) (*SupabaseAuthOutput, error)
}

// LocalAuthUsecase 本机 Postgres Auth（UUID + JWT + refresh 表）。
type LocalAuthUsecase struct {
	db     *gorm.DB
	jwt    *jwtmanager.Manager
	auth   config.AuthConfig
	server string
}

func NewLocalAuthUsecase(db *gorm.DB, jwtMgr *jwtmanager.Manager, authCfg config.AuthConfig, serverMode string) *LocalAuthUsecase {
	return &LocalAuthUsecase{db: db, jwt: jwtMgr, auth: authCfg, server: serverMode}
}

func (u *LocalAuthUsecase) Register(ctx context.Context, input RegisterInput) (*SupabaseAuthOutput, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" || !strings.Contains(email, "@") || input.Password == "" {
		return nil, ErrInvalidParams
	}
	if len(input.Password) < 6 {
		return nil, ErrInvalidParams
	}

	var existing entity.AuthUser
	err := u.db.WithContext(ctx).Where("email = ?", email).First(&existing).Error
	if err == nil {
		return nil, ErrUserExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("密码哈希失败: %w", err)
	}

	display := strings.TrimSpace(input.Username)
	if display == "" {
		display = strings.Split(email, "@")[0]
	}
	user := entity.AuthUser{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: string(hash),
		DisplayName:  display,
	}
	if err := u.db.WithContext(ctx).Create(&user).Error; err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	now := time.Now().UTC()
	profile := entity.Profile{
		ID:          user.ID,
		DisplayName: &display,
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}
	_ = u.db.WithContext(ctx).Create(&profile).Error

	return u.issueTokens(ctx, &user)
}

func (u *LocalAuthUsecase) Login(ctx context.Context, input LoginInput) (*SupabaseAuthOutput, error) {
	email := strings.ToLower(strings.TrimSpace(input.Username))
	if email == "" || input.Password == "" {
		return nil, ErrInvalidParams
	}
	if !strings.Contains(email, "@") {
		return nil, ErrInvalidCredentials
	}

	var user entity.AuthUser
	err := u.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAccountNotRegistered
	}
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		return nil, ErrInvalidCredentials
	}
	return u.issueTokens(ctx, &user)
}

func (u *LocalAuthUsecase) RefreshToken(ctx context.Context, refreshToken string) (*SupabaseAuthOutput, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, ErrInvalidCredentials
	}
	hash := hashToken(refreshToken)
	var row entity.AuthRefreshToken
	err := u.db.WithContext(ctx).Where("token_hash = ?", hash).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("查询 refresh token 失败: %w", err)
	}
	if time.Now().After(row.ExpiresAt) {
		_ = u.db.WithContext(ctx).Delete(&row).Error
		return nil, ErrInvalidCredentials
	}

	var user entity.AuthUser
	if err := u.db.WithContext(ctx).Where("id = ?", row.UserID).First(&user).Error; err != nil {
		return nil, ErrInvalidCredentials
	}

	_ = u.db.WithContext(ctx).Delete(&row).Error
	return u.issueTokens(ctx, &user)
}

func (u *LocalAuthUsecase) Logout(ctx context.Context, accessToken string) error {
	// 本地 access JWT 无服务端状态；尽力吊销该用户近期 refresh（可选：解析 sub）。
	if claims, err := u.jwt.ParseUUID(accessToken); err == nil {
		_ = u.db.WithContext(ctx).Where("user_id = ?", claims.Subject).Delete(&entity.AuthRefreshToken{}).Error
	}
	return nil
}

func (u *LocalAuthUsecase) SendPhoneOTP(ctx context.Context, phone string) error {
	if !u.auth.DevBypassEnabled(u.server) || !u.auth.IsDevTestPhone(phone) {
		return ErrPhoneLoginNotAvailable
	}
	return nil
}

func (u *LocalAuthUsecase) VerifyPhoneOTP(ctx context.Context, phone, otp string) (*SupabaseAuthOutput, error) {
	if !u.auth.DevBypassEnabled(u.server) || !u.auth.IsDevTestPhone(phone) {
		return nil, ErrPhoneLoginNotAvailable
	}
	if strings.TrimSpace(otp) != strings.TrimSpace(u.auth.DevTestOTP) {
		return nil, ErrInvalidOTP
	}

	digits := config.NormalizePhoneDigits(phone)
	email := fmt.Sprintf("%s@dev.test.local", digits)
	var user entity.AuthUser
	err := u.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		hash, herr := bcrypt.GenerateFromPassword([]byte(u.auth.DevTestPasswordOrDefault()), bcrypt.DefaultCost)
		if herr != nil {
			return nil, herr
		}
		user = entity.AuthUser{
			ID:           uuid.NewString(),
			Email:        email,
			Phone:        digits,
			PasswordHash: string(hash),
			DisplayName:  "dev-" + digits,
		}
		if err := u.db.WithContext(ctx).Create(&user).Error; err != nil {
			return nil, fmt.Errorf("创建测试用户失败: %w", err)
		}
		now := time.Now().UTC()
		name := user.DisplayName
		_ = u.db.WithContext(ctx).Create(&entity.Profile{
			ID: user.ID, DisplayName: &name, Phone: &digits, CreatedAt: &now, UpdatedAt: &now,
		}).Error
	} else if err != nil {
		return nil, err
	}
	return u.issueTokens(ctx, &user)
}

func (u *LocalAuthUsecase) issueTokens(ctx context.Context, user *entity.AuthUser) (*SupabaseAuthOutput, error) {
	access, err := u.jwt.GenerateUUID(user.ID, user.Email, user.DisplayName)
	if err != nil {
		return nil, err
	}
	rawRefresh, err := randomToken(32)
	if err != nil {
		return nil, err
	}
	row := entity.AuthRefreshToken{
		UserID:    user.ID,
		TokenHash: hashToken(rawRefresh),
		ExpiresAt: time.Now().Add(refreshTokenTTL),
	}
	if err := u.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, fmt.Errorf("保存 refresh token 失败: %w", err)
	}
	return &SupabaseAuthOutput{
		Token:        access,
		RefreshToken: rawRefresh,
		UserID:       user.ID,
		Username:     user.DisplayName,
		Email:        user.Email,
	}, nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
