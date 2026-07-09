// asynq_client.go Asynq 任务入队客户端。
package queue

import (
	"context"
	"fmt"

	"github.com/hibiken/asynq"
	"github.com/stvenfor/my_go_study/internal/usecase"
	"github.com/stvenfor/my_go_study/pkg/config"
)

// Client 封装 Asynq 入队操作。
type Client struct {
	inner *asynq.Client
}

// NewAsynqClient 创建 Asynq 客户端（复用 Redis 连接配置）。
func NewAsynqClient(cfg config.Config) *Client {
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}
	return &Client{inner: asynq.NewClient(redisOpt)}
}

// Close 关闭 Asynq 客户端。
func (c *Client) Close() error {
	return c.inner.Close()
}

// EnqueueRealtimePush 将 Realtime 推送任务入队。
func (c *Client) EnqueueRealtimePush(ctx context.Context, input usecase.RealtimePushInput) (string, error) {
	task, err := NewPushNotifyTask(PushNotifyPayload{
		UserID: input.UserID,
		Topic:  input.Topic,
		Title:  input.Title,
		Body:   input.Body,
		Name:   input.Name,
		Extra:  input.Extra,
	})
	if err != nil {
		return "", err
	}

	info, err := c.inner.EnqueueContext(ctx, task)
	if err != nil {
		return "", fmt.Errorf("enqueue push notify: %w", err)
	}
	return info.ID, nil
}

// EnqueueRealtimePushWithTaskID 将推送任务入队并指定幂等 Task ID。
func (c *Client) EnqueueRealtimePushWithTaskID(ctx context.Context, input usecase.RealtimePushInput, taskID string) (string, error) {
	task, err := NewPushNotifyTask(PushNotifyPayload{
		UserID: input.UserID,
		Topic:  input.Topic,
		Title:  input.Title,
		Body:   input.Body,
		Name:   input.Name,
		Extra:  input.Extra,
	})
	if err != nil {
		return "", err
	}
	opts := []asynq.Option{}
	if taskID != "" {
		opts = append(opts, asynq.TaskID(taskID))
	}
	info, err := c.inner.EnqueueContext(ctx, task, opts...)
	if err != nil {
		return "", fmt.Errorf("enqueue push notify: %w", err)
	}
	return info.ID, nil
}

// EnqueueBroadcastNotify 入队定时广播任务。
func (c *Client) EnqueueBroadcastNotify(ctx context.Context, p BroadcastNotifyPayload) (string, error) {
	task, err := NewBroadcastNotifyTask(p)
	if err != nil {
		return "", err
	}
	info, err := c.inner.EnqueueContext(ctx, task)
	if err != nil {
		return "", fmt.Errorf("enqueue broadcast notify: %w", err)
	}
	return info.ID, nil
}

// EnqueueSendSMS 将短信发送任务入队（生产 OTP 预留）。
func (c *Client) EnqueueSendSMS(ctx context.Context, phone, message string) (string, error) {
	task, err := NewSendSMSTask(SendSMSPayload{Phone: phone, Message: message})
	if err != nil {
		return "", err
	}
	info, err := c.inner.EnqueueContext(ctx, task)
	if err != nil {
		return "", fmt.Errorf("enqueue send sms: %w", err)
	}
	return info.ID, nil
}

// EnqueueJPushRegister 将极光推送注册任务入队（预留）。
func (c *Client) EnqueueJPushRegister(ctx context.Context, p JPushRegisterPayload) (string, error) {
	task, err := NewJPushRegisterTask(p)
	if err != nil {
		return "", err
	}
	info, err := c.inner.EnqueueContext(ctx, task)
	if err != nil {
		return "", fmt.Errorf("enqueue jpush register: %w", err)
	}
	return info.ID, nil
}
