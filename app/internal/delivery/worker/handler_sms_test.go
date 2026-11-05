// handler_sms_test.go 锁定 SMS 占位分支的可观测语义（清单条目 2）。
package worker

import (
	"context"
	"encoding/json"
	"testing"

	pkgqueue "core/queue"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

const smsPlaceholderMsg = "sms task NOT IMPLEMENTED: provider not integrated, task acked and dropped"

func TestHandleSendSMS_PlaceholderWarnsAndAcks(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	h := &Handler{log: zap.New(core)}

	payload, err := json.Marshal(pkgqueue.SendSMSPayload{Phone: "13800000000", Message: "code 123456"})
	if err != nil {
		t.Fatal(err)
	}
	// 未实现必须 ACK（返回 error 会走当前的无限重投），但要留下 Warn 痕迹。
	if err := h.handleSendSMS(context.Background(), payload); err != nil {
		t.Fatalf("placeholder SMS must ACK, got %v", err)
	}
	entries := logs.FilterMessage(smsPlaceholderMsg).All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 placeholder warn, got %d: %+v", len(entries), logs.All())
	}
	if got := entries[0].ContextMap()["status"]; got != "placeholder" {
		t.Fatalf("log status field = %v, want placeholder", got)
	}
}

func TestHandleSendSMS_BadPayloadNacksForRetry(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	h := &Handler{log: zap.New(core)}

	if err := h.handleSendSMS(context.Background(), []byte("{")); err == nil {
		t.Fatal("畸形载荷必须返回 error（交由消费侧重投/死信）")
	}
	if logs.Len() != 0 {
		t.Fatalf("畸形载荷不应打占位日志: %+v", logs.All())
	}
}
