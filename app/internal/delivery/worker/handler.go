// =============================================================================
// 文件：handler.go
// 作用：RabbitMQ Worker 侧的任务分发器（Push / SMS / JPush / 定时广播）。
//
// 【为什么放在 delivery 层而不是 pkg/queue？】
//
//	它持有 RealtimePushUsecase / HourlyNotifyUsecase / JPushUsecase / SessionRepository，
//	是业务逻辑而非基建。原先放在 pkg/queue 导致 pkg → internal/usecase 反向依赖，
//	模块化时成环。基建部分（连接、信封、消费循环）仍在 pkg/queue。
//
// =============================================================================
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"app/internal/usecase"
	"common/config"
	pkgqueue "core/queue"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"model/repository"
)

const schedulerDedupKeyFmt = "scheduler:dedup:%s"

// Handler 按任务类型分发业务逻辑。
type Handler struct {
	deliver  *usecase.RealtimePushUsecase
	sessions repository.SessionRepository
	enqueue  *pkgqueue.Client
	hourly   *usecase.HourlyNotifyUsecase
	jpushUC  *usecase.JPushUsecase
	cfg      config.Config
	redis    *redis.Client
	log      *zap.Logger
}

// NewHandler 创建任务处理器。
func NewHandler(
	deliver *usecase.RealtimePushUsecase,
	sessions repository.SessionRepository,
	enqueue *pkgqueue.Client,
	cfg config.Config,
	redisClient *redis.Client,
	log *zap.Logger,
) *Handler {
	return &Handler{
		deliver:  deliver,
		sessions: sessions,
		enqueue:  enqueue,
		hourly:   usecase.NewHourlyNotifyUsecase(cfg),
		cfg:      cfg,
		redis:    redisClient,
		log:      log,
	}
}

// WithJPush 注入极光设备登记 usecase（可选）。
func (h *Handler) WithJPush(uc *usecase.JPushUsecase) *Handler {
	if h != nil {
		h.jpushUC = uc
	}
	return h
}

// Dispatch 按 envelope.type 分发。实现 pkg/queue.Dispatcher。
func (h *Handler) Dispatch(ctx context.Context, body []byte) error {
	var env pkgqueue.Envelope
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("unmarshal envelope: %w", err)
	}
	switch env.Type {
	case pkgqueue.TypeRealtimePushNotify:
		return h.handlePushNotify(ctx, env.Payload)
	case pkgqueue.TypeScheduledBroadcastNotify:
		return h.handleBroadcastNotify(ctx, env.Payload)
	case pkgqueue.TypeSendSMS:
		return h.handleSendSMS(ctx, env.Payload)
	case pkgqueue.TypeJPushRegister:
		return h.handleJPushRegister(ctx, env.Payload)
	default:
		return fmt.Errorf("unknown task type: %s", env.Type)
	}
}

func (h *Handler) handlePushNotify(ctx context.Context, payload []byte) error {
	p, err := pkgqueue.ParsePushNotifyPayload(payload)
	if err != nil {
		return err
	}
	_, _, err = h.deliver.DeliverPush(ctx, usecase.RealtimePushInput{
		UserID: p.UserID,
		Topic:  p.Topic,
		Title:  p.Title,
		Body:   p.Body,
		Name:   p.Name,
		Extra:  p.Extra,
	})
	if err != nil {
		return fmt.Errorf("deliver push: %w", err)
	}
	return nil
}

func (h *Handler) handleBroadcastNotify(ctx context.Context, payload []byte) error {
	p, err := pkgqueue.ParseBroadcastNotifyPayload(payload)
	if err != nil {
		return err
	}

	slot := time.Now().In(h.cfg.Scheduler.Location()).Truncate(time.Hour)
	if p.ScheduleSlot != "" {
		if parsed, parseErr := time.Parse(time.RFC3339, p.ScheduleSlot); parseErr == nil {
			slot = parsed.In(h.cfg.Scheduler.Location()).Truncate(time.Hour)
		}
	}

	userIDs, err := h.sessions.ListActiveUserIDs(ctx)
	if err != nil {
		return fmt.Errorf("list active sessions: %w", err)
	}

	enqueued := 0
	skipped := 0
	for _, userID := range userIDs {
		if h.cfg.Auth.IsSessionExempt(userID, "") {
			skipped++
			continue
		}
		taskID := h.hourly.DedupTaskID(userID, slot)
		dedupKey := fmt.Sprintf(schedulerDedupKeyFmt, taskID)
		ok, err := h.redis.SetNX(ctx, dedupKey, 1, 2*time.Hour).Result()
		if err != nil {
			return fmt.Errorf("dedup check: %w", err)
		}
		if !ok {
			skipped++
			continue
		}

		input := h.hourly.BuildPushInput(userID, slot)
		if _, err := h.enqueue.EnqueuePushNotifyWithTaskID(ctx, pkgqueue.PushNotifyPayload{
			UserID: input.UserID,
			Topic:  input.Topic,
			Title:  input.Title,
			Body:   input.Body,
			Name:   input.Name,
			Extra:  input.Extra,
		}, taskID); err != nil {
			_, _ = h.redis.Del(ctx, dedupKey).Result()
			return fmt.Errorf("enqueue push for %s: %w", userID, err)
		}
		enqueued++
	}

	h.log.Info("hourly broadcast enqueued",
		zap.String("schedule_slot", slot.Format(time.RFC3339)),
		zap.Int("total_sessions", len(userIDs)),
		zap.Int("enqueued", enqueued),
		zap.Int("skipped", skipped),
	)
	return nil
}

func (h *Handler) handleSendSMS(ctx context.Context, payload []byte) error {
	var p pkgqueue.SendSMSPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("unmarshal sms payload: %w", err)
	}
	h.log.Info("sms task stub — integrate SMS provider in production",
		zap.String("phone", p.Phone),
	)
	return nil
}

func (h *Handler) handleJPushRegister(ctx context.Context, payload []byte) error {
	var p pkgqueue.JPushRegisterPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return fmt.Errorf("unmarshal jpush payload: %w", err)
	}
	if h.jpushUC == nil {
		h.log.Info("jpush register skipped — usecase nil",
			zap.String("userId", p.UserID),
			zap.String("deviceId", p.DeviceID),
		)
		return nil
	}
	_, err := h.jpushUC.RegisterDevice(ctx, usecase.RegisterDeviceInput{
		UserID:         p.UserID,
		DeviceID:       p.DeviceID,
		Platform:       p.Platform,
		RegistrationID: p.RegisterID,
	})
	if err != nil {
		return fmt.Errorf("jpush register: %w", err)
	}
	h.log.Info("jpush device registered",
		zap.String("userId", p.UserID),
		zap.String("deviceId", p.DeviceID),
		zap.String("platform", p.Platform),
	)
	return nil
}

// NewDeliveryPushUsecase 创建 Worker 侧推送投递 Usecase（经 Pub/Sub 广播）。
func NewDeliveryPushUsecase(
	events repository.RealtimeEventRepository,
	cfg config.Config,
	publisher *pkgqueue.FanoutPublisher,
) *usecase.RealtimePushUsecase {
	return usecase.NewRealtimePushUsecase(events, cfg, publisher, nil)
}
