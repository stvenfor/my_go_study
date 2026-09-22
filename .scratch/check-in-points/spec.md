# Check-in & Points

**Status:** ready-for-agent

用语：`CONTEXT.md` §签到与积分  
ADR：[docs/adr/0008-mall-points-and-mixed-payment.md](../../docs/adr/0008-mall-points-and-mixed-payment.md)

实现按 `issues/` 阻塞边，从 frontier（无 blocker 或 blocker 已完成）抓票；每票独立会话 `/implement`。跨仓：Go BFF 本仓库；Flutter 在 `my_ai_project`。

| # | Ticket | Blocked by |
|---|--------|------------|
| 01 | [积分账本 + 每日签到](./issues/01-points-ledger-check-in.md) | — |
| 02 | [首页每日签到弹窗](./issues/02-daily-check-in-dialog.md) | 01 |
| 03 | [成长任务进度与领取](./issues/03-growth-tasks-claim.md) | 01 |
| 04 | [商城双价 + 积分/混合支付 + 退分](./issues/04-mall-points-payment.md) | 01 |
| 05 | [签到页积分货架 + 商城价展示](./issues/05-check-in-shelf-mall-prices.md) | 01, 04 |

**Frontier 起点：** 只开 **01**。01 完成后可并行 02 / 03 / 04；05 等 04。

测试缝（已确认）：
1. **Points usecase** — 余额/流水、签到、任务进度与领取
2. **MallUsecase（扩展）** — SKU 双价、订单支付方式一致、积分支付与退分（内部调 Points）
3. **Flutter 签到体验** — BFF HTTP 客户端 + 每日弹窗频控 + 入口迁移

---

## Problem Statement

用户需要每日签到攒积分、在签到页看到日历与任务、并用积分（或积分+人民币）在现有商城里消费。今天日历入口只 toast，签到商城是首页「生活服务」下的假页面，商城只能人民币支付，没有积分账本。

## Solution

在「我的」下提供统一的签到页（承接原签到商城 UI）：真签到、真积分、真成长任务（登录 / 发动态 / 下单后手动领奖）。当天首次进首页且未签到时弹轻量窗可直接签。现有商城 SKU 支持仅人民币、仅积分、混合支付；签到页底部展示可用积分购买的货架。首页「生活服务」改为「即将上线」提示。

## User Stories

