# Acceptance Record — harden-h6-money-unit-audit

- 日期：2026-11-06
- Slice：`plans/slices/harden-h6-money-unit-audit.md`（权威执行单位，已完整读取）
- 角色：只读盘点 + 方案（**本轮 ONLY 不含落地**，落地在 `harden-h6b-transaction-money-unit`）
- 主交付物：`docs/money-unit-audit.md`（三套口径完整字段清单 + **跨界换算点清单** + 五步迁移方案 + h6b 输入契约）
- 取证基线：分支 `maoqian/harden-h6`，HEAD `e288553`

## status: **Partial**

note：方案与影响面已交付且可被 h6b 直接引用；但
（a）**§5.1 float 侧精度事故 SQL 已取证 —— 但执行者 = coordinator，非本 worker**。
worker 侧沙箱当时 PG 不可达（`nc -z 127.0.0.1 5432` 失败、`docker ps` 无输出），本 Slice 未执行任何 SQL；
coordinator 于 2026-11-06 在**同一台机器**以只读方式补跑同一条 SQL：
`docker exec my_go_study_postgres psql -U postgres -d my_go_study -At -c "SELECT count(*) FROM transactions WHERE amount IS NOT NULL AND amount <> round(amount,2)"`
→ `transactions` **0** / `transaction_records` **0** / `wys_used_car_order`（`amount IS NULL`）**0**；**零 DDL/DML**，未碰任何写路径。
**本结论只覆盖本机 `my_go_study`**，其它环境/生产库仍需各自跑；
且 **§5.2（numeric 侧 `amount * 100 <> trunc(amount * 100)`）与 §5.3（跨界 A–E 与账本求和不变式）仍未执行**，
「mall/payment/wallet 数据是否已失真」这一子问题本 Slice **无结论**，不得据此声称全量已核查；
（b）Brief 的验证命令 `grep -rnE 'float64|string.*numeric' model/entity/ | wc -l`（=38）
与「清单条数一致」这一验收项**本身不成立** —— 该 pattern 会漏掉 6 条无 `numeric` tag 的金额字符串读模型（见验收第 1 条），
需修订 pattern 后才能当门禁；本轮按 **44 条 + int64 分口径清单** 交付，故整体只记 Partial。
状态词依 `.harness/rules/status-enums.md` 五态之一；实现类结论一律留给 h6b。

---

## 逐条验收证据

### ☐ 1. 字段清单完备性**用排除法证明**（不接受「只列我想到的」）

**未达成** → 记 Partial。做法与结果：

| 步骤 | 命令 | 结果 |
|------|------|------|
| mandated grep | `grep -rnE 'float64|string.*numeric' model/entity/ \| wc -l` | **38** |
| 加 int64 的排除法 grep | `grep -rnE 'float64|string.*numeric|int64' model/entity/ \| wc -l` | **173** = 38 + 135，两分支可加和不重叠 ✓ |
| int64 总量 | `grep -rnE 'int64' model/entity/ \| wc -l` | **135** |
| **残差（四类 pattern 全取反）** | 见 audit 附录 | **0** → 135 行 int64 无遗漏 ✓ |

38 条**逐条**归入 MONEY / NOT-MONEY 并写理由，落表在 `docs/money-unit-audit.md` §2.1（38 行，编号 1-38 与 grep 输出一一对应）：
MONEY 24 条 / NOT-MONEY 14 条（24+14=38，与 grep 条数吻合；细分见 audit §2.1 小计行）。
Brief 点名的反例全部显式豁免并给出**依据而非印象**：

- `mall.go:162,177 CoverAspect` → NOT-MONEY，依据 `migrations/20260921180000_mall_schema.up.sql:40`
  的列注释「封面高/宽，瀑布流用」+ `CHECK 0.50..2.00`（`:33`）；
- `short_video.go:26 AspectRatio` → NOT-MONEY（宽高比）；
- `auto_finance.go:37,39,42,45,65` → NOT-MONEY（利率/比例），且其**源列是 BPS 整数**
  （`annual_rate_bps`/`min_down_payment_bps`/`commercial_rate_bps`，`align_gorm_only_schema.up.sql:126,128,133`），
  与钱分属两类，已按要求**单列分类**；
- `analytics_record.go` → 拆成两种：**#33/#34 `MetricRevenue`/`MetricCost` 语义是钱但性质是投放统计**
  （不可花、不可退、无借贷平衡、不参与结算）→ 判 NOT-MONEY-for-migration 但显式标注 money-adjacent；
  `MetricROI`/`MetricBounceRate`/`ScoreQuality`/`ScoreRisk` → 比值/评分，NOT-MONEY。

