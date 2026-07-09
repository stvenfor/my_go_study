// session_repo.go Redis 实现单设备登录会话存储。
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

const sessionKeyPrefix = "auth:session:"

type SessionRepository struct {
	client *redis.Client
}

func NewSessionRepository(client *redis.Client) *SessionRepository {
	return &SessionRepository{client: client}
}

func (r *SessionRepository) Get(ctx context.Context, userID string) (*repository.DeviceSession, error) {
	raw, err := r.client.Get(ctx, sessionKeyPrefix+userID).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var session repository.DeviceSession
	if err := json.Unmarshal(raw, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *SessionRepository) Save(ctx context.Context, userID string, session repository.DeviceSession, ttl time.Duration) error {
	raw, err := json.Marshal(session)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		return fmt.Errorf("session ttl 必须大于 0")
	}
	return r.client.Set(ctx, sessionKeyPrefix+userID, raw, ttl).Err()
}

func (r *SessionRepository) Delete(ctx context.Context, userID string) error {
	return r.client.Del(ctx, sessionKeyPrefix+userID).Err()
}

// ListActiveUserIDs 返回当前有登录 Session 的用户 ID 列表。
func (r *SessionRepository) ListActiveUserIDs(ctx context.Context) ([]string, error) {
	var (
		cursor  uint64
		userIDs []string
	)
	for {
		keys, next, err := r.client.Scan(ctx, cursor, sessionKeyPrefix+"*", 100).Result()
		if err != nil {
			return nil, err
		}
		for _, key := range keys {
			userID := strings.TrimPrefix(key, sessionKeyPrefix)
			if userID != "" {
				userIDs = append(userIDs, userID)
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return userIDs, nil
}
