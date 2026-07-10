package queue

import (
	"encoding/json"
	"testing"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

func TestFanoutMessageJSONRoundtrip(t *testing.T) {
	env := entity.RealtimeEnvelope{
		ID:    "evt_1",
		Type:  "event",
		Topic: entity.TopicSysNotify,
		Seq:   3,
		Payload: map[string]any{
			"name":     entity.EventSysNotifyShow,
			"notifyId": "abc-123",
			"title":    "t",
			"body":     "b",
		},
	}
	msg := FanoutMessage{
		UserID:   "user-1",
		Topic:    entity.TopicSysNotify,
		Envelope: env,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	var back FanoutMessage
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if back.Envelope.Type != "event" {
		t.Fatalf("type=%q want event", back.Envelope.Type)
	}
	if back.Envelope.Seq != 3 {
		t.Fatalf("seq=%d want 3", back.Envelope.Seq)
	}
	if back.UserID != "user-1" {
		t.Fatalf("userId=%q", back.UserID)
	}
}

func TestFanoutSubscriberHandleMessage(t *testing.T) {
	delivered := 0
	sub := NewFanoutSubscriber(nil, "ch", func(userID, topic string, envelope entity.RealtimeEnvelope) int {
		if userID != "user-1" || topic != entity.TopicSysNotify || envelope.Seq != 4 {
			t.Fatalf("unexpected broadcast user=%s topic=%s seq=%d", userID, topic, envelope.Seq)
		}
		delivered++
		return delivered
	}, nil)

	payload, _ := json.Marshal(FanoutMessage{
		UserID: "user-1",
		Topic:  entity.TopicSysNotify,
		Envelope: entity.RealtimeEnvelope{
			ID:    "evt_2",
			Type:  "event",
			Topic: entity.TopicSysNotify,
			Seq:   4,
			Payload: map[string]any{
				"name": entity.EventSysNotifyShow,
			},
		},
	})
	sub.handleMessage(string(payload))
	if delivered != 1 {
		t.Fatalf("delivered=%d want 1", delivered)
	}
}