**排除法暴露出的 Brief 自身缺口（这是本片对抗性最强的产出）**：
mandated pattern 要求同行出现 `numeric`，因此**没有 gorm `numeric` tag 的 decimal-as-string 读模型全部扫不出来**。
补扫 `grep -rnE '\b(Amount|Price|Balance|Fee|Cost|Total|LineAmount|SubsidyAmountCut)\b[[:space:]]+string' model/entity/`
→ 额外 **6 条金额字段**被漏（`docs/money-unit-audit.md` §2.2）：
`mall.go:179`（`MallShelfItem.Price`）、`mall.go:190`（`MallSKUOffer.Price`）、`mall.go:346`（`MallOrderListItem.Amount`
—— **Brief 自己在第 10 行引用了这条**，但用它给的 pattern 抓不到它）、`membership.go:64`（`MembershipPlan.Price`）、
`cash_wallet.go:65`、`cash_wallet.go:78`（两个 `Balance` 元字符串）。
→ 真实金额字段数 = 38 + 6 = **44**。「清单条数 = 38」这条验收项在 pattern 修订前**不可能为真**，故第 1 条不勾。

### ☑ 2. 每个金额字段标注 ①真实 DB 列类型（查 `migrations/*.sql`）②是否被 wallet/points/payment 链路引用 ③是否出现在 Flutter 契约

落表：`docs/money-unit-audit.md` §3（entity 声明 vs 迁移声明 vs **最终有效列型** 三列对照）+ §2.3 分口径表 + §8 Flutter 行。

**Brief 警示的「entity 与列类型不一致的先例」被证实，且比预想严重**：

1. `analytics_records.metric_*`：建表 `20260917000000_create_analytics_records.up.sql:20-26` 写 `DOUBLE PRECISION`，
   被 `20260924110000_align_column_drift.up.sql:123-128` 全量 `ALTER ... TYPE numeric`。
   该迁移头注（`:1-10`）自陈原因是「**迁移声明比实际更窄** …34 列类型偏窄（integer→bigint、**double precision→numeric**…）」
   → 即 AutoMigrate 实际形态与 entity tag 都跟迁移声明不一致，**前例确凿**。
2. **本片最严重的发现，且方向与 Brief 不同**：`transactions.amount` **不是 float-only**，是**裸 `numeric`（无精度无标度）**
   （`align_gorm_only_schema.up.sql:40`；`transaction_records.amount` 同 `:53`）
   而 Go 侧 `float64`（`transaction.go:16`、`transaction_record.go:14`）。
   → 后果：**DB 原样收下 float 累加残值（如 `0.30000000000000004`），Go 读回即失真**；
     「精度事故」的源头在**入站不 Round**，不只是「用了 float」。
3. **可空性与类型语言不符（新发现，Brief 未列）**：
   - `wys_used_car_order.amount`：`numeric(14,2) NOT NULL`（`20260922221000_used_car_order.up.sql:15`）
     → `align_column_drift.up.sql:103` **`DROP NOT NULL`**，而 Go 是**非指针 `float64`**（`used_car_order.go:138`）
     → **NULL 与 0 在代码里不可区分**；
   - `wys_membership_order.amount_fen`：`bigint NOT NULL CHECK (amount_fen > 0)`（`20260923140000_membership_subscribe.up.sql:20`）
     → `align_column_drift.up.sql:66` **`DROP NOT NULL`**，而 Go 是**非指针 `int64`**（`membership.go:48`）
     → **可空性与「必须为正」约束互相矛盾**：0 被 CHECK 挡住，脏值只能以 NULL 形态存在并被读成 0。
     核查 SQL §5.3(E) 就是量这个。
4. `wys_used_car_order` 是 `numeric(14,2)`（Brief 说对了），上限 9,999,999,999.99 元远超 float64 精确整数域需求。

**②是否被 wallet/points/payment 链路引用**（穷举后）：

- **`transactions` / `transaction_records` 完全没有被资金链路引用** —— 负结论，取证方式：
  `grep -rn "wallet\|points\|payment" --include="*.go" app/internal/transaction/` → 仅 1 条**注释**命中
  （`repo_supabase.go:217`，讲 memwallet 测试桩），无代码依赖；
  `grep -rnEi "sum\(amount|aggregate" --include="*.go" app/internal/` → **0 命中**（Go 侧不做金额聚合）。
  → 所以 h6b 迁 `transactions` **不会牵动 payment/wallet/points 事务链**，可独立发布（这是给 h6b 的关键解耦信息）。
- 被引用的金额字段（口径 A/B）：`payment.go:53,82,84`、`membership.go:48,63`、`cash_wallet.go:30,40,41`、
  `mall.go:210,255,278,281,308`、`points.go:33,43,44`、`auto_finance.go:21,23,24` —— 引用点逐条见 §4 表。

**③Flutter 契约**：`my_ai_project` 在本 worktree 的 `../../` **不存在**，实际位于 `/Users/qpy/Desktop/github/my_ai_project`；
只做静态取证（见第 8 条），**未跑 `dart analyze`** → 该子项运行时部分记 Partial。

