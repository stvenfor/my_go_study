package usecase_test

import (
	"context"
	"testing"
	"time"

	domainrepo "github.com/stvenfor/my_go_study/internal/domain/repository"
	"github.com/stvenfor/my_go_study/internal/usecase"
	"github.com/stvenfor/my_go_study/pkg/config"
)

type mockSessionRepo struct {
	sessions map[string]domainrepo.DeviceSession
	lastTTL  time.Duration
}

func (m *mockSessionRepo) Get(_ context.Context, userID string) (*domainrepo.DeviceSession, error) {
	session, ok := m.sessions[userID]
	if !ok {
		return nil, nil
	}
	copy := session
	return &copy, nil
}

func (m *mockSessionRepo) Save(_ context.Context, userID string, session domainrepo.DeviceSession, ttl time.Duration) error {
	if m.sessions == nil {
		m.sessions = make(map[string]domainrepo.DeviceSession)
	}
	m.sessions[userID] = session
	m.lastTTL = ttl
	return nil
}

func (m *mockSessionRepo) Delete(_ context.Context, userID string) error {
	delete(m.sessions, userID)
	return nil
}

func (m *mockSessionRepo) ListActiveUserIDs(_ context.Context) ([]string, error) {
	ids := make([]string, 0, len(m.sessions))
	for userID := range m.sessions {
		ids = append(ids, userID)
	}
	return ids, nil
}

func TestDeviceSessionIssueAndValidate(t *testing.T) {
	repo := &mockSessionRepo{}
	uc := usecase.NewDeviceSessionUsecase(repo, config.AuthConfig{SessionTTLHours: 24})

	sessionA, err := uc.IssueOnLogin(context.Background(), usecase.IssueSessionInput{
		UserID:   "user-1",
		DeviceID: "device-a",
		Platform: "ios",
	})
	if err != nil {
		t.Fatalf("issue A: %v", err)
	}
	if sessionA == "" {
		t.Fatal("expected session id")
	}
	if err := uc.Validate(context.Background(), "user-1", "", sessionA, "device-a"); err != nil {
		t.Fatalf("validate A: %v", err)
	}

	sessionB, err := uc.IssueOnLogin(context.Background(), usecase.IssueSessionInput{
		UserID:   "user-1",
		DeviceID: "device-b",
		Platform: "android",
	})
	if err != nil {
		t.Fatalf("issue B: %v", err)
	}
	if sessionB == sessionA {
		t.Fatal("expected new session id after re-login")
	}
	if err := uc.Validate(context.Background(), "user-1", "", sessionA, "device-a"); err != usecase.ErrSessionReplaced {
		t.Fatalf("expected replaced error, got %v", err)
	}
	if err := uc.Validate(context.Background(), "user-1", "", sessionB, "device-b"); err != nil {
		t.Fatalf("validate B: %v", err)
	}
}

func TestDeviceSessionSameDeviceStaleSessionID(t *testing.T) {
	repo := &mockSessionRepo{}
	uc := usecase.NewDeviceSessionUsecase(repo, config.AuthConfig{})

	sessionID, err := uc.IssueOnLogin(context.Background(), usecase.IssueSessionInput{
		UserID:   "user-1",
		DeviceID: "device-a",
		Platform: "ios",
	})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if err := uc.Validate(context.Background(), "user-1", "", "stale-session", "device-a"); err != usecase.ErrSessionInvalid {
		t.Fatalf("expected invalid session on same device, got %v", err)
	}
	if err := uc.Validate(context.Background(), "user-1", "", sessionID, "device-a"); err != nil {
		t.Fatalf("validate current session: %v", err)
	}
}

func TestDeviceSessionRenewOnRefreshSameDevice(t *testing.T) {
	repo := &mockSessionRepo{}
	uc := usecase.NewDeviceSessionUsecase(repo, config.AuthConfig{})

	oldSession, err := uc.IssueOnLogin(context.Background(), usecase.IssueSessionInput{
		UserID:   "user-1",
		DeviceID: "device-a",
		Platform: "ios",
	})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	newSession, err := uc.RenewOnRefresh(context.Background(), usecase.RenewSessionInput{
		UserID:    "user-1",
		DeviceID:  "device-a",
		Platform:  "ios",
		SessionID: oldSession,
	})
	if err != nil {
		t.Fatalf("renew: %v", err)
	}
	if newSession == "" || newSession == oldSession {
		t.Fatalf("expected new session id, got %q", newSession)
	}
	if err := uc.Validate(context.Background(), "user-1", "", newSession, "device-a"); err != nil {
		t.Fatalf("validate renewed session: %v", err)
	}
}

func TestDeviceSessionRenewOnRefreshOtherDevice(t *testing.T) {
	repo := &mockSessionRepo{}
	uc := usecase.NewDeviceSessionUsecase(repo, config.AuthConfig{})

	_, err := uc.IssueOnLogin(context.Background(), usecase.IssueSessionInput{
		UserID:   "user-1",
		DeviceID: "device-a",
		Platform: "ios",
	})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	_, err = uc.RenewOnRefresh(context.Background(), usecase.RenewSessionInput{
		UserID:   "user-1",
		DeviceID: "device-b",
		Platform: "android",
	})
	if err != usecase.ErrSessionReplaced {
		t.Fatalf("expected replaced error, got %v", err)
	}
}

