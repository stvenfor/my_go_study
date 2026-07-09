package usecase_test

import (
	"context"
	"testing"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/usecase"
)

type mockPresenceRepo struct {
	onlineCount int
	lastUserID  string
	lastDevice  string
	wasOnline   bool
}

func (m *mockPresenceRepo) SetOnline(_ context.Context, userID, device string) (int, error) {
	m.lastUserID = userID
	m.lastDevice = device
	m.wasOnline = true
	m.onlineCount = 3
	return m.onlineCount, nil
}

func (m *mockPresenceRepo) SetOffline(_ context.Context, userID string) (int, error) {
	m.lastUserID = userID
	m.wasOnline = false
	m.onlineCount = 2
	return m.onlineCount, nil
}

type mockPresenceBroadcaster struct {
	excludeUserID string
	topic         string
	envelope      entity.RealtimeEnvelope
	delivered     int
}

func (m *mockPresenceBroadcaster) BroadcastToTopicExcept(excludeUserID, topic string, envelope entity.RealtimeEnvelope) int {
	m.excludeUserID = excludeUserID
	m.topic = topic
	m.envelope = envelope
	return m.delivered
}

func TestRealtimePresenceReport(t *testing.T) {
	repo := &mockPresenceRepo{}
	broadcaster := &mockPresenceBroadcaster{delivered: 2}
	uc := usecase.NewRealtimePresenceUsecase(repo, broadcaster)

	out, delivered, err := uc.Report(context.Background(), usecase.RealtimePresenceInput{
		UserID: "user-a",
		Online: true,
		Device: "ios",
	})
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if delivered != 2 {
		t.Fatalf("delivered=%d", delivered)
	}
	if broadcaster.excludeUserID != "user-a" {
		t.Fatalf("exclude=%s", broadcaster.excludeUserID)
	}
	if broadcaster.topic != entity.TopicPresenceBulk {
		t.Fatalf("topic=%s", broadcaster.topic)
	}
	if out.Payload["name"] != entity.EventPresenceUpdate {
		t.Fatalf("name=%v", out.Payload["name"])
	}
	if out.Payload["userId"] != "user-a" {
		t.Fatalf("userId=%v", out.Payload["userId"])
	}
	if out.Payload["onlineCount"] != 3 {
		t.Fatalf("onlineCount=%v", out.Payload["onlineCount"])
	}
}