### ☑ 3. **跨界换算点必须穷举**（mall↔payment、membership↔payment、wallet/points↔mall），每条给 `file:line` 与谁做 Round

主表：`docs/money-unit-audit.md` §4（含 §4.6「谁做 Round」汇总）。**这一条是本 Slice 的首要产出。**

| 边界 | 换算点 | 谁做 Round | 精度 |
|------|--------|-------------|------|
| **mall(B) ↔ payment(A)** | `app/internal/mall/service.go:553` `ParseYuanToFen(order.Amount)` → `:563 AmountFen: fen` | **mall 侧** | **无损** |
| 　同上（本地模拟渠道双台账） | `mall.go:308`（`wys_mall_payment.amount numeric(10,2)`）与 `payment.go:53`（`amount_fen bigint`）**并存** | — | 非换算错误，是**双真相源**，记 Deferred |
| **membership(A) ↔ payment(A)** | **0 处** —— `PriceFen` 直通 `AmountFen`：`app/internal/membership/service.go:183,201,214,253,285,334,386,444` | 无 | 直通，**不该被顺手改** |
| 　membership(A) ↔ wallet(A) | `membership/service.go:237` `walletSvc.Debit(ctx, userID, plan.PriceFen, ...)` | 无 | 同为分，无损 |
| 　目录价 A→B（对外展示） | `model/entity/membership.go:90` `Price: FormatFenToYuan(fen)` | entity | 整数运算，精确 |
| **wallet(A) ↔ mall(B)** | 扣款 `app/internal/mall/service.go:707` → `:711 walletSvc.Debit(fen)` | **mall 侧** | **无损** |
| 　退款 | `app/internal/mall/service.go:756` → `:764 walletSvc.Credit(fen)` | **mall 侧** | **无损**；**复用订单原值不重算** → 无「退多退少一分」 |
| 　充值入站 | `app/internal/wallet/service.go:115 ParseYuanToFen`（+ `:116` 夹 `[MinFen,MaxFen]`） | wallet | 无损，但**上限 50000.00 元** |
| 　余额出站 | `wallet/service.go:36`、`:143 FormatFenToYuan` | wallet | 精确 |
| **points ↔ mall** | **不存在换算（负结论）**：积分是 SKU 上**平行价**（`mall.go:210` 元 + `mall.go:211` 积分两列并存），`MallSKUPaymentMode`（`model/entity/mall.go:88-100`）按谁非零派生三态；结算时 `mall/service.go:699` 扣积分与 `:707` 扣钱包**在同事务内并列全额执行，互不折算** | 无 | **全仓查不到任何积分↔元汇率常量或换算函数** → 禁止后续「补汇率」 |

**舍入方向与丢精度侧**（为什么「拒绝」比「四舍五入」安全，是本轮给出的可复用结论）：

- `entity.ParseYuanToFen`（`model/entity/cash_money.go:10-22`）：`big.Rat × 100` 后 `if !r.IsInt() { return 0, err }`
  → **>2 位小数直接报错，绝不静默进位**。mall/wallet/渠道入站全走这道门 → 判定 **✅ 安全**。
- 唯一的**静默吃精度**点：`app/internal/mall/service.go:106` `normalizeMoney`（后台改价）
  → `formatMoney(r) = r.FloatString(2)`（`mall/service.go:825-827`）**四舍五入、无告警**。
- 唯一**刻意让资金过 float 的出站点**：`app/internal/payment/service.go:285`
  `fmt.Sprintf("%.2f", float64(amountFen)/100.0)`（支付宝下单报文）→ 数值上不致错（分口径 ≤1e10 在 2^53 内），
  但**模式上是 float 参与资金**，且 `%.2f` 会进位。建议改 `FormatFenToYuan`，属另一 Slice。
- 渠道入站：微信 v2 `total_fee` 本就是分 → `verify.go:60 strconv.ParseInt`，**无换算**；
  支付宝 `total_amount` 是元 → `channel.go:129` / `verify.go:118 ParseYuanToFen`，**拒绝式**。
- `auto_finance`（§4.5，风险最高，因为「算」而非「存」）：落库与运算全程 `int64` 分（值得肯定），
  但 `purchase_calculator/service.go:255 yuanToFen = int64(math.Round(y*100))` **对客户端入参静默进位**、
  `:259 fenToYuan = float64(f)/100` 出站转 float、`:281-287` 月供经 `math.Pow` 指数运算后单次 Round；
  `:154 totalRepayFen = monthlyFen * termMonths` → **末期舍入被吸收进「利息合计」**。
  ⚠ **`auto_finance.go:23 SubsidyValue` 一列两义**（`subsidy_type` 决定是 bps 还是 fen，`auto_finance.go:9-11`），
  消费点 `service.go:143`（当 bps 减利率）与 `:149`（当 fen 减本金）→ **误读即错 100 倍**。
