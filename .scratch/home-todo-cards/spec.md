# Home Todo Cards

**Status:** ready-for-agent

用语：`CONTEXT.md` §首页待办卡  
ADR：[docs/adr/0009-home-todo-card-type-vs-size.md](../../docs/adr/0009-home-todo-card-type-vs-size.md)

实现按 `issues/` 阻塞边，从 frontier（无 blocker 或 blocker 已完成）抓票；每票独立会话 `/implement`。跨仓：Go BFF 本仓库；Flutter 在 `my_ai_project`。

| # | Ticket | Blocked by |
|---|--------|------------|
| 01 | [入店申请 + 首页聚合（仅 partner）+ Flutter 大卡与确认/拒绝](./issues/01-join-application-partner-card.md) | — |
| 02 | [待跟进客户 + 聚合中卡 + Flutter 中卡与只读列表](./issues/02-follow-up-customer-card.md) | 01 |
| 03 | [售后预约 + 聚合中卡 + Flutter 只读列表](./issues/03-after-sales-appointment-card.md) | 01 |
| 04 | [店务审核单 + 聚合小卡 + Flutter 只读列表](./issues/04-store-review-order-card.md) | 01 |
| 05 | [Flutter 装箱与左右滑（完整规则）](./issues/05-flutter-packing-pager.md) | 02, 03, 04 |

**Frontier 起点：** 只开 **01**。01 完成后可并行 02 / 03 / 04；05 等 02–04。

测试缝（已确认）：
1. **HomeTodo usecase** — 当前店 + 有效权限 + 各域 count → 有序待办卡列表（省略 count=0 / 无权限）
2. **各域 usecase** — 入店申请（含确认/拒绝）；跟进客户 / 售后预约 / 店务审核单（只读列表 + count 口径）
3. **Flutter 首页待办体验** — type→尺寸、2×2 装箱、≤2 小卡包裹、PageView；聚合 HTTP；四列表页；新伙伴确认/拒绝

---

## Problem Statement

首页「新伙伴待确认」等目前是等宽小卡 mock，不能区分中卡/大卡，也不能按 2×2 格左右滑分页。店管需要在首页一眼看到当前店真实待办（入店待审、跟进客户、售后预约、订单待审），并点进列表处理；排版要稳定：大卡优先，中卡可与小卡混排。

## Solution

Go BFF 提供当前店的首页待办卡聚合接口，以及四个域的最小真实数据与列表（入店申请可确认/拒绝）。卡片只下发 **类型** 与文案/计数/路由；客户端按类型映射小/中/大尺寸并装箱分页。Flutter 替换 mock 快捷卡为该聚合，实现中/大卡与左右滑，并提供四个列表页。

## User Stories

