# Used Car Orders（二手车业务单）

**Status:** done

用语：`CONTEXT.md` §二手车  
ADR：[docs/adr/0014-used-car-order.md](../../docs/adr/0014-used-car-order.md)

实现按 `issues/` 阻塞边，从 frontier 抓票；每票独立会话 `/implement`。跨仓：Go BFF 本仓库；Flutter 在 `my_ai_project`。

| # | Ticket | Blocked by |
|---|--------|------------|
| 01 | [摘要 + 列表（API + Flutter 列表）](./issues/01-summary-and-list.md) | — |
| 02 | [详情（API + Flutter 详情）](./issues/02-detail.md) | 01 |
| 03 | [新建 + 客户选择（API + Flutter 提交）](./issues/03-create-and-customers.md) | 02 |
| 04 | [首页待办「二手车待审」](./issues/04-home-todo-pending-review.md) | 01 |
| 05 | [「我的」收支入口 + 二手车文案纠正](./issues/05-ledger-entry-and-copy.md) | 01 |

**Frontier：** 全部完成。

```text
01 → 02 → 03  (done)
01 → 04       (done)
01 → 05       (done)
```

测试缝（已确认）：

1. **HTTP `/api/v1/used-car-orders*`（主缝）** — SessionAuth；本人当前店；summary / 列表(kind+status) / 详情 / 新建 / customers
2. **UsedCarOrderUsecase（内存 repo）** — 非法 kind、缺必填车况、跨店/非本人不可读、统计口径
3. **`GET /api/v1/home/todo-cards`** — `used_car_pending_review`：`store_admin` + 全店待审 count>0；与 `order_pending_review` 并存
4. **Flutter** — 首页「二手车」→ 业务单体验；「我的」→「收支」仍打 `transactions`

---

## Problem Statement

首页与「我的」入口叫「二手车」，副文案是「置换/专卖/估价」，实际却打开个人收支账本（`transactions`）：列表是收入/支出，没有车辆、客户、审核态，也谈不上订单。销售无法按店务流程提交/查看二手车单；店管也没有独立的「二手车待审」待办。需要把产品语义纠正为店务业务单，并给出列表与详情（及本切必要的新建）接口，同时把原账本挪到「收支」入口。

## Solution

新建 **二手车业务单** 域（表 `wys_used_car_order`，HTTP `/api/v1/used-car-orders`）：销售在当前店为自己提交置换/专卖/收车单，看本人摘要/列表/详情；审核态对齐成交发票四态（本切只读）；首页待办新增小卡 **二手车待审**（全店待审、`store_admin`）。原 `transactions` UI 改由「我的」**收支**入口承接；首页「二手车」进入新业务单体验并美化列表/详情/新建。

## User Stories