- transaction 侧的 Round：**0 处** —— `grep -rnE 'Round|Trunc|decimal|ParseFloat' --include='*.go' app/internal/transaction/` → 无命中；
  入站 `dto/request/transaction_request.go:8,17`（`Amount float64` / `*float64`，`binding:"required"` 只管非零不管精度），
  落库 `repo_postgres.go:90`（Create）与 `:124-125`（Update）原样写入裸 `numeric`。
  → 客户端直收另一处：`app/internal/used_car/handler.go:94`（`Amount float64` 来自 JSON body，无精度与上限校验），
  唯一门槛是 `used_car/service.go:127` 的 `in.Amount <= 0`（拒非正数，**不拒小数位**）。

**示范（正向结论，供 h6b 抄）**：`mall` 下单聚合是全仓最严谨的资金运算 ——
`mall/service.go:303` `total := big.NewRat(0,1)`、`:342` 解析单价、`:346 Mul(unit, qty)`、`:347 Add`，
只在落库/展示时 `FloatString(2)`（`:364 LineAmount`、`:380 Amount`）。

### ☑ 4. 精度事故评估：给出「现有数据是否已失真」的**可执行**核查 SQL

**SQL 已交付；§5.1 已补跑取证，§5.2/§5.3 仍未执行**（本条按「交付可执行 SQL」这一 Brief 字面要求打勾，数据侧结论只覆盖 float 口径，整体仍是 Partial）。产物在 `docs/money-unit-audit.md` §5：

- §5.1 float 侧（C 口径）：`transactions` / `transaction_records` 的 `amount <> round(amount, 2)`
  （Brief 要求的口径；本文另加 `IS NOT NULL` 守卫，因为 `align_column_drift` 已放宽过 NOT NULL）
  + 样本查询 + `wys_used_car_order` 的 `amount IS NULL` 计数（对应第 2 条发现的可空性坑）；
- §5.2 numeric 侧（B 口径）：对 mall 全部 5 个金额列做 `amount * 100 <> trunc(amount * 100)` 的 UNION 计数
  （`line_amount` 无 CHECK，见下，所以它是最需要这条的一列）；
- §5.3 **跨口径一致性 5 条**（超出 Brief 要求，但这才是真会暴露事故的）：
  (A) `wys_mall_order.amount*100` vs `wys_payment_records.amount_fen`；
  (B) `wys_cash_wallet.balance_fen` vs `sum(wys_cash_ledger.delta_fen)`；
  (C) 积分同形；
  (D) `wys_mall_order.amount` vs `sum(line_amount)`；
  (E) `wys_membership_order.amount_fen IS NULL` 计数。

执行分工（**不隐瞒**）：worker 侧 `nc -z 127.0.0.1 5432` → **closed**、`docker ps` → 无输出，本 Slice 一条 SQL 都没跑过。
Docker 起来后，**coordinator**（非本 worker）于 2026-11-06 以只读方式补跑了 §5.1 三条计数：
`transactions` **0** / `transaction_records` **0** / `wys_used_car_order`（`amount IS NULL`）**0**，零 DDL/DML
（完整命令见本文开头 note（a））。**该结论只覆盖本机 `my_go_study`**，其它环境/生产库需各自跑。
§5.2（numeric 侧）与 §5.3（跨口径 5 条）**仍未执行** → B/A 口径「是否已失真」本 Slice 无结论，遗留见 §10 与残留表 1b。

### ☑ 5. 迁移方案含「加列→回填→双写→切读→删列」SQL 草案 + 每步回滚点与停机评估

`docs/money-unit-audit.md` §6.2 五步表（每步都有 SQL 草案 / 回滚点 / 停机评估 三列）：

1. **加列**：`ALTER TABLE ... ADD COLUMN IF NOT EXISTS amount_fen bigint`（**先不加 NOT NULL/CHECK**，避免存量不满足直接失败）
   → 无停机（PG `ADD COLUMN` 无默认值是 catalog-only，不重写表）；回滚 = `DROP COLUMN`。
2. **回填**：**前置硬门槛 = 第 4 条 §5.1(1) 返回 0 行**；按 id 分批 5k，`round(amount*100)::bigint`，
   `WHERE amount_fen IS NULL` 保证幂等可重跑 → 无停机（需避高峰，先在 scratch 量时间）。
3. **双写**：`repo_postgres.go:90`（Create）与 `:124-125`（Update）两个入口都补 `amount_fen`，读仍走旧列
   → 无停机；回滚 = 关双写，旧列未破坏故读不受影响。
