// metrics_access_test.go — harden-h7：/metrics 只本机可达的可跑证明。
package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestMetricsPeerAllowed(t *testing.T) {
	allow := parseMetricsAllowCIDRs("172.16.0.0/12,10.0.0.0/8", nil)
	cases := []struct {
		remote string
		want   bool
	}{
		{"127.0.0.1:50000", true},     // 本机 IPv4
		{"[::1]:50000", true},         // 本机 IPv6
		{"192.168.2.56:50000", false}, // 局域网 IP
		{"192.168.65.1:50000", false}, // docker-proxy 网关：经端口映射的外部访问
		{"172.18.0.5:50000", true},    // Docker 内网 Prometheus（显式放行）
		{"10.1.2.3:50000", true},      // 自定义 Docker 网段（显式放行）
		{"8.8.8.8:50000", false},      // 公网
		{"", false},                   // 拿不到对端地址不放行
		{"not-an-ip:50000", false},
	}
	for _, tc := range cases {
		if got := metricsPeerAllowed(tc.remote, allow); got != tc.want {
			t.Errorf("metricsPeerAllowed(%q)=%v want %v", tc.remote, got, tc.want)
		}
	}

	// 默认（无 METRICS_ALLOW_CIDRS）：只放行 loopback，Docker 内网也不开。
	none := parseMetricsAllowCIDRs("", nil)
	if metricsPeerAllowed("172.18.0.5:50000", none) {
		t.Fatal("默认未配置放行网段时不应放行 Docker 内网 IP")
	}
	if !metricsPeerAllowed("127.0.0.1:50000", none) {
		t.Fatal("默认必须放行 loopback")
	}
}

// 走真实 Setup：证明守卫确实挂在 /metrics 路由上，且 127.0.0.1 仍可取、LAN IP 取不到。
func TestSetupMetricsLoopbackOnly(t *testing.T) {
	t.Setenv("METRICS_ALLOW_CIDRS", "")
	gin.SetMode(gin.TestMode)
	r := Setup(Options{Log: zap.NewNop(), Mode: gin.TestMode})

	cases := []struct {
		name   string
		remote string
		xff    string
		want   int
	}{
		{"loopback", "127.0.0.1:50000", "", http.StatusOK},
		{"loopback-v6", "[::1]:50000", "", http.StatusOK},
		{"lan-ip", "192.168.2.56:50000", "", http.StatusForbidden},
		{"docker-gateway", "192.168.65.1:50000", "", http.StatusForbidden},
		{"spoofed-xff", "192.168.2.56:50000", "127.0.0.1", http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			req.RemoteAddr = tc.remote
			if tc.xff != "" {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("remote=%q xff=%q 得到 %d，期望 %d", tc.remote, tc.xff, w.Code, tc.want)
			}
		})
	}
}

// compose 注入 Docker 网段后，Prometheus 的 app:8080/metrics 抓取仍可达。
func TestSetupMetricsAllowsDockerInternalScrape(t *testing.T) {
	t.Setenv("METRICS_ALLOW_CIDRS", "172.16.0.0/12,10.0.0.0/8")
	gin.SetMode(gin.TestMode)
	r := Setup(Options{Log: zap.NewNop(), Mode: gin.TestMode})

	for _, remote := range []string{"172.18.0.5:50000", "10.1.2.3:50000"} {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		req.RemoteAddr = remote
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("Docker 内网抓取 remote=%q 得到 %d，期望 200", remote, w.Code)
		}
	}

	// 放行内网不等于放行经 docker-proxy 的局域网访问。
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.RemoteAddr = "192.168.65.1:50000"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("docker-proxy 网关应被拒绝，实得 %d", w.Code)
	}
}
