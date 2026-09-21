## 1. Schema

- [x] 1.1 Add up/down migration and embedded SQL for mall tables (category, product, sku, virtual_code, cart, order, order_item, order_log, payment, refund, audit_log) with no physical FKs; money `numeric(10,2)`; `payment_channel` 1–4
- [x] 1.2 Grant `mall.catalog.write` to `store_admin` and `store_staff`
- [x] 1.3 Append the same tables to `scripts/navicat_local_schema.sql`

## 2. Domain and persistence

- [x] 2.1 Add mall entities and postgres repository (catalog, cart, order, mock pay)
- [x] 2.2 Enforce kind rules on write: physical needs stock; virtual needs deliver_type; reject mixed kinds under one product

## 3. HTTP

- [x] 3.1 Register mall routes only when `auth.provider=local`, behind SessionAuth and matching `user_id`
- [x] 3.2 Buyer: list on-shelf products, cart upsert, create order with idempotency key and snapshots, read own order
- [x] 3.3 Buyer: pay with channel 1–4 always succeeds locally (payment row + fulfill); cancel only while unpaid; repeat pay idempotent
- [x] 3.4 Store staff: write product/SKU/codes with `mall.catalog.write` for that store (category table ready; HTTP create deferred)

## 4. Verify

- [x] 4.1 Note mall tables and mock pay in `docs/local-auth-postgres.md`
- [x] 4.2 Unit/integration checks: physical without address fails; virtual-only ok; pay any channel writes success; repeat pay no double stock; bad channel rejected; mismatched user_id stores nothing
