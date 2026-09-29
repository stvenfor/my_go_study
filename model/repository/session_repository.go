// session_repository.go 定义单设备登录会话仓储接口。
package repository

import (
	"context"
	"time"
)

// DeviceSession 用户当前唯一 mobile 会话。
type DeviceSession struct {
	SessionID string `json:"session_id"`
	DeviceID  string `json:"device_id"`
	Platform  string `json:"platform"`
	CreatedAt int64  `json:"created_at"`
}

// SessionRepository 读写 Redis 中的用户活跃会话。
type SessionRepository interface {
	Get(ctx context.Context, userID string) (*DeviceSession, error)
	Save(ctx context.Context, userID string, session DeviceSession, ttl time.Duration) error
	Delete(ctx context.Context, userID string) error
	ListActiveUserIDs(ctx context.Context) ([]string, error)
}
