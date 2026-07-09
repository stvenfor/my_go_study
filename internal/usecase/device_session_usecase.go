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
	if stored.SessionID != sessionID || stored.DeviceID != deviceID {
		return ErrSessionReplaced
	}
	return nil
}
