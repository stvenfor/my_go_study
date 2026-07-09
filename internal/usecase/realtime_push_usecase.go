// =============================================================================
// 文件：realtime_push_usecase.go
// 作用：Go → Flutter 推送（如 sys.notify 通知）
//
// 【为什么先写 Redis 再广播？】
//   1. 用户离线时 delivered=0，但 sync 仍能补拉
//   2. seq 持久化后，客户端可按序号去重
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

type RealtimePushUsecase struct {
	events repository.RealtimeEventRepository
	cfg    config.Config
	hub    RealtimeBroadcaster
}

// RealtimeBroadcaster 接口：Usecase 不依赖具体 Hub，方便单元测试 mock。
type RealtimeBroadcaster interface {
	BroadcastToUser(userID, topic string, envelope entity.RealtimeEnvelope) int
}

func NewRealtimePushUsecase(
	events repository.RealtimeEventRepository,
	cfg config.Config,
	hub RealtimeBroadcaster,
) *RealtimePushUsecase {
	return &RealtimePushUsecase{events: events, cfg: cfg, hub: hub}
}

func (u *RealtimePushUsecase) PushToUser(ctx context.Context, input RealtimePushInput) (entity.RealtimeEnvelope, int, error) {
	if input.UserID == "" {
		return entity.RealtimeEnvelope{}, 0, fmt.Errorf("userId 不能为空")
	}

	topic := input.Topic
	if topic == "" {
		topic = entity.TopicSysNotify // 默认推系统通知
	}
	name := input.Name
	if name == "" {
		name = entity.EventSysNotifyShow
	}

	// 每个用户独立 seq，INCR 保证单调递增
	seq, err := u.events.NextSeq(ctx, input.UserID)
	if err != nil {
		return entity.RealtimeEnvelope{}, 0, err
	}

	notifyID := uuid.NewString() // 客户端按 notifyId 去重，防 sync+实时双份
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
