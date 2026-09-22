# Car Purchase Calculator（购车计算器）

**Status:** done

用语：`CONTEXT.md` §购车计算  
ADR：[docs/adr/0012-car-purchase-calculator.md](../../docs/adr/0012-car-purchase-calculator.md)

实现按 `issues/` 阻塞边，从 frontier 抓票；每票独立会话 `/implement`。跨仓：Go BFF 本仓库；Flutter 在 `my_ai_project`。

| # | Ticket | Blocked by |
|---|--------|------------|
| 01 | [种子金融产品 + 服务端全款/贷款报价 API](./issues/01-seed-products-and-quote-api.md) | — |
| 02 | [Flutter 购车计算器页接真报价](./issues/02-flutter-calculator-ui.md) | 01 |

**Frontier：** 全部完成。

```text
01 → 02  (done)
```

测试缝（已确认）：

1. **PurchaseQuote usecase（主缝）** — 金融产品列表读模型；全款/贷款购车报价（购置税、险、手续费、贴息、等额本息月供）；校验首付/期数/产品约束
2. **HTTP 公开报价 API** — 无 SessionAuth 的产品列表 + 报价；错误码与响应形状稳定
3. **Flutter 购车计算器页** — 手填裸车价、选全款/贷款与产品、展示服务端报价；游客可进

---

## Problem Statement

买家在成交前想自己算清：全款大概要付多少（含税险杂费），贷款的话首付和月供是多少。今天 App 没有可信的购车估算；若只在客户端用写死利率，容易和真实金融产品脱节，也无法与后续门店/平台产品表对齐。

## Solution

提供买家自助的 **购车计算器**：从平台 **金融产品** 目录选产品，手填裸车价，由 BFF 算出权威 **购车报价**（全款方案或贷款方案）。第一期为 C-lite：法定近似购置税、交强险固定额、商业险比例粗算、一次性手续费、厂商贴息；贷款仅等额本息。与新车成交无关，不落方案库。

## User Stories

1. As a guest buyer, I want to open the car purchase calculator without logging in, so that I can estimate costs before committing to an account.
2. As a guest buyer, I want to enter a vehicle price manually, so that I am not blocked by a missing vehicle catalog.
3. As a guest buyer, I want to choose a cash purchase plan, so that I can see total amount due if I pay in full.
4. As a guest buyer, I want to choose a loan purchase plan, so that I can compare financing against cash.
5. As a guest buyer, I want to see a list of available auto finance products, so that I can pick a bank or manufacturer-finance style offer.
6. As a guest buyer, I want each finance product to show rate, allowed terms, and minimum down-payment ratio, so that I understand constraints before quoting.
7. As a guest buyer on loan path, I want to set down payment (amount or ratio within product rules) and term, so that the quote matches my budget.
8. As a guest buyer, I want the quote to include approximate purchase tax using taxable price ≈ bare price / 1.13 × 10%, so that the number matches common China dealer talk-tracks.
9. As a guest buyer, I want to override the taxable price used for purchase tax, so that NEVs or special cases are not stuck on the default formula.
10. As a guest buyer, I want compulsory traffic insurance as a fixed default amount, so that cash and loan totals include a realistic mandatory line.
11. As a guest buyer, I want commercial insurance estimated as bare price × configurable ratio (and optionally disabled), so that I get a rough all-in number without an underwriting engine.
12. As a guest buyer on loan path, I want a one-time loan fee included when the product defines it, so that fee-bait rates are not understated.
13. As a guest buyer on loan path, I want manufacturer interest subsidy (fixed rate cut or fixed amount cut) applied when the product defines it, so that promo offers are visible in the quote.
14. As a guest buyer on loan path, I want monthly payment computed with equal installment (等额本息) only, so that the schedule matches the most common consumer presentation.
15. As a guest buyer, I want the displayed quote to come from the server, so that I cannot accidentally trust a stale or tampered local formula.
16. As a guest buyer, I want clear validation when down payment or term violates the selected product, so that I know how to fix inputs.
17. As a guest buyer, I want cash and loan quotes to share the same tax/insurance cost lines, so that comparing modes is fair.
18. As a logged-in buyer, I want the same calculator behavior as a guest in v1, so that login is not a gate for estimation.
19. As a product operator (dev), I want 2–3 seed finance products on API boot, so that LAN demos work without an admin UI.
20. As a developer, I want purchase calculator domain language aligned with CONTEXT.md, so that agents do not confuse it with deal invoices.
21. As a developer, I want calculator APIs to remain independent of current store and deal-invoice tables, so that buyer self-serve does not require store membership.
22. As a Flutter user, I want a clear entry to the calculator from the app (home or tools area as implemented), so that the feature is discoverable.
23. As a Flutter user, I want loading and error states when the quote API fails, so that empty or wrong numbers are not shown as truth.
24. As a QA engineer, I want usecase tests locking tax, fee, subsidy, and amortization math, so that regressions fail without UI flakiness.
25. As a future buyer, I want “save my plan” deferred, so that v1 stays shippable without a plans list.
26. As a future operator, I want per-store product catalogs and contract-level deposit/early-repayment rules deferred, so that C-full does not block C-lite.

