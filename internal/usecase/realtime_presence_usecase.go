// =============================================================================
// 文件：realtime_presence_usecase.go
// 作用：Flutter → Go → 其他 Flutter（presence.report → presence.update）
//
// 【为什么不给 presence 也写 seq？】
//   在线状态是瞬时信息，断线重连后 sync 历史 presence 意义不大；
//   省略 seq 简化实现，Flutter acceptSeq(null) 会直接通过。
// =============================================================================
package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
)

type RealtimePresenceInput struct {
	UserID string
	Online bool
	Device string
}

type RealtimePresenceBroadcaster interface {
	BroadcastToTopicExcept(excludeUserID, topic string, envelope entity.RealtimeEnvelope) int
}

type RealtimePresenceUsecase struct {
	presence    repository.PresenceRepository
	broadcaster RealtimePresenceBroadcaster
}

func NewRealtimePresenceUsecase(
	presence repository.PresenceRepository,
	broadcaster RealtimePresenceBroadcaster,
) *RealtimePresenceUsecase {
	return &RealtimePresenceUsecase{
		presence:    presence,
		broadcaster: broadcaster,
	}
}

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

	// 广播给除上报者外的所有订阅 presence.bulk 的连接
	delivered := u.broadcaster.BroadcastToTopicExcept(
		input.UserID,
		entity.TopicPresenceBulk,
		envelope,
	)
	return envelope, delivered, nil
}
