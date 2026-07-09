// supabase_auth_usecase.go 通过 Supabase Auth 实现邮箱密码注册与登录。
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/supabase-community/gotrue-go/types"
	pkgsb "github.com/stvenfor/my_go_study/pkg/supabase"
)

var (
	// ErrEmailConfirmationRequired 注册成功但需邮箱验证后才能登录。
	ErrEmailConfirmationRequired = errors.New("email confirmation required")
	// ErrAccountNotRegistered 邮箱尚未注册。
	ErrAccountNotRegistered = errors.New("account not registered")
	// ErrSupabaseUnavailable 无法连接 Supabase 服务。
	ErrSupabaseUnavailable = errors.New("supabase unavailable")
)

// SupabaseAuthOutput Supabase 认证成功结果。
type SupabaseAuthOutput struct {
	Token    string
	UserID   string
	Username string
	Email    string
}

// SupabaseAuthUsecase Supabase 邮箱密码认证用例。
type SupabaseAuthUsecase struct {
	sb *pkgsb.Client
}

// NewSupabaseAuthUsecase 创建 Supabase 认证用例。
func NewSupabaseAuthUsecase(sb *pkgsb.Client) *SupabaseAuthUsecase {
	return &SupabaseAuthUsecase{sb: sb}
}

// Register 使用邮箱密码在 Supabase 注册。
func (u *SupabaseAuthUsecase) Register(ctx context.Context, input RegisterInput) (*SupabaseAuthOutput, error) {
	_ = ctx
	email := strings.TrimSpace(input.Email)
	if email == "" || input.Password == "" {
		return nil, ErrInvalidParams
	}
	if len(input.Password) < 6 {
		return nil, ErrInvalidParams
	}

	metadata := map[string]interface{}{}
	if name := strings.TrimSpace(input.Username); name != "" {
		metadata["display_name"] = name
	}

	resp, err := u.sb.Anon.Auth.Signup(types.SignupRequest{
		Email:    email,
		Password: input.Password,
		Data:     metadata,
	})
	if err != nil {
		return nil, mapSupabaseAuthError(err)
	}

	user := resp.User
	if user.ID.String() == "00000000-0000-0000-0000-000000000000" && resp.Session.User.ID.String() != "00000000-0000-0000-0000-000000000000" {
		user = resp.Session.User
	}

	if resp.Session.AccessToken == "" {
		if user.ID.String() != "00000000-0000-0000-0000-000000000000" {
			return &SupabaseAuthOutput{
				UserID:   user.ID.String(),
				Username: resolveSupabaseUsername(user, input.Username),
				Email:    user.Email,
			}, ErrEmailConfirmationRequired
		}
		return nil, ErrEmailConfirmationRequired
	}

	return supabaseAuthOutputFromSession(resp.Session, input.Username), nil
}

// Login 使用邮箱密码登录 Supabase 并返回 access token。
func (u *SupabaseAuthUsecase) Login(ctx context.Context, input LoginInput) (*SupabaseAuthOutput, error) {
	_ = ctx
	email := strings.TrimSpace(input.Username)
	if email == "" || input.Password == "" {
		return nil, ErrInvalidParams
	}
	if !strings.Contains(email, "@") {
		return nil, ErrInvalidCredentials
	}

	token, err := u.sb.Anon.Auth.SignInWithEmailPassword(email, input.Password)
	if err != nil {
		mapped := mapSupabaseAuthError(err)
		if errors.Is(mapped, ErrInvalidCredentials) {
			return nil, u.refineInvalidCredentials(email, mapped)
		}
		return nil, mapped
	}
	return supabaseAuthOutputFromToken(token, email), nil
}

func (u *SupabaseAuthUsecase) refineInvalidCredentials(email string, fallback error) error {
	exists, err := u.sb.EmailRegistered(email)
	if err != nil || !u.sb.HasServiceRole() {
		return fallback
	}
	if !exists {
		return ErrAccountNotRegistered
	}
	return ErrInvalidCredentials
}

func supabaseAuthOutputFromToken(token *types.TokenResponse, fallbackUsername string) *SupabaseAuthOutput {
	return supabaseAuthOutputFromSession(token.Session, fallbackUsername)
}

func supabaseAuthOutputFromSession(session types.Session, fallbackUsername string) *SupabaseAuthOutput {
	user := session.User
	return &SupabaseAuthOutput{
		Token:    session.AccessToken,
		UserID:   user.ID.String(),
		Username: resolveSupabaseUsername(user, fallbackUsername),
		Email:    user.Email,
	}
}

func resolveSupabaseUsername(user types.User, fallback string) string {
	if name, ok := user.UserMetadata["display_name"].(string); ok {
		name = strings.TrimSpace(name)
		if name != "" {
			return name
		}
	}
	if user.Email != "" {
		if local := strings.Split(user.Email, "@")[0]; local != "" {
			return local
		}
	}
	fallback = strings.TrimSpace(fallback)
	if fallback != "" {
		return fallback
	}
	return user.Email
}

func mapSupabaseAuthError(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "user already registered"),
		strings.Contains(msg, "already been registered"),
		strings.Contains(msg, "email address is already registered"):
		return ErrUserExists
	case strings.Contains(msg, "invalid login credentials"),
		strings.Contains(msg, "invalid_grant"),
		strings.Contains(msg, "invalid email or password"):
		return ErrInvalidCredentials
	case strings.Contains(msg, "signup is disabled"):
		return ErrInvalidParams
	case strings.Contains(msg, "password should be at least"),
		strings.Contains(msg, "weak password"):
		return ErrInvalidParams
	case strings.Contains(msg, "invalid api key"),
		strings.Contains(msg, "no api key"),
		strings.Contains(msg, "jwt"):
		return ErrInvalidCredentials
	case strings.Contains(msg, "timeout"),
		strings.Contains(msg, "deadline exceeded"),
		strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "no such host"),
		strings.Contains(msg, "i/o timeout"):
		return ErrSupabaseUnavailable
	default:
		return fmt.Errorf("supabase auth 失败: %w", err)
	}
}