1. As a logged-in salesperson, I want the Home grid「二手车」to open used-car business orders (not the personal ledger), so that the label matches the job.
2. As a logged-in salesperson, I want a Mine entry「收支」that opens the former transactions list/detail, so that the personal ledger is still reachable under the right name.
3. As a logged-in salesperson without a current store, I want used-car-order APIs to fail clearly (same pattern as deal invoices / home todos), so that I know to pick a store first.
4. As a logged-in salesperson, I want a list-page summary with my display name, avatar, store position label, store name, and my four stats (submitted / pending review / approved / rejected) for the current store, so that the header matches the新车成交 feel.
5. As a logged-in salesperson, I want to paginate my own used-car orders in the current store, so that I can browse history.
6. As a logged-in salesperson, I want to filter the list by business kind (置换 / 专卖 / 收车), so that I can focus on one workflow.
7. As a logged-in salesperson, I want to filter the list by status (all / pending review / approved / rejected), so that tabs mirror成交发票.
8. As a logged-in salesperson, I want each list row to show kind, vehicle model, amount with the kind-appropriate label, status badge, and key dates, so that the list is scannable and not a ledger.
9. As a logged-in salesperson, I want to open order detail by id for my own orders only, so that peer orders stay private in v1.
10. As a logged-in salesperson on detail, I want to see customer name/phone, full vehicle fields (model, plate, VIN, mileage, model year), amount with kind label, optional image, status, reject reason when rejected, and read-only rating stars when rated, so that the page tells a complete story.
11. As a logged-in salesperson, I want to create a new used-car order with kind, customer, all required vehicle fields, and amount, so that work can enter the system without seed-only demos.
12. As a logged-in salesperson creating an order, I want status to become 待审核 by default, so that review is the next human step.
13. As a logged-in salesperson creating an order, I want optional local image preview with nullable `image_url` on the server, so that photos are not blocked on object storage.
14. As a logged-in salesperson, I want a customer picker for the current store (paginated, optional name/phone search) under the used-car-orders resource, so that create flow reuses `wys_store_customer` without inventing a new customer table.
15. As a logged-in salesperson, I want create validation to reject missing/blank required vehicle fields, unknown kind, non-positive amount, or customer not in current store, so that bad data never lands.
16. As a store admin, I want a Home todo card「二手车待审」when the current store has pending used-car orders, so that review work is visible beside「订单待审核」.
17. As a store admin, I want that card to be small-tier, count all pending orders in the store (not only mine), and require `store_admin`, so that it matches other small review cards.
18. As a non-admin member, I want「二手车待审」hidden even if pending exists, so that the card does not leak manager work.
19. As a client packing Home todos, I want `used_car_pending_review` as its own type (not folded into `order_pending_review`), so that copy and routing stay distinct.
20. As a salesperson, I want amount UI labels to be 补差价 / 成交价 / 收车价 by kind while the API keeps one `amount` field, so that forms stay simple.
21. As a salesperson, I want status values aligned with成交发票 (0–3 codes and the same four UI labels), so that agents and clients reuse mental models.
22. As a salesperson, I want `rating_stars` displayed read-only when status is rated, without a rating write API in this slice, so that seeded demos look complete.
23. As a salesperson viewing a rejected order, I want `reject_reason` read-only, without a reject write API in this slice, so that detail narrative works from seed/manual DB updates.
24. As a developer, I want used-car orders stored in `wys_used_car_order`, not in `transactions`, `wys_mall_order`, or `wys_store_review_order`, so that domains stay separated.
25. As a developer, I want SessionAuth on all used-car-order and customer-picker endpoints, so that access is bound to the session user and current store.
26. As a Flutter user, I want the used-car list/detail/create UI polished (not income/expense tags), so that the product no longer looks like a ledger.
27. As a Flutter user, I want empty/loading/error states on list and detail against the real BFF, so that LAN demos fail loudly when misconfigured.
28. As a QA engineer, I want usecase tests locking create validation, ownership scope, filters, and summary stats, so that regressions fail without UI flakiness.
29. As a QA engineer, I want an HTTP smoke path (script or make target) for summary/list/detail/create, so that LAN verification is one command.
30. As a future store admin, I want approve/reject/resubmit/rating write APIs deferred, so that v1 can ship list/detail/create first.
31. As a future product owner, I want估价 as a separate capability deferred, so that kind enum stays 置换/专卖/收车 only.
32. As a developer, I want domain language locked in CONTEXT.md and ADR 0013, so that agents do not rename「二手车业务单」back to transactions.

## Implementation Decisions

### Modules & seams

- **UsedCarOrder usecase (new):** summary, list (kind + status filters), get by id (owner + current store), create (default pending), list customers for current store. Controllers thin.
- **Repository + `wys_used_car_order`:** GORM/local Postgres; seed rows covering kinds and all four statuses for LAN.
- **HTTP:** `/api/v1/used-car-orders` under SessionAuth (same envelope as deal invoices: `{ code, message, data, timestamp }`, snake_case, `page`/`size`).
- **Home todo aggregation (extend):** new type `used_car_pending_review`; count `status = pending` for current store; gate `store_admin`; small card; do not change `order_pending_review` source (`wys_store_review_order`).
- **Flutter (`my_ai_project`):** replace used-car routes’ data source from `TransactionApi` to used-car-order API; polish list/detail/create; add Mine「收支」→ existing transactions screens; Home「二手车」stays label, new destination.
- **Dependency direction:** Used-car orders may read `wys_store_customer` and current-store membership helpers. Must not write mall orders, store review orders, deal invoices, or transactions. Personal ledger keeps using `transactions` APIs unchanged.

### Domain rules (frozen in grilling)