1. As a logged-in user, I want to open the check-in page from the second top-right icon on Mine, so that I can see my check-in calendar and points.
2. As a logged-in user, I want the former check-in mall screen to live under Mine, so that I do not hunt for it under Home life services.
3. As a user tapping Home「生活服务」, I want a toast「生活服务即将上线」, so that I am not sent into the check-in flow from the wrong entry.
4. As a logged-in user opening Home for the first time that local calendar day without having checked in, I want a light daily check-in dialog, so that I can check in without opening the full page.
5. As a logged-in user who already checked in today, I want Home not to show the daily dialog, so that I am not interrupted.
6. As a logged-in user who already dismissed or completed the daily dialog today, I want Home not to show it again the same local day, so that it only appears once per day.
7. As a logged-out user, I want Home never to show the daily check-in dialog, so that I am not prompted into a half-auth flow.
8. As a logged-out user opening the check-in page, I want to be guided to log in, so that check-in and points require a real session.
9. As a logged-in user, I want to check in at most once per Shanghai calendar day, so that rewards cannot be farmed by repeating the action.
10. As a logged-in user checking in, I want to receive 5 points on a normal day, so that daily habit is rewarded.
11. As a logged-in user on streak day 3, I want that day's check-in to credit 10 points total, so that streaks feel worthwhile.
12. As a logged-in user on streak day 7, I want that day's check-in to credit 20 points total, so that weekly streaks feel worthwhile.
13. As a logged-in user, I want my check-in page to show recent calendar status and current points balance labeled only as「积分」, so that I am not confused by growth/coin aliases.
14. As a logged-in user, I want a daily「每日登录」growth task (+5) separate from check-in, so that simply having a session that day can be claimed once.
15. As a logged-in user who published a community post today, I want a「发动态」growth task (+10) to become claimable, so that content creation is rewarded.
16. As a logged-in user who successfully paid a mall order today, I want a「商城下单」growth task (+20) to become claimable, so that purchasing is rewarded.
17. As a logged-in user with a completed-but-unclaimed growth task, I want to tap「领取」to credit points, so that rewards are intentional and visible.
18. As a logged-in user, I want claiming a task to be idempotent for that Shanghai day, so that double taps do not double credit.
19. As a logged-in user, I want my points balance to be global across stores, so that changing current store does not wipe or split my wallet.
20. As a logged-in user, I want a points ledger of credits and debits, so that I can trust where points came from or went.
21. As a seller/catalog operator, I want SKUs to carry `price_cny` and `price_points` (either may be zero), so that products can be CNY-only, points-only, or mixed.
22. As a buyer, I want mixed SKUs to require paying the full CNY price and the full points price together, so that pricing rules stay simple.
23. As a buyer, I want points to be a mall payment channel with its own payment row, so that settlement and refunds are auditable.
24. As a buyer of a mixed order, I want the order paid only when both the points payment and the CNY-channel payment succeed, so that partial payment is not treated as complete.
25. As a buyer, I want points deducted at pay time (not at order create), so that abandoned carts do not freeze my balance.
26. As a buyer whose points payment fails or whose unpaid/paid-cancellable order is cancelled after points were deducted, I want points fully restored, so that the wallet stays correct.
27. As a buyer, I want an order to be rejected if its SKUs do not all share the same product payment mode, so that I am prompted to split the cart rather than hit undefined pay states.
28. As a buyer on the check-in page, I want a shelf of SKUs with `price_points > 0`, so that I can redeem without leaving the check-in experience for discovery.
29. As a buyer tapping a shelf item from the check-in page, I want the existing mall product detail and checkout flow, so that redeem reuses commerce infrastructure.
30. As a buyer browsing the normal mall, I want dual prices visible where relevant, so that I know whether I need CNY, points, or both.
31. As a developer, I want SessionAuth on all check-in/points/task APIs, so that rewards are bound to the authenticated user id.
32. As a developer, I want Community and Mall to remain unaware of points UI, only exposing facts Points needs (posted today / paid today), so that contexts stay decoupled.
33. As a user on Flutter, I want the daily dialog frequency key stored locally like the community convention dialog, so that popup spam is controlled even if the network is flaky.
34. As a user whose device local day differs from Shanghai day near midnight, I want eligibility still decided by the server Shanghai day, so that cross-timezone farming is limited.
35. As a QA engineer, I want automated tests at the Points and Mall usecase seams, so that streak math, claim idempotency, mixed pay, and refunds are locked without UI flakiness.

## Implementation Decisions

### Modules & seams

- **Points usecase (new, primary seam):** check-in for today, streak computation, balance, ledger append, growth-task progress snapshot, claim task reward. Controllers stay thin.
- **MallUsecase (extend existing seam):** SKU pricing fields, derive product payment mode, enforce order payment homogeneity at create, points payment channel at pay, refund points via Points on cancel/pay failure. Does not own the wallet.
- **Flutter check-in experience:** migrate route/entry to Mine calendar icon; Home life-services toast; daily dialog; wire Check-in Page to BFF; points-labeled UI; embed points shelf using mall list filtered by `price_points > 0`.
- **Dependency direction:** Mall → Points (debit/credit). Points may query Community/Mall read facts (or small ports) for「posted today」/「paid order today」. Community must not depend on Points.

### Domain rules (frozen in grilling)

