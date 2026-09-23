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

	existing, err := u.findOpenByEmail(ctx, email)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	if existing != nil {
		return nil, ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("密码哈希失败: %w", err)
	}

	user := newLocalUser(email, registerUserName(input.Username, email), "", string(hash))
	if err := u.db.WithContext(ctx).Create(&user).Error; err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

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

	user, err := u.findOpenByEmail(ctx, email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrAccountNotRegistered
	}
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	now := time.Now().UTC()
	if err := loginBlockReason(user, now); err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		applyFailedLogin(user, now)
		if saveErr := u.saveLoginState(ctx, user); saveErr != nil {
			return nil, saveErr
		}
		return nil, ErrInvalidCredentials
	}
	applySuccessfulLogin(user, now)
	if err := u.saveLoginState(ctx, user); err != nil {
		return nil, err
	}
	return u.issueTokens(ctx, user)
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

	var user entity.User
	if err := u.db.WithContext(ctx).Where("user_id = ?", row.UserID).First(&user).Error; err != nil {
		return nil, ErrInvalidCredentials
	}
	if user.DeletedAt != nil {
		_ = u.db.WithContext(ctx).Delete(&row).Error
		return nil, ErrAccountNotRegistered
	}
	if user.Status == entity.UserStatusDisabled {
		_ = u.db.WithContext(ctx).Where("user_id = ?", user.UserID).Delete(&entity.AuthRefreshToken{}).Error
		return nil, ErrAccountDisabled
	}

	_ = u.db.WithContext(ctx).Delete(&row).Error
	return u.issueTokens(ctx, &user)
}

func (u *LocalAuthUsecase) Logout(ctx context.Context, accessToken string) error {
	if claims, err := u.jwt.ParseUUID(accessToken); err == nil {
		_ = u.db.WithContext(ctx).Where("user_id = ?", claims.Subject).Delete(&entity.AuthRefreshToken{}).Error
	}
	return nil
}

// Deactivate 注销当前账号：只写 deleted_at，并撤销该 user_id 的 refresh token。
func (u *LocalAuthUsecase) Deactivate(ctx context.Context, userID string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ErrInvalidParams
	}
	now := time.Now().UTC()
	res := u.db.WithContext(ctx).Model(&entity.User{}).
		Where("user_id = ? AND deleted_at IS NULL", userID).
		Updates(map[string]any{"deleted_at": now, "updated_at": now})
	if res.Error != nil {
		return fmt.Errorf("注销用户失败: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrAccountNotRegistered
	}
	if err := u.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&entity.AuthRefreshToken{}).Error; err != nil {
		return fmt.Errorf("撤销 refresh token 失败: %w", err)
	}
	return nil
}

// AllowsRequest 已登录请求在账号注销或停用时拒绝。锁定不打断已有会话。
func (u *LocalAuthUsecase) AllowsRequest(ctx context.Context, userID string) error {
	userID = strings.TrimSpace(userID)
	var user entity.User
	err := u.db.WithContext(ctx).Where("user_id = ?", userID).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) || user.DeletedAt != nil {
		return ErrAccountNotRegistered
	}
	if err != nil {
		return err
	}
	if user.Status == entity.UserStatusDisabled {
		return ErrAccountDisabled
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
	user, err := u.findOpenByEmail(ctx, email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		hash, herr := bcrypt.GenerateFromPassword([]byte(u.auth.DevTestPasswordOrDefault()), bcrypt.DefaultCost)
		if herr != nil {
			return nil, herr
		}
		created := newLocalUser(email, "dev-"+digits, digits, string(hash))
		if err := u.db.WithContext(ctx).Create(&created).Error; err != nil {
			return nil, fmt.Errorf("创建测试用户失败: %w", err)
		}
		user = &created
	} else if err != nil {
		return nil, err
	}
	return u.issueTokens(ctx, user)
}

// LoginOrRegisterHuawei 用华为 UnionID 找/建本地账号；有手机号则优先合并到已有手机号用户。
// email 占位 hw_{unionID}@huawei.local。
func (u *LocalAuthUsecase) LoginOrRegisterHuawei(ctx context.Context, unionID, openID, phone, nickname string) (*SupabaseAuthOutput, error) {
	unionID = strings.TrimSpace(unionID)
	if unionID == "" {
		return nil, ErrInvalidParams
	}
	digits := config.NormalizePhoneDigits(phone)
	email := fmt.Sprintf("hw_%s@huawei.local", unionID)

	user, err := u.findOpenByEmail(ctx, email)
	if errors.Is(err, gorm.ErrRecordNotFound) && digits != "" {
		user, err = u.findOpenByPhone(ctx, digits)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		hash, herr := bcrypt.GenerateFromPassword([]byte(uuid.NewString()), bcrypt.DefaultCost)
		if herr != nil {
			return nil, herr
		}
		name := strings.TrimSpace(nickname)
		if name == "" {
			name = "华为用户"
		}
		if len(name) > 64 {
			name = name[:64]
		}
		created := newLocalUser(email, name, digits, string(hash))
		if digits != "" {
			now := time.Now().UTC()
			created.PhoneVerifiedAt = &now
		}
		if err := u.db.WithContext(ctx).Create(&created).Error; err != nil {
			return nil, fmt.Errorf("创建华为用户失败: %w", err)
		}
		return u.issueTokens(ctx, &created)
	}
	if err != nil {
		return nil, err
	}

	updates := map[string]any{"updated_at": time.Now().UTC()}
	if user.Email != email && (strings.HasSuffix(user.Email, "@dev.test.local") || user.Email == "") {
		updates["email"] = email
		user.Email = email
	}
	if digits != "" && user.Phone == "" {
		updates["phone"] = digits
		user.Phone = digits
		now := time.Now().UTC()
		updates["phone_verified_at"] = now
		user.PhoneVerifiedAt = &now
	}
	if n := strings.TrimSpace(nickname); n != "" && (user.UserName == "华为用户" || user.UserName == "") {
		if len(n) > 64 {
			n = n[:64]
		}
		updates["user_name"] = n
		user.UserName = n
	}
	_ = openID // OpenID 仅联调用；账号主键用 UnionID。
	if len(updates) > 1 {
		_ = u.db.WithContext(ctx).Model(&entity.User{}).Where("user_id = ?", user.UserID).Updates(updates).Error
	}
	return u.issueTokens(ctx, user)
}

