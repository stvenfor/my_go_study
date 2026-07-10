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

type mockEventRepo struct{}

func (m *mockEventRepo) NextSeq(ctx context.Context, userID string) (int64, error) {
	return 1, nil
}

func (m *mockEventRepo) Append(ctx context.Context, userID string, event entity.RealtimeEnvelope, retention int) error {
	return nil
}

func (m *mockEventRepo) ListSince(ctx context.Context, userID string, sinceSeq int64, topics []string, retention int) ([]entity.RealtimeEnvelope, int64, error) {
	return nil, sinceSeq, nil
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
	pushAsync := true
	hub := &mockBroadcaster{}
	cfg := config.Config{Queue: config.QueueConfig{Enabled: true, PushAsync: &pushAsync}}
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

func TestRealtimePushUsecaseSyncWhenPushAsyncDisabled(t *testing.T) {
	pushAsync := false
	hub := &mockBroadcaster{}
	cfg := config.Config{Queue: config.QueueConfig{Enabled: true, PushAsync: &pushAsync}}
	enqueuer := &mockEnqueuer{taskID: "task-abc"}
	uc := usecase.NewRealtimePushUsecase(&mockEventRepo{}, cfg, hub, enqueuer)

	out, err := uc.PushToUser(context.Background(), usecase.RealtimePushInput{
		UserID: "u1",
		Title:  "t",
		Body:   "b",
	})
	if err != nil {
		t.Fatalf("push: %v", err)
	}
	if out.Queued || out.Delivered != 1 {
		t.Fatalf("unexpected output: %+v", out)
	}
}
