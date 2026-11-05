// client.go RabbitMQ 任务入队客户端。
package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"common/config"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// Client 封装 RabbitMQ 入队操作。
type Client struct {
	conn     *amqp.Connection
	ch       *amqp.Channel
	queue    string
	dlq      string
	dlqWired bool
	// confirms 非 nil 表示开启 publisher confirm；conflictMu 保证
	// 「发布 → 读一条确认」串行，读到的确认一定对应刚发的那条。
	confirms  chan amqp.Confirmation
	publishMu sync.Mutex
}

// NewClient 创建 RabbitMQ 客户端并声明队列（含死信拓扑）。
func NewClient(cfg config.Config, log *zap.Logger) (*Client, error) {
	if log == nil {
		log = zap.NewNop()
	}
	conn, err := amqp.Dial(cfg.Queue.RabbitMQURL())
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}
	topo := newTaskTopology(cfg)
	dlqWired, err := ensureTopology(conn, topo, log)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq channel: %w", err)
	}
	client := &Client{conn: conn, ch: ch, queue: topo.queue, dlq: topo.dlq, dlqWired: dlqWired}
	if cfg.Queue.UsePublisherConfirm() {
		if err := ch.Confirm(false); err != nil {
			_ = ch.Close()
			_ = conn.Close()
			return nil, fmt.Errorf("rabbitmq confirm mode: %w", err)
		}
		client.confirms = ch.NotifyPublish(make(chan amqp.Confirmation, 1))
	}
	return client, nil
}

// Queue 返回任务队列名。
func (c *Client) Queue() string { return c.queue }

// DLQ 返回死信队列名。
func (c *Client) DLQ() string { return c.dlq }

// DLQWired 死信拓扑是否声明成功（false = 406 降级沿用旧队列）。
func (c *Client) DLQWired() bool { return c.dlqWired }

// DLQDepth 返回死信队列当前消息数。
//
// 走独立 channel，避免与发布争用同一 channel；QueueInspect 失败会关闭所在
// channel，这里用完即关，故每次新建。
func (c *Client) DLQDepth() (int, error) {
	if c == nil {
		return 0, errors.New("queue client is nil")
	}
	ch, err := c.conn.Channel()
	if err != nil {
		return 0, fmt.Errorf("rabbitmq channel: %w", err)
	}
	defer func() { _ = ch.Close() }()
	q, err := ch.QueueInspect(c.dlq)
	if err != nil {
		return 0, fmt.Errorf("inspect dlq %s: %w", c.dlq, err)
	}
	return q.Messages, nil
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
	c.publishMu.Lock()
	defer c.publishMu.Unlock()

	err = c.ch.PublishWithContext(ctx, "", c.queue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		MessageId:    env.TaskID,
		Body:         body,
	})
	if err != nil {
		return "", err
	}
	// broker 未落盘（连接断开 / 内存告警拒收）时 PublishWithContext 仍可能返回 nil，
	// 必须靠 confirm 才能判定失败；失败一律上抛，调用方不得当成功。
	if c.confirms != nil {
		select {
		case conf, ok := <-c.confirms:
			if !ok {
				return "", errors.New("publisher confirm channel closed")
			}
			if !conf.Ack {
				return "", errors.New("publisher nack: broker 未接收消息")
			}
		case <-ctx.Done():
			return "", fmt.Errorf("wait publisher confirm: %w", ctx.Err())
		}
	}
	return env.TaskID, nil
}

// Enqueue 入队任意已构造好的信封。
// 定时任务（payment 查单/对账）用它，避免每加一种任务就抄一份 6 行样板。
func (c *Client) Enqueue(ctx context.Context, env Envelope) (string, error) {
	id, err := c.publish(ctx, env)
	if err != nil {
		return "", fmt.Errorf("enqueue %s: %w", env.Type, err)
	}
	return id, nil
}

// EnqueuePushNotify 将 Realtime 推送任务入队。
//
// 入参用 queue 自己的 PushNotifyPayload，而非 usecase.RealtimePushInput：
// 本包是基建层，不能反向依赖业务层。业务侧的类型转换见
// internal/repository/queue/push_enqueuer.go。
func (c *Client) EnqueuePushNotify(ctx context.Context, p PushNotifyPayload) (string, error) {
	env, err := NewPushNotifyTask(p)
	if err != nil {
		return "", err
	}
	id, err := c.publish(ctx, env)
	if err != nil {
		return "", fmt.Errorf("enqueue push notify: %w", err)
	}
	return id, nil
}

// EnqueuePushNotifyWithTaskID 将推送任务入队并指定幂等 Task ID。
func (c *Client) EnqueuePushNotifyWithTaskID(ctx context.Context, p PushNotifyPayload, taskID string) (string, error) {
	env, err := newEnvelope(TypeRealtimePushNotify, p, taskID)
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
