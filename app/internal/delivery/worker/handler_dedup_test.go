// handler_dedup_test.go 锁定「入队失败后删去重键失败」不再被静默吞掉（清单条目 22）。
package worker

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"

	"common/metrics"
	dto "github.com/prometheus/client_model/go"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

const dedupDelFailedName = "push_dedup_del_failed"

func degradedCounterValue(t *testing.T, name string) float64 {
	t.Helper()
	m := &dto.Metric{}
	if err := metrics.DegradedPathsTotal.WithLabelValues(name).Write(m); err != nil {
		t.Fatal(err)
	}
	return m.GetCounter().GetValue()
}

// TestCleanupDedupAfterEnqueueError_DelFailurePropagates 断言删键失败时：
//  1. Handler 返回 error（原始 enqueue error 仍可被 errors.Is 命中，且文案反映 Del 失败）；
//  2. degraded_path_total{push_dedup_del_failed} +1；
//  3. 打了一条 Error 日志。
//
// Del 失败用强制 dial 失败制造真实错误，不依赖本机端口占用、不引入新 mock 依赖。
// handleBroadcastNotify 的入队失败分支直接 `return h.cleanupDedupAfterEnqueueError(...)`，
// 故本方法返回 error 即 handleBroadcastNotify 返回 error（交既有重试/死信）。
func TestCleanupDedupAfterEnqueueError_DelFailurePropagates(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:1",
		Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return nil, errors.New("forced dial failure")
		},
		MaxRetries: -1,
	})
	defer rdb.Close()

	core, logs := observer.New(zap.ErrorLevel)
	h := &Handler{redis: rdb, log: zap.New(core)}

	enqueueErr := errors.New("forced enqueue failure")
	before := degradedCounterValue(t, dedupDelFailedName)

	err := h.cleanupDedupAfterEnqueueError(context.Background(), "scheduler:dedup:u1", "u1", enqueueErr)
	if err == nil {
		t.Fatal("删键失败时 Handler 必须返回 error，不能静默留下坏的去重键")
	}
	if !errors.Is(err, enqueueErr) {
		t.Fatalf("返回的 error 应保留原始 enqueue error: %v", err)
	}
	if !strings.Contains(err.Error(), "cleanup dedup key") {
		t.Fatalf("返回的 error 应反映 Del 失败: %v", err)
	}
	if after := degradedCounterValue(t, dedupDelFailedName); after-before != 1 {
		t.Fatalf("degraded_path_total{%s} 应 +1, before=%v after=%v", dedupDelFailedName, before, after)
	}
	if got := logs.FilterMessage("cleanup dedup key failed after enqueue failure").Len(); got != 1 {
		t.Fatalf("应打 1 条 Error 日志, got %d: %+v", got, logs.All())
	}
}
