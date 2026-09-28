package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stvenfor/my_go_study/pkg/metrics"
	dto "github.com/prometheus/client_model/go"
)

func TestPrometheusHTTPIncrementsCounter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(PrometheusHTTP())
	r.GET("/api/v1/ping", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	before := counterValue(t, "GET", "/api/v1/ping", "2xx")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	after := counterValue(t, "GET", "/api/v1/ping", "2xx")
	if after < before+1 {
		t.Fatalf("counter did not increase: before=%v after=%v", before, after)
	}
}

func counterValue(t *testing.T, method, route, class string) float64 {
	t.Helper()
	m := &dto.Metric{}
	if err := metrics.HTTPRequestsTotal.WithLabelValues(method, route, class).Write(m); err != nil {
		t.Fatal(err)
	}
	return m.GetCounter().GetValue()
}
