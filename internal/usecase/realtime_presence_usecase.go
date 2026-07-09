package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

// RealtimePresenceInput 客户端 presence.report 输入。
type RealtimePresenceInput struct {
	UserID string
	Online bool
	Device string
}

// RealtimePresenceBroadcaster 向 topic 广播（排除发送者）。
type RealtimePresenceBroadcaster interface {
	BroadcastToTopicExcept(excludeUserID, topic string, envelope entity.RealtimeEnvelope) int
}

// RealtimePresenceUsecase 处理 presence.report 并广播 presence.update。
type RealtimePresenceUsecase struct {
	presence repository.PresenceRepository
	broadcaster RealtimePresenceBroadcaster
}

// NewRealtimePresenceUsecase 创建 presence 用例。
func NewRealtimePresenceUsecase(
	presence repository.PresenceRepository,
	broadcaster RealtimePresenceBroadcaster,
) *RealtimePresenceUsecase {
	return &RealtimePresenceUsecase{
		presence:    presence,
		broadcaster: broadcaster,
	}
}

// Report 处理客户端上报并向其他订阅者广播 presence.update。
func (u *RealtimePresenceUsecase) Report(ctx context.Context, input RealtimePresenceInput) (entity.RealtimeEnvelope, int, error) {
	if input.UserID == "" {
		return entity.RealtimeEnvelope{}, 0, fmt.Errorf("userId 不能为空")
	}

	var onlineCount int
	var err error
	if input.Online {
		onlineCount, err = u.presence.SetOnline(ctx, input.UserID, input.Device)
	} else {
		onlineCount, err = u.presence.SetOffline(ctx, input.UserID)
	}
	if err != nil {
		return entity.RealtimeEnvelope{}, 0, err
	}

	envelope := entity.RealtimeEnvelope{
		ID:    fmt.Sprintf("evt_presence_%d", time.Now().UnixMilli()),
		Type:  "event",
		Topic: entity.TopicPresenceBulk,
		TS:    time.Now().UnixMilli(),
		Payload: map[string]any{
			"name":        entity.EventPresenceUpdate,
			"userId":      input.UserID,
			"online":      input.Online,
			"onlineCount": onlineCount,
		},
	}
	if input.Device != "" {
		envelope.Payload["device"] = input.Device
	}

	delivered := u.broadcaster.BroadcastToTopicExcept(
		input.UserID,
		entity.TopicPresenceBulk,
		envelope,
	)
	return envelope, delivered, nil
}
