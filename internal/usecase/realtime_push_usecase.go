// =============================================================================
// 文件：realtime_push_usecase.go
// 作用：Go → Flutter 推送（如 sys.notify 通知）
//
// 【为什么先写 Redis 再广播？】
//   1. 用户离线时 delivered=0，但 sync 仍能补拉
//   2. seq 持久化后，客户端可按序号去重
//
// 【异步模式】queue.enabled=true 时 HTTP 入队 Asynq，Worker 投递并经 Pub/Sub 广播。
// =============================================================================
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"github.com/stvenfor/my_go_study/pkg/config"
)

type RealtimePushInput struct {
	UserID string
	Topic  string
	Title  string
	Body   string
	Name   string
	Extra  map[string]any
}

// RealtimePushOutput 推送结果（同步或异步入队）。
type RealtimePushOutput struct {
	Envelope  entity.RealtimeEnvelope
	Delivered int
	Queued    bool
	TaskID    string
}

// PushEnqueuer 异步入队接口（Asynq 实现）。
type PushEnqueuer interface {
	EnqueueRealtimePush(ctx context.Context, input RealtimePushInput) (taskID string, err error)
}

type RealtimePushUsecase struct {
	events repository.RealtimeEventRepository
	cfg    config.Config
	hub    RealtimeBroadcaster
	queue  PushEnqueuer
}

// RealtimeBroadcaster 接口：Usecase 不依赖具体 Hub，方便单元测试 mock。
type RealtimeBroadcaster interface {
	BroadcastToUser(userID, topic string, envelope entity.RealtimeEnvelope) int
}

func NewRealtimePushUsecase(
	events repository.RealtimeEventRepository,
	cfg config.Config,
	hub RealtimeBroadcaster,
	queue PushEnqueuer,
) *RealtimePushUsecase {
	return &RealtimePushUsecase{events: events, cfg: cfg, hub: hub, queue: queue}
}

// PushToUser 推送通知：queue.enabled 时异步入队，否则同步投递。
func (u *RealtimePushUsecase) PushToUser(ctx context.Context, input RealtimePushInput) (RealtimePushOutput, error) {
	if input.UserID == "" {
		return RealtimePushOutput{}, fmt.Errorf("userId 不能为空")
	}

	if u.cfg.Queue.UseAsyncPush() && u.queue != nil {
		taskID, err := u.queue.EnqueueRealtimePush(ctx, input)
		if err != nil {
			return RealtimePushOutput{}, err
		}
		return RealtimePushOutput{
			Queued:    true,
			TaskID:    taskID,
			Delivered: -1,
		}, nil
	}

	envelope, delivered, err := u.DeliverPush(ctx, input)
	if err != nil {
		return RealtimePushOutput{}, err
	}
	return RealtimePushOutput{
		Envelope:  envelope,
		Delivered: delivered,
	}, nil
}

// DeliverPush 同步投递：写 Redis 事件日志并广播（Worker 或同步模式调用）。
func (u *RealtimePushUsecase) DeliverPush(ctx context.Context, input RealtimePushInput) (entity.RealtimeEnvelope, int, error) {
	if input.UserID == "" {
		return entity.RealtimeEnvelope{}, 0, fmt.Errorf("userId 不能为空")
	}
	if u.hub == nil {
		return entity.RealtimeEnvelope{}, 0, fmt.Errorf("broadcaster 未配置")
	}

	topic := input.Topic
	if topic == "" {
		topic = entity.TopicSysNotify
	}
	name := input.Name
	if name == "" {
		name = entity.EventSysNotifyShow
	}

	seq, err := u.events.NextSeq(ctx, input.UserID)
	if err != nil {
		return entity.RealtimeEnvelope{}, 0, err
	}

	notifyID := uuid.NewString()
	payload := map[string]any{
		"name":     name,
		"notifyId": notifyID,
		"title":    input.Title,
		"body":     input.Body,
	}
	for k, v := range input.Extra {
		payload[k] = v
	}

	envelope := entity.RealtimeEnvelope{
		ID:      fmt.Sprintf("evt_%d", time.Now().UnixMilli()),
		Type:    "event",
		Topic:   topic,
		TS:      time.Now().UnixMilli(),
		Seq:     seq,
		Payload: payload,
	}

	if err := u.events.Append(ctx, input.UserID, envelope, u.cfg.Realtime.EventRetention); err != nil {
		return entity.RealtimeEnvelope{}, 0, err
	}

	delivered := u.hub.BroadcastToUser(input.UserID, topic, envelope)
	return envelope, delivered, nil
}
