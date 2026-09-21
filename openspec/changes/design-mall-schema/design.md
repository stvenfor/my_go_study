## Context

See proposal.md — Why. Local mode has `users.user_id`, `wys_store`, and store roles. New business tables use `wys_mall_*`. Authenticated routes require matching `user_id`. Mock pay always succeeds for channels 1–4 without calling PSPs.

## Goals / Non-Goals

**Goals:**

- Shared SPU/SKU catalog; money as `numeric(10,2)`; no physical FKs; soft delete on catalog
- Order line snapshots (title, cover, specs, price); order snapshots receiver
- Local pay: any of channels 1–4 writes a successful payment and fulfills in one transaction

**Non-Goals:**

- Real Alipay/WeChat/Apple/Huawei HTTP, refunds UI, shipping carriers, coupons, sharding

## Decisions

### 1. Tables (no physical FKs)

| Table | Role |
|-------|------|
| `wys_mall_category` | store category, soft delete |
| `wys_mall_product` | SPU, `kind` 0/1 |
| `wys_mall_sku` | price, specs JSONB, physical `stock_qty`, virtual deliver fields |
| `wys_mall_virtual_code` | redeem pool |
| `wys_mall_cart_item` | PK (user_id, sku_id) |
| `wys_mall_order` | order_no, idempotency_key, status, payment_channel, amount, receiver |
| `wys_mall_order_item` | snapshots |
| `wys_mall_order_log` | status transitions |
| `wys_mall_payment` | payment_no, payment_channel 1–4, status, channel_payload |
| `wys_mall_refund` | same channel enum (schema ready; pay path does not create refunds yet) |
| `wys_mall_audit_log` | payment.success etc. |

### 2. Mock pay

`POST .../pay` with `payment_channel` ∈ {1,2,3,4}: insert payment status=1, set order paid, fulfill stock/codes, log. No PSP. `channel_trade_no` is a local unique id. Repeat pay: if order already paid, return success, no second success payment.

Stock: `UPDATE ... WHERE stock_qty >= n RETURNING`. Codes: claim unused rows under row lock.

### 3. Auth

Local only. `mall.catalog.write` on `store_admin` and `store_staff`. Buyers need no mall permission to browse/order/pay.

## Risks / Trade-offs

- [Fake money] Mock pay writes real stock decrements → Accept for LAN; replace with webhook later without changing fulfillment SQL.
- [Oversell before pay] No hold → Pay fails closed if stock gone.

## Migration Plan

1. Embed `mall_schema.sql`, run after access control on local boot; mirror in `migrations/` and navicat script.
2. Rollback: drop `wys_mall_*` and revoke `mall.catalog.write`.

## Open Questions

None for this slice.
