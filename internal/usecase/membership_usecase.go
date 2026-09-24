package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stvenfor/my_go_study/internal/domain/entity"
	"github.com/stvenfor/my_go_study/internal/domain/repository"
	"github.com/stvenfor/my_go_study/pkg/config"
)

var (
	ErrMembershipPlanInvalid     = errors.New("会员套餐无效")
	ErrMembershipChannelInvalid  = errors.New("会员支付渠道无效")
	ErrMembershipTierMismatch    = errors.New("套餐与档位不匹配")
	ErrMembershipHuaweiInvalid   = errors.New("华为内购凭证无效")
	ErrMembershipHuaweiMismatch  = errors.New("华为商品与套餐不匹配")
	ErrMembershipHuaweiNotConfig = errors.New("华为 IAP 未配置")
	ErrMembershipAppleInvalid    = errors.New("苹果内购凭证无效")
	ErrMembershipAppleMismatch   = errors.New("苹果商品与套餐不匹配")
	ErrMembershipAppleNotConfig  = errors.New("苹果 IAP 未配置")
)

// MembershipMe 权益 + 目录。
type MembershipMe struct {
	Entitlements []MembershipEntitlementDTO `json:"entitlements"`
	Catalog      []entity.MembershipPlan    `json:"catalog"`
}

// MembershipEntitlementDTO 对外权益读模型。
type MembershipEntitlementDTO struct {
	Tier          string `json:"tier"`
	ExpiresAt     string `json:"expires_at"`
	Active        bool   `json:"active"`
	SourceChannel string `json:"source_channel"`
	CTA           string `json:"cta"` // 开通 | 续费
}

// MembershipBuyoutResult 买断结果。
type MembershipBuyoutResult struct {
	OrderID       int64                  `json:"order_id"`
	Channel       string                 `json:"channel"`
	AmountFen     int64                  `json:"amount_fen"`
	Status        string                 `json:"status"` // paid | pending
	Entitlement   *MembershipEntitlementDTO `json:"entitlement,omitempty"`
	PrepayChannel string                 `json:"prepay_channel,omitempty"`
	PrepayParams  map[string]any         `json:"prepay_params,omitempty"`
}

// HuaweiVerifyInput 客户端上报。
type HuaweiVerifyInput struct {
	ProductID        string
	PurchaseToken    string
	PurchaseOrderID  string
	SubscriptionID   string
	JWSPurchaseOrder string
}

// AppleVerifyInput 客户端上报（StoreKit 交易 + 可选 receipt）。
type AppleVerifyInput struct {
	ProductID             string
	TransactionID         string
	OriginalTransactionID string
	ReceiptData           string // base64 app receipt；可空（dev）
}

// MembershipUsecase 会员订阅。
type MembershipUsecase struct {
	repo    repository.MembershipRepository
	wallet  *CashWalletUsecase
	pay     *PaymentUsecase
	huawei  HuaweiSubscriptionVerifier
	apple   AppleReceiptVerifier
	devMode bool
}

// HuaweiSubscriptionVerifier 验订阅；本地可注入 mock。
type HuaweiSubscriptionVerifier interface {
	Configured() bool
	Verify(ctx context.Context, in HuaweiVerifyInput, plan entity.MembershipPlan) (*HuaweiVerifyResult, error)
}

// HuaweiVerifyResult 验单通过后的有效期提示（月数由 plan 决定时可忽略 ExpiresAt）。
type HuaweiVerifyResult struct {
	PurchaseToken  string
	SubscriptionID string
	AckRequired    bool
}

// AppleReceiptVerifier 验苹果票据。
type AppleReceiptVerifier interface {
	Configured() bool
	Verify(ctx context.Context, in AppleVerifyInput, plan entity.MembershipPlan) (*AppleVerifyResult, error)
}

// AppleVerifyResult 验单结果。
type AppleVerifyResult struct {
	OriginalTransactionID string
	TransactionID         string
}

func NewMembershipUsecase(
	repo repository.MembershipRepository,
	wallet *CashWalletUsecase,
	pay *PaymentUsecase,
	huawei HuaweiSubscriptionVerifier,
	apple AppleReceiptVerifier,
	appEnv string,
) *MembershipUsecase {
	return &MembershipUsecase{
		repo:    repo,
		wallet:  wallet,
		pay:     pay,
		huawei:  huawei,
		apple:   apple,
		devMode: config.IsLabAppEnv(appEnv),
	}
}

