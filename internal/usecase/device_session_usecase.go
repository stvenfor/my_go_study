// device_session_usecase.go 单设备登录：登录签发 session，业务 API 校验 session。
package usecase

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	domainrepo "github.com/stvenfor/my_go_study/internal/domain/repository"
	"github.com/stvenfor/my_go_study/pkg/config"
)

var (
	ErrSessionInvalid   = errors.New("session invalid")
	ErrSessionReplaced  = errors.New("session replaced by other device")
	ErrInvalidPlatform  = errors.New("invalid platform")
	ErrInvalidDeviceID  = errors.New("invalid device id")
)

const (
	MsgSessionInvalid  = "会话无效，请重新登录"
	MsgSessionReplaced = "账号已在其他设备登录，请重新登录"
)

type DeviceSessionUsecase struct {
	sessions domainrepo.SessionRepository
	cfg      config.AuthConfig
}

func NewDeviceSessionUsecase(sessions domainrepo.SessionRepository, cfg config.AuthConfig) *DeviceSessionUsecase {
	return &DeviceSessionUsecase{sessions: sessions, cfg: cfg}
}

type IssueSessionInput struct {
	UserID   string
	Email    string
	DeviceID string
	Platform string
}

// IsExempt 账号是否豁免单设备 session 限制。
func (u *DeviceSessionUsecase) IsExempt(userID, email string) bool {
	return u.cfg.IsSessionExempt(userID, email)
}

// IssueOnLogin 登录成功后签发新 session；普通用户覆盖旧设备，白名单用户仅返回 session_id。
func (u *DeviceSessionUsecase) IssueOnLogin(ctx context.Context, input IssueSessionInput) (string, error) {
	deviceID := strings.TrimSpace(input.DeviceID)
	platform := strings.ToLower(strings.TrimSpace(input.Platform))
	if deviceID == "" {
		return "", ErrInvalidDeviceID
	}
	if platform != "android" && platform != "ios" {
		return "", ErrInvalidPlatform
	}
	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return "", ErrInvalidParams
	}

	sessionID := uuid.NewString()
	if u.cfg.IsSessionExempt(userID, input.Email) {
		log.Printf("[auth] session exempt login user=%s email=%s device=%s platform=%s",
			userID, strings.TrimSpace(input.Email), deviceID, platform)
		return sessionID, nil
	}

	session := domainrepo.DeviceSession{
		SessionID: sessionID,
		DeviceID:  deviceID,
		Platform:  platform,
		CreatedAt: time.Now().Unix(),
	}
	if err := u.sessions.Save(ctx, userID, session, u.cfg.SessionTTL()); err != nil {
		return "", err
	}
	return sessionID, nil
}

// RevokeOnLogout 主动退出时删除当前活跃 session。
func (u *DeviceSessionUsecase) RevokeOnLogout(ctx context.Context, userID, email, sessionID, deviceID string) error {
	userID = strings.TrimSpace(userID)
	if u.cfg.IsSessionExempt(userID, email) {
		return nil
	}
	if err := u.Validate(ctx, userID, email, sessionID, deviceID); err != nil {
		return err
	}
	return u.sessions.Delete(ctx, userID)
}

// Validate 校验请求携带的 session 是否为当前活跃会话；白名单用户直接放行。
func (u *DeviceSessionUsecase) Validate(ctx context.Context, userID, email, sessionID, deviceID string) error {
	userID = strings.TrimSpace(userID)
	if u.cfg.IsSessionExempt(userID, email) {
		return nil
	}

	sessionID = strings.TrimSpace(sessionID)
	deviceID = strings.TrimSpace(deviceID)
	if sessionID == "" || deviceID == "" {
		return ErrSessionInvalid
	}

	stored, err := u.sessions.Get(ctx, userID)
	if err != nil {
		return err
	}
	if stored == nil {
		return ErrSessionInvalid
	}

	// 同 session 但 device_id 变更：允许客户端升级稳定 device_id（如 iOS 模拟器占位 id）。
	if stored.SessionID == sessionID && stored.DeviceID != deviceID {
		updated := *stored
		updated.DeviceID = deviceID
		if err := u.sessions.Save(ctx, userID, updated, u.cfg.SessionTTL()); err != nil {
			return err
		}
		return nil
	}

	if stored.DeviceID != deviceID {
		return ErrSessionReplaced
	}
	if stored.SessionID != sessionID {
		// 同设备 session_id 过期，客户端应走 refresh 续期，不应视为其他设备互踢。
		return ErrSessionInvalid
	}
	return nil
}

// RenewSessionInput refresh 时续期单设备 session 的入参。
type RenewSessionInput struct {
	UserID    string
	Email     string
	DeviceID  string
	Platform  string
	SessionID string
}

// RenewOnRefresh 同设备 refresh 时续期 session 并返回最新 session_id；其他设备仍拒绝。
func (u *DeviceSessionUsecase) RenewOnRefresh(ctx context.Context, input RenewSessionInput) (string, error) {
	deviceID := strings.TrimSpace(input.DeviceID)
	platform := strings.ToLower(strings.TrimSpace(input.Platform))
	userID := strings.TrimSpace(input.UserID)
	clientSessionID := strings.TrimSpace(input.SessionID)

	if userID == "" {
		return "", ErrInvalidParams
	}
	if deviceID == "" {
		return "", ErrInvalidDeviceID
	}
	if platform != "" && platform != "android" && platform != "ios" {
		return "", ErrInvalidPlatform
	}
	if platform == "" {
		platform = "ios"
	}

	newSessionID := uuid.NewString()
	if u.cfg.IsSessionExempt(userID, input.Email) {
		if clientSessionID != "" {
			return clientSessionID, nil
		}
		return newSessionID, nil
	}

	stored, err := u.sessions.Get(ctx, userID)
	if err != nil {
		return "", err
	}
	if stored == nil {
		session := domainrepo.DeviceSession{
			SessionID: newSessionID,
			DeviceID:  deviceID,
			Platform:  platform,
			CreatedAt: time.Now().Unix(),
		}
		if err := u.sessions.Save(ctx, userID, session, u.cfg.SessionTTL()); err != nil {
			return "", err
		}
		return newSessionID, nil
	}

	if stored.DeviceID == deviceID {
		session := domainrepo.DeviceSession{
			SessionID: newSessionID,
			DeviceID:  deviceID,
			Platform:  platform,
			CreatedAt: time.Now().Unix(),
		}
		if err := u.sessions.Save(ctx, userID, session, u.cfg.SessionTTL()); err != nil {
			return "", err
		}
		return newSessionID, nil
	}

	// device_id 变更但客户端仍持有当前 session_id：视为同设备 id 迁移，续期并更新 device。
	if clientSessionID != "" && stored.SessionID == clientSessionID {
		session := domainrepo.DeviceSession{
			SessionID: newSessionID,
			DeviceID:  deviceID,
			Platform:  platform,
			CreatedAt: time.Now().Unix(),
		}
		if err := u.sessions.Save(ctx, userID, session, u.cfg.SessionTTL()); err != nil {
			return "", err
		}
		return newSessionID, nil
	}

	return "", ErrSessionReplaced
}
