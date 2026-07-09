package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	presenceOnlineSetKey = "presence:online"
	presenceUserKeyFmt   = "presence:user:%s"
	presenceUserTTL      = 5 * time.Minute
)

// PresenceRepository Redis 在线状态仓储。
type PresenceRepository struct {
	client *redis.Client
}

// NewPresenceRepository 创建 Presence 仓储。
func NewPresenceRepository(client *redis.Client) *PresenceRepository {
	return &PresenceRepository{client: client}
}

// SetOnline 标记用户在线。
func (r *PresenceRepository) SetOnline(ctx context.Context, userID, device string) (int, error) {
	userKey := fmt.Sprintf(presenceUserKeyFmt, userID)
	pipe := r.client.Pipeline()
	pipe.Set(ctx, userKey, device, presenceUserTTL)
	pipe.SAdd(ctx, presenceOnlineSetKey, userID)
	pipe.SCard(ctx, presenceOnlineSetKey)
	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return int(cmds[2].(*redis.IntCmd).Val()), nil
}

// SetOffline 标记用户离线。
func (r *PresenceRepository) SetOffline(ctx context.Context, userID string) (int, error) {
	userKey := fmt.Sprintf(presenceUserKeyFmt, userID)
	pipe := r.client.Pipeline()
	pipe.Del(ctx, userKey)
	pipe.SRem(ctx, presenceOnlineSetKey, userID)
	pipe.SCard(ctx, presenceOnlineSetKey)
	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return int(cmds[2].(*redis.IntCmd).Val()), nil
}
