package config

import (
	"testing"
)

func TestRealtimeWSURLUsesPublicHost(t *testing.T) {
	cfg := RealtimeConfig{
		WsPath:       "/realtime/v1/connect",
		PublicWSHost: "192.168.1.23",
	}
	got := cfg.WSURL(8080)
	want := "ws://192.168.1.23:8080/realtime/v1/connect"
	if got != want {
		t.Fatalf("WSURL = %q, want %q", got, want)
	}
}

func TestRealtimeWSURLDefaultsLocalhost(t *testing.T) {
	cfg := RealtimeConfig{WsPath: "/realtime/v1/connect"}
	got := cfg.WSURL(8080)
	want := "ws://127.0.0.1:8080/realtime/v1/connect"
	if got != want {
		t.Fatalf("WSURL = %q, want %q", got, want)
	}
}
