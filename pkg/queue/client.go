// client.go RabbitMQ 任务入队客户端。
package queue

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stvenfor/my_go_study/internal/usecase"
	"github.com/stvenfor/my_go_study/pkg/config"
)

// Client 封装 RabbitMQ 入队操作。
type Client struct {
	conn  *amqp.Connection
	ch    *amqp.Channel
	queue string
}

// NewClient 创建 RabbitMQ 客户端并声明队列。
func NewClient(cfg config.Config) (*Client, error) {
	conn, err := amqp.Dial(cfg.Queue.RabbitMQURL())
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq channel: %w", err)
	}
	qName := cfg.Queue.RabbitMQQueue()
	if _, err := ch.QueueDeclare(qName, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq declare: %w", err)
	}
	return &Client{conn: conn, ch: ch, queue: qName}, nil
}

// Close 关闭 RabbitMQ 连接。
func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	var first error
	if c.ch != nil {
		if err := c.ch.Close(); err != nil && first == nil {
			first = err
		}
	}
	if c.conn != nil {
		if err := c.conn.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (c *Client) publish(ctx context.Context, env Envelope) (string, error) {
	body, err := json.Marshal(env)
	if err != nil {
		return "", fmt.Errorf("marshal envelope: %w", err)
	}
	err = c.ch.PublishWithContext(ctx, "", c.queue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    env.TaskID,
		Body:         body,
	})
	if err != nil {
		return "", err
	}
	return env.TaskID, nil
}

// EnqueueRealtimePush 将 Realtime 推送任务入队。
func (c *Client) EnqueueRealtimePush(ctx context.Context, input usecase.RealtimePushInput) (string, error) {
	env, err := NewPushNotifyTask(PushNotifyPayload{
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
	id, err := c.publish(ctx, env)
	if err != nil {
		return "", fmt.Errorf("enqueue push notify: %w", err)
	}
	return id, nil
}

// EnqueueRealtimePushWithTaskID 将推送任务入队并指定幂等 Task ID。
func (c *Client) EnqueueRealtimePushWithTaskID(ctx context.Context, input usecase.RealtimePushInput, taskID string) (string, error) {
	env, err := newEnvelope(TypeRealtimePushNotify, PushNotifyPayload{
		UserID: input.UserID,
		Topic:  input.Topic,
		Title:  input.Title,
		Body:   input.Body,
		Name:   input.Name,
		Extra:  input.Extra,
	}, taskID)
	if err != nil {
		return "", err
	}
	id, err := c.publish(ctx, env)
	if err != nil {
		return "", fmt.Errorf("enqueue push notify: %w", err)
	}
	return id, nil
}

// EnqueueBroadcastNotify 入队定时广播任务。
func (c *Client) EnqueueBroadcastNotify(ctx context.Context, p BroadcastNotifyPayload) (string, error) {
	env, err := NewBroadcastNotifyTask(p)
	if err != nil {
		return "", err
	}
	id, err := c.publish(ctx, env)
	if err != nil {
		return "", fmt.Errorf("enqueue broadcast notify: %w", err)
	}
	return id, nil
}

// EnqueueSendSMS 将短信发送任务入队（生产 OTP 预留）。
func (c *Client) EnqueueSendSMS(ctx context.Context, phone, message string) (string, error) {
	env, err := NewSendSMSTask(SendSMSPayload{Phone: phone, Message: message})
	if err != nil {
		return "", err
	}
	id, err := c.publish(ctx, env)
	if err != nil {
		return "", fmt.Errorf("enqueue send sms: %w", err)
	}
	return id, nil
}

// EnqueueJPushRegister 将极光推送注册任务入队（预留）。
func (c *Client) EnqueueJPushRegister(ctx context.Context, p JPushRegisterPayload) (string, error) {
	env, err := NewJPushRegisterTask(p)
	if err != nil {
		return "", err
	}
	id, err := c.publish(ctx, env)
	if err != nil {
		return "", fmt.Errorf("enqueue jpush register: %w", err)
	}
	return id, nil
}
