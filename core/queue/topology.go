// topology.go 任务队列的死信拓扑（DLX + DLQ）声明。
//
// RabbitMQ 队列参数不可变：对已存在、args 不同的队列再 QueueDeclare 会返回
// 406 PRECONDITION_FAILED，并且**服务端会关掉所在 channel**。因此：
//   - 声明必须走独立 channel，不能污染发布/消费用的长驻 channel；
//   - 命中 406 时不能删队列（会丢消息），只能降级为「沿用旧队列」，
//     由运维按 scripts/rabbitmq-ensure-dlq.sh 在维护窗口重建。
package queue

import (
	"errors"
	"fmt"

	"common/config"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// taskTopology 任务队列 + 死信交换机 + 死信队列的名字组合。
type taskTopology struct {
	queue string
	dlx   string
	dlq   string
}

func newTaskTopology(cfg config.Config) taskTopology {
	q := cfg.Queue
	return taskTopology{
		queue: q.RabbitMQQueue(),
		dlx:   q.DeadLetterExchange(),
		dlq:   q.DeadLetterQueue(),
	}
}

// queueArgs 主队列参数。
//
// 发布走默认 exchange、routing key = 队列名，所以死信也按同一 routing key
// 绑定到 DLQ，Nack(requeue=false) 的消息即可落到 <queue>.dlq。
func (t taskTopology) queueArgs() amqp.Table {
	return amqp.Table{
		"x-dead-letter-exchange":    t.dlx,
		"x-dead-letter-routing-key": t.queue,
	}
}

// declare 在给定 channel 上声明完整拓扑；出错时该 channel 可能已被服务端关闭。
func (t taskTopology) declare(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(t.dlx, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dlx %s: %w", t.dlx, err)
	}
	// DLQ 自身不再挂死信，避免失败消息绕圈。
	if _, err := ch.QueueDeclare(t.dlq, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dlq %s: %w", t.dlq, err)
	}
	if err := ch.QueueBind(t.dlq, t.queue, t.dlx, false, nil); err != nil {
		return fmt.Errorf("bind dlq %s to dlx %s: %w", t.dlq, t.dlx, err)
	}
	if _, err := ch.QueueDeclare(t.queue, true, false, false, false, t.queueArgs()); err != nil {
		return fmt.Errorf("declare queue %s: %w", t.queue, err)
	}
	return nil
}

// ensureTopology 声明任务队列拓扑，返回「死信是否已生效」。
//
// 406（旧队列参数不含 DLX）不算致命：降级为沿用旧队列并返回 false，
// 让既有联调继续可用；其余错误原样返回。
func ensureTopology(conn *amqp.Connection, t taskTopology, log *zap.Logger) (bool, error) {
	if log == nil {
		log = zap.NewNop()
	}
	if ch, err := conn.Channel(); err == nil {
		declareErr := t.declare(ch)
		_ = ch.Close()
		if declareErr == nil {
			return true, nil
		}
		if !isPreconditionFailed(declareErr) {
			return false, fmt.Errorf("declare task topology: %w", declareErr)
		}
		log.Warn("task queue args mismatch (406): 沿用旧队列，死信未生效；请在维护窗口执行 scripts/rabbitmq-ensure-dlq.sh 重建",
			zap.String("queue", t.queue),
			zap.String("dlq", t.dlq),
			zap.String("status", "degraded"),
			zap.Error(declareErr),
		)
	}

	// 兜底：不带 args 声明，保证旧队列仍可用。
	ch, err := conn.Channel()
	if err != nil {
		return false, fmt.Errorf("rabbitmq channel: %w", err)
	}
	defer func() { _ = ch.Close() }()
	if _, err := ch.QueueDeclare(t.queue, true, false, false, false, nil); err != nil {
		return false, fmt.Errorf("declare legacy queue %s: %w", t.queue, err)
	}
	return false, nil
}

// isPreconditionFailed 判断是否 406（amqp091 可能以 Error 值或 *Error 指针返回）。
func isPreconditionFailed(err error) bool {
	var e amqp.Error
	if errors.As(err, &e) {
		return e.Code == amqp.PreconditionFailed
	}
	var pe *amqp.Error
	if errors.As(err, &pe) && pe != nil {
		return pe.Code == amqp.PreconditionFailed
	}
	return false
}
