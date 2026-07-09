package ws_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	redisrepo "github.com/stvenfor/my_go_study/internal/repository/redis"
	"github.com/stvenfor/my_go_study/internal/usecase"
	"github.com/stvenfor/my_go_study/pkg/config"
)

func TestWSAuthFlowLive(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("redis not available:", err)
	}
	defer client.Close()

	cfg := config.Config{
		Server: config.ServerConfig{Port: 8080},
		Realtime: config.RealtimeConfig{
			WsPath:           "/realtime/v1/connect",
			TicketTTLSeconds: 120,
			PublicWSHost:     "127.0.0.1",
		},
	}
	repo := redisrepo.NewWSTicketRepository(client)
	ticketUC := usecase.NewRealtimeTicketUsecase(repo, cfg)
	out, err := ticketUC.Issue(ctx, usecase.RealtimeTicketInput{
		UserID:   "live-ws-user",
		Platform: "test",
		ConnID:   "conn_live",
	})
	if err != nil {
		t.Fatalf("issue ticket: %v", err)
	}

	conn, resp, err := websocket.DefaultDialer.Dial(out.WSURL, nil)
	if err != nil {
		t.Skip("ws server not running:", err, resp)
	}
	defer conn.Close()

	raw, _ := json.Marshal(entity.RealtimeEnvelope{
		ID:   "auth_live",
		Type: "auth",
		TS:   time.Now().UnixMilli(),
		Payload: map[string]any{
			"ticket":   out.Ticket,
			"connId":   out.ConnID,
			"platform": "test",
		},
	})
	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read auth_ok: %v", err)
	}
	var envelope entity.RealtimeEnvelope
	if err := json.Unmarshal(msg, &envelope); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if envelope.Type != "auth_ok" {
		t.Fatalf("expected auth_ok got %s", envelope.Type)
	}
}
