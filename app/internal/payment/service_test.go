// service_test.go 支付预下单的「未配置」分支。
//
// 由 app/internal/usecase/third_party_auth_pay_test.go 拆出：该文件混装了
// 支付与第三方登录两组用例，批 3 切片时按域拆分（登录侧仍在 usecase，
// 见 usecase/third_party_auth_test.go）。
package payment

import (
	"context"
	"errors"
	"testing"

	"common/config"
)

func TestServiceNotConfigured(t *testing.T) {
	svc := NewService(config.ThirdPartyConfig{}, nil)
	_, err := svc.Prepay(context.Background(), PrepayInput{
		Channel:   "wechat",
		AmountFen: 100,
		Subject:   "test",
	})
	if !errors.Is(err, ErrPayNotConfigured) {
		t.Fatalf("want ErrPayNotConfigured, got %v", err)
	}

	_, err = svc.Prepay(context.Background(), PrepayInput{
		Channel:   "alipay",
		AmountFen: 100,
		Subject:   "test",
	})
	if !errors.Is(err, ErrPayNotConfigured) {
		t.Fatalf("want ErrPayNotConfigured, got %v", err)
	}

	_, err = svc.Prepay(context.Background(), PrepayInput{
		Channel:   "unknown",
		AmountFen: 100,
		Subject:   "test",
	})
	if !errors.Is(err, ErrPayChannelInvalid) {
		t.Fatalf("want ErrPayChannelInvalid, got %v", err)
	}
}
