package config

import "testing"

func TestAuthConfigIsSessionExempt(t *testing.T) {
	cfg := AuthConfig{
		SessionWhitelistUserIDs: []string{"user-uuid-1"},
		SessionWhitelistEmails:  []string{"Internal@Example.com"},
	}

	if !cfg.IsSessionExempt("user-uuid-1", "") {
		t.Fatal("expected user id whitelist match")
	}
	if !cfg.IsSessionExempt("", "internal@example.com") {
		t.Fatal("expected email whitelist match case-insensitive")
	}
	if cfg.IsSessionExempt("other-user", "other@example.com") {
		t.Fatal("expected non-whitelist user to be rejected")
	}
}
