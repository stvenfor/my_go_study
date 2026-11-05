// Package metrics registers Prometheus collectors for the HTTP API.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Registry is the process-wide registry used by /metrics and HTTP middleware.
var Registry = prometheus.NewRegistry()

var (
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests by method, route template, and status class.",
		},
		[]string{"method", "route", "status_class"},
	)
	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency by method and route template.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "route"},
	)
	ESTransactionIndexFailures = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "es_transaction_index_failures_total",
			Help: "Failed best-effort Elasticsearch index/delete for transactions.",
		},
	)
	// DegradedPathsTotal 统计「能返回成功但没做真事」的降级路径命中次数，
	// 标签取值与 docs/placeholder-inventory.md 的条目名一一对应。
	DegradedPathsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "degraded_path_total",
			Help: "Hits on placeholder/degraded code paths registered in docs/placeholder-inventory.md, by name.",
		},
		[]string{"name"},
	)
)

func init() {
	Registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		HTTPRequestsTotal,
		HTTPRequestDuration,
		ESTransactionIndexFailures,
		DegradedPathsTotal,
	)
}

// Handler serves Prometheus text exposition from Registry.
func Handler() http.Handler {
	return promhttp.HandlerFor(Registry, promhttp.HandlerOpts{EnableOpenMetrics: false})
}

// depthCollector 把「拉取式」的第三方队列长度暴露成 gauge。
//
// 之所以不用 GaugeFunc：GaugeFunc 无法表达采集失败，只能返回 0，
// 而 DLQ 读不到 ≠ 死信为空 —— 这里选择「失败就不出数」，让告警侧
// 靠 absent_metric 判断。
type depthCollector struct {
	desc *prometheus.Desc
	fn   func() (float64, error)
}

func (c *depthCollector) Describe(ch chan<- *prometheus.Desc) { ch <- c.desc }

func (c *depthCollector) Collect(ch chan<- prometheus.Metric) {
	v, err := c.fn()
	if err != nil {
		return
	}
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, v)
}

// NewDepthCollector 构造一个 gauge Collector，供调用方 Registry.MustRegister。
//
// labels 用 map[string]string 而非 prometheus.Labels：调用方（cmd 装配层）
// 不必为此再 import prometheus。
func NewDepthCollector(name, help string, labels map[string]string, fn func() (float64, error)) prometheus.Collector {
	return &depthCollector{
		desc: prometheus.NewDesc(name, help, nil, prometheus.Labels(labels)),
		fn:   fn,
	}
}

// StatusClass maps an HTTP status to a low-cardinality label.
func StatusClass(code int) string {
	switch {
	case code < 200:
		return "1xx"
	case code < 300:
		return "2xx"
	case code < 400:
		return "3xx"
	case code < 500:
		return "4xx"
	default:
		return "5xx"
	}
}
