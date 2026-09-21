package usecase

import (
	"errors"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
)

const (
	maxFailedLogins = 5
	accountLockFor  = 15 * time.Minute
)

var (
	ErrAccountDisabled = errors.New("account disabled")
	ErrAccountLocked   = errors.New("account locked")
)

// loginBlockReason 拒绝停用、未到期锁定，以及 status=2 且没有到期时间的行。
// 锁定已过期时返回 nil，交给成功登录把状态清回 0。
func loginBlockReason(user *entity.User, now time.Time) error {
	if user == nil || user.DeletedAt != nil {
		return ErrAccountNotRegistered
	}
	if user.Status == entity.UserStatusDisabled {
		return ErrAccountDisabled
	}
	if user.LockedUntil != nil && user.LockedUntil.After(now) {
		return ErrAccountLocked
	}
	if user.Status == entity.UserStatusLocked && (user.LockedUntil == nil || user.LockedUntil.After(now)) {
		return ErrAccountLocked
	}
	return nil
}

func applyFailedLogin(user *entity.User, now time.Time) {
	user.FailedLoginCount++
	if user.FailedLoginCount >= maxFailedLogins {
		user.Status = entity.UserStatusLocked
		until := now.Add(accountLockFor)
		user.LockedUntil = &until
	}
}

func applySuccessfulLogin(user *entity.User, now time.Time) {
	user.Status = entity.UserStatusActive
	user.FailedLoginCount = 0
	user.LockedUntil = nil
	at := now
	user.LastLoginAt = &at
}
