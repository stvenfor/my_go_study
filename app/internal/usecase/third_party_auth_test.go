// third_party_auth_test.go 第三方登录（微信 / 华为）的「未配置」分支。
//
// 由 app/internal/usecase/third_party_auth_pay_test.go 拆出：原文件混装支付与
// 第三方登录两组用例，批 3 把支付域切片后，登录侧留在 usecase（批 4 归 auth 域）。
package usecase

import (
	"context"
	"errors"
	"testing"

	"common/config"
)

func TestWeChatAuthUsecase_NotConfigured(t *testing.T) {
	uc := NewWeChatAuthUsecase(config.ThirdPartyConfig{}, nil)
	_, err := uc.LoginWithWechatCode(context.Background(), "code")
	if !errors.Is(err, ErrWechatLocalOnly) {
		t.Fatalf("want ErrWechatLocalOnly, got %v", err)
	}

	uc = NewWeChatAuthUsecase(config.ThirdPartyConfig{}, stubWechatAccount{})
	_, err = uc.LoginWithWechatCode(context.Background(), "code")
	if !errors.Is(err, ErrWechatNotConfigured) {
		t.Fatalf("want ErrWechatNotConfigured, got %v", err)
	}
}

func TestHuaweiAuthUsecase_NotConfigured(t *testing.T) {
	uc := NewHuaweiAuthUsecase(config.ThirdPartyConfig{}, nil)
	_, err := uc.LoginWithHuaweiCode(context.Background(), "code")
	if !errors.Is(err, ErrHuaweiLocalOnly) {
		t.Fatalf("want ErrHuaweiLocalOnly, got %v", err)
	}

	uc = NewHuaweiAuthUsecase(config.ThirdPartyConfig{}, stubHuaweiAccount{})
	_, err = uc.LoginWithHuaweiCode(context.Background(), "code")
	if !errors.Is(err, ErrHuaweiNotConfigured) {
		t.Fatalf("want ErrHuaweiNotConfigured, got %v", err)
	}
}

type stubWechatAccount struct{}

func (stubWechatAccount) LoginOrRegisterWechat(context.Context, string, string, string) (*SupabaseAuthOutput, error) {
	return nil, errors.New("unused")
}

type stubHuaweiAccount struct{}

func (stubHuaweiAccount) LoginOrRegisterHuawei(context.Context, string, string, string, string) (*SupabaseAuthOutput, error) {
	return nil, errors.New("unused")
}

func TestThirdPartyConfigHelpers(t *testing.T) {
	cfg := config.ThirdPartyConfig{
		WeChatAppID:        "wx",
		WeChatAppSecret:    "secret",
		HuaweiClientID:     "hw",
		HuaweiClientSecret: "hwsecret",
	}
	if !cfg.WeChatLoginConfigured() {
		t.Fatal("login should be configured")
	}
	if cfg.WeChatPayConfigured() {
		t.Fatal("pay should not be configured without mch")
	}
	if !cfg.HuaweiLoginConfigured() {
		t.Fatal("huawei login should be configured")
	}
}
