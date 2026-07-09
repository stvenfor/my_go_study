// =============================================================================
// 文件：ws_ticket_repo.go
// 层级：Repository —— 只负责 Redis 读写，不含业务判断
//
// 【为什么 Consume 用 GetDel 而不是 Get + Del？】
//   GetDel 是原子操作，避免两个 WS 同时用同一 ticket auth 的竞态。
// =============================================================================
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

const wsTicketKeyPrefix = "ws:ticket:"

type WSTicketRepository struct {
	client *redis.Client
}

func NewWSTicketRepository(client *redis.Client) *WSTicketRepository {
	return &WSTicketRepository{client: client}
}

func (r *WSTicketRepository) Save(ctx context.Context, ticket string, data repository.WSTicket, ttl time.Duration) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	// SET key value EX ttl
	return r.client.Set(ctx, wsTicketKeyPrefix+ticket, raw, ttl).Err()
}

func (r *WSTicketRepository) Consume(ctx context.Context, ticket string) (*repository.WSTicket, error) {
	key := wsTicketKeyPrefix + ticket
	raw, err := r.client.GetDel(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, fmt.Errorf("ticket 无效或已过期")
	}
	if err != nil {
		return nil, err
	}
	var data repository.WSTicket
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	return &data, nil
}
