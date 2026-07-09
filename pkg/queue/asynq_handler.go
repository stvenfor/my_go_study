// asynq_handler.go Asynq 任务处理器。
package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"github.com/stvenfor/my_go_study/internal/usecase"
	"github.com/stvenfor/my_go_study/pkg/config"
	"go.uber.org/zap"
)

const schedulerDedupKeyFmt = "scheduler:dedup:%s"

// Handler 注册 Asynq 任务处理函数。
type Handler struct {
	deliver    *usecase.RealtimePushUsecase
	sessions   repository.SessionRepository
	enqueue    *Client
	hourly     *usecase.HourlyNotifyUsecase
	cfg        config.Config
	redis      *redis.Client
	log        *zap.Logger
}

// NewHandler 创建任务处理器。
func NewHandler(
	deliver *usecase.RealtimePushUsecase,
	sessions repository.SessionRepository,
	enqueue *Client,
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

// Register 向 ServeMux 注册所有任务路由。
func (h *Handler) Register(mux *asynq.ServeMux) {
	mux.HandleFunc(TypeRealtimePushNotify, h.handlePushNotify)
	mux.HandleFunc(TypeScheduledBroadcastNotify, h.handleBroadcastNotify)
	mux.HandleFunc(TypeSendSMS, h.handleSendSMS)
	mux.HandleFunc(TypeJPushRegister, h.handleJPushRegister)
}

func (h *Handler) handlePushNotify(ctx context.Context, t *asynq.Task) error {
	payload, err := ParsePushNotifyPayload(t)
	if err != nil {
		return err
	}
	_, _, err = h.deliver.DeliverPush(ctx, usecase.RealtimePushInput{
		UserID: payload.UserID,
		Topic:  payload.Topic,
		Title:  payload.Title,
		Body:   payload.Body,
		Name:   payload.Name,
		Extra:  payload.Extra,
	})
	if err != nil {
		return fmt.Errorf("deliver push: %w", err)
	}
	return nil
}

func (h *Handler) handleBroadcastNotify(ctx context.Context, t *asynq.Task) error {
	payload, err := ParseBroadcastNotifyPayload(t)
	if err != nil {
		return err
	}

	slot := time.Now().In(h.cfg.Scheduler.Location()).Truncate(time.Hour)
	if payload.ScheduleSlot != "" {
		if parsed, parseErr := time.Parse(time.RFC3339, payload.ScheduleSlot); parseErr == nil {
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
		if _, err := h.enqueue.EnqueueRealtimePushWithTaskID(ctx, input, taskID); err != nil {
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

func (h *Handler) handleSendSMS(ctx context.Context, t *asynq.Task) error {
	var payload SendSMSPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal sms payload: %w", err)
	}
	h.log.Info("sms task stub — integrate SMS provider in production",
		zap.String("phone", payload.Phone),
	)
	return nil
}

func (h *Handler) handleJPushRegister(ctx context.Context, t *asynq.Task) error {
	var payload JPushRegisterPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal jpush payload: %w", err)
	}
	h.log.Info("jpush register task stub — integrate JPush SDK in production",
		zap.String("userId", payload.UserID),
		zap.String("deviceId", payload.DeviceID),
	)
	return nil
}

// NewAsynqServer 创建 Asynq Worker Server。
func NewAsynqServer(cfg config.Config) *asynq.Server {
	return asynq.NewServer(RedisClientOpt(cfg), asynq.Config{
		Concurrency: cfg.Queue.AsynqConcurrency(),
	})
}

// NewDeliveryPushUsecase 创建 Worker 侧推送投递 Usecase（经 Pub/Sub 广播）。
func NewDeliveryPushUsecase(
	events repository.RealtimeEventRepository,
	cfg config.Config,
	publisher *FanoutPublisher,
) *usecase.RealtimePushUsecase {
	return usecase.NewRealtimePushUsecase(events, cfg, publisher, nil)
}
