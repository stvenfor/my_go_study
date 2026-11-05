// retry.go 消费侧的投递次数判定（有界重试 → 死信）。
package queue

import (
	amqp "github.com/rabbitmq/amqp091-go"
)

// retryCountHeader 是本消费端自己维护的重试计数头。
//
// 实测 RabbitMQ 3.13 对 requeue（Nack requeue=true）后的消息**不带**
// x-delivery-count，也不写 x-death —— 这两个头只在 broker 侧死信投递时才有。
// 所以「第几次尝试」只能由我们自己数：失败时 Ack 原消息并带 x-retry-count+1
// 重新发布，见 Consumer.retryLater。
const retryCountHeader = "x-retry-count"

// deliveryAttempt 返回该消息「第几次尝试」（含本次）以及计数是否可信。
//
// 计数来源按优先级：
//  1. x-retry-count —— 我们自己写的头，正常路径都命中它；
//  2. x-delivery-count / x-death —— 兼容升级前就已入队、或经 broker 侧重投的消息；
//  3. 三者都没有且 Redelivered=true → known=false，调用方按「已达上限」处理。
//     这条兜底是硬要求：宁可误投 DLQ，也不能因为读不到计数而无限重投。
func deliveryAttempt(d amqp.Delivery) (attempt int, known bool) {
	if n, ok := headerInt(d.Headers, retryCountHeader); ok {
		return int(n), true
	}
	if n, ok := headerInt(d.Headers, "x-delivery-count"); ok {
		return int(n) + 1, true
	}
	if n, ok := deathCount(d.Headers); ok {
		return n + 1, true
	}
	if d.Redelivered {
		return 0, false
	}
	return 1, true
}

// deathCount 解析 x-death（[]any，元素为 amqp.Table）里的最大 count。
func deathCount(headers amqp.Table) (int, bool) {
	raw, ok := headers["x-death"]
	if !ok {
		return 0, false
	}
	entries, ok := raw.([]any)
	if !ok || len(entries) == 0 {
		return 0, false
	}
	maxCount := int64(-1)
	for _, item := range entries {
		entry, ok := item.(amqp.Table)
		if !ok {
			continue
		}
		if c, ok := headerInt(entry, "count"); ok && c > maxCount {
			maxCount = c
		}
	}
	if maxCount < 0 {
		return 0, false
	}
	return int(maxCount), true
}

// headerInt 兼容 AMQP 解码出的多种整数宽度。
func headerInt(headers amqp.Table, key string) (int64, bool) {
	raw, ok := headers[key]
	if !ok {
		return 0, false
	}
	switch v := raw.(type) {
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint8:
		return int64(v), true
	case uint32:
		return int64(v), true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}
