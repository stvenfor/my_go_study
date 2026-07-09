package ws

import (
	"testing"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"go.uber.org/zap"
)

func TestHubBroadcastToUser(t *testing.T) {
	hub := NewHub()
	client := NewClient(hub, nil, zap.NewNop())
	client.userID = "u1"
	hub.Register(client)
	client.Subscribe([]string{entity.TopicSysNotify})

	delivered := hub.BroadcastToUser("u1", entity.TopicSysNotify, entity.RealtimeEnvelope{
		Type:  "event",
		Topic: entity.TopicSysNotify,
		Seq:   1,
	})
	if delivered != 1 {
		t.Fatalf("delivered=%d", delivered)
	}
}

func TestHubBroadcastToTopicExcept(t *testing.T) {
	hub := NewHub()

	reporter := NewClient(hub, nil, zap.NewNop())
	reporter.userID = "u1"
	hub.Register(reporter)
	reporter.Subscribe([]string{entity.TopicPresenceBulk})

	peer := NewClient(hub, nil, zap.NewNop())
	peer.userID = "u2"
	hub.Register(peer)
	peer.Subscribe([]string{entity.TopicPresenceBulk})

	other := NewClient(hub, nil, zap.NewNop())
	other.userID = "u3"
	hub.Register(other)
	other.Subscribe([]string{entity.TopicSysNotify})

	delivered := hub.BroadcastToTopicExcept("u1", entity.TopicPresenceBulk, entity.RealtimeEnvelope{
		Type:  "event",
		Topic: entity.TopicPresenceBulk,
		Payload: map[string]any{
			"name":   entity.EventPresenceUpdate,
			"userId": "u1",
			"online": true,
		},
	})
	if delivered != 1 {
		t.Fatalf("delivered=%d want 1 (only u2)", delivered)
	}
}
