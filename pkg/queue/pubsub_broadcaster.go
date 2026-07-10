// pubsub_broadcaster.go Redis Pub/Sub 多实例 WS 广播。
package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"go.uber.org/zap"
)

// FanoutMessage Pub/Sub 广播消息体。
type FanoutMessage struct {
	UserID   string                  `json:"userId"`
	Topic    string                  `json:"topic"`
	Envelope entity.RealtimeEnvelope `json:"envelope"`
}

// FanoutPublisher 通过 Redis Pub/Sub 向所有 BFF 实例广播。
type FanoutPublisher struct {
	redis   *redis.Client
	channel string
}

// NewFanoutPublisher 创建 Pub/Sub 发布器。
func NewFanoutPublisher(redisClient *redis.Client, channel string) *FanoutPublisher {
	return &FanoutPublisher{redis: redisClient, channel: channel}
}

// BroadcastToUser 发布到 Pub/Sub 频道，由各实例 Subscriber 投递本地 Hub。
func (p *FanoutPublisher) BroadcastToUser(userID, topic string, envelope entity.RealtimeEnvelope) int {
	msg := FanoutMessage{UserID: userID, Topic: topic, Envelope: envelope}
	data, err := json.Marshal(msg)
	if err != nil {
		return 0
	}
	if err := p.redis.Publish(context.Background(), p.channel, data).Err(); err != nil {
		return 0
	}
	return 0
}

// FanoutSubscriber 订阅 Pub/Sub 并在本地 Hub 广播。
type FanoutSubscriber struct {
	redis       *redis.Client
	channel     string
	broadcaster func(userID, topic string, envelope entity.RealtimeEnvelope) int
	log         *zap.Logger
}

// NewFanoutSubscriber 创建 Pub/Sub 订阅器。
func NewFanoutSubscriber(
	redisClient *redis.Client,
	channel string,
	broadcaster func(userID, topic string, envelope entity.RealtimeEnvelope) int,
	log *zap.Logger,
) *FanoutSubscriber {
	if log == nil {
		log = zap.NewNop()
	}
	return &FanoutSubscriber{
		redis:       redisClient,
		channel:     channel,
		broadcaster: broadcaster,
		log:         log,
	}
}

// Run 阻塞监听 Pub/Sub，ctx 取消时退出。
func (s *FanoutSubscriber) Run(ctx context.Context) error {
	pubsub := s.redis.Subscribe(ctx, s.channel)
	defer func() {
		_ = pubsub.Close()
	}()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return fmt.Errorf("pubsub channel closed")
			}
			s.handleMessage(msg.Payload)
		}
	}
}

func (s *FanoutSubscriber) handleMessage(payload string) {
	var msg FanoutMessage
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		s.log.Warn("pubsub fanout unmarshal failed", zap.Error(err))
		return
	}
	if msg.UserID == "" || msg.Envelope.Type == "" {
		s.log.Warn("pubsub fanout drop invalid message",
			zap.String("userId", msg.UserID),
			zap.String("type", msg.Envelope.Type),
			zap.String("topic", msg.Topic),
		)
		return
	}
	delivered := s.broadcaster(msg.UserID, msg.Topic, msg.Envelope)
	s.log.Info("pubsub fanout delivered",
		zap.String("userId", msg.UserID),
		zap.String("topic", msg.Topic),
		zap.Int64("seq", msg.Envelope.Seq),
		zap.Int("delivered", delivered),
	)
}