func (u *MembershipUsecase) Me(ctx context.Context, userID string) (*MembershipMe, error) {
	rows, err := u.repo.ListEntitlements(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	byTier := map[string]entity.WysMembershipEntitlement{}
	for _, r := range rows {
		byTier[r.Tier] = r
	}
	out := make([]MembershipEntitlementDTO, 0, 2)
	for _, tier := range []string{entity.MembershipTierSVIP, entity.MembershipTierAISVIP} {
		dto := MembershipEntitlementDTO{Tier: tier, CTA: "开通", Active: false}
		if e, ok := byTier[tier]; ok {
			dto.ExpiresAt = e.ExpiresAt.UTC().Format(time.RFC3339)
			dto.SourceChannel = e.SourceChannel
			dto.Active = e.Active(now)
			if dto.Active {
				dto.CTA = "续费"
			}
		}
		out = append(out, dto)
	}
	return &MembershipMe{Entitlements: out, Catalog: entity.MembershipCatalog()}, nil
}

func (u *MembershipUsecase) Buyout(ctx context.Context, userID, planID, channel string) (*MembershipBuyoutResult, error) {
	channel = strings.ToLower(strings.TrimSpace(channel))
	if !entity.ValidMembershipBuyoutChannel(channel) {
		return nil, ErrMembershipChannelInvalid
	}
	plan, ok := entity.FindMembershipPlan(planID)
	if !ok || !plan.BuyoutEligible {
		return nil, ErrMembershipPlanInvalid
	}

	outTrade := strings.ReplaceAll(uuid.NewString(), "-", "")
	order := &entity.WysMembershipOrder{
		UserID:     userID,
		Tier:       plan.Tier,
		PlanID:     plan.PlanID,
		Channel:    channel,
		AmountFen:  plan.PriceFen,
		Status:     entity.MembershipOrderPending,
		OutTradeNo: outTrade,
		CreatedAt:  time.Now(),
	}
	if err := u.repo.CreateOrder(ctx, order); err != nil {
		return nil, err
	}

	if channel == entity.MembershipChannelBalance {
		return u.completeBalanceBuyout(ctx, userID, order, plan)
	}

	if u.pay == nil {
		return nil, ErrPayNotConfigured
	}
	prepay, err := u.pay.Prepay(ctx, PrepayInput{
		Channel:   channel,
		AmountFen: plan.PriceFen,
		Subject:   fmt.Sprintf("会员-%s-%s", plan.Tier, plan.Title),
	})
	if err != nil {
		return nil, err
	}
	return &MembershipBuyoutResult{
		OrderID:       order.OrderID,
		Channel:       channel,
		AmountFen:     plan.PriceFen,
		Status:        "pending",
		PrepayChannel: prepay.Channel,
		PrepayParams:  prepay.Params,
	}, nil
}

func (u *MembershipUsecase) completeBalanceBuyout(
	ctx context.Context,
	userID string,
	order *entity.WysMembershipOrder,
	plan entity.MembershipPlan,
) (*MembershipBuyoutResult, error) {
	if u.wallet == nil {
		return nil, ErrCashInsufficient
	}
	ref := fmt.Sprintf("m%d", order.OrderID)
	if _, err := u.wallet.Debit(ctx, userID, plan.PriceFen, entity.CashReasonMembershipPay, ref); err != nil {
		return nil, err
	}
	now := time.Now()
	if err := u.repo.MarkOrderPaid(ctx, order.OrderID, now); err != nil {
		_, _ = u.wallet.Credit(ctx, userID, plan.PriceFen, entity.CashReasonMembershipPay, ref+":rollback")
		return nil, err
	}
	ent, err := u.repo.ExtendEntitlement(ctx, userID, plan.Tier, entity.MembershipChannelBalance, plan.Months, now, "", "", "")
	if err != nil {
		_, _ = u.wallet.Credit(ctx, userID, plan.PriceFen, entity.CashReasonMembershipPay, ref+":rollback")
		return nil, err
	}
	return &MembershipBuyoutResult{
		OrderID:     order.OrderID,
		Channel:     entity.MembershipChannelBalance,
		AmountFen:   plan.PriceFen,
		Status:      "paid",
		Entitlement: entitlementDTO(*ent, now),
	}, nil
}

// ConfirmBuyout 微信/支付宝客户端成功后确认（幂等）。
func (u *MembershipUsecase) ConfirmBuyout(ctx context.Context, userID string, orderID int64) (*MembershipBuyoutResult, error) {
	order, err := u.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.UserID != userID {
		return nil, repository.ErrMembershipOrderOwner
	}
	plan, ok := entity.FindMembershipPlan(order.PlanID)
	if !ok {
		return nil, ErrMembershipPlanInvalid
	}
	now := time.Now()
	if order.Status == entity.MembershipOrderPaid {
		ent, err := u.repo.GetEntitlement(ctx, userID, order.Tier)
		if err != nil {
			return nil, err
		}
		var dto *MembershipEntitlementDTO
		if ent != nil {
			dto = entitlementDTO(*ent, now)
		}
		return &MembershipBuyoutResult{
			OrderID:     order.OrderID,
			Channel:     order.Channel,
			AmountFen:   order.AmountFen,
			Status:      "paid",
			Entitlement: dto,
		}, nil
	}
	if order.Channel != entity.MembershipChannelWeChat && order.Channel != entity.MembershipChannelAlipay {
		return nil, ErrMembershipChannelInvalid
	}
	if err := u.repo.MarkOrderPaid(ctx, order.OrderID, now); err != nil {
		if errors.Is(err, repository.ErrMembershipOrderPaid) {
			return u.ConfirmBuyout(ctx, userID, orderID)
		}
		return nil, err
	}
	ent, err := u.repo.ExtendEntitlement(ctx, userID, plan.Tier, order.Channel, plan.Months, now, "", "", "")
	if err != nil {
		return nil, err
	}
	return &MembershipBuyoutResult{
		OrderID:     order.OrderID,
		Channel:     order.Channel,
		AmountFen:   order.AmountFen,
		Status:      "paid",
		Entitlement: entitlementDTO(*ent, now),
	}, nil
}

// VerifyHuawei 验华为订阅并发放权益。
func (u *MembershipUsecase) VerifyHuawei(ctx context.Context, userID string, in HuaweiVerifyInput) (*MembershipEntitlementDTO, error) {
	productID := strings.TrimSpace(in.ProductID)
	token := strings.TrimSpace(in.PurchaseToken)
	if productID == "" || token == "" {
		return nil, ErrMembershipHuaweiInvalid
	}
	plan, ok := entity.FindMembershipPlanByHuaweiProduct(productID)
	if !ok {
		return nil, ErrMembershipHuaweiMismatch
	}

	var verified *HuaweiVerifyResult
	if u.huawei != nil && u.huawei.Configured() {
		res, err := u.huawei.Verify(ctx, in, plan)
		if err != nil {
			return nil, err
		}
		verified = res
	} else if u.devMode {
		// 本地未配 AGC 密钥时：仅校验 productId 映射与 token 非空。
		verified = &HuaweiVerifyResult{
			PurchaseToken:  token,
			SubscriptionID: strings.TrimSpace(in.SubscriptionID),
			AckRequired:    true,
		}
	} else {
		return nil, ErrMembershipHuaweiNotConfig
	}

	now := time.Now()
	ent, err := u.repo.ExtendEntitlement(
		ctx, userID, plan.Tier, entity.MembershipChannelHuawei, plan.Months, now,
		verified.PurchaseToken, verified.SubscriptionID, "",
	)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(token))
	outTrade := "hw-" + hex.EncodeToString(sum[:16])
	if existing, _ := u.repo.FindPaidByOutTradeNo(ctx, outTrade); existing == nil {
		_ = u.repo.CreateOrder(ctx, &entity.WysMembershipOrder{
			UserID:     userID,
			Tier:       plan.Tier,
			PlanID:     plan.PlanID,
			Channel:    entity.MembershipChannelHuawei,
			AmountFen:  plan.PriceFen,
			Status:     entity.MembershipOrderPaid,
			OutTradeNo: outTrade,
			CreatedAt:  now,
			PaidAt:     &now,
		})
	}
	return entitlementDTO(*ent, now), nil
}

