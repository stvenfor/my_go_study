package queue

import (
	"context"
	"errors"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// fakeAck 记录 Ack/Nack 调用，替代真 channel。
type fakeAck struct {
	acks        int
	nacks       int
	lastRequeue bool
}

func (f *fakeAck) Ack(tag uint64, multiple bool) error {
	f.acks++
	return nil
}

func (f *fakeAck) Nack(tag uint64, multiple bool, requeue bool) error {
	f.nacks++
	f.lastRequeue = requeue
	return nil
}

func (f *fakeAck) Reject(tag uint64, requeue bool) error { return nil }

type stubDispatcher struct {
	err         error
	deadline    time.Time
	hasDeadline bool
}

func (s *stubDispatcher) Dispatch(ctx context.Context, _ []byte) error {
	s.deadline, s.hasDeadline = ctx.Deadline()
	return s.err
}

// publishRecorder 替代 amqp.Channel.PublishWithContext。
type publishRecorder struct {
	calls      int
	routingKey string
	msg        amqp.Publishing
	err        error
}

func (r *publishRecorder) fn(_ context.Context, _, routingKey string, _, _ bool, msg amqp.Publishing) error {
	r.calls++
	r.routingKey = routingKey
	r.msg = msg
	return r.err
}

func newTestConsumer(maxRetries int, rec *publishRecorder) *Consumer {
	c := &Consumer{
		queue:      "q",
		dlq:        "q.dlq",
		maxRetries: maxRetries,
		timeout:    time.Second,
		dlqWired:   true,
		log:        zap.NewNop(),
	}
	c.publish = rec.fn
	return c
}

func delivery(ack amqp.Acknowledger, redelivered bool, headers amqp.Table) amqp.Delivery {
	return amqp.Delivery{
		Acknowledger: ack,
		MessageId:    "m1",
		Body:         []byte("{}"),
		Redelivered:  redelivered,
		Headers:      headers,
	}
}

func TestHandleSuccessAcks(t *testing.T) {
	ack := &fakeAck{}
	c := newTestConsumer(3, &publishRecorder{})
	c.handle(context.Background(), &stubDispatcher{}, delivery(ack, false, nil))
	if ack.acks != 1 || ack.nacks != 0 {
		t.Fatalf("acks=%d nacks=%d", ack.acks, ack.nacks)
	}
}

// 未到上限：重投一条带 x-retry-count=2 的新消息，然后 Ack 原消息（不靠 requeue 计数）。
func TestHandleBelowLimitRepublishesWithCount(t *testing.T) {
	ack := &fakeAck{}
	rec := &publishRecorder{}
	c := newTestConsumer(3, rec)
	c.handle(context.Background(), &stubDispatcher{err: errors.New("boom")}, delivery(ack, false, nil))
	if rec.calls != 1 {
		t.Fatalf("期望重投 1 次，calls=%d", rec.calls)
	}
	if rec.routingKey != "q" {
		t.Fatalf("重投目标队列=%q", rec.routingKey)
	}
	if got := rec.msg.Headers[retryCountHeader]; got != int64(2) {
		t.Fatalf("%v=%v，期望 int64(2)", retryCountHeader, got)
	}
	if string(rec.msg.Body) != "{}" || rec.msg.MessageId != "m1" {
		t.Fatalf("重投内容未保持原样: body=%s id=%s", rec.msg.Body, rec.msg.MessageId)
	}
	if ack.acks != 1 || ack.nacks != 0 {
		t.Fatalf("重投成功后应 Ack 原消息；acks=%d nacks=%d", ack.acks, ack.nacks)
	}
}

// 已有 x-retry-count=2 → 本次是第 2 次尝试，仍低于上限 → 投第 3 次。
func TestHandleIncrementsExistingRetryCount(t *testing.T) {
	ack := &fakeAck{}
	rec := &publishRecorder{}
	c := newTestConsumer(3, rec)
	d := delivery(ack, false, amqp.Table{retryCountHeader: int64(2)})
	c.handle(context.Background(), &stubDispatcher{err: errors.New("boom")}, d)
	if rec.msg.Headers[retryCountHeader] != int64(3) {
		t.Fatalf("next=%v 期望 3", rec.msg.Headers[retryCountHeader])
	}
}

// 重投失败时绝不能 Ack 原消息（会丢任务），退回队列等下次。
func TestHandleRepublishFailureKeepsMessage(t *testing.T) {
	ack := &fakeAck{}
	rec := &publishRecorder{err: errors.New("channel closed")}
	c := newTestConsumer(3, rec)
	c.handle(context.Background(), &stubDispatcher{err: errors.New("boom")}, delivery(ack, false, nil))
	if ack.acks != 0 {
		t.Fatalf("重投失败不得 Ack，acks=%d", ack.acks)
	}
	if ack.nacks != 1 || !ack.lastRequeue {
		t.Fatalf("期望 Nack(requeue=true)，nacks=%d requeue=%v", ack.nacks, ack.lastRequeue)
	}
}

func TestHandleAtLimitGoesToDLQ(t *testing.T) {
	ack := &fakeAck{}
	rec := &publishRecorder{}
	c := newTestConsumer(3, rec)
	// x-retry-count=3 → 本次即第 3 次尝试，已达上限
	d := delivery(ack, false, amqp.Table{retryCountHeader: int64(3)})
	c.handle(context.Background(), &stubDispatcher{err: errors.New("boom")}, d)
	if rec.calls != 0 {
		t.Fatalf("已达上限不应再重投，calls=%d", rec.calls)
	}
	if ack.nacks != 1 || ack.lastRequeue {
		t.Fatalf("期望 Nack(requeue=false) 进死信，nacks=%d requeue=%v", ack.nacks, ack.lastRequeue)
	}
}

func TestHandleLegacyXDeliveryCountAtLimit(t *testing.T) {
	ack := &fakeAck{}
	c := newTestConsumer(3, &publishRecorder{})
	d := delivery(ack, true, amqp.Table{"x-delivery-count": int64(2)})
	c.handle(context.Background(), &stubDispatcher{err: errors.New("boom")}, d)
	if ack.nacks != 1 || ack.lastRequeue {
		t.Fatalf("x-delivery-count=2 即第 3 次尝试，应进死信；nacks=%d requeue=%v", ack.nacks, ack.lastRequeue)
	}
}

func TestHandleXDeathCountReached(t *testing.T) {
	ack := &fakeAck{}
	c := newTestConsumer(2, &publishRecorder{})
	d := delivery(ack, true, amqp.Table{
		"x-death": []any{amqp.Table{"reason": "rejected", "count": int64(1), "queue": "q"}},
	})
	c.handle(context.Background(), &stubDispatcher{err: errors.New("boom")}, d)
	if ack.nacks != 1 || ack.lastRequeue {
		t.Fatalf("x-death=1 即第 2 次尝试，应进死信；nacks=%d requeue=%v", ack.nacks, ack.lastRequeue)
	}
}

// 计数拿不到但消息是重复投递时，宁可进死信也不能无限重投（硬门槛）。
func TestHandleUnknownCountOnRedeliveryTreatsAsExhausted(t *testing.T) {
	ack := &fakeAck{}
	rec := &publishRecorder{}
	c := newTestConsumer(3, rec)
	d := delivery(ack, true, nil)
	c.handle(context.Background(), &stubDispatcher{err: errors.New("boom")}, d)
	if rec.calls != 0 {
		t.Fatalf("未知计数不得再重投，calls=%d", rec.calls)
	}
	if ack.nacks != 1 || ack.lastRequeue {
		t.Fatalf("未知计数 + redelivered 应停止 requeue；nacks=%d requeue=%v", ack.nacks, ack.lastRequeue)
	}
}

func TestHandleAppliesPerMessageTimeout(t *testing.T) {
	ack := &fakeAck{}
	c := newTestConsumer(3, &publishRecorder{})
	c.timeout = 3 * time.Second
	sp := &stubDispatcher{}
	c.handle(context.Background(), sp, delivery(ack, false, nil))
	if !sp.hasDeadline {
		t.Fatal("Dispatch 的 ctx 应带单条超时 deadline")
	}
	if d := time.Until(sp.deadline); d < 2*time.Second || d > 4*time.Second {
		t.Fatalf("deadline 偏移异常：until=%v", d)
	}
}

// 只有 deadline 不够：真正卡死的 handler 必须被单条超时截断，否则会占住整个 prefetch 窗口。
type blockUntilCtx struct{ returned time.Duration }

func (b *blockUntilCtx) Dispatch(ctx context.Context, _ []byte) error {
	start := time.Now()
	<-ctx.Done()
	b.returned = time.Since(start)
	return ctx.Err()
}

func TestHandleTimeoutTruncatesSlowDispatch(t *testing.T) {
	ack := &fakeAck{}
	rec := &publishRecorder{}
	c := newTestConsumer(3, rec)
	c.timeout = 50 * time.Millisecond
	sp := &blockUntilCtx{}
	done := make(chan struct{})
	go func() { defer close(done); c.handle(context.Background(), sp, delivery(ack, false, nil)) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handle 被慢任务拖住，单条超时未生效")
	}
	if sp.returned < 40*time.Millisecond || sp.returned > 500*time.Millisecond {
		t.Fatalf("Dispatch 应在 ~50ms 超时处返回，实际 %v", sp.returned)
	}
	if rec.calls != 1 || ack.acks != 1 {
		t.Fatalf("超时应按失败重试处理；republish=%d acks=%d", rec.calls, ack.acks)
	}
}

// 关机取消不应传染在途任务：外层 ctx cancel 后 Dispatch 仍要拿到可用 ctx。
func TestHandleIgnoresParentCancellation(t *testing.T) {
	ack := &fakeAck{}
	c := newTestConsumer(3, &publishRecorder{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sp := &stubDispatcher{}
	c.handle(ctx, sp, delivery(ack, false, nil))
	if !sp.hasDeadline {
		t.Fatal("期望仍带单条超时 deadline")
	}
	if ack.acks != 1 {
		t.Fatalf("外层已取消但任务成功，应 Ack；acks=%d", ack.acks)
	}
}
