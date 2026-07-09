package queue

import (
	"testing"

	"github.com/hibiken/asynq"
)

func TestNewBroadcastNotifyTask(t *testing.T) {
	task, err := NewBroadcastNotifyTask(BroadcastNotifyPayload{ScheduleSlot: "2026-07-10T10:00:00+08:00"})
	if err != nil {
		t.Fatalf("new broadcast task: %v", err)
	}
	if task.Type() != TypeScheduledBroadcastNotify {
		t.Fatalf("type=%s", task.Type())
	}
	got, err := ParseBroadcastNotifyPayload(task)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.ScheduleSlot == "" {
		t.Fatal("scheduleSlot empty")
	}
}

func TestNewPushNotifyTaskRoundTrip(t *testing.T) {
	task, err := NewPushNotifyTask(PushNotifyPayload{
		UserID: "user-1",
		Topic:  "sys.notify",
		Title:  "hello",
		Body:   "world",
		Name:   "sys.notify.show",
		Extra:  map[string]any{"k": "v"},
	})
	if err != nil {
		t.Fatalf("new task: %v", err)
	}
	if task.Type() != TypeRealtimePushNotify {
		t.Fatalf("type=%s", task.Type())
	}

	got, err := ParsePushNotifyPayload(task)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.UserID != "user-1" || got.Title != "hello" || got.Extra["k"] != "v" {
		t.Fatalf("unexpected payload: %+v", got)
	}
}

func TestParsePushNotifyPayloadInvalidJSON(t *testing.T) {
	_, err := ParsePushNotifyPayload(asynq.NewTask(TypeRealtimePushNotify, []byte("not-json")))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewSendSMSTask(t *testing.T) {
	task, err := NewSendSMSTask(SendSMSPayload{Phone: "13400000000", Message: "123456"})
	if err != nil {
		t.Fatalf("new sms task: %v", err)
	}
	if task.Type() != TypeSendSMS {
		t.Fatalf("type=%s", task.Type())
	}
}

func TestNewJPushRegisterTask(t *testing.T) {
	task, err := NewJPushRegisterTask(JPushRegisterPayload{
		UserID: "u1", DeviceID: "d1", Platform: "ios", RegisterID: "r1",
	})
	if err != nil {
		t.Fatalf("new jpush task: %v", err)
	}
	if task.Type() != TypeJPushRegister {
		t.Fatalf("type=%s", task.Type())
	}
}
