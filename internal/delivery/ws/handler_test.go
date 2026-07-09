package ws_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

func TestWSAuthFlow(t *testing.T) {
	wsURL := os.Getenv("TEST_WS_URL")
	ticket := os.Getenv("TEST_WS_TICKET")
	connID := os.Getenv("TEST_WS_CONN_ID")
	if wsURL == "" || ticket == "" {
		t.Skip("skip integration: set TEST_WS_URL and TEST_WS_TICKET")
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	defer conn.Close()

	raw, err := json.Marshal(entity.RealtimeEnvelope{
		ID:   "auth_1",
		Type: "auth",
		TS:   time.Now().UnixMilli(),
		Payload: map[string]any{
			"ticket":   ticket,
			"connId":   connID,
			"platform": "test",
		},
	})
	if err != nil {
		t.Fatalf("marshal auth: %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, raw); err != nil {
		t.Fatalf("write auth: %v", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("deadline: %v", err)
	}
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read auth_ok: %v", err)
	}
	var resp entity.RealtimeEnvelope
	if err := json.Unmarshal(msg, &resp); err != nil {
		t.Fatalf("parse auth_ok: %v", err)
	}
	if resp.Type != "auth_ok" {
		t.Fatalf("expected auth_ok, got %s", resp.Type)
	}
}
