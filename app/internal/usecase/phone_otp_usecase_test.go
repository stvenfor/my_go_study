package usecase

import (
	"context"
	"testing"

	"common/config"
	pkgsb "core/supabase"
)

func TestPhoneOTPUsecase_SendPhoneOTP(t *testing.T) {
	auth := config.AuthConfig{
		DevTestPhone: "13400000000",
		DevTestOTP:   "123456",
	}

	uc := NewPhoneOTPUsecase(nil, auth, "dev")
	if err := uc.SendPhoneOTP(context.Background(), "13400000000"); err != ErrSupabaseUnavailable {
		t.Fatalf("expected ErrSupabaseUnavailable, got %v", err)
	}

	uc = NewPhoneOTPUsecase(&pkgsb.Client{}, auth, "prod")
	if err := uc.SendPhoneOTP(context.Background(), "13400000000"); err != ErrPhoneLoginNotAvailable {
		t.Fatalf("expected ErrPhoneLoginNotAvailable in prod, got %v", err)
	}

	uc = NewPhoneOTPUsecase(&pkgsb.Client{}, auth, "dev")
	if err := uc.SendPhoneOTP(context.Background(), "13800000000"); err != ErrPhoneLoginNotAvailable {
		t.Fatalf("expected ErrPhoneLoginNotAvailable for non-test phone, got %v", err)
	}
	if err := uc.SendPhoneOTP(context.Background(), "+8613400000000"); err != nil {
		t.Fatalf("expected nil for test phone, got %v", err)
	}
}

func TestPhoneOTPUsecase_VerifyPhoneOTP_Validation(t *testing.T) {
	auth := config.AuthConfig{
		DevTestPhone: "13400000000",
		DevTestOTP:   "123456",
	}
	uc := NewPhoneOTPUsecase(&pkgsb.Client{}, auth, "dev")

	if _, err := uc.VerifyPhoneOTP(context.Background(), "13400000000", "000000"); err != ErrInvalidOTP {
		t.Fatalf("expected ErrInvalidOTP, got %v", err)
	}
	if _, err := uc.VerifyPhoneOTP(context.Background(), "13400000000", "123456"); err != ErrSupabaseUnavailable {
		t.Fatalf("expected ErrSupabaseUnavailable without service role, got %v", err)
	}
}

// parity-p0-5 硬门槛：APP_ENV=production 时固定测试号 + 固定码 123456 不得换到登录态。
func TestPhoneOTP_ProductionRejectsDevFixedCode(t *testing.T) {
	auth := config.AuthConfig{DevTestPhone: "13400000000", DevTestOTP: "123456"}

	if _, err := NewPhoneOTPUsecase(&pkgsb.Client{}, auth, "production").
		VerifyPhoneOTP(context.Background(), "13400000000", "123456"); err != ErrPhoneLoginNotAvailable {
		t.Fatalf("supabase prod verify: want ErrPhoneLoginNotAvailable, got %v", err)
	}

	local := NewLocalAuthUsecase(nil, nil, auth, "production")
	if err := local.SendPhoneOTP(context.Background(), "13400000000"); err != ErrPhoneLoginNotAvailable {
		t.Fatalf("local prod send: want ErrPhoneLoginNotAvailable, got %v", err)
	}
	if _, err := local.VerifyPhoneOTP(context.Background(), "13400000000", "123456"); err != ErrPhoneLoginNotAvailable {
		t.Fatalf("local prod verify: want ErrPhoneLoginNotAvailable, got %v", err)
	}

	// 联调环境：发送仍是 no-op 成功（Placeholder，见 docs/placeholder-inventory.md），登录放行。
	if err := NewLocalAuthUsecase(nil, nil, auth, "dev").SendPhoneOTP(context.Background(), "13400000000"); err != nil {
		t.Fatalf("dev send should no-op succeed, got %v", err)
	}
}

func TestAuthConfig_IsDevTestPhone(t *testing.T) {
	auth := config.AuthConfig{DevTestPhone: "13400000000,13400000001,13400000002,13400000003,13400000004"}
	cases := []struct {
		phone string
		want  bool
	}{
		{"13400000000", true},
		{"+8613400000000", true},
		{"13400000001", true},
		{"13400000002", true},
		{"13400000003", true},
		{"13400000004", true},
		{"13800000000", false},
	}
	for _, tc := range cases {
		if got := auth.IsDevTestPhone(tc.phone); got != tc.want {
			t.Fatalf("phone %s: want %v got %v", tc.phone, tc.want, got)
		}
	}
}

func TestDevTestDisplayName(t *testing.T) {
	cases := map[string]string{
		"13400000000": "测试甲",
		"13400000001": "测试乙",
		"13400000002": "测试丙",
		"13400000003": "测试丁",
		"13400000004": "测试戊",
	}
	for phone, want := range cases {
		if got := config.DevTestDisplayName(phone); got != want {
			t.Fatalf("%s: want %s got %s", phone, want, got)
		}
	}
}