func TestDeviceSessionRenewOnRefreshDeviceMigration(t *testing.T) {
	repo := &mockSessionRepo{}
	uc := usecase.NewDeviceSessionUsecase(repo, config.AuthConfig{})

	oldSession, err := uc.IssueOnLogin(context.Background(), usecase.IssueSessionInput{
		UserID:   "user-1",
		DeviceID: "ios",
		Platform: "ios",
	})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	newSession, err := uc.RenewOnRefresh(context.Background(), usecase.RenewSessionInput{
		UserID:    "user-1",
		DeviceID:  "stable-device-id",
		Platform:  "ios",
		SessionID: oldSession,
	})
	if err != nil {
		t.Fatalf("renew with migration: %v", err)
	}
	if err := uc.Validate(context.Background(), "user-1", "", newSession, "stable-device-id"); err != nil {
		t.Fatalf("validate migrated device: %v", err)
	}
}

func TestDeviceSessionWhitelistExempt(t *testing.T) {
	repo := &mockSessionRepo{}
	uc := usecase.NewDeviceSessionUsecase(repo, config.AuthConfig{
		SessionWhitelistUserIDs: []string{"user-wl"},
	})

	sessionA, err := uc.IssueOnLogin(context.Background(), usecase.IssueSessionInput{
		UserID:   "user-wl",
		Email:    "internal@example.com",
		DeviceID: "device-a",
		Platform: "ios",
	})
	if err != nil {
		t.Fatalf("issue A: %v", err)
	}
	sessionB, err := uc.IssueOnLogin(context.Background(), usecase.IssueSessionInput{
		UserID:   "user-wl",
		Email:    "internal@example.com",
		DeviceID: "device-b",
		Platform: "android",
	})
	if err != nil {
		t.Fatalf("issue B: %v", err)
	}
	if len(repo.sessions) != 0 {
		t.Fatalf("whitelist login should not write redis session, got %d entries", len(repo.sessions))
	}
	if err := uc.Validate(context.Background(), "user-wl", "internal@example.com", sessionA, "device-a"); err != nil {
		t.Fatalf("validate A: %v", err)
	}
	if err := uc.Validate(context.Background(), "user-wl", "internal@example.com", sessionB, "device-b"); err != nil {
		t.Fatalf("validate B: %v", err)
	}
	if err := uc.Validate(context.Background(), "user-wl", "internal@example.com", "stale-session", "device-a"); err != nil {
		t.Fatalf("whitelist should ignore stale session id: %v", err)
	}
}

func TestDeviceSessionInvalidPlatform(t *testing.T) {
	uc := usecase.NewDeviceSessionUsecase(&mockSessionRepo{}, config.AuthConfig{})
	_, err := uc.IssueOnLogin(context.Background(), usecase.IssueSessionInput{
		UserID:   "user-1",
		DeviceID: "device-a",
		Platform: "web",
	})
	if err != usecase.ErrInvalidPlatform {
		t.Fatalf("expected invalid platform, got %v", err)
	}
}

func TestDeviceSessionMissingHeaders(t *testing.T) {
	repo := &mockSessionRepo{}
	uc := usecase.NewDeviceSessionUsecase(repo, config.AuthConfig{})
	sessionID, err := uc.IssueOnLogin(context.Background(), usecase.IssueSessionInput{
		UserID:   "user-1",
		DeviceID: "device-a",
		Platform: "ios",
	})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if err := uc.Validate(context.Background(), "user-1", "", "", "device-a"); err != usecase.ErrSessionInvalid {
		t.Fatalf("expected invalid session, got %v", err)
	}
	if err := uc.Validate(context.Background(), "user-1", "", sessionID, ""); err != usecase.ErrSessionInvalid {
		t.Fatalf("expected invalid session, got %v", err)
	}
}

func TestDeviceSessionRevokeOnLogout(t *testing.T) {
	repo := &mockSessionRepo{}
	uc := usecase.NewDeviceSessionUsecase(repo, config.AuthConfig{SessionTTLHours: 0})

	sessionID, err := uc.IssueOnLogin(context.Background(), usecase.IssueSessionInput{
		UserID:   "user-1",
		DeviceID: "device-a",
		Platform: "ios",
	})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if repo.lastTTL != 0 {
		t.Fatalf("expected ttl=0, got %v", repo.lastTTL)
	}
	if err := uc.RevokeOnLogout(context.Background(), "user-1", "", sessionID, "device-a"); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, ok := repo.sessions["user-1"]; ok {
		t.Fatal("session should be deleted after logout")
	}
	if err := uc.Validate(context.Background(), "user-1", "", sessionID, "device-a"); err != usecase.ErrSessionInvalid {
		t.Fatalf("expected invalid session after logout, got %v", err)
	}
}
