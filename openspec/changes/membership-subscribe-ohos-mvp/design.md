## Context

See proposal.md — Why. Flutter membership renew UI lives in `my_ai_project/features/pay` (`MembershipRenewController`, mock plans, WeChat/Alipay via `/api/v1/payments/prepay`). Cash wallet already supports balance debit for mall (`payment_channel=6`). Huawei IAP Kit server patterns: JWT + JWS verify + SubscriptionService (sample: iapkit-sample-serverdemo). Mall `payment_channel=4` stays local-sim and out of this change.

## Goals / Non-Goals

**Goals:**

- Keep SVIP / AI SVIP dual tabs; plans become 1m / 6m / 12m per tier
- HarmonyOS: four pay methods with semantics **C** (buyout vs auto-renew IAP)
- Server-owned entitlement (`expires_at` + source); open and renew on same page
- Sandbox-ready path for Huawei subscription verify + finish/ack

**Non-Goals:**

- Redesigning membership visual system beyond plan/channel wiring
- Real AppGallery review packaging (hide non-IAP channels)
- Apple IAP; mall Huawei channel hardening
- Fine-grained feature gates per entitlement flag

## Decisions

### 1. Product semantics C (locked)

| Channel | Product meaning | Entitlement effect |
|---------|-----------------|--------------------|
| WeChat / Alipay / Balance | Prepaid buyout | `expires_at = max(now, expires_at) + duration` |
| Huawei IAP | Auto-renewable subscription | Sync from Huawei subscription status; renewals via notification or client restore |

Alternatives rejected: A (IAP-only on store builds) deferred to release packaging; B (all channels identical auto-renew) impossible without Huawei owning all renewals.

### 2. Keep dual tabs (locked)

Two tiers remain: `svip`, `ai_svip`. Each has three SKUs: `*_1m`, `*_6m`, `*_12m`. Mapping tables:

- Buyout: internal `plan_id` → duration months + price fen
- IAP: internal `plan_id` → AGC `productId` (six auto-renew products minimum)

### 3. Entitlement model (single row per user per tier)

`wys_membership_entitlement`: `user_id`, `tier`, `expires_at`, `source_channel`, `huawei_purchase_token` (nullable), `updated_at`.

Open = insert/update when expired or absent; renew = extend. Active Huawei auto-renew may refuse overlapping buyout of same tier (or allow stack — **default: allow buyout to extend `expires_at` floor while IAP remains source of truth for renewal events**). Simpler MVP: any successful pay extends/sets expiry; Huawei webhook/query can move `expires_at` forward on renew.

### 4. API shape

| Method | Path | Role |
|--------|------|------|
| GET | `/api/v1/membership/me` | Current entitlements + catalog (plans/prices) |
| POST | `/api/v1/membership/buyout` | `tier`, `plan_id`, `channel`=`wechat`\|`alipay`\|`balance`；微信/支付宝先走既有 prepay，客户端付成功后再带 `out_trade_no` 或会话校验调用 confirm；MVP 可先 **balance 同步开通**，微信/支付宝 **prepay + client success callback confirm**（本地可信任 session + amount 校验；正式环境再加异步 notify） |
| POST | `/api/v1/membership/huawei/verify` | Client posts JWS / purchaseToken / productId；server verifies, upserts entitlement, acks delivery |
| POST | `/api/v1/membership/huawei/notifications` | Optional key-event webhook (phase 2 if timebox) |

Reuse `/api/v1/payments/prepay` for WeChat/Alipay order params; do not overload mall pay.

### 5. Flutter OHOS IAP bridge

Prefer MethodChannel → ArkTS IAP Kit (`createPurchase` / `finishPurchase` / `queryPurchases`) over stock `in_app_purchase_ohos` until verification fields are reliable. Gateway branches: `ohos + huawei` → bridge; balance → wallet + buyout; wechat/alipay → existing gateway.

### 6. Page strategy

Refactor existing `membership_renew_page` in place (not a second route). CTA label: inactive/expired →「开通」; active →「续费」. Payment method list on OHOS shows four options; other platforms keep wechat/alipay (+ balance if wallet available).

## Risks / Trade-offs

- [AppGallery policy vs four channels] → Mitigation: debug builds show all four; store packaging tracked as follow-up, not MVP blocker
- [in_app_purchase_ohos incomplete receipts] → Mitigation: native MethodChannel + raw JWS
- [WeChat/Alipay confirm without notify] → Mitigation: MVP session-bound confirm; document notify as hardening
- [IAP + buyout double-pay confusion] → Mitigation: UI copy distinguishes「自动续费」vs「时长买断」; same tier single entitlement row

## Migration Plan

1. Add membership tables + seed plan catalog in config/DB
2. Ship GET me + balance buyout first (easiest E2E)
3. Wire WeChat/Alipay confirm
4. Huawei verify + finishPurchase + cold-start restore
5. Sandbox acceptance record

Rollback: feature-flag Huawei/balance rows on client; leave tables in place.

## Open Questions

- Exact fen prices for 1m/6m/12m × two tiers (use placeholder prices until product fills AGC)
- Whether AI SVIP and SVIP can both be active simultaneously (**default yes**, independent rows)
