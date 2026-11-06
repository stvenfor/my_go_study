# Slice — harden-h6-money-unit-audit

> **范围修订（2026-11-06）**：人已批「方案 + 直接落地 transaction 一条链路」。落地部分**拆出去**成 [`harden-h6b-transaction-money-unit`](./harden-h6b-transaction-money-unit.md)；本 Slice 只做**方案与影响面**，并必须先纠正一条我自己写错的前提（见下）。

## ⚠ 前提更正：金额口径是**三套**，不是两套

| 口径 | 代表字段（实测） | 出处 |
|------|------------------|------|
| **整数分 `int64`** | `AmountFen`、`LocalAmountFen`、`ChannelAmountFen`、`PriceFen` | `model/entity/payment.go:53,82,84`、`model/entity/membership.go:48,63`；DB `amount_fen bigint CHECK (amount_fen > 0)`（`migrations/20260923140000_membership_subscribe.up.sql:20`） |
| **`string` + `numeric(10,2)`**（decimal-as-string） | `Amount string gorm:"type:numeric(10,2)"`、`LineAmount` | `model/entity/mall.go:255,281,308,346`；DB `amount numeric(10,2)`（`migrations/20260921180000_mall_schema.up.sql:101,134,161,191`） |
| **`float64`** | `Amount float64` | `model/entity/transaction.go:16,28,36`、`transaction_record.go:14`、`used_car_order.go:138,180`、`auto_finance.go:52,60-70`；DB `amount numeric(14,2)`（`migrations/20260922221000_used_car_order.up.sql:15`）与裸 `numeric`（`migrations/20260923167500_align_gorm_only_schema.up.sql:40,53`） |

→ 原先「payment/wallet/mall 是整数分」这句**对 mall 是错的**：mall 走 decimal-as-string（本身是合理做法，但与 payment 的分口径不一致）。**两者之间的换算点比 float 更值得查**，是本 Slice 最有价值的产出。

## Slice Brief
- SOURCE_MODULE: `model/entity`（金额相关实体）、`app/internal/{transaction,used_car,auto_finance,mall,wallet,points,payment,membership}`
- TARGET_MODULE: 一份落盘的**口径与迁移方案**文档（本 Slice 不改代码）
- Source entry: 见上表 + 客户端直收 `app/internal/used_car/handler.go:94`（`Amount float64` 来自 JSON body，无精度与上限校验）、`app/internal/delivery/http/dto/request/transaction_request.go:8,17`
  - 反例（不是钱，别误伤）：`model/entity/mall.go:162,177 CoverAspect`（图片宽高比）、`analytics_record.go`、`short_video.go`；`auto_finance.go:37-45` 是**百分比/利率**，性质不同，必须单列分类
- Target entry: `docs/money-unit-audit.md` —— 三套口径的完整字段清单 + **跨界换算点清单** + 迁移到单一口径的方案，并直接作为 h6b 的输入契约
- 本轮 ONLY: **只读盘点 + 方案**（落地交 h6b）
- 不做: 不改任何实体、迁移、handler、DTO；不动 Flutter；不做「顺手加个校验」（要加就单独立 Slice）
- 验收:
  - [ ] 字段清单完备性**用排除法证明**：`grep -rnE 'float64|string.*numeric|int64' model/entity/` 命中金额语义的每条都要么归入「资金/金额类」要么显式标为「非钱（比例/尺寸/统计）」并写明理由 —— 不接受「只列我想到的」
  - [ ] 每个金额字段标注：①当前 DB 列类型（**查 `migrations/*.sql`，不要只信 entity tag**，本仓已有 entity 与列类型不一致的先例）②是否已被 `wallet`/`points`/`payment` 事务链路引用 ③是否出现在 Flutter 契约
  - [ ] **跨界换算点必须穷举**：至少覆盖 `mall` ↔ `payment`（decimal 元 ↔ 整数分）、`membership` ↔ `payment`、`wallet`/`points` ↔ `mall`（余额/积分抵扣），每条给 `file:line` 与「谁做 Round、往哪个方向丢精度」
  - [ ] 精度事故评估：给出「现有数据是否已失真」的可执行核查 SQL —— float 侧 `amount <> round(amount, 2)`；numeric 侧 `amount * 100 <> trunc(amount * 100)`
  - [ ] 迁移方案含：加列→回填→双写→切读→删列的 SQL 草案 + 每步回滚点与停机评估
  - [ ] **显式标注 Supabase 路径不可控**：`app/internal/transaction/repo_supabase.go:137,174` 直接把 `amount` 写进云端表，本仓 `migrations/` 管不到它 → 方案必须回答「`auth.provider=supabase` 时怎么办」（含「该路径不做迁移」也是一种结论）
  - [ ] 连带影响列出：PostgREST/RLS、ES mapping（`app/internal/transaction/repo_es.go:93` 写 `amount`，改单位需 reindex，见 `docs/elasticsearch-transactions.md`）、Flutter `TransactionApi`
  - [ ] **h6b 输入契约**：文档末尾给出「transaction 链路目标口径 + **对外 JSON 是否变**」的明确推荐（我的默认倾向：JSON 保持元、只改存储与内部计算，避免与 Flutter 同步发版），供 h6b 直接引用
  - [ ] 状态词：本 Slice 交付方案 → **Partial**（不许写 Migrated，落地在 h6b）
- 文件白名单:
  - `docs/money-unit-audit.md`（新建）
  - `docs/placeholder-inventory.md`（追加式：登记「金额三套口径」条目与状态）
  - `docs/acceptance-records/2026-11-06-harden-h6-money-unit-audit.md`
  - `plans/slices/harden-h6-money-unit-audit.md`（仅 Context Card）
- 文件黑名单:
  - `model/**`、`app/**`、`migrations/**`、`configs/**`
  - 任何 `_test.go`
- 验证命令:
  - `grep -rnE 'float64|string.*numeric' model/entity/ | wc -l`（与清单条数一致）
  - `git diff --stat` 证明本 Slice **零代码改动**（只允许 docs/ 变化）
  - `./scripts/load-env.sh psql -h 127.0.0.1 -U postgres -d <只读查询>` —— 只跑 `SELECT`，**禁止任何 DDL/DML**（业务库 `my_go_study` 属破坏性边界）
- 证据: `docs/acceptance-records/2026-11-06-harden-h6-money-unit-audit.md` + `docs/money-unit-audit.md`

## Context Card — harden-h6-money-unit-audit
- 已完成:
- 未做/Deferred:
- 关键文件:
- harness: post ok?
- 下一 Slice 建议:
- 已知坑:
