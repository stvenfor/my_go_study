package queue

import (
	"testing"
)

func TestNewBroadcastNotifyTask(t *testing.T) {
	env, err := NewBroadcastNotifyTask(BroadcastNotifyPayload{ScheduleSlot: "2026-07-10T10:00:00+08:00"})
	if err != nil {
		t.Fatalf("new broadcast task: %v", err)
	}
	if env.Type != TypeScheduledBroadcastNotify {
		t.Fatalf("type=%s", env.Type)
	}
	got, err := ParseBroadcastNotifyPayload(env.Payload)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.ScheduleSlot == "" {
		t.Fatal("scheduleSlot empty")
	}
}

func TestNewPushNotifyTaskRoundTrip(t *testing.T) {
	env, err := NewPushNotifyTask(PushNotifyPayload{
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
	if env.Type != TypeRealtimePushNotify {
		t.Fatalf("type=%s", env.Type)
	}
	if env.TaskID == "" {
		t.Fatal("task_id empty")
	}

	got, err := ParsePushNotifyPayload(env.Payload)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.UserID != "user-1" || got.Title != "hello" || got.Extra["k"] != "v" {
		t.Fatalf("unexpected payload: %+v", got)
	}
}

func TestParsePushNotifyPayloadInvalidJSON(t *testing.T) {
	_, err := ParsePushNotifyPayload([]byte("not-json"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewSendSMSTask(t *testing.T) {
	env, err := NewSendSMSTask(SendSMSPayload{Phone: "13400000000", Message: "123456"})
	if err != nil {
		t.Fatalf("new sms task: %v", err)
	}
	if env.Type != TypeSendSMS {
		t.Fatalf("type=%s", env.Type)
	}
}

func TestNewJPushRegisterTask(t *testing.T) {
	env, err := NewJPushRegisterTask(JPushRegisterPayload{
		UserID: "u1", DeviceID: "d1", Platform: "ios", RegisterID: "r1",
	})
	if err != nil {
		t.Fatalf("new jpush task: %v", err)
	}
	if env.Type != TypeJPushRegister {
		t.Fatalf("type=%s", env.Type)
	}
}

func TestHourMatchesCron(t *testing.T) {
	if !hourMatchesCron("0 10-19 * * *", 10) || !hourMatchesCron("0 10-19 * * *", 19) {
		t.Fatal("expected 10 and 19 in range")
	}
	if hourMatchesCron("0 10-19 * * *", 9) || hourMatchesCron("0 10-19 * * *", 20) {
		t.Fatal("expected 9 and 20 out of range")
	}
	if !hourMatchesCron("0 * * * *", 3) {
		t.Fatal("expected * to match")
	}
}
