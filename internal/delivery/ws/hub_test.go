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
