package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	redisrepo "github.com/stvenfor/my_go_study/internal/repository/redis"
	"github.com/stvenfor/my_go_study/internal/usecase"
	"github.com/stvenfor/my_go_study/pkg/config"
)

func TestRealtimeTicketIssueAndConsume(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("redis not available:", err)
	}
	defer client.Close()

	cfg := config.Config{
		Server: config.ServerConfig{Port: 8080},
		Realtime: config.RealtimeConfig{
			TicketTTLSeconds: 120,
			WsPath:           "/realtime/v1/connect",
		},
	}
	repo := redisrepo.NewWSTicketRepository(client)
	uc := usecase.NewRealtimeTicketUsecase(repo, cfg)

	out, err := uc.Issue(ctx, usecase.RealtimeTicketInput{
		UserID:   "test-user",
		Platform: "test",
	})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if out.Ticket == "" || out.WSURL == "" {
		t.Fatalf("unexpected output: %+v", out)
	}

	data, err := uc.Consume(ctx, out.Ticket)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if data.UserID != "test-user" {
		t.Fatalf("userId=%s", data.UserID)
	}

	_, err = uc.Consume(ctx, out.Ticket)
	if err == nil {
		t.Fatal("expected ticket one-time consume error")
	}
}

func TestRealtimeSyncAppendAndList(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("redis not available:", err)
	}
	defer client.Close()

	userID := "sync-user-" + time.Now().Format("150405")
	cfg := config.Config{Realtime: config.RealtimeConfig{EventRetention: 50}}
	eventRepo := redisrepo.NewRealtimeEventRepository(client)
	syncUC := usecase.NewRealtimeSyncUsecase(eventRepo, cfg)

	seq, err := eventRepo.NextSeq(ctx, userID)
	if err != nil {
		t.Fatalf("next seq: %v", err)
	}
	if err := eventRepo.Append(ctx, userID, entity.RealtimeEnvelope{
		ID: "e1", Type: "event", Topic: entity.TopicSysNotify,
		TS: time.Now().UnixMilli(), Seq: seq,
		Payload: map[string]any{"name": "sys.notify.show", "title": "t"},
	}, cfg.Realtime.EventRetention); err != nil {
		t.Fatalf("append: %v", err)
	}

	out, err := syncUC.Sync(ctx, usecase.RealtimeSyncInput{
		UserID:   userID,
		SinceSeq: 0,
		Topics:   []string{entity.TopicSysNotify},
	})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if len(out.Events) == 0 {
		t.Fatal("expected events")
	}
}
