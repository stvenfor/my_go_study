package usecase

import (
	"testing"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

func TestApplyFailedLoginLocksOnFifth(t *testing.T) {
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	user := &entity.User{Status: entity.UserStatusActive}
	for i := 0; i < 4; i++ {
		applyFailedLogin(user, now)
	}
	if user.Status != entity.UserStatusActive || user.FailedLoginCount != 4 {
		t.Fatalf("before fifth: status=%d count=%d", user.Status, user.FailedLoginCount)
	}
	applyFailedLogin(user, now)
	if user.Status != entity.UserStatusLocked || user.FailedLoginCount != 5 {
		t.Fatalf("fifth: status=%d count=%d", user.Status, user.FailedLoginCount)
	}
	if user.LockedUntil == nil || !user.LockedUntil.Equal(now.Add(15*time.Minute)) {
		t.Fatalf("locked_until=%v", user.LockedUntil)
	}
	if err := loginBlockReason(user, now.Add(time.Minute)); err != ErrAccountLocked {
		t.Fatalf("during lock: %v", err)
	}
}

func TestExpiredLockAllowsLogin(t *testing.T) {
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	until := now.Add(-time.Minute)
	user := &entity.User{
		Status:           entity.UserStatusLocked,
		FailedLoginCount: 5,
		LockedUntil:      &until,
	}
	if err := loginBlockReason(user, now); err != nil {
		t.Fatalf("expired lock blocked: %v", err)
	}
	applySuccessfulLogin(user, now)
	if user.Status != entity.UserStatusActive || user.FailedLoginCount != 0 || user.LockedUntil != nil {
		t.Fatalf("cleared: status=%d count=%d until=%v", user.Status, user.FailedLoginCount, user.LockedUntil)
	}
	if user.LastLoginAt == nil || !user.LastLoginAt.Equal(now) {
		t.Fatalf("last_login_at=%v", user.LastLoginAt)
	}
}

func TestDisabledAndDeletedLogin(t *testing.T) {
	now := time.Now().UTC()
	disabled := &entity.User{Status: entity.UserStatusDisabled}
	if err := loginBlockReason(disabled, now); err != ErrAccountDisabled {
		t.Fatalf("disabled: %v", err)
	}
	deletedAt := now
	deleted := &entity.User{Status: entity.UserStatusActive, DeletedAt: &deletedAt}
	if err := loginBlockReason(deleted, now); err != ErrAccountNotRegistered {
		t.Fatalf("deleted: %v", err)
	}
}

func TestNewLocalUserIgnoresClientID(t *testing.T) {
	const clientID = "client-supplied"
	user := newLocalUser("a@b.com", "ann", "", "hash")
	if user.UserID == "" || user.UserID == clientID {
		t.Fatalf("user_id=%q", user.UserID)
	}
	if user.Email != "a@b.com" || user.UserName != "ann" || user.Status != entity.UserStatusActive || user.FailedLoginCount != 0 || user.DeletedAt != nil {
		t.Fatalf("user=%+v", user)
	}
}