4. **切读**：读路径以 `amount_fen` 为准、对外换算回元；单 flag 可切回 → 无停机，建议灰度（先 lab 后 lan）。
5. **收紧 + 删列**：`SET NOT NULL` + `ADD CONSTRAINT chk_transactions_amount_fen CHECK (amount_fen >= 0)`，
   观察 ≥1 周后 `DROP COLUMN amount` → **约束校验全表扫（大表长事务，需评估 `lock_timeout`）**；
   **删列不可逆** → 单独一次发布 + 事先 `pg_dump` 单表备份。

约束方向附带前置核查：`SELECT count(*) FROM transactions WHERE amount < 0`（`transactions.type` 才表达收支方向，
`transaction.go:14`）→ 若现网已有负值，降级为无 CHECK 并在 record 说明（与 h6b 验收第 2/3 条同口径）。

§6.1 给出**目标口径推荐及四条理由**（含「不引入第三方 decimal 库」，依据 AGENTS.md 全局约束 9
「勿盲跑 `go mod tidy` / 升级依赖必须单独一次改动 + 全量 `make test`」）；
§6.3 给出**其余链路各自的最短路径**，并明确哪些**不该照抄** transaction 方案
（`transaction_records` 遗留表建议不迁移；`mall` 的 B 口径不动只修 `normalizeMoney`；`auto_finance` 是纯契约变更需与 Flutter 同发；
`payment`/`membership`/`wallet`/`points` **已是目标态**）。

### ☑ 6. **显式标注 Supabase 路径不可控**，并回答「`auth.provider=supabase` 时怎么办」

`docs/money-unit-audit.md` §7，含取证与**明确结论**：

- 事实：`app/internal/transaction/repo_supabase.go:137`（Create `payload["amount"] = input.Amount`）
  与 `:174`（Update `payload["amount"] = *input.Amount`）把 float 金额直写**云端表**；
  两处都经 `client.WithUserToken(accessToken)`（`:128`/`:159`）→ 受云端 RLS 管辖，**本仓 `migrations/` 管不到**。
- **结论：`provider=supabase` 路径本轮不做迁移，显式记为 `Degraded`**（四条理由见 §7），
  并按 h6b 验收第 5 条要求登记进 `docs/placeholder-inventory.md`
  （**已追加条目 #22 金额三套口径 `Partial`、#23 Supabase 不迁移 `Degraded`、#24 `normalizeMoney` 静默舍入 `Degraded`**；
  同时在「状态速览」表 `:46-48` 加了三行索引）。
- 「该路径不做迁移」按 Brief 允许，**作为一种合法结论**写出，而不是留空。
- 残留风险已点名：本仓默认 provider 现为 local，`docs/local-auth-postgres.md` 支持回切 supabase；
  一旦回切，`amount_fen` 在云端不存在会**直接写失败** → 给出解除条件（云端补列，或代码显式禁止该组合）。

### ☑ 7. 连带影响列出：PostgREST/RLS、ES mapping、Flutter `TransactionApi`

`docs/money-unit-audit.md` §8 表格，七行（超出 Brief 要求的三条）：

- **PostgREST / RLS**：云端加列须同步 RLS policy + PostgREST schema 缓存（`NOTIFY pgrst`）；
  **不得**为省事改用 service_role（AGENTS 硬约束 + `.harness/rules/auth-token-hygiene.md`）。
- **ES mapping**：`app/internal/transaction/repo_es.go:48` `"amount": { "type": "double" }`，写入 `:93`，
  搜索结果解析 `:207`；reindex 脚本 `scripts/reindex_transactions_es.sh:15` 同为 `double`。
  → ES mapping **不可原地改 type**，换单位**必然 reindex**；建议 `scaled_float(scaling_factor:100)` 或 `long`。
  并点出**搜索语义坑**：若 `amount` 变分，用户按元搜金额会 100 倍偏（`docs/elasticsearch-transactions.md`）。
- **Flutter `TransactionApi`**：`features/home/lib/home/model/transaction_model.dart:22,39`
  （`(json['amount'] as num?)?.toDouble()` / `final double amount`）
  → **对外 JSON 保持「元」则 Flutter 零改动**（即第 9 条推荐）；改 `amount_fen` 则跨仓同步发版。
- 另外补的 4 行（都是证据，不是推测）：Flutter **同时承受三套口径**
  （`double`：`used_car_order_models.dart:120,148`、`purchase_calculator_models.dart:65,71`、`transaction_list_item.dart:118`；
  `String`：`mall_order.dart:68,79,139,156`、`:21`、`wallet_api.dart:106`；
  `int` 分：`wallet_models.dart:12,21,65,66,74,109,110,115`、`membership_models.dart:40`）
  → **服务端单方收敛会打破其中一侧**；Realtime/Push 载荷同批；
  **`wys_mall_refund` 僵尸表**：迁移有建表 + `CHECK (amount > 0)`（`mall_schema.up.sql:184-199`）但实体**只有表名常量**
  （`mall.go:63`）、无 struct、无写入方，真实退款走 `walletSvc.Credit`（`mall/service.go:756-764`）
  → **审计时别去这张表找退款，会得出「零退款」的错误结论**。

