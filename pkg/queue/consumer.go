// consumer.go RabbitMQ 任务消费循环。
package queue

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stvenfor/my_go_study/pkg/config"
	"go.uber.org/zap"
)

// Consumer RabbitMQ 消费者。
type Consumer struct {
	conn     *amqp.Connection
	ch       *amqp.Channel
	queue    string
	prefetch int
	log      *zap.Logger
}

// NewConsumer 创建消费者并声明队列。
func NewConsumer(cfg config.Config, log *zap.Logger) (*Consumer, error) {
	if log == nil {
		log = zap.NewNop()
	}
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
	prefetch := cfg.Queue.Prefetch()
	if err := ch.Qos(prefetch, 0, false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq qos: %w", err)
	}
	return &Consumer{conn: conn, ch: ch, queue: qName, prefetch: prefetch, log: log}, nil
}

// Prefetch 返回 QoS prefetch。
func (c *Consumer) Prefetch() int { return c.prefetch }

// Queue 返回队列名。
func (c *Consumer) Queue() string { return c.queue }

// Close 关闭连接。
func (c *Consumer) Close() error {
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

// Run 阻塞消费直至 ctx 取消。
func (c *Consumer) Run(ctx context.Context, h *Handler) error {
	deliveries, err := c.ch.Consume(c.queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("rabbitmq consume: %w", err)
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("rabbitmq deliveries closed")
			}
			if err := h.Dispatch(ctx, d.Body); err != nil {
				c.log.Error("task failed", zap.Error(err), zap.String("message_id", d.MessageId))
				_ = d.Nack(false, true)
				continue
			}
			_ = d.Ack(false)
		}
	}
}
