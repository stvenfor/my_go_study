package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/supabase-community/gotrue-go/types"
	"github.com/stvenfor/my_go_study/pkg/config"
	pkgsb "github.com/stvenfor/my_go_study/pkg/supabase"
)

var (
	ErrInvalidOTP             = errors.New("invalid otp")
	ErrPhoneLoginNotAvailable = errors.New("phone login not available")
)

// PhoneOTPUsecase 处理测试环境手机号 OTP 登录。
type PhoneOTPUsecase struct {
	sb         *pkgsb.Client
	auth       config.AuthConfig
	serverMode string
}

// NewPhoneOTPUsecase 创建手机号 OTP 用例。
func NewPhoneOTPUsecase(sb *pkgsb.Client, auth config.AuthConfig, serverMode string) *PhoneOTPUsecase {
	return &PhoneOTPUsecase{sb: sb, auth: auth, serverMode: serverMode}
}

// SendPhoneOTP 发送短信验证码；dev 测试号 no-op。
func (u *PhoneOTPUsecase) SendPhoneOTP(ctx context.Context, phone string) error {
	if u.sb == nil {
		return ErrSupabaseUnavailable
	}
	if !u.auth.DevBypassEnabled(u.serverMode) {
		return ErrPhoneLoginNotAvailable
	}
	if !u.auth.IsDevTestPhone(phone) {
		return ErrPhoneLoginNotAvailable
	}
	return nil
}

// VerifyPhoneOTP 校验验证码并签发 Supabase session。
func (u *PhoneOTPUsecase) VerifyPhoneOTP(ctx context.Context, phone, otp string) (*SupabaseAuthOutput, error) {
	if u.sb == nil {
		return nil, ErrSupabaseUnavailable
	}
	if !u.auth.DevBypassEnabled(u.serverMode) {
		return nil, ErrPhoneLoginNotAvailable
	}
	if !u.auth.IsDevTestPhone(phone) {
		return nil, ErrPhoneLoginNotAvailable
	}
	if strings.TrimSpace(otp) != strings.TrimSpace(u.auth.DevTestOTP) {
		return nil, ErrInvalidOTP
	}
	if !u.sb.HasServiceRole() {
		return nil, ErrSupabaseUnavailable
	}

	digits := config.NormalizePhoneDigits(phone)
	e164 := config.ToE164China(phone)
	devEmail := config.DevPhoneEmail(digits)
	password := u.auth.DevTestPasswordOrDefault()
	displayName := fmt.Sprintf("用户%s", lastNDigits(digits, 4))

	if _, err := u.sb.EnsureDevPhoneUser(e164, devEmail, password, displayName); err != nil {
		return nil, ErrSupabaseUnavailable
	}

	var token *types.TokenResponse
	err := u.withAuthTimeout(ctx, func(ctx context.Context) error {
		var callErr error
		token, callErr = u.sb.Anon.Auth.SignInWithEmailPassword(devEmail, password)
		return callErr
	})
	if err != nil {
		return nil, mapSupabaseAuthError(err)
	}
	return supabaseAuthOutputFromToken(token, displayName), nil
}

func (u *PhoneOTPUsecase) withAuthTimeout(ctx context.Context, fn func(context.Context) error) error {
	authUC := &SupabaseAuthUsecase{}
	return authUC.withAuthTimeout(ctx, fn)
}

func lastNDigits(digits string, n int) string {
	if len(digits) <= n {
		return digits
	}
	return digits[len(digits)-n:]
}
