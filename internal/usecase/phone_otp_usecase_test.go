package usecase

import (
	"context"
	"testing"

	"github.com/stvenfor/my_go_study/pkg/config"
	pkgsb "github.com/stvenfor/my_go_study/pkg/supabase"
)

func TestPhoneOTPUsecase_SendPhoneOTP(t *testing.T) {
	auth := config.AuthConfig{
		DevTestPhone: "13400000000",
		DevTestOTP:   "123456",
	}

	uc := NewPhoneOTPUsecase(nil, auth, "debug")
	if err := uc.SendPhoneOTP(context.Background(), "13400000000"); err != ErrSupabaseUnavailable {
		t.Fatalf("expected ErrSupabaseUnavailable, got %v", err)
	}

	uc = NewPhoneOTPUsecase(&pkgsb.Client{}, auth, "release")
	if err := uc.SendPhoneOTP(context.Background(), "13400000000"); err != ErrPhoneLoginNotAvailable {
		t.Fatalf("expected ErrPhoneLoginNotAvailable in release, got %v", err)
	}

	uc = NewPhoneOTPUsecase(&pkgsb.Client{}, auth, "debug")
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
	uc := NewPhoneOTPUsecase(&pkgsb.Client{}, auth, "debug")

	if _, err := uc.VerifyPhoneOTP(context.Background(), "13400000000", "000000"); err != ErrInvalidOTP {
		t.Fatalf("expected ErrInvalidOTP, got %v", err)
	}
	if _, err := uc.VerifyPhoneOTP(context.Background(), "13400000000", "123456"); err != ErrSupabaseUnavailable {
		t.Fatalf("expected ErrSupabaseUnavailable without service role, got %v", err)
	}
}

func TestAuthConfig_IsDevTestPhone(t *testing.T) {
	auth := config.AuthConfig{DevTestPhone: "13400000000"}
	cases := []struct {
		phone string
		want  bool
	}{
		{"13400000000", true},
		{"+8613400000000", true},
		{"13800000000", false},
	}
	for _, tc := range cases {
		if got := auth.IsDevTestPhone(tc.phone); got != tc.want {
			t.Fatalf("phone %s: want %v got %v", tc.phone, tc.want, got)
		}
	}
}