1. As a store member with the right permissions, I want homepage todo cards for my **current store**, so that switching stores changes what I see.
2. As a user without a current store, I want an empty todo-card list (or a clear client empty state), so that I am not shown another store’s work.
3. As a user with `member.write`, I want a **新伙伴待确认** card when there is at least one pending 入店申请, so that I can review join requests.
4. As a user without `member.write`, I want the partner-pending card omitted, so that I do not see actions I cannot take.
5. As a store admin, I want **待跟进客户** when overdue follow-ups exist, so that I can work the queue.
6. As a store admin, I want **售后预约** when pending future-or-today appointments exist, so that arrivals are not missed.
7. As a store admin, I want **订单待审核** when store review orders await review, so that staff work is visible on Home.
8. As a user with zero count for a type, I want that card omitted, so that the home strip only shows live work.
9. As a client, I want cards ordered `partner_pending` → `follow_up_customer` → `after_sales_appointment` → `order_pending_review`, so that large tier comes before medium before small without client re-sorting by size.
10. As a client, I want each card to include `type`, `title`, `subtitle`, `action_label`, `action_route`, `count`, and optional `image_url`, so that UI can render without hardcoding copy (except size map and fallback art).
11. As a client, I want **no `size` field** from the server, so that type→size stays a single client table shared conceptually with ADR 0009.
12. As a Flutter user, I want `partner_pending` rendered as a **large** card (4 cells), so that join approvals dominate the page.
13. As a Flutter user, I want `follow_up_customer` and `after_sales_appointment` as **medium** cards (2 cells, full row), so that they share a page with small cards when needed.
14. As a Flutter user, I want `order_pending_review` as a **small** card (1 cell), so that it fills remaining slots.
15. As a Flutter user with only ≤2 small cards and no medium/large, I want a single wrapped row without horizontal paging, so that empty grid space is not reserved.
16. As a Flutter user with more work than that, I want horizontal pages of at most 4 cells, so that I can swipe between batches.
17. As a Flutter packer, I want valid pages of: 1 large; or 2 medium; or 4 small; or 1 medium + 1–2 small, so that layout matches the product grid rules.
18. As a Flutter packer, I want at most one large card per page, so that a large card never shares a page.
19. As a user tapping a card CTA, I want navigation via `action_route` (with type→route fallback), so that deep links can change without a client release for copy-only moves.
20. As an applicant, I want to submit an 入店申请 for a store, so that membership is not silently granted.
21. As a store member with `member.write`, I want a list of pending 入店申请 for the current store, so that I can review who is waiting.
22. As a store member with `member.write`, I want to **approve** an application and have the user become a 门店成员, so that they can work in the store.
23. As a store member with `member.write`, I want to **reject** an application, so that bad requests do not linger as pending.
24. As the system, I want approving an already-member applicant to be a safe no-op or clear error without duplicate members, so that `wys_store_member` stays one row per user per store.
25. As a store admin, I want a read-only list of 待跟进客户 where `next_follow_up_at <= now`, so that the card count matches the list.
26. As a store admin, I want a read-only list of 售后预约 that are pending and scheduled for today or later, so that expired items do not inflate the home card.
27. As a store admin, I want a read-only list of 店务审核单 awaiting review, separate from mall buyer orders, so that staff review is not tangled with `wys_mall_order` payment states.
28. As a developer, I want all of these APIs behind SessionAuth, so that counts are bound to the authenticated user and current store.
29. As a developer, I want first-phase permission map: partner → `member.write`; follow-up / after-sales / order-review → `store_admin`, so that gating is explicit and small.
30. As a QA engineer, I want HomeTodo usecase tests that omit zero/unauthorized types and preserve sort order, so that aggregation stays stable.
31. As a QA engineer, I want domain usecase tests for approve/reject join applications and count definitions, so that home numbers match lists.
32. As a Flutter engineer, I want packing unit tests for page compositions (1L, 2M, 1M+2S, ≤2S wrap), so that swipe layout does not regress.
33. As a LAN demo user, I want seed data that can produce at least one large, one medium, and one small card together, so that packing and swipe can be verified on device.
34. As a user whose subtitle includes a count, I want the subtitle to stay consistent with `count`, so that UI does not invent a second number.
35. As a future client (e.g. KMP), I want the same aggregate JSON contract, so that packing can be reimplemented without a new BFF shape.

## Implementation Decisions

### Modules & seams

- **HomeTodo usecase (new, primary aggregate seam):** resolve current store; for each known type, if actor has permission and domain count > 0, append a card with server-owned copy + `action_route` + `count`; emit in fixed type order. Controllers thin.
- **Domain usecases (new):** Store Join Application (create/list/approve/reject); Follow-up Customer (list + count); After-sales Appointment (list + count); Store Review Order (list + count). HomeTodo depends on count ports only—not on HTTP DTOs.
- **Flutter home todo experience:** replace `HomeQuickAction` mock grid with todo-card strip; type→size table; packer; PageView vs wrap; API client; four list screens; approve/reject on partner list.
- **ADR 0009:** server never sends `size` or `pages[]`.

### Domain rules (frozen in grilling)

| Type key | UI title | Size (client) | Count definition | Permission (phase 1) |
|----------|----------|---------------|------------------|----------------------|
| `partner_pending` | 新伙伴待确认 | large (4) | pending 入店申请 in current store | `member.write` |
| `follow_up_customer` | 待跟进客户 | medium (2, full row) | customers with `next_follow_up_at <= now` | `store_admin` |
| `after_sales_appointment` | 售后预约 | medium (2, full row) | pending appointments with date ≥ today (Shanghai day) | `store_admin` |
| `order_pending_review` | 订单待审核 | small (1) | store review orders awaiting review | `store_admin` |

