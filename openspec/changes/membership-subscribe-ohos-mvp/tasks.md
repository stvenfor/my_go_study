## 1. Go — schema & catalog

- [x] 1.1 Add `wys_membership_entitlement` (+ optional payment/ledger row) migration and embedded SQL; `user_id`+`tier` unique
- [x] 1.2 Define plan catalog (svip/ai_svip × 1m/6m/12m) with placeholder fen prices and Huawei `productId` map in config
- [x] 1.3 Wire entity + repository + DI in `cmd/api/main.go`

## 2. Go — buyout APIs

- [x] 2.1 `GET /api/v1/membership/me` returns entitlements + catalog
- [x] 2.2 `POST /api/v1/membership/buyout` for `channel=balance`: debit wallet + extend `expires_at` atomically; reject insufficient funds
- [x] 2.3 Buyout confirm for wechat/alipay after prepay (session-bound, amount/plan check, idempotent)
- [x] 2.4 Usecase tests: balance success/fail, expiry extend from max(now, previous), dual-tier independence

## 3. Go — Huawei subscription

- [x] 3.1 Config + JWT client for IAP server APIs (AGC key from env); `.env.example` keys
- [x] 3.2 `POST /api/v1/membership/huawei/verify`: JWS/subscription query, upsert entitlement, ack delivery
- [x] 3.3 Reject forged/mismatched productId; idempotent verify
- [x] 3.4 (Optional if timebox) `POST /api/v1/membership/huawei/notifications` stub + doc
- [x] 3.5 Short API doc `docs/membership-subscribe-api.md` + acceptance notes path

## 4. Flutter — membership page (keep dual tabs)

- [x] 4.1 Replace mock plans with 1m/6m/12m per SVIP and AI SVIP; CTA 开通/续费 from `/membership/me`
- [x] 4.2 Extend `PaymentMethodType` with balance + huawei; OHOS shows all four with buyout vs auto-renew copy
- [x] 4.3 Gateway: balance → buyout API; wechat/alipay → prepay + confirm; refresh me after success
- [x] 4.4 OHOS MethodChannel (or verified plugin) for createPurchase / finishPurchase / queryPurchases → verify API
- [x] 4.5 Cold-start restore unfinished Huawei purchases

## 5. Sandbox acceptance

- [ ] 5.1 AGC: create six auto-renew products + sandbox tester; map productIds in Go config
- [ ] 5.2 E2E: balance buyout; wechat or alipay buyout (dev); Huawei subscribe + kill/reopen restore
- [x] 5.3 Write acceptance record under `docs/acceptance-records/`