func (u *LocalAuthUsecase) findOpenByPhone(ctx context.Context, phone string) (*entity.User, error) {
	var user entity.User
	err := u.db.WithContext(ctx).Where("phone = ? AND deleted_at IS NULL", phone).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// LoginOrRegisterWechat 用微信 openid 找/建本地账号（email 占位 wx_{openid}@wechat.local）。
func (u *LocalAuthUsecase) LoginOrRegisterWechat(ctx context.Context, openID, nickname, avatarURL string) (*SupabaseAuthOutput, error) {
	openID = strings.TrimSpace(openID)
	if openID == "" {
		return nil, ErrInvalidParams
	}
	email := fmt.Sprintf("wx_%s@wechat.local", openID)
	user, err := u.findOpenByEmail(ctx, email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		hash, herr := bcrypt.GenerateFromPassword([]byte(uuid.NewString()), bcrypt.DefaultCost)
		if herr != nil {
			return nil, herr
		}
		name := strings.TrimSpace(nickname)
		if name == "" {
			name = "微信用户"
		}
		if len(name) > 64 {
			name = name[:64]
		}
		created := newLocalUser(email, name, "", string(hash))
		created.AvatarURL = strings.TrimSpace(avatarURL)
		if err := u.db.WithContext(ctx).Create(&created).Error; err != nil {
			return nil, fmt.Errorf("创建微信用户失败: %w", err)
		}
		user = &created
	} else if err != nil {
		return nil, err
	} else {
		updates := map[string]any{"updated_at": time.Now().UTC()}
		if n := strings.TrimSpace(nickname); n != "" && user.UserName == "微信用户" {
			if len(n) > 64 {
				n = n[:64]
			}
			updates["user_name"] = n
			user.UserName = n
		}
		if a := strings.TrimSpace(avatarURL); a != "" && user.AvatarURL == "" {
			updates["avatar_url"] = a
			user.AvatarURL = a
		}
		if len(updates) > 1 {
			_ = u.db.WithContext(ctx).Model(&entity.User{}).Where("user_id = ?", user.UserID).Updates(updates).Error
		}
	}
	return u.issueTokens(ctx, user)
}

func (u *LocalAuthUsecase) issueTokens(ctx context.Context, user *entity.User) (*SupabaseAuthOutput, error) {
	if user.DeletedAt != nil || user.Status == entity.UserStatusDisabled {
		return nil, ErrAccountDisabled
	}
	access, err := u.jwt.GenerateUUID(user.UserID, user.Email, user.UserName)
	if err != nil {
		return nil, err
	}
	rawRefresh, err := randomToken(32)
	if err != nil {
		return nil, err
	}
	row := entity.AuthRefreshToken{
		UserID:    user.UserID,
		TokenHash: hashToken(rawRefresh),
		ExpiresAt: time.Now().Add(refreshTokenTTL),
	}
	if err := u.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, fmt.Errorf("保存 refresh token 失败: %w", err)
	}
	return &SupabaseAuthOutput{
		Token:        access,
		RefreshToken: rawRefresh,
		UserID:       user.UserID,
		Username:     user.UserName,
		Email:        user.Email,
		Status:       user.Status,
		Phone:        user.Phone,
		AvatarURL:    user.AvatarURL,
	}, nil
}

func (u *LocalAuthUsecase) findOpenByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	err := u.db.WithContext(ctx).Where("email = ? AND deleted_at IS NULL", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (u *LocalAuthUsecase) saveLoginState(ctx context.Context, user *entity.User) error {
	locked := any(gorm.Expr("NULL"))
	if user.LockedUntil != nil {
		locked = *user.LockedUntil
	}
	last := any(gorm.Expr("NULL"))
	if user.LastLoginAt != nil {
		last = *user.LastLoginAt
	}
	return u.db.WithContext(ctx).Model(&entity.User{}).Where("user_id = ?", user.UserID).Updates(map[string]any{
		"status":             user.Status,
		"failed_login_count": user.FailedLoginCount,
		"locked_until":       locked,
		"last_login_at":      last,
		"updated_at":         time.Now().UTC(),
	}).Error
}

func newLocalUser(email, userName, phone, hash string) entity.User {
	now := time.Now().UTC()
	return entity.User{
		UserID:            uuid.NewString(),
		UserName:          userName,
		Email:             email,
		Phone:             phone,
		PasswordHash:      hash,
		Status:            entity.UserStatusActive,
		FailedLoginCount:  0,
		PasswordChangedAt: &now,
	}
}

func registerUserName(username, email string) string {
	name := strings.TrimSpace(username)
	if name == "" {
		name = strings.Split(email, "@")[0]
	}
	if name == "" {
		return "user"
	}
	return name
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
