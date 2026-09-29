// domain_routes_test.go 批 1 切片域的路由集回归护栏。
//
// 对照基线是 router 包里的 access_routes.go / cash_wallet_routes.go /
// home_todo_routes.go / points_routes.go（2026-09-29 切片前），
// 逐个 verb+path 提出来后固化在这里 —— 域切片最容易悄悄丢路由，
// 这个测试保证「切片前后注册的路由完全一致」。
package router

import (
	"slices"
	"sort"
	"strings"
	"testing"

	"app/internal/access"
	"app/internal/home"
	"app/internal/points"
	"app/internal/shared"
	"app/internal/wallet"
	"github.com/gin-gonic/gin"
)

const batch1Want = `DELETE /api/v1/roles/assignments
DELETE /api/v1/stores/:store_id/members/:user_id
DELETE /api/v1/wallet/cards/:card_id
GET /api/v1/home/after-sales-appointments
GET /api/v1/home/follow-up-customers
GET /api/v1/home/join-applications
GET /api/v1/home/store-review-orders
GET /api/v1/home/todo-cards
GET /api/v1/me/permissions
GET /api/v1/points/status
GET /api/v1/points/tasks
GET /api/v1/wallet
GET /api/v1/wallet/cards
GET /api/v1/wallet/ledger
POST /api/v1/home/join-applications/:application_id/approve
POST /api/v1/home/join-applications/:application_id/reject
POST /api/v1/points/check-in
POST /api/v1/points/tasks/:code/claim
POST /api/v1/roles/assignments
POST /api/v1/stores
POST /api/v1/stores/:store_id/join-applications
POST /api/v1/stores/:store_id/members
POST /api/v1/wallet/cards
POST /api/v1/wallet/cards/:card_id/default
POST /api/v1/wallet/recharge`

func batch1Routers(sessionAuth gin.HandlerFunc) []shared.Registrable {
	return []shared.Registrable{
		access.NewRouter(access.NewService(nil), sessionAuth, nil),
		home.NewRouter(home.NewService(nil, nil), sessionAuth, nil),
		points.NewRouter(points.NewService(nil, nil), sessionAuth, nil),
		wallet.NewRouter(wallet.NewService(nil), sessionAuth, nil),
	}
}

// TestBatch1DomainRoutes 四域注册的路由必须与切片前基线逐条一致。
func TestBatch1DomainRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	for _, r := range batch1Routers(func(c *gin.Context) { c.Next() }) {
		r.Register(engine)
	}

	got := make([]string, 0, len(engine.Routes()))
	for _, r := range engine.Routes() {
		got = append(got, r.Method+" "+r.Path)
	}
	sort.Strings(got)

	want := splitLines(batch1Want)
	sort.Strings(want)

	if !slices.Equal(got, want) {
		t.Fatalf("路由集与基线不一致\n got=%d want=%d\n缺失: %v\n多余: %v",
			len(got), len(want), missing(want, got), missing(got, want))
	}
}

// TestBatch1DomainRoutesSkippedWithoutSessionAuth sessionAuth 为 nil 时不应注册任何路由。
func TestBatch1DomainRoutesSkippedWithoutSessionAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	for _, r := range batch1Routers(nil) {
		r.Register(engine)
	}
	if n := len(engine.Routes()); n != 0 {
		t.Fatalf("sessionAuth 为 nil 时不应注册路由，实际 %d 条", n)
	}
}

func splitLines(s string) []string {
	return strings.Split(strings.TrimSpace(s), "\n")
}

func missing(want, got []string) []string {
	var out []string
	for _, w := range want {
		if !slices.Contains(got, w) {
			out = append(out, w)
		}
	}
	return out
}
