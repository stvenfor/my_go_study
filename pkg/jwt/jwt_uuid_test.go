package jwt

import (
	"testing"

	"github.com/stvenfor/my_go_study/pkg/config"
)

func TestGenerateAndParseUUID(t *testing.T) {
	m := NewManager(config.JWTConfig{Secret: "test-secret", ExpireHours: 1})
	token, err := m.GenerateUUID("550e8400-e29b-41d4-a716-446655440000", "a@b.com", "alice")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := m.ParseUUID(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("sub=%q", claims.Subject)
	}
	if claims.Email != "a@b.com" {
		t.Fatalf("email=%q", claims.Email)
	}
}