- Sort = table order above (already large→medium→medium→small).
- Same-size medium order: follow-up before after-sales.
- count=0 or missing permission → omit card.
- Scope = **当前店** (`users.current_store_id`); not all stores summed.
- 入店申请: applicant applies → `pending|approved|rejected`; approve creates/ensures 门店成员; reject closes pending.
- 店务审核单 ≠ mall order; do not overload `wys_mall_order`.
- Medium card occupies a **full row** in the 2×2 page.
- ≤2 small-only → wrap one row, no pager; otherwise page by 4 cells; last page may be partial.
- Writes in phase 1: join approve/reject required; other three domains read-only lists OK if write would balloon the first tickets.

### Schema (directional)

- `wys_` tables for: store join applications; store customers (with next follow-up time); after-sales appointments; store review orders. Status enums as needed for pending/approved/rejected or pending/done.
- Do not reuse mall order rows for 订单待审核.
- Existing `wys_store_member` remains the membership truth after approve; applications are the pending queue in front of it.
- Seed LAN rows so aggregate can return mixed sizes for packing QA.

### API contracts (directional, snake_case JSON)

- `GET /api/v1/home/todo-cards` — SessionAuth; ordered array of:
  - `type`, `title`, `subtitle`, `action_label`, `action_route`, `count`, `image_url` (nullable)
- Domain routes (names flexible in first ticket, behavior normative):
  - Join applications: create (applicant), list pending (store), approve, reject
  - Follow-up customers: list for current store (filter overdue)
  - After-sales appointments: list pending upcoming/today
  - Store review orders: list awaiting review
- Exact path strings may be finalized in implementation tickets; aggregate path above is preferred default.

### Flutter

- Remove reliance on hardcoded `HomeQuickAction` mock for these four titles.
- Client constant map: type → size + optional fallback route if `action_route` empty.
- Packing algorithm covered by unit tests; UI uses packer output for PageView children.
- List pages: one per type; partner page has approve/reject actions calling BFF.

### Auth / gating

- SessionAuth on all endpoints.
- Effective permissions as in access control; phase-1 type→permission map above. Platform admin behavior: follow existing access patterns (may see store-scoped data only with current store set—do not invent a cross-store home strip in phase 1).

## Testing Decisions

- Good tests assert **external behavior** at seams: which cards appear, order, omitted zeros/unauthorized; approve creates membership and clears pending count; follow-up/appointment/order counts match list filters. Do not assert SQL or private packer helpers’ names.
- **HomeTodo usecase:** table-driven permission × count matrices; fixed type order; empty current store.
- **Domain usecases:** join approve/reject transitions; count predicates with fixed clocks where dates matter (`Asia/Shanghai` for appointment “today”).
- **Flutter:** packer unit tests for 1L, 2M, 1M+2S, 4S, ≤2S wrap/no pager; widget/http smoke if prior art exists.
- Prior art: `access_usecase_test.go` for permission + current store; community/check-in Flutter prefs and SessionAuth clients.

## Out of Scope

- Server-side `size` or prebuilt `pages[]`.
- Per-type fine-grained permissions beyond the phase-1 map (can refine later).
- Full CRM / appointment scheduling product / mall staff order console.
- Cross-store aggregated todo strip.
- KMP implementation in the same tickets (contract should not block it later).
- Push notifications when new applications arrive.
- Auto-credit or points hooks for completing these todos.
- Changing existing immediate `POST .../members` admin upsert beyond coexisting with the application flow.

## Further Notes

- Multi-session build: next `/to-tickets` should tracer-bullet schema + join application + aggregate first, then other domains’ read models, then Flutter packer/UI, then partner write UI.
- Companion Flutter repo: `/Users/mac/Desktop/github/my_ai_project` (independent git).
- Today’s Flutter mock lives in `HomeRepository.loadDashboard()` / `HomeQuickActionGrid`; replace that strip, leave unrelated dashboard mocks alone unless they block.
- Glossary and ADR already capture vocabulary and type-vs-size split; keep tickets from re-litigating packing rules.
