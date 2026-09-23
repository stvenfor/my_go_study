package entity

import "time"

const (
	WysMembershipEntitlementTable = "wys_membership_entitlement"
	WysMembershipOrderTable       = "wys_membership_order"

	MembershipTierSVIP   = "svip"
	MembershipTierAISVIP = "ai_svip"

	MembershipChannelWeChat  = "wechat"
	MembershipChannelAlipay  = "alipay"
	MembershipChannelBalance = "balance"
	MembershipChannelHuawei  = "huawei"
	MembershipChannelApple   = "apple"

	MembershipOrderPending int16 = 0
	MembershipOrderPaid    int16 = 1
)

// WysMembershipEntitlement 每用户每档一条权益。
type WysMembershipEntitlement struct {
	UserID                     string    `json:"user_id" gorm:"primaryKey"`
	Tier                       string    `json:"tier" gorm:"primaryKey"`
	ExpiresAt                  time.Time `json:"expires_at"`
	SourceChannel              string    `json:"source_channel"`
	HuaweiPurchaseToken        string    `json:"huawei_purchase_token,omitempty"`
	HuaweiSubscriptionID       string    `json:"huawei_subscription_id,omitempty"`
	AppleOriginalTransactionID string    `json:"apple_original_transaction_id,omitempty"`
	UpdatedAt                  time.Time `json:"updated_at"`
}

func (WysMembershipEntitlement) TableName() string { return WysMembershipEntitlementTable }

// Active 当前是否在有效期内。
func (e WysMembershipEntitlement) Active(now time.Time) bool {
	return e.ExpiresAt.After(now)
}

// WysMembershipOrder 买断待支付/已支付单（微信/支付宝/余额）。
type WysMembershipOrder struct {
	OrderID    int64      `json:"order_id" gorm:"primaryKey"`
	UserID     string     `json:"user_id"`
	Tier       string     `json:"tier"`
	PlanID     string     `json:"plan_id"`
	Channel    string     `json:"channel"`
	AmountFen  int64      `json:"amount_fen"`
	Status     int16      `json:"status"`
	OutTradeNo string     `json:"out_trade_no,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	PaidAt     *time.Time `json:"paid_at,omitempty"`
}

func (WysMembershipOrder) TableName() string { return WysMembershipOrderTable }

// MembershipPlan 套餐目录项。
type MembershipPlan struct {
	PlanID            string `json:"plan_id"`
	Tier              string `json:"tier"`
	Title             string `json:"title"`
	Months            int    `json:"months"`
	PriceFen          int64  `json:"price_fen"`
	Price             string `json:"price"` // 元，两位小数
	HuaweiProductID   string `json:"huawei_product_id"`
	AppleProductID    string `json:"apple_product_id"`
	BuyoutEligible    bool   `json:"buyout_eligible"`
	AutoRenewEligible bool   `json:"auto_renew_eligible"`
}

// MembershipCatalog 固定 2 档 × 3 时长。
func MembershipCatalog() []MembershipPlan {
	return []MembershipPlan{
		plan("svip_1m", MembershipTierSVIP, "1个月", 1, 3000, "wys_svip_1m", "wys_svip_1m"),
		plan("svip_6m", MembershipTierSVIP, "6个月", 6, 15000, "wys_svip_6m", "wys_svip_6m"),
		plan("svip_12m", MembershipTierSVIP, "1年", 12, 28000, "wys_svip_12m", "wys_svip_12m"),
		plan("ai_svip_1m", MembershipTierAISVIP, "1个月", 1, 4800, "wys_ai_svip_1m", "wys_ai_svip_1m"),
		plan("ai_svip_6m", MembershipTierAISVIP, "6个月", 6, 24000, "wys_ai_svip_6m", "wys_ai_svip_6m"),
		plan("ai_svip_12m", MembershipTierAISVIP, "1年", 12, 39800, "wys_ai_svip_12m", "wys_ai_svip_12m"),
	}
}

func plan(id, tier, title string, months int, fen int64, huaweiPID, applePID string) MembershipPlan {
	return MembershipPlan{
		PlanID:            id,
		Tier:              tier,
		Title:             title,
		Months:            months,
		PriceFen:          fen,
		Price:             FormatFenToYuan(fen),
		HuaweiProductID:   huaweiPID,
		AppleProductID:    applePID,
		BuyoutEligible:    true,
		AutoRenewEligible: true,
	}
}

// FindMembershipPlan 按 plan_id 查找。
func FindMembershipPlan(planID string) (MembershipPlan, bool) {
	for _, p := range MembershipCatalog() {
		if p.PlanID == planID {
			return p, true
		}
	}
	return MembershipPlan{}, false
}

// FindMembershipPlanByHuaweiProduct 按 AGC productId 反查。
func FindMembershipPlanByHuaweiProduct(productID string) (MembershipPlan, bool) {
	for _, p := range MembershipCatalog() {
		if p.HuaweiProductID == productID {
			return p, true
		}
	}
	return MembershipPlan{}, false
}

// FindMembershipPlanByAppleProduct 按 App Store productId 反查。
func FindMembershipPlanByAppleProduct(productID string) (MembershipPlan, bool) {
	for _, p := range MembershipCatalog() {
		if p.AppleProductID == productID {
			return p, true
		}
	}
	return MembershipPlan{}, false
}

// ValidMembershipTier 档位合法。
func ValidMembershipTier(tier string) bool {
	return tier == MembershipTierSVIP || tier == MembershipTierAISVIP
}

// ValidMembershipBuyoutChannel 买断渠道。
func ValidMembershipBuyoutChannel(ch string) bool {
	switch ch {
	case MembershipChannelWeChat, MembershipChannelAlipay, MembershipChannelBalance:
		return true
	default:
		return false
	}
}
