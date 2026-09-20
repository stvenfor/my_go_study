package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

const conversationKeyPrefix = "sse:conv:"

// ConversationRepository Redis 停留会话仓储。
type ConversationRepository struct {
	client *redis.Client
}

// NewConversationRepository 创建会话仓储。
func NewConversationRepository(client *redis.Client) *ConversationRepository {
	return &ConversationRepository{client: client}
}

func conversationKey(userID, conversationID string) string {
	return fmt.Sprintf("%s%s:%s", conversationKeyPrefix, userID, conversationID)
}

// Create 创建空会话。
func (r *ConversationRepository) Create(ctx context.Context, userID string, ttl time.Duration) (string, error) {
	id := "conv_" + uuid.NewString()
	raw, err := json.Marshal([]entity.ChatTurn{})
	if err != nil {
		return "", err
	}
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	if err := r.client.Set(ctx, conversationKey(userID, id), raw, ttl).Err(); err != nil {
		return "", err
	}
	return id, nil
}

// Get 读取会话。
func (r *ConversationRepository) Get(ctx context.Context, userID, conversationID string) ([]entity.ChatTurn, error) {
	raw, err := r.client.Get(ctx, conversationKey(userID, conversationID)).Bytes()
	if err == redis.Nil {
		return nil, repository.ErrConversationNotFound
	}
	if err != nil {
		return nil, err
	}
	var turns []entity.ChatTurn
	if err := json.Unmarshal(raw, &turns); err != nil {
		return nil, err
	}
	if turns == nil {
		turns = []entity.ChatTurn{}
	}
	return turns, nil
}

// SaveTurns 覆写并续期。
func (r *ConversationRepository) SaveTurns(ctx context.Context, userID, conversationID string, turns []entity.ChatTurn, ttl time.Duration) error {
	if turns == nil {
		turns = []entity.ChatTurn{}
	}
	raw, err := json.Marshal(turns)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	key := conversationKey(userID, conversationID)
	return r.client.Set(ctx, key, raw, ttl).Err()
}

// Touch 滑动续期。
func (r *ConversationRepository) Touch(ctx context.Context, userID, conversationID string, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	key := conversationKey(userID, conversationID)
	ok, err := r.client.Expire(ctx, key, ttl).Result()
	if err != nil {
		return err
	}
	if !ok {
		return repository.ErrConversationNotFound
	}
	return nil
}

var _ repository.ConversationRepository = (*ConversationRepository)(nil)
