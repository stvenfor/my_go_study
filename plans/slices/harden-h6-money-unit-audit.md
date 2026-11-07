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
  - `docs/money-unit-audit.md` 落盘：三套口径 44 条字段清单（mandated grep 38 条逐条 MONEY/NOT-MONEY 归类 + 排除法补 6 条盲区）
  - **跨界换算点穷举**（§4，首要产出）：mall↔payment 1 处、membership↔payment **0 处（负结论）**、
    wallet↔mall 2 处、points↔mall **无汇率（负结论）**、渠道边界 4 处、auto_finance 9 处，每条给 `file:line` + 谁做 Round + 丢精度方向
  - §3 entity 声明 vs `migrations/*.sql` **最终有效列型** 分歧表（12 行）
  - §5 精度事故核查 SQL 三组（float 侧 / numeric 侧 / 跨口径一致性 5 条）
  - §6.2 五步迁移（加列→回填→双写→切读→删列）每步含 SQL 草案 + 回滚点 + 停机评估；§6.1 目标口径推荐及四条理由
  - §7 Supabase 路径不可控 → 显式结论「该路径本轮不迁移」，登记 `Degraded`（inventory #23）
  - §8 连带影响 7 行（PostgREST/RLS、ES reindex、Flutter `TransactionApi` + Flutter 三套口径并存、Realtime、`wys_mall_refund` 僵尸表）
  - §9 h6b 输入契约 6 条（**含明确推荐：对外 JSON 不变**）
  - `docs/placeholder-inventory.md` 追加条目 **#22 金额三套口径（Partial）/ #23 Supabase 不迁移（Degraded）/ #24 `normalizeMoney` 静默舍入（Degraded）**
  - 零代码改动：`model/**`、`app/**`、`migrations/**`、`configs/**`、`*_test.go` 全部未动
- 未做/Deferred:
  - **§5 核查 SQL 全部未执行** → 「现有数据是否已失真」**无结论**（沙箱 PostgreSQL 未监听，`nc -z 127.0.0.1 5432` = closed）
  - 验收第 1 条的「清单条数 = 38」**不成立** → mandated pattern 漏 6 条，需修订 pattern 才能当门禁
  - Flutter 仅静态 `file:line` 取证，未跑 `dart analyze`
  - 14 项残留逐条列表见 acceptance-record「残留」表（normalizeMoney 舍入、`payment/service.go:285` 出站 float、
    `subsidy_value` 一列两义、可空性 vs 非指针、`wys_mall_refund` 僵尸表、Mixed 双计费、mall 金额双台账、遗留表等）
- 关键文件:
  - `docs/money-unit-audit.md`（主交付）
  - `docs/acceptance-records/2026-11-06-harden-h6-money-unit-audit.md`（证据）
  - `docs/placeholder-inventory.md` #22/#23/#24
  - 换算门：`model/entity/cash_money.go:10-22`（`ParseYuanToFen` **拒绝式**）、`:25-37`（`FormatFenToYuan` 整数运算）
  - 示范（严谨）：`app/internal/mall/service.go:303,342,346,347,364,380`（`big.Rat` 全程精确）
  - 反例（脏值入口）：`app/internal/transaction/repo_postgres.go:90,124-125` → 裸 `numeric`
  - 静默舍入（唯一）：`app/internal/mall/service.go:106` → `:816-827`
- harness: post ok? **未运行** —— 本仓 `make agent-post` / `check-brief` 机检脚本尚未落地（`.harness/rules/slice-contract.md` 标注「若已落地」）；
  本 Slice 依 `.harness/rules/no-silent-accept.md` 以「验证命令 + acceptance-record + Context Card」三项人工核对收口，状态只记 **Partial**。
  另：`go build` / `go test` 在本沙箱**无法运行**（无网络，`proxy.golang.org` 全 timeout → 「updates to go.mod needed」）；
  `go.work`/`go.mod`/`go.sum` 对 HEAD **零 diff**、代码目录零改动 → 编译不受影响，但**不得**称构建已验证为绿。
- 下一 Slice 建议:
  1. **先在 scratch/只读库跑 audit §5 的核查 SQL**（否则 h6b 的回填硬门槛无法判定），把「现网是否已失真」变成事实
  2. 执行 `plans/slices/harden-h6b-transaction-money-unit.md`（其验收第 1-3 条直接引用本 Slice §6.2 与 §5.1）
  3. 「`normalizeMoney` 舍入显式化」小 Slice（改动面最小、危害明确）
  4. `auto_finance`（float 入参 Round + 月供 float 指数 + `subsidy_value` 一列两义）专项 Slice
- 已知坑:
  - **别信 mandated grep 的「完备」**：`float64|string.*numeric` 要求同行有 `numeric`，
    故**无 tag 的 decimal-as-string 读模型全部漏**（6 条，含 Brief 自己引用的 `mall.go:346`）
  - **别信 entity gorm tag 当列型**：`20260924110000_align_column_drift.up.sql:1-10` 的存在就是为此，
    其头注自陈「34 列类型偏窄（double precision→numeric…）」
  - **`transactions.amount` 是裸 `numeric`，不是 float 列** → DB 能存下 >2dp 脏值，别假设「PG 帮我截到 2 位」
  - **别给积分补汇率**（§4.3 负结论查证过）；Mixed 模式是「现金 + 积分**双重计费**」不是抵扣，属产品口径
  - **别顺手改 `membership`↔`payment`**：零换算，任何「统一」都是在制造风险
  - 可空性坑：`wys_used_car_order.amount` / `wys_membership_order.amount_fen` 被 `align_column_drift:103,66` 放宽为 NULL，
    Go 侧却是非指针 `float64`/`int64` → **NULL 与 0 不可区分**；且 `amount_fen` 的 `CHECK (>0)` 与可空性互相矛盾
  - `wys_mall_refund` 表有列约束**但无 Go 实体、无写入方** → 去它那里查退款会得出「零退款」的错误结论
  - 本 worktree 的 `../../my_ai_project` **不存在**；Flutter 实际在 `/Users/qpy/Desktop/github/my_ai_project`
