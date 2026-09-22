// mall.go 门店商城：商品、购物车、订单、本地模拟支付。
package entity

import (
	"encoding/json"
	"time"
)

const (
	MallKindPhysical int16 = 0
	MallKindVirtual  int16 = 1

	MallProductDraft    int16 = 0
	MallProductOnShelf  int16 = 1
	MallProductOffShelf int16 = 2

	MallSKUOff int16 = 0
	MallSKUOn  int16 = 1

	MallDeliverRedeemCode int16 = 0
	MallDeliverContentURL int16 = 1

	MallCodeUnused int16 = 0
	MallCodeIssued int16 = 1
	MallCodeVoid   int16 = 2

	MallOrderUnpaid    int16 = 0
	MallOrderPaid      int16 = 1
	MallOrderFulfilled int16 = 2
	MallOrderCancelled int16 = 3
	MallOrderClosed    int16 = 4

	// MallOrderPayTTL 待支付订单支付窗口。超时后惰性取消为已取消。
	MallOrderPayTTL = 15 * time.Minute

	MallPayAlipay    int16 = 1
	MallPayWeChat    int16 = 2
	MallPayAppleIAP  int16 = 3
	MallPayHuaweiIAP int16 = 4
	MallPayPoints    int16 = 5

	MallPayModeCNY    int16 = 1
	MallPayModePoints int16 = 2
	MallPayModeMixed  int16 = 3

	MallPaymentPending int16 = 0
	MallPaymentSuccess int16 = 1
	MallPaymentFailed  int16 = 2

	PermMallCatalogWrite = "mall.catalog.write"
)

const (
	WysMallCategoryTable    = "wys_mall_category"
	WysMallProductTable     = "wys_mall_product"
	WysMallSKUTable         = "wys_mall_sku"
	WysMallVirtualCodeTable = "wys_mall_virtual_code"
	WysMallCartItemTable    = "wys_mall_cart_item"
	WysMallOrderTable       = "wys_mall_order"
	WysMallOrderItemTable   = "wys_mall_order_item"
	WysMallOrderLogTable    = "wys_mall_order_log"
	WysMallPaymentTable     = "wys_mall_payment"
	WysMallRefundTable      = "wys_mall_refund"
	WysMallAuditLogTable    = "wys_mall_audit_log"
)

// ValidMallPaymentChannel 本地模拟支付允许的渠道（含积分）。
func ValidMallPaymentChannel(ch int16) bool {
	switch ch {
	case MallPayAlipay, MallPayWeChat, MallPayAppleIAP, MallPayHuaweiIAP, MallPayPoints:
		return true
	default:
		return false
	}
}

// ValidMallCNYChannel 人民币渠道（不含积分）。
func ValidMallCNYChannel(ch int16) bool {
	switch ch {
	case MallPayAlipay, MallPayWeChat, MallPayAppleIAP, MallPayHuaweiIAP:
		return true
	default:
		return false
	}
}

// MallSKUPaymentMode 由 CNY 价与积分价派生支付方式。
func MallSKUPaymentMode(priceCNY string, pricePoints int64) int16 {
	cnyZero := IsZeroMoney(priceCNY)
	ptsZero := pricePoints <= 0
	switch {
	case !cnyZero && ptsZero:
		return MallPayModeCNY
	case cnyZero && !ptsZero:
		return MallPayModePoints
	case !cnyZero && !ptsZero:
		return MallPayModeMixed
	default:
		return MallPayModeCNY
	}
}

// IsZeroMoney 判断金额字符串是否为零。
func IsZeroMoney(s string) bool {
	s = trimSpaceMoney(s)
	if s == "" || s == "0" || s == "0.0" || s == "0.00" {
		return true
	}
	return false
}

func trimSpaceMoney(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\t') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t') {
		s = s[:len(s)-1]
	}
	return s
}

// MallOrderPayDeadline 待支付截止时刻；非待支付返回 nil。
func MallOrderPayDeadline(status int16, createdAt time.Time) *time.Time {
	if status != MallOrderUnpaid {
		return nil
	}
	t := createdAt.Add(MallOrderPayTTL)
	return &t
}

// MallOrderPayExpired 待支付是否已过支付窗口。
func MallOrderPayExpired(status int16, createdAt, now time.Time) bool {
	if status != MallOrderUnpaid {
		return false
	}
	return !now.Before(createdAt.Add(MallOrderPayTTL))
}

