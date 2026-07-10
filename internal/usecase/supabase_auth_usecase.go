// =============================================================================
// 文件：supabase_auth_usecase.go
// 层级：Usecase —— 登录/注册的「业务大脑」
//
// 数据流：UserHandler → 本文件 → pkg/supabase (gotrue-go) → Supabase Cloud
//
// 【初学者】Register 与 Login 区别：
//   Register → Auth.Signup，可能需邮箱验证才返回 token
//   Login    → SignInWithEmailPassword，成功必有 access_token
// =============================================================================
package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/supabase-community/gotrue-go/types"
	pkgsb "github.com/stvenfor/my_go_study/pkg/supabase"
)

const supabaseAuthTimeout = 20 * time.Second

var (
	ErrEmailConfirmationRequired = errors.New("email confirmation required")
	ErrAccountNotRegistered        = errors.New("account not registered")
	ErrSupabaseUnavailable         = errors.New("supabase unavailable")
)

type SupabaseAuthOutput struct {
	Token        string // Supabase access_token，Flutter 存本地
	RefreshToken string // Supabase refresh_token，用于静默续期
	UserID       string // UUID
	Username     string // 展示名
	Email        string
}

type SupabaseAuthUsecase struct {
	sb *pkgsb.Client
}

func NewSupabaseAuthUsecase(sb *pkgsb.Client) *SupabaseAuthUsecase {
	return &SupabaseAuthUsecase{sb: sb}
}

// Register 在 Supabase 创建账号。
func (u *SupabaseAuthUsecase) Register(ctx context.Context, input RegisterInput) (*SupabaseAuthOutput, error) {
	email := strings.TrimSpace(input.Email)
	if email == "" || input.Password == "" {
		return nil, ErrInvalidParams
	}
	if len(input.Password) < 6 {
		return nil, ErrInvalidParams
	}

	metadata := map[string]interface{}{}
	if name := strings.TrimSpace(input.Username); name != "" {
		metadata["display_name"] = name // 存 Supabase user_metadata
	}

	var resp *types.SignupResponse
	err := u.withAuthTimeout(ctx, func(ctx context.Context) error {
		var callErr error
		resp, callErr = u.sb.Anon.Auth.Signup(types.SignupRequest{
			Email:    email,
			Password: input.Password,
			Data:     metadata,
		})
		return callErr
	})
	if err != nil {
		return nil, mapSupabaseAuthError(err)
	}
	if resp == nil {
		return nil, ErrSupabaseUnavailable
	}

	user := resp.User
	// 部分 Supabase 版本 Signup 用户 ID 在 Session 里
	if user.ID.String() == "00000000-0000-0000-0000-000000000000" && resp.Session.User.ID.String() != "00000000-0000-0000-0000-000000000000" {
		user = resp.Session.User
	}

	// 开启邮箱验证时 Signup 不返回 token
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

// Login 邮箱密码登录。input.Username 实际是邮箱（Flutter 传入）。
func (u *SupabaseAuthUsecase) Login(ctx context.Context, input LoginInput) (*SupabaseAuthOutput, error) {
	email := strings.TrimSpace(input.Username)
	if email == "" || input.Password == "" {
		return nil, ErrInvalidParams
	}
	if !strings.Contains(email, "@") {
		return nil, ErrInvalidCredentials // 强制邮箱格式
	}

	var token *types.TokenResponse
	err := u.withAuthTimeout(ctx, func(ctx context.Context) error {
		var callErr error
		token, callErr = u.sb.Anon.Auth.SignInWithEmailPassword(email, input.Password)
		return callErr
	})
	if err != nil {
		mapped := mapSupabaseAuthError(err)
		if errors.Is(mapped, ErrInvalidCredentials) {
			// 有 service_role 时可区分「未注册」与「密码错」
			return nil, u.refineInvalidCredentials(email, mapped)
		}
		return nil, mapped
	}
	return supabaseAuthOutputFromToken(token, email), nil
}

// RefreshToken 使用 refresh_token 换取新的 access_token。
func (u *SupabaseAuthUsecase) RefreshToken(ctx context.Context, refreshToken string) (*SupabaseAuthOutput, error) {
	if u.sb == nil {
		return nil, ErrSupabaseUnavailable
	}
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, ErrInvalidParams
	}

	var token *types.TokenResponse
	err := u.withAuthTimeout(ctx, func(ctx context.Context) error {
		var callErr error
		token, callErr = u.sb.Anon.Auth.RefreshToken(refreshToken)
		return callErr
	})
	if err != nil {
		return nil, mapSupabaseAuthError(err)
	}
	return supabaseAuthOutputFromToken(token, ""), nil
}

// Logout 撤销 Supabase 侧 refresh token（需有效 access_token）。
func (u *SupabaseAuthUsecase) Logout(ctx context.Context, accessToken string) error {
	if u.sb == nil {
		return ErrSupabaseUnavailable
	}
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return ErrInvalidParams
	}
	return u.withAuthTimeout(ctx, func(ctx context.Context) error {
		return u.sb.Anon.Auth.WithToken(accessToken).Logout()
	})
}

// withAuthTimeout 防止 Supabase 网络卡住占满 HTTP  worker。
func (u *SupabaseAuthUsecase) withAuthTimeout(ctx context.Context, fn func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, supabaseAuthTimeout)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- fn(ctx)
	}()

	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return ErrSupabaseUnavailable
		}
		return ctx.Err()
	case err := <-errCh:
		return err
	}
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
		Token:        session.AccessToken,
		RefreshToken: session.RefreshToken,
		UserID:       user.ID.String(),
		Username:     resolveSupabaseUsername(user, fallbackUsername),
		Email:        user.Email,
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

// mapSupabaseAuthError 把 Supabase 英文错误映射为项目内 sentinel error，供 Handler 翻译中文。
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
		strings.Contains(msg, "invalid jwt"),
		strings.Contains(msg, "jwt signature"):
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