## Implementation Decisions

### Modules & seams

- **PurchaseQuote usecase (new, primary seam):** `ListFinanceProducts`; `QuoteCash`; `QuoteLoan` (names indicative). Owns C-lite formulas and product constraint checks. Controllers stay thin.
- **Finance product repository:** persist platform catalog rows (`wys_` prefix); `EnsureSeed` on API startup (same pattern as mall/community/deal-invoice seeds).
- **HTTP delivery:** public (no SessionAuth) routes under something like `/api/v1/purchase-calculator/products` and `/api/v1/purchase-calculator/quote`. Optional auth must not be required for v1.
- **Flutter calculator experience:** input bare price, mode toggle cash/loan, product picker, down payment & term, display server quote breakdown; guest-accessible entry.
- **Dependency direction:** Calculator must not depend on Deal Invoice, Store Current Store, or Mall checkout. No writes to deal or order tables.

### Domain rules (frozen in grilling)

- Actor: buyer self-serve (guest OK).
- Independent of 新车成交 / 成交发票.
- Platform finance product catalog; not per-store in v1.
- Server-authoritative 购车报价.
- Loan amortization: 等额本息 only.
- C-lite lines: purchase tax (法定近似), compulsory insurance fixed, commercial insurance % of bare price (optional off), one-time fee, manufacturer subsidy (rate cut or amount cut).
- Out of v1 math: deposit occupancy, early-repayment penalty into quote, equal-principal amortization.
- Input: manual bare price; optional taxable price override; no vehicle catalog.
- No quote persistence /「我的方案」in v1.
- Seed 2–3 products; no admin API/UI in v1.

### Formula notes (directional, lock in usecase tests)

- 购置税默认：`taxable = bare_price / 1.13`，`purchase_tax = taxable * 0.10`；request may override `taxable`.
- 交强险：product or global default fixed amount (seed picks a sensible passenger-car default).
- 商业险：`bare_price * commercial_rate`；rate 0 or flag disables.
- 贷款额：由裸车相关应贷基数按产品规则推导（首付后余额）；贴息按产品类型先减利率或减总额再摊月供——具体字段在实现票里固定，测试用金样例锁住。
- 等额本息月供：标准公式；返回月供、利息合计、还款总额、分项明细。

### Schema (directional)

- `wys_auto_finance_product` (name indicative): id, display name, annual rate, allowed terms (months), min down-payment ratio, one-time fee, subsidy type/value, compulsory insurance default, commercial rate, enabled flag, timestamps.
- No quote/history table in v1.
- Seed ≥2 products (e.g. one bank-like, one manufacturer-subsidy-like).

### API (directional)

- `GET .../products` → list enabled products (public).
- `POST .../quote` → body: mode `cash|loan`, bare_price, optional taxable_price, product_id (loan), down_payment, term_months, optional disable_commercial; response: line items + totals + (loan) monthly payment / interest total.
- Validation errors for unknown product, term not allowed, down payment below minimum, non-positive prices.

### Flutter

- New or reused settings/tools surface for calculator UI; call public BFF endpoints; do not treat local math as source of truth.
- Cross-repo: implement UI in `my_ai_project` against this contract.

## Testing Decisions

- Prefer **external behavior** at the PurchaseQuote usecase: given product + inputs → exact money breakdown (use fixed decimal/int cents strategy consistent with repo money handling).
- HTTP tests optional/smoke: public access (no auth header) returns 200 for happy path; 4xx on validation.
- Prior art: `home_todo_usecase_test.go`, points/mall usecase tests — table-driven cases, in-memory or seeded repo.
- Flutter: manual LAN check sufficient for v1 unless an existing widget test harness is trivial to extend; no requirement for golden tests.
- One runnable check minimum for non-trivial amortization + tax (Ponytail).

## Out of Scope

- Per-store finance catalogs and store admin UI
- Saving / listing / sharing「我的方案」
- Vehicle model catalog and MSRP binding
- 等额本金
- Deposit and early-repayment penalty as computed quote lines
- Real bank/OEM API integration
- Tie-in to 新车成交, 店务审核单, or mall orders
- OCR / credit application / e-sign
- Pushing calculator entry into sales advisor current-store workflows

## Further Notes

- First intentional **public business** read/compute API beyond auth/health; keep rate-limit concerns out of v1 unless abuse shows up in LAN.
- Q3 grilled as「合同级」intent; v1 explicitly **C-lite** — do not silently expand into C-full inside implement tickets.
- After tickets are approved via `/to-tickets`, frontier starts at the first vertical slice that can demo quote math end-to-end.