// WysMallCategory 门店类目。
type WysMallCategory struct {
	CategoryID int64      `json:"category_id" gorm:"primaryKey"`
	StoreID    int        `json:"store_id"`
	ParentID   *int64     `json:"parent_id,omitempty"`
	Name       string     `json:"name"`
	Sort       int        `json:"sort"`
	Status     int16      `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at,omitempty"`
}

func (WysMallCategory) TableName() string { return WysMallCategoryTable }

// WysMallProduct 商品 SPU。
type WysMallProduct struct {
	ProductID   int64      `json:"product_id" gorm:"primaryKey"`
	StoreID     int        `json:"store_id"`
	CategoryID  *int64     `json:"category_id,omitempty"`
	Kind        int16      `json:"kind"`
	Title       string     `json:"title"`
	CoverURL    *string    `json:"cover_url,omitempty"`
	CoverAspect float64    `json:"cover_aspect" gorm:"type:numeric(4,2);default:1"`
	Status      int16      `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

func (WysMallProduct) TableName() string { return WysMallProductTable }

// MallShelfItem 在售列表扁平行（瀑布流卡片）。
type MallShelfItem struct {
	ProductID   int64   `json:"product_id"`
	SKUID       int64   `json:"sku_id"`
	Title       string  `json:"title"`
	CoverURL    string  `json:"cover_url"`
	CoverAspect float64 `json:"cover_aspect"`
	Kind        int16   `json:"kind"`
	Price       string  `json:"price"`
	PricePoints int64   `json:"price_points"`
	PaymentMode int16   `json:"payment_mode"`
	Subtitle    string  `json:"subtitle,omitempty"`
}

// MallProductWithSKUs 买家列表用（兼容旧结构）。
type MallProductWithSKUs struct {
	Product WysMallProduct `json:"product"`
	SKUs    []WysMallSKU   `json:"skus"`
}

// MallSKUOffer 买家可见规格。不含发放地址。
type MallSKUOffer struct {
	SKUID       int64           `json:"sku_id"`
	Title       string          `json:"title"`
	Specs       json.RawMessage `json:"specs"`
	Price       string          `json:"price"`
	PricePoints int64           `json:"price_points"`
	PaymentMode int16           `json:"payment_mode"`
	StockQty    int             `json:"stock_qty"`
	DeliverType *int16          `json:"deliver_type,omitempty"`
}

// MallProductDetail 在售商品详情。
type MallProductDetail struct {
	Product WysMallProduct `json:"product"`
	SKUs    []MallSKUOffer `json:"skus"`
}

// WysMallSKU 规格与价格。金额用字符串承载 numeric，避免 float。
type WysMallSKU struct {
	SKUID       int64          `json:"sku_id" gorm:"column:sku_id;primaryKey"`
	ProductID   int64          `json:"product_id"`
	SKUCode     string         `json:"sku_code" gorm:"column:sku_code"`
	Title       string         `json:"title"`
	Specs       json.RawMessage `json:"specs" gorm:"type:jsonb"`
	Price       string          `json:"price" gorm:"type:numeric(10,2)"` // CNY
	PricePoints int64           `json:"price_points" gorm:"column:price_points"`
	StockQty    int            `json:"stock_qty"`
	Version     int            `json:"version"`
	Status      int16          `json:"status"`
	DeliverType *int16         `json:"deliver_type,omitempty"`
	ContentURL  *string        `json:"content_url,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   *time.Time     `json:"deleted_at,omitempty"`
}

func (WysMallSKU) TableName() string { return WysMallSKUTable }

