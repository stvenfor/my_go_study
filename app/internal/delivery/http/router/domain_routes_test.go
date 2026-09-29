// domain_routes_test.go 已切片域的路由集回归护栏。
//
// 批 1 对照基线是 router 包里的 access_routes.go / cash_wallet_routes.go /
// home_todo_routes.go / points_routes.go（2026-09-29 切片前）；
// 批 2 对照基线是 used_car_order_routes.go / new_car_follow_routes.go /
// after_sales_zone_routes.go / deal_invoice_routes.go /
// purchase_calculator_routes.go（同一提交）；
// 批 3 对照基线是 mall_routes.go / membership_routes.go / payment_routes.go /
// community_routes.go / short_video_routes.go / transaction_routes.go（同一提交）。
// 逐个 verb+path 由脚本从基线文件机械提取后固化在这里 —— 域切片最容易悄悄丢
// 路由，这个测试保证「切片前后注册的路由完全一致」。
package router

import (
	"slices"
	"sort"
	"strings"
	"testing"

	"app/internal/access"
	"app/internal/after_sales"
	"app/internal/community"
	"app/internal/deal_invoice"
	"app/internal/home"
	"app/internal/mall"
	"app/internal/membership"
	"app/internal/new_car_follow"
	"app/internal/payment"
	"app/internal/points"
	"app/internal/purchase_calculator"
	"app/internal/shared"
	"app/internal/short_video"
	"app/internal/transaction"
	"app/internal/used_car"
	"app/internal/wallet"
	"common/config"
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

// batch3Want 批 3（电商 + 社区 6 域）基线，50 条。
// 脚本从 mall_routes.go / membership_routes.go / payment_routes.go /
// community_routes.go / short_video_routes.go / transaction_routes.go 机械提取。
// 注意 transaction 是两组（/transactions legacy + /transactions/manage 统一格式）。
const batch3Want = `DELETE /api/v1/community/posts/:id
DELETE /api/v1/community/posts/:id/like
DELETE /api/v1/community/users/:id/follow
DELETE /api/v1/short-videos/:id
DELETE /api/v1/short-videos/:id/like
DELETE /api/v1/transactions/:id
DELETE /api/v1/transactions/manage/:id
GET /api/v1/community/posts
GET /api/v1/community/posts/:id/comments
GET /api/v1/community/search
GET /api/v1/community/topics
GET /api/v1/community/topics/search
GET /api/v1/mall/cart
GET /api/v1/mall/orders
GET /api/v1/mall/orders/:order_id
GET /api/v1/mall/stores/:store_id/products
GET /api/v1/mall/stores/:store_id/products/:product_id
GET /api/v1/membership/me
GET /api/v1/short-videos
GET /api/v1/short-videos/:id
GET /api/v1/short-videos/profile
GET /api/v1/transactions
GET /api/v1/transactions/:id
GET /api/v1/transactions/manage
GET /api/v1/transactions/manage/:id
GET /api/v1/transactions/search
POST /api/v1/community/posts
POST /api/v1/community/posts/:id/comments
POST /api/v1/community/posts/:id/like
POST /api/v1/community/users/:id/follow
POST /api/v1/mall/cart
POST /api/v1/mall/orders
POST /api/v1/mall/orders/:order_id/cancel
POST /api/v1/mall/orders/:order_id/pay
POST /api/v1/mall/products
POST /api/v1/mall/products/:product_id/skus
POST /api/v1/mall/skus/:sku_id/codes
POST /api/v1/membership/apple/verify
POST /api/v1/membership/buyout
POST /api/v1/membership/buyout/confirm
POST /api/v1/membership/huawei/notifications
POST /api/v1/membership/huawei/verify
POST /api/v1/payments/prepay
POST /api/v1/short-videos
POST /api/v1/short-videos/:id/like
POST /api/v1/short-videos/:id/view
POST /api/v1/transactions
POST /api/v1/transactions/manage
PUT /api/v1/transactions/:id
PUT /api/v1/transactions/manage/:id`

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

// batch3Routers 批 3 六域。全部为受保护域（切片前 mall/membership/community/
// short_video 由 IsLocalProvider 门控、transaction/payment 由 businessAuth 门控），
// 都要 sessionAuth 才会注册，故统一传 sessionAuth。
func batch3Routers(sessionAuth gin.HandlerFunc) []shared.Registrable {
	return []shared.Registrable{
		mall.NewRouter(mall.NewService(nil, nil, nil, nil), sessionAuth, nil),
		membership.NewRouter(membership.NewService(nil, nil, nil, nil, nil, ""), sessionAuth, nil),
		community.NewRouter(community.NewService(nil, nil, nil), sessionAuth, nil),
		short_video.NewRouter(short_video.NewService(nil, 0), sessionAuth, nil),
		transaction.NewRouter(transaction.NewService(nil), sessionAuth, nil),
		payment.NewRouter(payment.NewService(config.ThirdPartyConfig{}), sessionAuth, nil),
	}
}

// TestBatch3DomainRoutes 批 3 六域注册的路由必须与切片前基线逐条一致（50 条）。
func TestBatch3DomainRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	for _, r := range batch3Routers(func(c *gin.Context) { c.Next() }) {
		r.Register(engine)
	}

	got := make([]string, 0, len(engine.Routes()))
	for _, r := range engine.Routes() {
		got = append(got, r.Method+" "+r.Path)
	}
	sort.Strings(got)

	want := splitLines(batch3Want)
	sort.Strings(want)

	if !slices.Equal(got, want) {
		t.Fatalf("路由集与基线不一致\n got=%d want=%d\n缺失: %v\n多余: %v",
			len(got), len(want), missing(want, got), missing(got, want))
	}
}

// TestBatch3DomainRoutesSkippedWithoutSessionAuth sessionAuth 为 nil 时，
// 批 3 六域都不应注册任何路由。
func TestBatch3DomainRoutesSkippedWithoutSessionAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	for _, r := range batch3Routers(nil) {
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
