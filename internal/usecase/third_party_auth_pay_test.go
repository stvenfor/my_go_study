package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/stvenfor/my_go_study/pkg/config"
)

func TestPaymentUsecase_NotConfigured(t *testing.T) {
	uc := NewPaymentUsecase(config.ThirdPartyConfig{})
	_, err := uc.Prepay(context.Background(), PrepayInput{
		Channel:   "wechat",
		AmountFen: 100,
		Subject:   "test",
	})
	if !errors.Is(err, ErrPayNotConfigured) {
		t.Fatalf("want ErrPayNotConfigured, got %v", err)
	}

	_, err = uc.Prepay(context.Background(), PrepayInput{
		Channel:   "alipay",
		AmountFen: 100,
		Subject:   "test",
	})
	if !errors.Is(err, ErrPayNotConfigured) {
		t.Fatalf("want ErrPayNotConfigured, got %v", err)
	}

	_, err = uc.Prepay(context.Background(), PrepayInput{
		Channel:   "unknown",
		AmountFen: 100,
		Subject:   "test",
	})
	if !errors.Is(err, ErrPayChannelInvalid) {
		t.Fatalf("want ErrPayChannelInvalid, got %v", err)
	}
}

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
