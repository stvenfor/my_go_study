package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stvenfor/my_go_study/pkg/config"
)

func TestValidateAccessToken_DirectUserObject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"id":    "user-123",
			"email": "test@example.com",
			"phone": "",
		})
	}))
	defer server.Close()

	cfg := config.SupabaseConfig{
		URL:     server.URL,
		AnonKey: "anon-key",
	}
	user, err := ValidateAccessToken(context.Background(), cfg, "token-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != "user-123" || user.Email != "test@example.com" {
		t.Fatalf("unexpected user: %+v", user)
	}
}

func TestValidateAccessToken_WrappedUserObject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"user": map[string]string{
				"id":    "user-456",
				"email": "wrap@example.com",
			},
		})
	}))
	defer server.Close()

	cfg := config.SupabaseConfig{URL: server.URL, AnonKey: "anon-key"}
	user, err := ValidateAccessToken(context.Background(), cfg, "token-xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != "user-456" {
		t.Fatalf("unexpected user: %+v", user)
	}
}
