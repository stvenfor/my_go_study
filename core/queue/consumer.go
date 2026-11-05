// consumer.go RabbitMQ 任务消费循环。
package queue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"common/config"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// Dispatcher 消费侧分发契约。
//
// 用接口而非具体的 *Handler：业务分发器已迁出本包
// （见 internal/delivery/worker），基建层不应知道它长什么样。
type Dispatcher interface {
	Dispatch(ctx context.Context, body []byte) error
}

// Consumer RabbitMQ 消费者。
type Consumer struct {
	conn       *amqp.Connection
	ch         *amqp.Channel
	queue      string
	dlq        string
	prefetch   int
	maxRetries int
	timeout    time.Duration
	dlqWired   bool
	log        *zap.Logger
	inflight   sync.WaitGroup
	// publish 注入 amqp.Channel.PublishWithContext；单测替换成 fake。
	publish func(ctx context.Context, exchange, routingKey string, mandatory, immediate bool, msg amqp.Publishing) error
}

// NewConsumer 创建消费者并声明队列（含死信拓扑）。
func NewConsumer(cfg config.Config, log *zap.Logger) (*Consumer, error) {
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
	prefetch := cfg.Queue.Prefetch()
	if err := ch.Qos(prefetch, 0, false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("rabbitmq qos: %w", err)
	}
	return &Consumer{
		conn:       conn,
		ch:         ch,
		queue:      topo.queue,
		dlq:        topo.dlq,
		prefetch:   prefetch,
		maxRetries: cfg.Queue.MaxRetries(),
		timeout:    cfg.Queue.HandlerTimeout(),
		dlqWired:   dlqWired,
		log:        log,
		publish:    ch.PublishWithContext,
	}, nil
}

// Prefetch 返回 QoS prefetch（同时在途的消息上限）。
func (c *Consumer) Prefetch() int { return c.prefetch }

// Queue 返回队列名。
func (c *Consumer) Queue() string { return c.queue }

// DLQ 返回死信队列名。
func (c *Consumer) DLQ() string { return c.dlq }

// DLQWired 死信拓扑是否声明成功。
//
// false = 旧队列参数冲突（406）后降级沿用：重试上限仍生效，但消息不会真的进 DLQ。
func (c *Consumer) DLQWired() bool { return c.dlqWired }

// Wait 等待在途消息处理结束。
//
// 优雅退出顺序：cancel() → Run 返回（此后不会再 Add 新任务）→ Wait() → Close()。
func (c *Consumer) Wait() { c.inflight.Wait() }

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

// Run 消费消息直至 ctx 取消；每条消息独立 goroutine，并发上限 = prefetch。
func (c *Consumer) Run(ctx context.Context, h Dispatcher) error {
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
			c.inflight.Add(1)
			go func() {
				defer c.inflight.Done()
				c.handle(ctx, h, d)
			}()
		}
	}
}

// handle 处理单条消息：成功 Ack，失败按重试上限决定 requeue 还是进死信。
//
// 在途计数由 Run 在 spawn 处 Add/Done，这里不碰 WaitGroup，便于单测直接调用。
func (c *Consumer) handle(ctx context.Context, h Dispatcher, d amqp.Delivery) {
	// WithoutCancel：关机时不再往在途任务传取消，收尾交给 Wait()；
	// 单条卡死仍由 HandlerTimeout 兜住。
	taskCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), c.timeout)
	err := h.Dispatch(taskCtx, d.Body)
	cancel()

	if err == nil {
		if ackErr := d.Ack(false); ackErr != nil {
			c.log.Error("ack failed (broker will redeliver)", zap.Error(ackErr), zap.String("message_id", d.MessageId))
		}
		return
	}

	attempt, known := deliveryAttempt(d)
	// 计数不可得的重复投递一律按超限处理：
	// 「必然失败的消息不会无限循环」是硬门槛，宁可误投 DLQ。
	exhausted := (known && attempt >= c.maxRetries) || (!known && d.Redelivered)
	if exhausted {
		if nackErr := d.Nack(false, false); nackErr != nil {
			c.log.Error("nack to DLQ failed", zap.Error(nackErr), zap.String("message_id", d.MessageId))
			return
		}
		c.log.Warn("task exhausted retries, moved to DLQ",
			zap.String("dlq", c.dlq),
			zap.Bool("dlq_wired", c.dlqWired),
			zap.Int("attempt", attempt),
			zap.Bool("attempt_known", known),
			zap.Error(err),
		)
		return
	}

	// 靠 Nack(requeue=true) 数不出次数（broker 不写计数头），
	// 所以重投由我们做：带 x-retry-count+1 重新发布，再 Ack 原消息。
	if retryErr := c.retryLater(ctx, d, attempt+1); retryErr != nil {
		c.log.Error("republish for retry failed, requeueing original",
			zap.Error(retryErr),
			zap.String("message_id", d.MessageId),
		)
		// 原消息不能 Ack（会连带丢内容），退回队列等下次再来。
		if nackErr := d.Nack(false, true); nackErr != nil {
			c.log.Error("nack(requeue) failed", zap.Error(nackErr), zap.String("message_id", d.MessageId))
		}
		return
	}
	// 先重投成功、后 Ack：中间崩溃只会重复一次（at-least-once），不会丢。
	if ackErr := d.Ack(false); ackErr != nil {
		c.log.Error("ack after republish failed (duplicate possible)",
			zap.Error(ackErr), zap.String("message_id", d.MessageId))
	}
	c.log.Warn("task failed, republished for retry",
		zap.Int("finished_attempt", attempt),
		zap.Int("next_attempt", attempt+1),
		zap.Int("max_retries", c.maxRetries),
		zap.String("message_id", d.MessageId),
		zap.Error(err),
	)
}

// retryLater 把消息带新的重试计数重新发布回任务队列（落到队尾，不阻塞头部）。
func (c *Consumer) retryLater(ctx context.Context, d amqp.Delivery, nextAttempt int) error {
	if c.publish == nil {
		return errors.New("publish not initialised")
	}
	headers := amqp.Table{}
	for k, v := range d.Headers {
		headers[k] = v
	}
	headers[retryCountHeader] = int64(nextAttempt)
	return c.publish(ctx, "", c.queue, false, false, amqp.Publishing{
		ContentType:  d.ContentType,
		DeliveryMode: amqp.Persistent,
		MessageId:    d.MessageId,
		Headers:      headers,
		Body:         d.Body,
	})
}
