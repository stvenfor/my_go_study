# Cash Wallet (人民币钱包)

**Status:** ready-for-agent

用语：`CONTEXT.md` §钱包（现金）  
ADR：[docs/adr/0015-mall-cny-refund-to-wallet.md](../../docs/adr/0015-mall-cny-refund-to-wallet.md)  
相关：积分见 §签到与积分、[ADR 0008](../../docs/adr/0008-mall-points-and-mixed-payment.md)（两本账，勿混）

实现按 `issues/` 阻塞边，从 frontier 抓票；每票独立会话 `/implement`。跨仓：Go BFF 本仓库；Flutter 在 `my_ai_project`。

| # | Ticket | Blocked by |
|---|--------|------------|
| 01 | [钱包账本 + 绑卡 + 充值 + Flutter 钱包页](./issues/01-wallet-ledger-cards-recharge.md) | — |
| 02 | [商城余额支付 + 人民币退回钱包](./issues/02-mall-balance-pay-refund.md) | 01 |
| 03 | [Flutter 商城支付选渠道（含余额）](./issues/03-flutter-mall-channel-picker.md) | 02 |
| 04 | [API/联调文档 + 验收记录](./issues/04-docs-acceptance.md) | 02, 03 |

**Frontier 起点：** 只开 **01**。01 完成后开 02；03 等 02；04 等 02+03。

测试缝（已确认）：
1. **Wallet usecase** — 余额/流水、绑卡、充值
2. **MallUsecase（扩展）** — 余额支付渠道、取消/失败人民币退回钱包
3. **Flutter 钱包体验** — `features/wallet` + 我的入口 + 商城渠道选择

---

## Problem Statement

用户在「我的」点「我的钱包」只有「开发中」toast；没有人民币余额、银行卡、充值，商城支付也选不了余额。积分是另一本账，不能当现金用。已付人民币的商城单取消时，也没有明确的现金退回去向。

## Solution

为每个账号开一本全局 **钱包**（人民币）：可看余额与流水、本地模拟绑卡、充值入账；商城人民币支付增加 **余额支付**；订单取消或人民币支付失败且已扣款时，按 ADR-0015 **退回钱包**（不退原 PSP）。积分与会员 Mock 支付保持不动。

## User Stories

1. As a logged-in user, I want to open「我的钱包」from Mine quick services, so that I reach a real wallet screen instead of a toast.
2. As a logged-out user opening the wallet entry, I want to be guided to log in, so that wallet actions require a session.
3. As a logged-in user, I want to see my wallet balance in CNY, so that I know how much cash I can spend.
4. As a logged-in user, I want a wallet ledger of credits and debits in reverse time order, so that I can trust where money came from or went.
5. As a logged-in user, I want my cash wallet to be global across stores, so that changing current store does not split or wipe my balance.
6. As a logged-in user, I want points and cash balance to remain separate, so that I am not confused which currency pays what.
7. As a logged-in user, I want to bind a simulated bank card with bank name and last four digits, so that recharge can pick a card.
8. As a logged-in user, I want to set one default bank card, so that recharge can preselect it.
9. As a logged-in user, I want to list and remove bound cards, so that I can manage what is shown on the wallet page.
10. As a logged-in user, I want the system never to store a full card number, so that local-sim data stays minimally sensitive.
11. As a logged-in user, I want to recharge with a free-entered amount between 0.01 and 50000 CNY, so that I can top up what I need.
12. As a logged-in user, I want optional UI preset amounts for recharge, so that common top-ups are one tap without changing the API contract.
13. As a logged-in user, I want to choose a simulated recharge channel (default/bound card, Alipay, or WeChat), so that recharge mirrors familiar pay UX while staying local-sim.
14. As a logged-in user completing recharge, I want my balance and ledger to update immediately, so that I can spend without waiting for a real PSP.
15. As a buyer of a CNY-only mall order, I want to select balance as the payment channel, so that I can pay without Alipay/WeChat.
16. As a buyer with insufficient wallet balance, I want balance pay to be rejected with a clear error, so that I know to recharge instead of partial-pay.
17. As a buyer of a mixed mall order, I want the CNY leg payable with balance while the points leg still uses points, so that mixed pricing still works.
18. As a buyer of a points-only mall order, I want balance pay to be rejected, so that cash cannot settle points prices.
19. As a buyer whose CNY was taken (any CNY channel including balance) and who cancels before fulfillment, or whose CNY pay fails after debit, I want that CNY credited to the wallet, so that money is not lost in local-sim.
20. As a buyer who paid with balance then cancels, I want the same wallet credit path as other CNY channels, so that refund behavior is uniform (ADR-0015).
21. As a buyer, I want balance deducted at pay time (not at order create), so that abandoned carts do not freeze cash.
22. As a developer, I want Mall to call Wallet for debit/credit and not own cash balance, so that one ledger remains the source of truth.
23. As a developer, I want SessionAuth on all wallet APIs, so that balances bind to the authenticated user id.
24. As a Flutter user, I want wallet UI in a dedicated `features/wallet` module, so that membership Mock pay stays in `features/pay`.
25. As a Flutter mall buyer, I want a payment channel picker that includes balance (and shows current balance), so that I am not stuck on hardcoded Alipay.
26. As a QA engineer, I want automated tests at Wallet and Mall usecase seams, so that recharge, insufficient balance, mixed CNY-via-balance, and cancel credit are locked without UI flakiness.
27. As a user, I want no withdraw-to-card and no pay-password gate in this phase, so that scope stays shippable on LAN BFF.
28. As a catalog operator, I want existing Alipay/WeChat/IAP CNY channels to keep working, so that balance is additive not a rewrite.
29. As a user viewing ledger reasons, I want distinguishable labels for recharge, mall pay debit, and mall refund credit, so that the list is readable.
30. As a user with no cards yet, I want recharge via Alipay/WeChat sim still available, so that first top-up does not require binding a card first.