// VerifyApple 验苹果订阅并发放权益。
func (u *MembershipUsecase) VerifyApple(ctx context.Context, userID string, in AppleVerifyInput) (*MembershipEntitlementDTO, error) {
	productID := strings.TrimSpace(in.ProductID)
	txID := strings.TrimSpace(in.TransactionID)
	if productID == "" || txID == "" {
		return nil, ErrMembershipAppleInvalid
	}
	plan, ok := entity.FindMembershipPlanByAppleProduct(productID)
	if !ok {
		return nil, ErrMembershipAppleMismatch
	}

	var verified *AppleVerifyResult
	if u.apple != nil && u.apple.Configured() {
		res, err := u.apple.Verify(ctx, in, plan)
		if err != nil {
			return nil, err
		}
		verified = res
	} else if u.devMode {
		orig := strings.TrimSpace(in.OriginalTransactionID)
		if orig == "" {
			orig = txID
		}
		verified = &AppleVerifyResult{
			OriginalTransactionID: orig,
			TransactionID:         txID,
		}
	} else {
		return nil, ErrMembershipAppleNotConfig
	}

	now := time.Now()
	ent, err := u.repo.ExtendEntitlement(
		ctx, userID, plan.Tier, entity.MembershipChannelApple, plan.Months, now,
		"", "", verified.OriginalTransactionID,
	)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(verified.TransactionID))
	outTrade := "ap-" + hex.EncodeToString(sum[:16])
	if existing, _ := u.repo.FindPaidByOutTradeNo(ctx, outTrade); existing == nil {
		_ = u.repo.CreateOrder(ctx, &entity.WysMembershipOrder{
			UserID:     userID,
			Tier:       plan.Tier,
			PlanID:     plan.PlanID,
			Channel:    entity.MembershipChannelApple,
			AmountFen:  plan.PriceFen,
			Status:     entity.MembershipOrderPaid,
			OutTradeNo: outTrade,
			CreatedAt:  now,
			PaidAt:     &now,
		})
	}
	return entitlementDTO(*ent, now), nil
}

func entitlementDTO(e entity.WysMembershipEntitlement, now time.Time) *MembershipEntitlementDTO {
	active := e.Active(now)
	cta := "开通"
	if active {
		cta = "续费"
	}
	return &MembershipEntitlementDTO{
		Tier:          e.Tier,
		ExpiresAt:     e.ExpiresAt.UTC().Format(time.RFC3339),
		Active:        active,
		SourceChannel: e.SourceChannel,
		CTA:           cta,
	}
}
