package redis

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

const (
	realtimeSeqKeyPrefix   = "realtime:seq:"
	realtimeEventsKeyPrefix = "realtime:events:"
)

// RealtimeEventRepository Redis 事件仓储。
type RealtimeEventRepository struct {
	client *redis.Client
}

// NewRealtimeEventRepository 创建事件仓储。
func NewRealtimeEventRepository(client *redis.Client) *RealtimeEventRepository {
	return &RealtimeEventRepository{client: client}
}

func (r *RealtimeEventRepository) NextSeq(ctx context.Context, userID string) (int64, error) {
	return r.client.Incr(ctx, realtimeSeqKeyPrefix+userID).Result()
}

func (r *RealtimeEventRepository) Append(ctx context.Context, userID string, event entity.RealtimeEnvelope, retention int) error {
	if retention <= 0 {
		retention = 200
	}
	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	key := realtimeEventsKeyPrefix + userID
	pipe := r.client.Pipeline()
	pipe.RPush(ctx, key, raw)
	pipe.LTrim(ctx, key, int64(-retention), -1)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *RealtimeEventRepository) ListSince(ctx context.Context, userID string, sinceSeq int64, topics []string, retention int) ([]entity.RealtimeEnvelope, int64, error) {
	key := realtimeEventsKeyPrefix + userID
	rawItems, err := r.client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, sinceSeq, err
	}

	topicSet := make(map[string]struct{}, len(topics))
	for _, t := range topics {
		t = strings.TrimSpace(t)
		if t != "" {
			topicSet[t] = struct{}{}
		}
	}

	var events []entity.RealtimeEnvelope
	var latestSeq int64 = sinceSeq
	for _, raw := range rawItems {
		var event entity.RealtimeEnvelope
		if err := json.Unmarshal([]byte(raw), &event); err != nil {
			continue
		}
		if event.Seq <= sinceSeq {
			continue
		}
		if len(topicSet) > 0 {
			if _, ok := topicSet[event.Topic]; !ok {
				continue
			}
		}
		events = append(events, event)
		if event.Seq > latestSeq {
			latestSeq = event.Seq
		}
	}
	return events, latestSeq, nil
}

var _ repository.RealtimeEventRepository = (*RealtimeEventRepository)(nil)