### ☑ 8. **h6b 输入契约**：文档末尾给出「transaction 链路目标口径 + 对外 JSON 是否变」的明确推荐

`docs/money-unit-audit.md` §9 共 6 条：

1. 目标口径 = **整数分 `int64`（`amount_fen bigint`）**，走 §6.2 五步，回填硬门槛 = §5.1(1) 零行、禁止 `trunc`；
2. **对外 JSON 不变** —— `amount` 字段名与「元」语义保持原样，
   `dto/request/transaction_request.go:8,17` 与 `dto/response/transaction.go:16` 本轮不动；
   → **本 Slice 正式确认 h6b 默认倾向成立**，理由：与 Flutter 同步发版超出本轮 + 每步可独立回滚 + §8 证据显示客户端已承受三套口径；
3. 约束方向 `CHECK (amount_fen >= 0)`，负向由 `type` 表达；执行前先跑负值核查，有则降级并留痕；
4. 新金额运算照 `big.Rat`（`mall/service.go:303-347`）或整数分 + `divRound`（`purchase_calculator/service.go:267-275`），
   **不要**照 `repo_postgres.go:90`；
5. **不要顺手改的清单**（membership / wallet / points / payment / mall）—— 这是第 3 条穷举的直接收益；
6. 必须留痕的不对称：Supabase（§7）+ ES（§8）→ 都进 placeholder-inventory。

### ☑ 9. 状态词：本 Slice 交付方案 → **Partial**（不许写 Migrated）

本 record 标题行即 `status: Partial`；**未在任何地方主张本片已达成迁移态或已完成数据核查**
（文中出现的 `Migrated` / 「已核查」字样，均为「不许写」「不得声称」这类禁止性表述，见第 12 行与 audit §5/§10）。
残留逐条列在本 record 末尾「残留（Deferred / Partial 明细）」表（14 项），并同步进 audit §10。
依 `.harness/rules/no-silent-accept.md`：第 1、4 条验收未达成 → **不勾**，只勾已达成者（2、3、5、6、7、8、9）。

---

## 前提更正复核（合同要求「先纠正一个错前提」）

| 合同说法 | 复核结论 | 证据 |
|----------|----------|------|
| 金额口径是**三套**不是两套 | **成立** | §1/§2 三套表 + 44 条字段清单 |
| 整数分：`payment.go:53,82,84`、`membership.go:48,63` | **成立** | §2.3；`amount_fen bigint CHECK(>0)` = `payment_ledger.up.sql:8,21`；`membership_subscribe.up.sql:20` |
| decimal-as-string：`mall.go:255,281,308,346` + DB `numeric(10,2)` | **成立但补充** | `mall.go:346` **不是**由 mandated grep 命中（该行无 `numeric` tag）；另发现 `:179/:190` 同类盲区（§2.2） |
| float64：`transaction.go:16,28,36`、`transaction_record.go:14`、`used_car_order.go:138,180`、`auto_finance.go:52,60-70` | **成立，但 `auto_finance` 需二次修正** | auto_finance 的 float64 **全部是 DTO/读模型，不落库**；落库列是 `bigint` 分 + BPS（`align_gorm_only_schema.up.sql:124-136`）。真正 float64 **落库** 只有 `transactions`、`transaction_records`（裸 numeric）与 `wys_used_car_order`（numeric(14,2)） |
| **mall 不是分口径，别照旧说法写** | **已按此执行** | §1 结论 1、§4.1 明确 mall→payment 由 mall 侧换算 |
| 最有价值产出 = 跨界换算点清单（含谁做 Round） | **交付** | §4 六小节 + §4.6 汇总表；含两条负结论（membership↔payment 0 处、points↔元无汇率） |
| DB 列类型查 `migrations/*.sql`，别只信 entity tag | **已按此执行，并发现 4 类分歧** | §3 分歧表；含「可空性 vs 非指针类型」这一 Brief 未预见的一类 |
| 严格只读、零代码改动 | **达成** | 见下「验证命令」`git diff --stat` |

---

## 验证命令执行结果

