// prometheus.go 记录低基数 HTTP 指标（route 用 Gin FullPath 模板）。
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"common/metrics"
)

// PrometheusHTTP 统计请求次数与延迟；跳过 /metrics 与 /health 避免自扰。
func PrometheusHTTP() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/metrics" || path == "/health" {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		method := c.Request.Method
		status := c.Writer.Status()
		metrics.HTTPRequestsTotal.WithLabelValues(method, route, metrics.StatusClass(status)).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(method, route).Observe(time.Since(start).Seconds())
	}
}