## Implementation Decisions

### Modules & seams

- **Wallet usecase (new, primary seam):** ensure wallet row, balance read, ledger list, bind/list/update-default/delete card, recharge (sim success → credit). Controllers stay thin.
- **MallUsecase (extend existing seam):** add balance as CNY payment channel (`6`); on pay debit Wallet; on cancel / CNY pay failure after money taken, credit Wallet per ADR-0015. Does not own balance. Points path unchanged.
- **Flutter wallet experience:** new `features/wallet`; Mine「我的钱包」→ wallet home; recharge + cards + ledger; mall order pay UI gains channel picker including balance.
- **Dependency direction:** Mall → Wallet (debit/credit). Wallet must not depend on Mall.

### Domain rules (frozen in grilling)

- One global cash wallet per user account; separate from points.
- Local-sim bank cards: `bank_name`, `card_last4`, optional `holder_name`, `is_default`; no full PAN; no debit/credit card type split.
- Recharge: free amount 0.01–50000; channels sim card / Alipay / WeChat; presets UI-only.
- Balance payment = CNY channel; insufficient → reject whole CNY leg; no balance+other CNY split pay.
- Mixed order: balance may pay CNY row; points row still points.
- Debit cash at pay action; CNY refunds credit wallet (not original PSP); no withdraw this phase; no pay password this phase.
- Channel code for balance: `6` (alongside existing 1–5).

### Schema (directional)

- `wys_` tables: cash wallet (balance), wallet ledger (amount, direction, reason, optional ref ids, timestamps), bank cards (user_id, fields above, unique constraints as needed for default).
- Mall payment channel check constraint / validators include `6`.
- Prefer reuse patterns from `wys_points_wallet` / `wys_points_ledger`; do not overload points tables for CNY.
- Existing `wys_mall_refund` may remain unused or lightly recorded; **wallet ledger credit is the user-visible cash refund truth** this phase.

### API contracts (directional, snake_case JSON)

- SessionAuth group under `/api/v1/wallet/...` roughly: GET summary (balance + optional cards peek), GET ledger (paginated), cards CRUD + set default, POST recharge `{ amount, channel, card_id? }`.
- Mall pay body already takes `payment_channel`; accept `6` for CNY legs; errors for insufficient balance and points-only misuse.
- Exact paths finalized in first implementation ticket; behavior above is normative.

### Flutter

- Replace Mine wallet toast with navigation into `features/wallet`.
- Do not fold cash wallet into `features/pay` membership Mock.
- Mall pay: stop hardcoding Alipay-only; picker shows available CNY channels for the order mode (and points when required).
- Optional recharge amount chips are client-only.

### Auth / gating

- Same SessionAuth gate as mall / points business APIs. No anonymous wallet.

## Testing Decisions

- Good tests assert **external behavior** at the seam: recharge increases balance + ledger; delete/default card rules; pay channel `6` debits; insufficient rejects without partial debit; mixed CNY-via-balance + points; cancel after CNY taken credits wallet (for balance and for other CNY channels under ADR-0015). Do not assert SQL shape or private helpers.
- **Wallet usecase:** table-driven recharge bounds, card default uniqueness, ledger ordering.
- **MallUsecase:** channel `6` success/fail; cancel credit via Wallet fake or in-memory adapter; points-only rejects `6`; mixed two-leg still required.
- **Flutter:** navigation from Mine; channel picker includes balance when CNY due; prefer small widget/unit tests over goldens.
- Prior art: points wallet/ledger tests; mall points payment tests; Mine menu routing.

## Out of Scope

- Withdraw / cash-out to bank card.
- Pay password / old 小葵花 payPwd APIs.
- Real PSP recharge or original-channel CNY refunds.
- Balance + Alipay (or other) split settlement of one CNY price.
- Per-store cash wallets.
- Using balance to pay points prices.
- Membership / `features/pay` real charging.
- Real UnionPay bind / KYC / full PAN storage.
- Freezing/hold at order create.
- Admin tooling for adjusting balances.
- Changing points domain beyond coexistence.

## Further Notes

- Multi-session: split via `/to-tickets` after this spec; work blockers-first; `/clear` between `/implement` sessions.
- Companion Flutter repo: `my_ai_project` (sibling of `my_code_study`, not a submodule).
- 「通知支付」in the original ask maps to **mall order payment** channel selection, not Realtime `sys.notify`.
- Local-sim recharge success is immediate; document in API docs that production PSP would replace the sim adapter later without changing wallet ledger semantics.