| Brief 要求 | 命令 | 结果 |
|------------|------|------|
| 零代码改动（只允许 docs/ 变化） | `git diff --stat` | **docs/ 三个文件** + `plans/slices/` Context Card 一处（白名单第 4 条明文允许，见下「白名单自证」）；`model/**`、`app/**`、`migrations/**`、`configs/**`、`*_test.go` **全部 0 改动** ✓ |
| 清单条数一致 | `grep -rnE 'float64|string.*numeric' model/entity/ \| wc -l` | **38**（清单 §2.1 恰 38 行一一对应）**但 pattern 漏 6 条 → 该项不勾**，详见验收第 1 条 |
| 排除法完备 | `grep -rnE 'float64\|string.*numeric\|int64' model/entity/ \| wc -l` | **173 = 38 + 135** ✓；int64 四类残差 **0** ✓ |
| 只读 psql 核查 | `./scripts/load-env.sh psql ...` | **本 worker 未执行** —— PG 5432 closed、`docker ps` 无输出；本 Slice 全程未连接任何业务库 `my_go_study`，**零 DDL/DML** ✓。（§5.1 的三条计数由 **coordinator** 后续只读补跑，见开头 note（a）；§5.2/§5.3 仍未跑） |
| 未碰 git 状态 | `git status` | 无 add/commit/push；未改 git config ✓ |
| （附加自证）编译未受影响 | `go -C app build ./cmd/api`、`go -C app test ./internal/transaction/` | **未能运行** —— 沙箱无网络：`proxy.golang.org` 全部 `dial tcp ... i/o timeout`，Go 因「`updates to go.mod needed`」拒绝构建。`git diff --stat HEAD -- go.work '*/go.mod' '*/go.sum'` 为**空**，`model/` `app/` `core/` `common/` `configs/` 全零改动 → **编译状态由构造保证不受本片影响**（本轮不改代码，本就不要求构建绿）。**未跑 `go mod tidy`**（AGENTS.md 全局约束 9 + `**/go.mod` 属黑名单） |

### 白名单自证

合同要求「`git diff --stat` 必须只有 docs/ 变化」，而 Brief 文件白名单第 4 条明文允许
`plans/slices/harden-h6-money-unit-audit.md`（仅 Context Card）。两者有轻微张力，处理方式：

- **已填** Context Card（`.harness/rules/no-silent-accept.md` 要求实现/审计类必须有 Context Card，缺则不得 Accept）；
- 只改 `## Context Card` 小节下的 6 个空条目，**未动** Brief 的 ONLY / 不做 / 验收 / 白名单 / 黑名单任何一行；
- 此处留痕，供人判断是否回退该处改动。**代码目录零改动**这一硬要求不受影响。

---

## 残留（Deferred / Partial 明细）

| # | 残留 | 状态 | 归口 |
|---|------|------|------|
| 1 | §5.1 float 侧精度事故 SQL（`transactions` / `transaction_records` / `wys_used_car_order`） | **已取证（本机数据 0 行）** —— 执行者 coordinator，非本 worker；见上文 note（a）的命令与结果 | h6b 验收第 3 条「必须 0 行才允许回填」这一硬门槛在**本机 `my_go_study`** 上已满足；**其它环境/生产库仍需各自跑，本结论仅覆盖本机** |
| 1b | §5.2 numeric 侧 + §5.3 跨界/账本求和 SQL 仍**未执行** | **Partial**（需人/CI 在 scratch 或只读副本跑） | h6b 涉及 mall/wallet/payment 的回填前前置门槛 |
| 2 | mandated grep pattern 需修订（补 6 条盲区）才能当完备性门禁 | **Partial** | 下一 Slice 或直接采纳 audit §2.2 的补充 pattern |
| 3 | Supabase `provider=supabase` 路径金额口径不迁移 | **Degraded**（已登记 inventory） | §7；解除条件：云端补列或禁止该 provider 组合 |
| 4 | ES `amount` 仍 `double`，改单位需 reindex | **Deferred**（h6b「不做」第 3 条明令不动） | 与 mapping 变更同批的独立 Slice |
| 5 | `mall/service.go:106 normalizeMoney` 静默四舍五入 | **Deferred** | 新 Slice（Brief「不做」禁止「顺手加校验」） |
| 6 | `payment/service.go:285` 资金出站过 float + `%.2f` 进位 | **Deferred** | 支付加固 Slice |
| 7 | `purchase_calculator` 入参 `math.Round` 静默进位 + 月供 float 指数 + 末期舍入进利息 | **Deferred** | auto_finance 专项 |
| 8 | `auto_finance.subsidy_value` 一列两义（bps 或 fen） | **Deferred** | auto_finance 专项 |
| 9 | `wys_used_car_order.amount` / `wys_membership_order.amount_fen` 可空 vs Go 非指针 | **Deferred** | 需先定 NULL 语义（业务决策） |
| 10 | `wys_mall_refund` 僵尸表（有列约束无写入方） | **Deferred**（`Removed` 候选） | 清理 Slice |
| 11 | Mixed 支付模式 = 现金 + 积分**双重计费**而非抵扣（证据链见 audit §4.3 五步 + 资损判断） | **Deferred**（语义歧义，非已证实资损） | **需人裁决产品口径**；裁决前 h6b 不得动 `service.go:380-382 / 695-713` |
| 12 | mall 金额「双台账」（`wys_mall_payment` 元 vs `wys_payment_records` 分） | **Deferred** | h6b 之后单独立 Slice |
| 13 | Flutter 契约仅静态取证，未跑 `dart analyze` | **Partial** | 跨仓 Brief |
| 14 | `transaction_records` 遗留表 | **Deferred**（建议不迁移 → `Removed`/`Deferred`） | §6.3 |