// WysMallVirtualCode 兑换码。
type WysMallVirtualCode struct {
	CodeID      int64     `json:"code_id" gorm:"primaryKey"`
	SKUID       int64     `json:"sku_id" gorm:"column:sku_id"`
	Code        string    `json:"code"`
	Status      int16     `json:"status"`
	OrderItemID *int64    `json:"order_item_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func (WysMallVirtualCode) TableName() string { return WysMallVirtualCodeTable }

// WysMallCartItem 购物车行。
type WysMallCartItem struct {
	UserID    string    `json:"user_id" gorm:"primaryKey;size:64"`
	SKUID     int64     `json:"sku_id" gorm:"column:sku_id;primaryKey"`
	Qty       int       `json:"qty"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (WysMallCartItem) TableName() string { return WysMallCartItemTable }

// WysMallOrder 订单。
type WysMallOrder struct {
	OrderID         int64      `json:"order_id" gorm:"primaryKey"`
	OrderNo         string     `json:"order_no"`
	StoreID         int        `json:"store_id"`
	BuyerUserID     string     `json:"buyer_user_id" gorm:"size:64"`
	IdempotencyKey  string     `json:"idempotency_key"`
	Status          int16      `json:"status"`
	PaymentChannel  *int16     `json:"payment_channel,omitempty"`
	Amount          string     `json:"amount" gorm:"type:numeric(10,2)"`
	TotalPoints     int64      `json:"total_points"`
	PaymentMode     int16      `json:"payment_mode"`
	ReceiverName    *string    `json:"receiver_name,omitempty"`
	ReceiverPhone   *string    `json:"receiver_phone,omitempty"`
	ReceiverAddress *string    `json:"receiver_address,omitempty"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (WysMallOrder) TableName() string { return WysMallOrderTable }

// WysMallOrderItem 订单行快照。
type WysMallOrderItem struct {
	ItemID       int64          `json:"item_id" gorm:"primaryKey"`
	OrderID      int64          `json:"order_id"`
	SKUID        int64          `json:"sku_id" gorm:"column:sku_id"`
	ProductID    int64          `json:"product_id"`
	Kind         int16          `json:"kind"`
	ProductTitle string         `json:"product_title"`
	CoverURL     *string        `json:"cover_url,omitempty"`
	Specs        json.RawMessage `json:"specs" gorm:"type:jsonb"`
	Price        string          `json:"price" gorm:"type:numeric(10,2)"`
	PricePoints  int64           `json:"price_points"`
	Qty          int             `json:"qty"`
	LineAmount   string          `json:"line_amount" gorm:"type:numeric(10,2)"`
	LinePoints   int64           `json:"line_points"`
	ContentURL   *string        `json:"content_url,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

func (WysMallOrderItem) TableName() string { return WysMallOrderItemTable }

// WysMallOrderLog 状态变更。
type WysMallOrderLog struct {
	LogID       int64     `json:"log_id" gorm:"primaryKey"`
	OrderID     int64     `json:"order_id"`
	FromStatus  *int16    `json:"from_status,omitempty"`
	ToStatus    int16     `json:"to_status"`
	ActorUserID string    `json:"actor_user_id" gorm:"size:64"`
	CreatedAt   time.Time `json:"created_at"`
}

func (WysMallOrderLog) TableName() string { return WysMallOrderLogTable }

// WysMallPayment 支付单。
type WysMallPayment struct {
	PaymentID      int64          `json:"payment_id" gorm:"primaryKey"`
	PaymentNo      string         `json:"payment_no"`
	OrderID        int64          `json:"order_id"`
	UserID         string         `json:"user_id" gorm:"size:64"`
	PaymentChannel int16          `json:"payment_channel"`
	Amount         string         `json:"amount" gorm:"type:numeric(10,2)"`
	Status         int16          `json:"status"`
	ChannelTradeNo *string        `json:"channel_trade_no,omitempty"`
	ChannelPayload json.RawMessage `json:"channel_payload" gorm:"type:jsonb"`
	PaidAt         *time.Time      `json:"paid_at,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func (WysMallPayment) TableName() string { return WysMallPaymentTable }

// WysMallAuditLog 审计。
type WysMallAuditLog struct {
	AuditID     int64           `json:"audit_id" gorm:"primaryKey"`
	ActorUserID string          `json:"actor_user_id" gorm:"size:64"`
	Action      string          `json:"action"`
	TargetType  string          `json:"target_type"`
	TargetID    string          `json:"target_id"`
	Detail      json.RawMessage `json:"detail" gorm:"type:jsonb"`
	CreatedAt   time.Time       `json:"created_at"`
}

func (WysMallAuditLog) TableName() string { return WysMallAuditLogTable }

// MallOrderDetail 订单详情。
type MallOrderDetail struct {
	Order         WysMallOrder       `json:"order"`
	Items         []WysMallOrderItem `json:"items"`
	Payments      []WysMallPayment   `json:"payments,omitempty"`
	PayDeadlineAt *time.Time         `json:"pay_deadline_at,omitempty"`
}

// MallOrderListItem 买家订单列表行（含行快照，够画卡片）。
type MallOrderListItem struct {
	OrderID       int64              `json:"order_id"`
	OrderNo       string             `json:"order_no"`
	StoreID       int                `json:"store_id"`
	Status        int16              `json:"status"`
	Amount        string             `json:"amount"`
	CreatedAt     time.Time          `json:"created_at"`
	PayDeadlineAt *time.Time         `json:"pay_deadline_at,omitempty"`
	Items         []WysMallOrderItem `json:"items"`
}
