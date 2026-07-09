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

// RealtimePushInput 推送通知请求。
type RealtimePushInput struct {
	UserID string
	Topic  string
	Title  string
	Body   string
	Name   string
	Extra  map[string]any
}

// RealtimePushUsecase 事件推送用例。
type RealtimePushUsecase struct {
	events repository.RealtimeEventRepository
	cfg    config.Config
	hub    RealtimeBroadcaster
}

// RealtimeBroadcaster WS 广播接口（由 Hub 实现）。
type RealtimeBroadcaster interface {
	BroadcastToUser(userID, topic string, envelope entity.RealtimeEnvelope) int
}

// NewRealtimePushUsecase 创建推送用例。
func NewRealtimePushUsecase(
	events repository.RealtimeEventRepository,
	cfg config.Config,
	hub RealtimeBroadcaster,
) *RealtimePushUsecase {
	return &RealtimePushUsecase{events: events, cfg: cfg, hub: hub}
}

// PushToUser 向指定用户推送 event 并持久化供 sync。
func (u *RealtimePushUsecase) PushToUser(ctx context.Context, input RealtimePushInput) (entity.RealtimeEnvelope, int, error) {
	if input.UserID == "" {
		return entity.RealtimeEnvelope{}, 0, fmt.Errorf("userId 不能为空")
	}
	topic := input.Topic
	if topic == "" {
		topic = entity.TopicSysNotify
	}
	name := input.Name
	if name == "" {
		name = "sys.notify.show"
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
		ID:    fmt.Sprintf("evt_%d", time.Now().UnixMilli()),
		Type:  "event",
		Topic: topic,
		TS:    time.Now().UnixMilli(),
		Seq:   seq,
		Payload: payload,
	}

	if err := u.events.Append(ctx, input.UserID, envelope, u.cfg.Realtime.EventRetention); err != nil {
		return entity.RealtimeEnvelope{}, 0, err
	}

	delivered := u.hub.BroadcastToUser(input.UserID, topic, envelope)
	return envelope, delivered, nil
}