## 残留 11 补充：Mixed「现金 + 积分双重计费」取证与资损判断（待人裁决）

完整五步证据链在 `docs/money-unit-audit.md` §4.3，此处只给结论骨架：

| 环节 | file:line | 事实 |
|------|-----------|------|
| Mixed 如何被判定 | `model/entity/mall.go:88-100`（`:97` 返回 `MallPayModeMixed`；常量 `:44`=3） | `price` 非零**且** `price_points>0` 即为 Mixed |
| 订单两笔金额来源 | `app/internal/mall/service.go:342-349` → `:380-382` | `Amount` = ΣCNY 全额、`TotalPoints` = Σ积分全额，**无任何抵扣项** |
| 结算两笔并列扣减 | `service.go:672-673`（两个 needs 同时为真）、`:695-701`（扣积分）、`:703-713`（扣钱包） | 同一事务内**各按全额扣**；`:678` 的「强制积分渠道」分支 Mixed 进不去 |
| 渠道支付路径同样双收 | `service.go:585` + `:608-616`；现金核对在 `app/internal/payment/repo_postgres.go:102` | 渠道收了全额现金，积分**照扣**；`:582-584` 注释自述「混合单（积分 + 人民币）的积分在这一步才扣」 |
| 设计意图旁证 | `service.go:551`、`:675-676` `ErrNeedCNYChannel` 文案「混合或人民币订单需使用人民币支付渠道」 | Mixed 被要求**必须**走现金渠道，说明双收是有意的，不是「积分可替代现金」 |
| 可达性 | `handler.go:76,92` → `service.go:118,127`（只校验 `>=0`，不禁止同时非零）；`repo_seed.go:84` `{"SEED-PTS-MIX","积分加价礼包","9.90",50}`；`service_test.go:36` | 运营可造、种子已造、单测当合法态 —— **不是理论路径** |

**是否构成资损：本 Slice 判定为「不确定 —— 取决于产品口径」，需人裁决。**
- 口径 A「现金+积分组合购买」→ 代码自洽（`积分加价礼包` 这个标题就是这个意思）→ **不构成资损**。
- 口径 B「积分抵扣部分现金」→ 用户为同一件商品付两份；退款侧 `:769-773` 两笔都原额退回，
  所以亏的不是平台而是**用户被多收**（平台侧多收），属过度收费 / 客诉与合规风险，**不是账面资损**。
- **未取证的**：线上/本机 `wys_mall_sku` 里实际有多少 Mixed SKU、有多少成交（本 worker 未连库，coordinator 只跑了 §5.1 三条计数）
  → 影响面大小**无数据**，因此这条不能升级为「已证实事故」，按 `.harness/rules/status-enums.md` 记 **Deferred**。

## 下一 Slice 建议

1. **跑 §5.2 / §5.3**（scratch 或只读副本）—— §5.1 float 侧已由 coordinator 补跑取证（本机 0 行，见开头 note（a）），
   但 numeric 侧与跨口径求和不变式仍无结论；否则涉及 mall/wallet/payment 的回填门槛无法判定；
2. 再执行 `plans/slices/harden-h6b-transaction-money-unit.md`（其验收第 1-3 条可直接引用本文 §6.2 与 §5.1）；
3. 然后开「`normalizeMoney` 舍入显式化」小 Slice（改动面最小、危害明确，见残留 5）；
4. `auto_finance` 与 `subsidy_value` 单独立 Slice（残留 7/8）。

## 已知坑（写给下一位）

- **别信 `grep ... | wc -l` 的「完备」**：本 Slice 的 mandated pattern 亲自漏掉 6 条，其中一条 Brief 自己引用过（`mall.go:346`）。
- **别信 entity 的 gorm tag 当列类型**：`align_column_drift` 这个迁移的存在就是为此，其头注 `:1-10` 是最佳二手证据。
- **`transactions.amount` 不是 float 列，是裸 `numeric`** —— 所以「DB 侧存了脏值」是完全可能的，别按「PG 会帮我截到 2 位」假设开工。
- **别给积分补汇率**：§4.3 的负结论是查证过的（全仓无汇率），Mixed 模式的双计费是产品问题不是精度问题。
- **别顺手改 `membership`↔`payment`**：它俩零换算，任何「统一」改动都是在制造风险。
- 本 worktree 的 `../../my_ai_project` 不存在；Flutter 实际在 `/Users/qpy/Desktop/github/my_ai_project`。
