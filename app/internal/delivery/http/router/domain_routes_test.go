// domain_routes_test.go 已切片域的路由集回归护栏。
//
// 批 1 对照基线是 router 包里的 access_routes.go / cash_wallet_routes.go /
// home_todo_routes.go / points_routes.go（2026-09-29 切片前）；
// 批 2 对照基线是 used_car_order_routes.go / new_car_follow_routes.go /
// after_sales_zone_routes.go / deal_invoice_routes.go /
// purchase_calculator_routes.go（同一提交）。逐个 verb+path 由脚本从基线
// 文件机械提取后固化在这里 —— 域切片最容易悄悄丢路由，这个测试保证
// 「切片前后注册的路由完全一致」。
package router

import (
	"slices"
	"sort"
	"strings"
	"testing"

	"app/internal/access"
	"app/internal/after_sales"
	"app/internal/deal_invoice"
	"app/internal/home"
	"app/internal/new_car_follow"
	"app/internal/points"
	"app/internal/purchase_calculator"
	"app/internal/shared"
	"app/internal/used_car"
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

// batch2Want 批 2（汽车业务 5 域）基线，25 条。
const batch2Want = `GET /api/v1/after-sales/pending-appointments
GET /api/v1/after-sales/records
GET /api/v1/after-sales/records/:record_id
GET /api/v1/deal-invoices
GET /api/v1/deal-invoices/:invoice_id
GET /api/v1/deal-invoices/customers
GET /api/v1/deal-invoices/summary
GET /api/v1/new-car-follow-files
GET /api/v1/new-car-follow-files/:file_id
GET /api/v1/new-car-follow-files/:file_id/logs
GET /api/v1/new-car-follow-files/customers
GET /api/v1/new-car-follow-files/summary
GET /api/v1/purchase-calculator/products
GET /api/v1/used-car-orders
GET /api/v1/used-car-orders/:order_id
GET /api/v1/used-car-orders/customers
GET /api/v1/used-car-orders/summary
PATCH /api/v1/deal-invoices/:invoice_id/resubmit
PATCH /api/v1/new-car-follow-files/:file_id
POST /api/v1/after-sales/records
POST /api/v1/deal-invoices
POST /api/v1/new-car-follow-files
POST /api/v1/new-car-follow-files/:file_id/logs
POST /api/v1/purchase-calculator/quote
POST /api/v1/used-car-orders`

func batch1Routers(sessionAuth gin.HandlerFunc) []shared.Registrable {
	return []shared.Registrable{
		access.NewRouter(access.NewService(nil), sessionAuth, nil),
		home.NewRouter(home.NewService(nil, nil), sessionAuth, nil),
		points.NewRouter(points.NewService(nil, nil), sessionAuth, nil),
		wallet.NewRouter(wallet.NewService(nil), sessionAuth, nil),
	}
}

// batch2AuthedRouters 批 2 里需要身份的四域（purchase_calculator 是公开域，单列）。
func batch2AuthedRouters(sessionAuth gin.HandlerFunc) []shared.Registrable {
	return []shared.Registrable{
		used_car.NewRouter(used_car.NewService(nil, nil), sessionAuth, nil),
		new_car_follow.NewRouter(new_car_follow.NewService(nil, nil), sessionAuth, nil),
		after_sales.NewRouter(after_sales.NewService(nil, nil), sessionAuth, nil),
		deal_invoice.NewRouter(deal_invoice.NewService(nil, nil), sessionAuth, nil),
	}
}

// TestBatch2PublicDomainRoutes 公开域不经 SessionAuth 也应注册（2 条）。
func TestBatch2PublicDomainRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	purchase_calculator.NewRouter(purchase_calculator.NewService(nil)).Register(engine)

	got := make([]string, 0, len(engine.Routes()))
	for _, r := range engine.Routes() {
		got = append(got, r.Method+" "+r.Path)
	}
	sort.Strings(got)

	want := []string{
		"GET /api/v1/purchase-calculator/products",
		"POST /api/v1/purchase-calculator/quote",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("公开域路由不一致\n got=%v want=%v", got, want)
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

// TestBatch2DomainRoutes 汽车业务四域 + 公开域注册的路由必须与切片前基线逐条一致。
func TestBatch2DomainRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	for _, r := range batch2AuthedRouters(func(c *gin.Context) { c.Next() }) {
		r.Register(engine)
	}
	purchase_calculator.NewRouter(purchase_calculator.NewService(nil)).Register(engine)

	got := make([]string, 0, len(engine.Routes()))
	for _, r := range engine.Routes() {
		got = append(got, r.Method+" "+r.Path)
	}
	sort.Strings(got)

	want := splitLines(batch2Want)
	sort.Strings(want)

	if !slices.Equal(got, want) {
		t.Fatalf("路由集与基线不一致\n got=%d want=%d\n缺失: %v\n多余: %v",
			len(got), len(want), missing(want, got), missing(got, want))
	}
}

// TestBatch2DomainRoutesSkippedWithoutSessionAuth sessionAuth 为 nil 时，
// 需身份的四域不应注册任何路由。
func TestBatch2DomainRoutesSkippedWithoutSessionAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	for _, r := range batch2AuthedRouters(nil) {
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