- Actor v1: salesperson own orders in **current store**; create allowed; approve/reject/resubmit/rating write out of scope.
- Kinds: `trade_in` | `consign` | `purchase` (UI: 置换 / 专卖 / 收车) — exact wire strings fixed in first implement ticket; must be stable.
- Status: same int16 map as成交发票 — 0 pending_review, 1 approved_pending_rating, 2 rated, 3 rejected.
- Required on create: kind, customer_id (in store), vehicle_model, plate_no, vin, mileage_km, model_year, amount (> 0).
- Amount: single numeric field; UI label by kind (补差价 / 成交价 / 收车价).
- Optional: `image_url` (often empty); `reject_reason` / `rating_stars` read-only in v1.
- Summary stats: **uploader × current store** — submitted(=all), pending_review, approved(status∈{1,2}), rejected.
- Todo count: **all uploaders × current store × pending**; visibility `store_admin`.
- Personal ledger: Mine「收支」only; not on Home grid beside 二手车.

### Schema (directional)

`wys_used_car_order` (names indicative):

| Column | Notes |
|--------|-------|
| order_id | bigserial PK |
| store_id | current store snapshot |
| uploader_user_id | session user |
| kind | smallint or text enum — pick one style consistent with repo |
| customer_id | FK-ish to `wys_store_customer` |
| customer_name / customer_phone | submit-time snapshot |
| vehicle_model, plate_no, vin | required strings |
| mileage_km | required int |
| model_year | required int |
| amount | required numeric |
| status | smallint 0–3 |
| image_url | nullable text |
| reject_reason | nullable text |
| rating_stars | nullable 1–5 |
| submitted_at, created_at, updated_at | timestamps |

Indexes: `(store_id, uploader_user_id, submitted_at DESC)`; `(store_id, status)` for todo counts; optional `(store_id, uploader_user_id, kind, status)`.

Reuse `wys_store_customer` as-is (phone already required from deal-invoice work).

### API (directional)

Prefix: `/api/v1/used-car-orders` (SessionAuth).

| Method | Path | Notes |
|--------|------|-------|
| GET | `/summary` | profile + four stats (self × current store) |
| GET | `/` | page/size; optional `kind`, `status` (`all`\|`pending_review`\|`approved`\|`rejected`) |
| GET | `/:order_id` | detail; self only |
| POST | `/` | create → status 0 |
| GET | `/customers` | current-store customers; optional `q` |

List/detail JSON includes kind code + display hints as needed by Flutter; amount raw number; client maps label by kind.

Home todo: extend `GET /api/v1/home/todo-cards` with type `used_car_pending_review`, title/subtitle/action_label/action_route/count; client maps type→small size.

### Flutter

- Keep routes under home used-car paths where practical; swap models/API.
- List: kind chips/tabs, status tabs, summary header, polished cards (vehicle + amount label + status).
- Detail: sections for customer, vehicle, amount, status/reason/stars, optional image.
- Create: kind picker, customer picker, required vehicle fields, amount, optional local image.
- Mine: add「收支」→ legacy transaction list/detail; remove「二手车」subtitle that promises 估价 if估价 is out of scope (or soften copy to 置换/专卖/收车).

## Testing Decisions

- Good tests assert **external behavior** only (API/usecase outcomes), not GORM SQL or widget trees.
- **Primary:** HTTP smoke for authz scope + create/list/detail/summary happy paths and validation 4xx.
- **Usecase (mem repo):** prior art `deal_invoice_usecase_test.go` — ownership, filters, stats, create defaults.
- **Home todo:** extend existing home-todo usecase/tests so `used_car_pending_review` appears/omits correctly beside `order_pending_review`.
- **Flutter:** manual LAN against `make lan-up` / `make run`; no new E2E framework.
- Ponytail: at least one runnable Go check that fails if create validation or stats break.

## Out of Scope

- Approve / reject / resubmit / rating write APIs or admin review UI
- Real object-storage upload / CDN
- 估价 product, finance calculator tie-in, mall checkout for cars
- Folding counts into `wys_store_review_order` or `order_pending_review`
- Store-wide salesperson list (seeing others’ orders) in the sales list API
- Deleting or migrating historical `transactions` rows
- Supabase-only path quirks beyond existing dual-provider patterns already used by peer modules
- KMP / RN / other clients beyond Flutter for this slice

## Further Notes

- Next process step: **`/to-tickets`** — split into tracer bullets (suggested shape: 01 table+seed+summary+list API → 02 detail+create+customers → 03 home todo type → 04 Flutter list/detail polish + entry swap → 05 Flutter create + Mine「收支」). Adjust edges in to-tickets; do not start `/implement` from this file alone.
- Prefer copying deal-invoice response shapes and status filter parsing to minimize drift.
- If wire kind strings need bilingual codes, lock them in ticket 01 and add to CONTEXT Avoid list only if a synonym war appears.