- One global points ledger per user account.
- UI vocabulary: only「积分」(no parallel 成长值/i车币 as currencies).
- Check-in & tasks: Shanghai (`Asia/Shanghai`) natural day.
- Daily dialog frequency: device-local `yyyy-MM-dd`, once per day; eligibility to check in from server.
- Check-in rewards: 5 / 10 on streak day 3 / 20 on streak day 7 (points for that day total).
- Growth tasks (daily, manual claim): login +5, community post +10, mall paid order +20. Login ≠ check-in.
- Product payment mode from SKU `price_cny` + `price_points`.
- Mixed = fixed full CNY + full points; no sliding offset.
- Points payment channel; mixed = two payment rows; both success ⇒ paid.
- Debit points at pay action; full restore on fail/cancel when deducted.
- Order payment homogeneity required.

### Schema (directional)

- Points wallet / ledger tables (`wys_` prefix): balance (or derived), ledger entries with reason codes (check_in, task_*, mall_pay, mall_refund, …), check-in days, task claim records per user per Shanghai day.
- Mall SKU: add points price beside existing CNY price; migrate existing `price` semantics to `price_cny` (or keep `price` as CNY alias in DB with API exposing both). Payment channel enum gains points.
- Seed at least one points-only and one mixed SKU for LAN demo.

### API contracts (directional, snake_case JSON)

- Authenticated SessionAuth group under something like `/api/v1/points/...` and/or `/api/v1/check-in/...` for: balance (+ optional ledger), today status / check-in POST, tasks list with progress + claim POST.
- Mall product/SKU/order/pay responses expose CNY price, points price, payment mode, and accept points channel on pay; create order returns clear error when homogeneity fails.
- Exact path names may be finalized in the first implementation ticket; behavior above is normative.

### Flutter

- Mine `onCalendarTap` → check-in page route (move registration from Home module to Settings/Mine as appropriate).
- Home「生活服务」→ toast only.
- Daily check-in dialog on Home first entry; prefs key pattern like community convention ack date.
- Replace mock coins/growth with points balance and server calendar/tasks.
- Reminder toggle on the mock page: out of scope unless already trivial local-only; do not build push.

### Auth / gating

- Same gate as mall business APIs (local SessionAuth). No anonymous check-in credits.

## Testing Decisions

- Good tests assert **external behavior** at the seam: inputs/outputs and invariants (cannot double check-in; claim twice same day no-ops or errors without double credit; mixed pay needs both legs; cancel restores points; homogeneity rejection). Do not assert SQL shape or private helpers.
- **Points usecase:** table-driven day/streak/claim cases with fixed clocks in `Asia/Shanghai`.
- **MallUsecase:** pay with points channel; mixed success; cross-mode create rejected; cancel after points pay restores balance (via Points fake or real in-memory adapter).
- **Flutter:** unit/widget tests for dialog gating (logged out / already checked in / prefs already shown today); navigation entry smoke if prior art exists. Prefer small tests over golden screenshots.
- Prior art: mall usecase tests; community convention dialog prefs pattern; SessionAuth middleware tests.

## Out of Scope

- Points-as-discount / sliding offset mixed payment.
- Per-store points wallets.
- Separate points-only catalog app.
- Growth tasks: 回复客户消息、邀请新销售顾问, or auto-credit without claim.
- Task engine config admin UI; keep seed/constants.
- Real Alipay/WeChat SDK changes beyond local simulated CNY channels already in mall.
- Check-in push reminders / notification scheduling.
- Cross-device dialog frequency sync (server-side「dialog shown」ack).
- Changing community post domain beyond exposing「posted today」for task progress.

## Further Notes

- This is a multi-session build: after this spec, split via `/to-tickets` into tracer-bullet issues with blocking edges (schema/Points first → mall pay → Flutter entries/dialog → shelf).
- Companion Flutter repo: `my_ai_project` (not a git submodule of this repo).
- Existing mock `CheckInMallPage` is the visual starting point; behavior becomes server-backed.
- Near midnight, local dialog day and Shanghai eligibility can diverge by design; document in client copy only if UX confuses QA.
