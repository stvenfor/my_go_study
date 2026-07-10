package config_test

import (
	"testing"
	"time"

	"github.com/stvenfor/my_go_study/pkg/config"
)

func TestAuthConfigSessionTTL(t *testing.T) {
	if got := (config.AuthConfig{SessionTTLHours: 0}).SessionTTL(); got != 0 {
		t.Fatalf("SessionTTL(0) = %v, want 0", got)
	}
	if got := (config.AuthConfig{SessionTTLHours: 24}).SessionTTL(); got != 24*time.Hour {
		t.Fatalf("SessionTTL(24) = %v, want 24h", got)
	}
}
