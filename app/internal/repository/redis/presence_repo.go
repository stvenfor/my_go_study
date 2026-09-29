// =============================================================================
// 文件：presence_repo.go
// 作用：用 Redis Set 维护「当前在线用户集合」
//
// Redis 结构：
//   presence:online        → Set{userId1, userId2, ...}
//   presence:user:{userId} → 设备名，TTL 5 分钟（超时未心跳视为离线可扩展）
// =============================================================================
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

type PresenceRepository struct {
	client *redis.Client
}

func NewPresenceRepository(client *redis.Client) *PresenceRepository {
	return &PresenceRepository{client: client}
}

func (r *PresenceRepository) SetOnline(ctx context.Context, userID, device string) (int, error) {
	userKey := fmt.Sprintf(presenceUserKeyFmt, userID)
	// Pipeline：多条命令一次网络往返，减少延迟
	pipe := r.client.Pipeline()
	pipe.Set(ctx, userKey, device, presenceUserTTL)
	pipe.SAdd(ctx, presenceOnlineSetKey, userID)
	pipe.SCard(ctx, presenceOnlineSetKey) // 返回 Set 大小 = 在线人数
	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return int(cmds[2].(*redis.IntCmd).Val()), nil
}

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
