// service.go 会员订阅（目录 / 买断 / IAP 验单发权益）—— 域内业务逻辑。
//
// 由 app/internal/usecase/membership_usecase.go 迁入。四处改动：
//  1. MembershipUsecase → Service（MembershipMe / MembershipBuyoutResult 等
//     读模型本就带域前缀，保留以对外 JSON 语义不变）；
//  2. 错误哨兵下沉 repository.go（订单类）与本文件（套餐 / 通道 / IAP 类），
//     统一去掉 Membership 前缀；
//  3. 支付能力由同批切片的 payment 域提供：pay *payment.Service，
//     入参 payment.PrepayInput，错误比较 payment.payment.ErrPayNotConfigured；
//  4. 华为 / 苹果验单器留在本域（iap_apple.go / iap_huawei.go）。
package membership

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"app/internal/payment"
	"app/internal/wallet"
	"common/config"
	"github.com/google/uuid"
	"model/entity"
)

var (
	ErrPlanInvalid     = errors.New("会员套餐无效")
	ErrChannelInvalid  = errors.New("会员支付渠道无效")
	ErrTierMismatch    = errors.New("套餐与档位不匹配")
	ErrHuaweiInvalid   = errors.New("华为内购凭证无效")
	ErrHuaweiMismatch  = errors.New("华为商品与套餐不匹配")
	ErrHuaweiNotConfig = errors.New("华为 IAP 未配置")
	ErrAppleInvalid    = errors.New("苹果内购凭证无效")
	ErrAppleMismatch   = errors.New("苹果商品与套餐不匹配")
	ErrAppleNotConfig  = errors.New("苹果 IAP 未配置")
	// ErrPaymentNotConfirmed 客户端说付完了，但渠道侧还没有已核销流水。
	// 发权益必须以流水为准（P0-2），不能以回跳为准。
	ErrPaymentNotConfirmed = errors.New("支付尚未确认，请稍后重试")
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
	OrderID       int64                     `json:"order_id"`
	Channel       string                    `json:"channel"`
	AmountFen     int64                     `json:"amount_fen"`
	Status        string                    `json:"status"` // paid | pending
	Entitlement   *MembershipEntitlementDTO `json:"entitlement,omitempty"`
	PrepayChannel string                    `json:"prepay_channel,omitempty"`
	PrepayParams  map[string]any            `json:"prepay_params,omitempty"`
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

// Service 会员订阅。
type Service struct {
	repo      Repository
	walletSvc *wallet.Service
	pay       *payment.Service
	huawei    HuaweiSubscriptionVerifier
	apple     AppleReceiptVerifier
	devMode   bool
}

// 编译期确认会员域已实现渠道核销器（F2：membership_order 不再走 settled_no_settler 分支）。
var _ payment.Settler = (*Service)(nil)

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

func NewService(
	repo Repository,
	walletSvc *wallet.Service,
	pay *payment.Service,
	huawei HuaweiSubscriptionVerifier,
	apple AppleReceiptVerifier,
	appEnv string,
) *Service {
	u := &Service{
		repo:      repo,
		walletSvc: walletSvc,
		pay:       pay,
		huawei:    huawei,
		apple:     apple,
		devMode:   config.IsLabAppEnv(appEnv),
	}
	// 照 mall/service.go 的 WithPayment 形态自注册：pay 已在 main 注入本构造器，
	// 无需动 main.go。注册后回调核销 membership_order 直接发权益，不再靠回跳/查单。
	if u.pay != nil {
		u.pay.RegisterSettler(entity.PaymentRefMembershipOrder, u)
	}
	return u
}

func (u *Service) Me(ctx context.Context, userID string) (*MembershipMe, error) {
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

func (u *Service) Buyout(ctx context.Context, userID, planID, channel string) (*MembershipBuyoutResult, error) {
	channel = strings.ToLower(strings.TrimSpace(channel))
	if !entity.ValidMembershipBuyoutChannel(channel) {
		return nil, ErrChannelInvalid
	}
	plan, ok := entity.FindMembershipPlan(planID)
	if !ok || !plan.BuyoutEligible {
		return nil, ErrPlanInvalid
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
		return nil, payment.ErrPayNotConfigured
	}
	prepay, err := u.pay.Prepay(ctx, payment.PrepayInput{
		Channel:    channel,
		AmountFen:  plan.PriceFen,
		Subject:    fmt.Sprintf("会员-%s-%s", plan.Tier, plan.Title),
		UserID:     userID,
		RefType:    entity.PaymentRefMembershipOrder,
		RefID:      strconv.FormatInt(order.OrderID, 10),
		OutTradeNo: order.OutTradeNo,
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

func (u *Service) completeBalanceBuyout(
	ctx context.Context,
	userID string,
	order *entity.WysMembershipOrder,
	plan entity.MembershipPlan,
) (*MembershipBuyoutResult, error) {
	if u.walletSvc == nil {
		return nil, wallet.ErrCashInsufficient
	}
	ref := fmt.Sprintf("m%d", order.OrderID)
	now := time.Now()
	// 扣款、改单、发权益必须同一个事务：任一步失败就整体回滚。
	// 旧写法是在失败后调 Credit 补偿，而补偿用 `_, _ =` 写 —— 补偿失败没人知道，
	// 钱扣了、单没付、权益没发，账就少一笔（parity-p0-3 在 mall 域删掉的是同一段）。
	var ent *entity.WysMembershipEntitlement
	err := u.repo.Transaction(ctx, func(ctx context.Context) error {
		if _, err := u.walletSvc.Debit(ctx, userID, plan.PriceFen, entity.CashReasonMembershipPay, ref); err != nil {
			return err
		}
		if err := u.repo.MarkOrderPaid(ctx, order.OrderID, now); err != nil {
			return err
		}
		var err error
		ent, err = u.repo.ExtendEntitlement(ctx, userID, plan.Tier, entity.MembershipChannelBalance, plan.Months, now, "", "", "")
		return err
	})
	if err != nil {
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
func (u *Service) ConfirmBuyout(ctx context.Context, userID string, orderID int64) (*MembershipBuyoutResult, error) {
	order, err := u.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order.UserID != userID {
		return nil, ErrOrderOwner
	}
	plan, ok := entity.FindMembershipPlan(order.PlanID)
	if !ok {
		return nil, ErrPlanInvalid
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
		return nil, ErrChannelInvalid
	}
	// 渠道支付必须已有**已核销流水**才能发权益：客户端回跳不能当作收款成功。
	// 回调常常晚于回跳，所以这里同步向渠道查一次单（ConfirmRefPaid）。
	if u.pay == nil {
		// 没装配 payment 域就等于无法证明渠道收过钱，不发权益。
		return nil, ErrPaymentNotConfirmed
	}
	rec, err := u.pay.ConfirmRefPaid(ctx, entity.PaymentRefMembershipOrder, strconv.FormatInt(orderID, 10))
	if err != nil {
		if errors.Is(err, payment.ErrSettleNotCommitted) || errors.Is(err, payment.ErrRecordNotFound) {
			return nil, ErrPaymentNotConfirmed
		}
		return nil, err
	}
	tradeNo := ""
	if rec.ChannelTradeNo != nil {
		tradeNo = *rec.ChannelTradeNo
	}
	// 改单与发权益同事务：中途失败不能留下「单已付、权益没发」或反之。
	var ent *entity.WysMembershipEntitlement
	var alreadyPaid bool
	err = u.repo.Transaction(ctx, func(ctx context.Context) error {
		if err := u.repo.MarkOrderPaid(ctx, order.OrderID, now); err != nil {
			if errors.Is(err, ErrOrderPaid) {
				alreadyPaid = true
				return nil
			}
			return err
		}
		var extendErr error
		ent, extendErr = u.repo.ExtendEntitlement(ctx, userID, plan.Tier, order.Channel, plan.Months, now, tradeNo, "", "")
		return extendErr
	})
	if err != nil {
		return nil, err
	}
	if alreadyPaid {
		return u.ConfirmBuyout(ctx, userID, orderID)
	}
	return &MembershipBuyoutResult{
		OrderID:     order.OrderID,
		Channel:     order.Channel,
		AmountFen:   order.AmountFen,
		Status:      "paid",
		Entitlement: entitlementDTO(*ent, now),
	}, nil
}

// SettleChannelPaid 实现 payment.Settler：渠道回调核销后主动发放会员权益。
//
// 与 payment.settleByFact 同处一个事务：ctx 已带核销事务句柄，这里**不另开事务**，
// MarkOrderPaid / ExtendEntitlement 经 txdb.DB(ctx,..) 直接落进该事务；
// 任一步失败连流水一起回滚，不会出现「流水 paid、权益没发」的半状态。
// 体内零外部 HTTP/ES/MQ（沿用 P0-3 立的规矩）。
//
// 幂等 / 与 ConfirmBuyout 并发：两条链都以 MarkOrderPaid 的
// `WHERE status=pending + RowsAffected==0 → ErrOrderPaid` 为原子守卫，
// 只有先抢到的那条能走到 ExtendEntitlement，回调重投或双链并发都只加一个周期。
func (u *Service) SettleChannelPaid(ctx context.Context, refType, refID string, rec *entity.WysPaymentRecord) error {
	if refType != entity.PaymentRefMembershipOrder {
		return ErrOrderNotFound
	}
	orderID, err := strconv.ParseInt(refID, 10, 64)
	if err != nil || orderID <= 0 {
		return ErrOrderNotFound
	}
	order, err := u.repo.GetOrder(ctx, orderID)
	if err != nil {
		return err
	}
	plan, ok := entity.FindMembershipPlan(order.PlanID)
	if !ok {
		return ErrPlanInvalid
	}
	// 流水归属校验：防止拿别人的已核销流水去给本单发权益。
	if rec != nil && rec.UserID != "" && rec.UserID != order.UserID {
		return ErrOrderOwner
	}
	now := time.Now()
	tradeNo := ""
	if rec != nil && rec.ChannelTradeNo != nil {
		tradeNo = *rec.ChannelTradeNo
	}
	if err := u.repo.MarkOrderPaid(ctx, orderID, now); err != nil {
		if errors.Is(err, ErrOrderPaid) {
			return nil // 已发放（回调重投 / 与 ConfirmBuyout 并发），幂等返回
		}
		return err
	}
	_, err = u.repo.ExtendEntitlement(ctx, order.UserID, order.Tier, order.Channel, plan.Months, now, tradeNo, "", "")
	return err
}

// VerifyHuawei 验华为订阅并发放权益。
func (u *Service) VerifyHuawei(ctx context.Context, userID string, in HuaweiVerifyInput) (*MembershipEntitlementDTO, error) {
	productID := strings.TrimSpace(in.ProductID)
	token := strings.TrimSpace(in.PurchaseToken)
	if productID == "" || token == "" {
		return nil, ErrHuaweiInvalid
	}
	plan, ok := entity.FindMembershipPlanByHuaweiProduct(productID)
	if !ok {
		return nil, ErrHuaweiMismatch
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
		return nil, ErrHuaweiNotConfig
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
func (u *Service) VerifyApple(ctx context.Context, userID string, in AppleVerifyInput) (*MembershipEntitlementDTO, error) {
	productID := strings.TrimSpace(in.ProductID)
	txID := strings.TrimSpace(in.TransactionID)
	if productID == "" || txID == "" {
		return nil, ErrAppleInvalid
	}
	plan, ok := entity.FindMembershipPlanByAppleProduct(productID)
	if !ok {
		return nil, ErrAppleMismatch
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
		return nil, ErrAppleNotConfig
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
