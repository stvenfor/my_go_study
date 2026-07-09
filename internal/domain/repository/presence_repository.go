package repository

import "context"

// PresenceRepository 在线状态存储（Redis）。
type PresenceRepository interface {
	// SetOnline 标记用户在线，返回当前在线人数。
	SetOnline(ctx context.Context, userID, device string) (onlineCount int, err error)
	// SetOffline 标记用户离线，返回当前在线人数。
	SetOffline(ctx context.Context, userID string) (onlineCount int, err error)
}
