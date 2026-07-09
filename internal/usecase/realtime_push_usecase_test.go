package usecase_test

import (
	"context"
	"testing"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/usecase"
	"github.com/stvenfor/my_go_study/pkg/config"
)

type mockBroadcaster struct {
	calls int
}

func (m *mockBroadcaster) BroadcastToUser(userID, topic string, envelope entity.RealtimeEnvelope) int {
	m.calls++
	return 1
}

type mockEnqueuer struct {
	taskID string
}

func (m *mockEnqueuer) EnqueueRealtimePush(ctx context.Context, input usecase.RealtimePushInput) (string, error) {
	return m.taskID, nil
}

func TestRealtimePushUsecaseDeliverRequiresBroadcaster(t *testing.T) {
	cfg := config.Config{Queue: config.QueueConfig{Enabled: false}}
	uc := usecase.NewRealtimePushUsecase(nil, cfg, nil, nil)

	_, _, err := uc.DeliverPush(context.Background(), usecase.RealtimePushInput{UserID: "u1"})
	if err == nil {
		t.Fatal("expected error without broadcaster")
	}
}

func TestRealtimePushUsecaseEnqueueWhenQueueEnabled(t *testing.T) {
	hub := &mockBroadcaster{}
	cfg := config.Config{Queue: config.QueueConfig{Enabled: true}}
	enqueuer := &mockEnqueuer{taskID: "task-abc"}
	uc := usecase.NewRealtimePushUsecase(nil, cfg, hub, enqueuer)

	out, err := uc.PushToUser(context.Background(), usecase.RealtimePushInput{
		UserID: "u1",
		Title:  "t",
		Body:   "b",
	})
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if !out.Queued || out.TaskID != "task-abc" || out.Delivered != -1 {
		t.Fatalf("unexpected output: %+v", out)
	}
	if hub.calls != 0 {
		t.Fatal("should not broadcast when queued")
	}
}
