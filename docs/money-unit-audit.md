# 金额口径审计（Money Unit Audit）

> Slice：[`plans/slices/harden-h6-money-unit-audit.md`](../plans/slices/harden-h6-money-unit-audit.md)
> 状态：**Partial**（本文件只出方案与影响面；落地在 [`harden-h6b-transaction-money-unit`](../plans/slices/harden-h6b-transaction-money-unit.md)）
> 取证时间：2026-11-06，基于分支 `maoqian/harden-h6` HEAD `e288553`
> 本文所有 `file:line` 均为该 HEAD 实测，不是记忆；分类依据见 §2。

---

## 0. 结论摘要（先读这 8 条）

1. **口径确为三套**，原 Brief 里「payment/wallet/mall 是整数分」对 mall 是错的：mall 走
   decimal-as-string（`string` + `numeric(10,2)`），内部用 `big.Rat` 算，是**合理做法**，只是与 payment 的分口径不一致。
2. **第二条前提也要修正**：`auto_finance.go:37-45,52,60-70` 的 `float64` **不是存储口径**，是对外 DTO 的「元」读模型；
   落库列全是 `bigint` 的分或 BPS（`migrations/20260923167500_align_gorm_only_schema.up.sql:124-136`）。
   真正的「float64 落库」只有 4 处：`transactions.amount`、`transaction_records.amount`、`wys_used_car_order.amount`（+ 它们的 DTO）。
3. **最高价值发现不是 float，而是 entity 与 DB 列类型分歧**：`transactions.amount` 是**裸 `numeric`（无精度无标度）**，
   Go 用 `float64` 读写 —— DB 侧能存下 `0.30000000000000004` 这类脏值，Go 侧读回即失真。见 §3。
4. **跨界换算点只有 1 类是真的「跨」**：`mall`（decimal 元）↔ `payment`/`wallet`（整数分），
   共 3 处，全部由 **mall 侧**用 `entity.ParseYuanToFen` 完成，且该函数 **>2 位小数直接报错、不四舍五入** ——
   这条边界是**安全的**。见 §4.1/§4.3。
5. **`membership` ↔ `payment` 没有换算点**（两侧同为 `int64` 分，`PriceFen` 直通 `AmountFen`）；
   **`points` 与元之间不存在任何汇率**（积分是 SKU 上的平行价，不是抵扣率）。两条**负结论**见 §4.2/§4.3。
6. **真正的静默丢精度点只有 1 处**：`app/internal/mall/service.go:106` 的 `normalizeMoney`，
   后台改价时把 >2 位小数**四舍五入吃掉**（`big.Rat.FloatString(2)`），无告警无报错。见 §4.6 与 §5。
7. **`auto_finance` 的 `subsidy_value` 是一列两义**（`bps` 或 `fen`，取决于 `subsidy_type`）——
   单位写在注释里而不是类型里，是最容易被后人误算的字段。见 §4.5。
8. **核查 SQL 只跑了一半，且执行者不是本 worker**：沙箱内 PostgreSQL 未运行（`127.0.0.1:5432` 关闭），
   worker 一条 SQL 都没跑；Docker 起来后由 **coordinator** 只读补跑 §5.1 三条计数 →
   `transactions` 0 / `transaction_records` 0 / `wys_used_car_order`（NULL）0，**仅覆盖本机 `my_go_study`**。
   §5.2 / §5.3 **仍是待人执行的产物**，故整体状态只能是 **Partial**。

---

## 1. 三套口径的定义与边界

| 口径 | Go 类型 | DB 列类型 | 代表域 | 是否可无损互转 |
|------|---------|-----------|--------|----------------|
| **A 整数分** | `int64`（`*Fen`） | `bigint` | payment / membership / wallet / auto_finance(落库) | 与 B 在「≤2 位小数」前提下可无损互转 |
| **B decimal-as-string** | `string` | `numeric(10,2)` | mall | 同上 |
| **C float64** | `float64` | **裸 `numeric`** / `numeric(14,2)` / `double precision`(ES) | transaction / used_car / analytics | **不可无损**：二进制浮点无「分」概念 |

**口径 A 与 B 的区别只在于「值在进程内怎么表达」**，两者在数学上都精确到分。
**C 是不同的东西**：`float64` 无法精确表示 `0.1`/`0.2`，且本仓 C 侧**没有任何一处做 `Round(_,2)`**
（`grep -rnE 'Round|Trunc|decimal|ParseFloat' app/internal/transaction/` → 0 命中）。

→ 结论：**A/B 需要「统一表达」，C 需要「统一精度」**。这两个问题的解法不同，不能一刀切合并成一个 Slice。

---

## 2. 字段清单（排除法证明，不接受「只列想到的」）

### 2.1  mandated grep 的 38 条 —— 逐条归类

验证命令 `grep -rnE 'float64|string.*numeric' model/entity/ | wc -l` = **38**，下表 **38 行**，一一对应。
判定列：**MONEY**（资金/金额语义，纳入迁移范围）/ **NOT-MONEY**（比例、尺寸、统计，显式豁免并给理由）。

| # | file:line | 字段 | 类型 | 判定 | 理由 |
|---|-----------|------|------|------|------|
| 1 | `model/entity/mall.go:162` | `WysMallProduct.CoverAspect` | `float64` `numeric(4,2)` | NOT-MONEY | 封面高/宽比（`migrations/20260921180000_mall_schema.up.sql:40` 注释「封面高/宽，瀑布流用」），有 `CHECK 0.50..2.00` |
| 2 | `model/entity/mall.go:177` | `MallShelfItem.CoverAspect` | `float64` | NOT-MONEY | 同上的列表读模型 |
| 3 | `model/entity/mall.go:210` | `WysMallSKU.Price` | `string` `numeric(10,2)` | **MONEY (B)** | SKU 人民币价，`CHECK (price >= 0)` |
| 4 | `model/entity/mall.go:255` | `WysMallOrder.Amount` | `string` `numeric(10,2)` | **MONEY (B)** | 订单合计，是 §4.1 换算的输入 |
| 5 | `model/entity/mall.go:278` | `WysMallOrderItem.Price` | `string` `numeric(10,2)` | **MONEY (B)** | 成交价快照 |
| 6 | `model/entity/mall.go:281` | `WysMallOrderItem.LineAmount` | `string` `numeric(10,2)` | **MONEY (B)** | 行小计 = 单价 × 数量，由 `big.Rat` 算出 |
| 7 | `model/entity/mall.go:308` | `WysMallPayment.Amount` | `string` `numeric(10,2)` | **MONEY (B)** | 商城支付单（本地模拟渠道，与口径 A 的 `wys_payment_records` 并存 → 见 §4.1「双台账」） |
| 8 | `model/entity/auto_finance.go:37` | `FinanceProductDTO.AnnualRatePercent` | `float64` | NOT-MONEY | 年利率百分数，源列 `annual_rate_bps bigint` |
| 9 | `model/entity/auto_finance.go:39` | `MinDownPaymentPercent` | `float64` | NOT-MONEY | 首付比例，源列 `min_down_payment_bps bigint` |
| 10 | `model/entity/auto_finance.go:40` | `OneTimeFee` | `float64` | **MONEY (C，瞬时)** | 一次性手续费「元」读模型，源列 `one_time_fee_fen bigint`；**不落库** |
| 11 | `model/entity/auto_finance.go:42` | `SubsidyRateCutPercent` | `float64` | NOT-MONEY | 贴息利率（百分点） |
| 12 | `model/entity/auto_finance.go:43` | `SubsidyAmountCut` | `float64` | **MONEY (C，瞬时)** | 贴息金额「元」，源 `subsidy_value`（分）—— 见 §4.5 一列两义 |
| 13 | `model/entity/auto_finance.go:44` | `CompulsoryInsurance` | `float64` | **MONEY (C，瞬时)** | 交强险「元」，源列 `compulsory_insurance_fen bigint` |
| 14 | `model/entity/auto_finance.go:45` | `CommercialRatePercent` | `float64` | NOT-MONEY | 商业险费率，源列 `commercial_rate_bps bigint` |
| 15 | `model/entity/auto_finance.go:52` | `PurchaseQuoteLine.Amount` | `float64` | **MONEY (C，瞬时)** | 报价分项「元」，由 `fenToYuan` 派生 |
| 16 | `model/entity/auto_finance.go:60` | `PurchaseQuoteDTO.BarePrice` | `float64` | **MONEY (C，瞬时)** | 同上，报价读模型 |
| 17 | `model/entity/auto_finance.go:61` | `TaxablePrice` | `float64` | **MONEY (C，瞬时)** | 同上 |
| 18 | `model/entity/auto_finance.go:62` | `DownPayment` | `float64` | **MONEY (C，瞬时)** | 同上 |
| 19 | `model/entity/auto_finance.go:63` | `LoanAmount` | `float64` | **MONEY (C，瞬时)** | 同上 |
| 20 | `model/entity/auto_finance.go:65` | `EffectiveAnnualRate` | `float64` | NOT-MONEY | 实际年利率百分数 |
| 21 | `model/entity/auto_finance.go:66` | `MonthlyPayment` | `float64` | **MONEY (C，瞬时)** | 月供「元」，由等额本息 float 公式算出 → §4.5 有真精度风险 |
| 22 | `model/entity/auto_finance.go:67` | `TotalInterest` | `float64` | **MONEY (C，瞬时)** | 利息合计 |
| 23 | `model/entity/auto_finance.go:68` | `TotalRepayment` | `float64` | **MONEY (C，瞬时)** | 还款总额 |
| 24 | `model/entity/auto_finance.go:69` | `InitialPayment` | `float64` | **MONEY (C，瞬时)** | 首期支出 |
| 25 | `model/entity/auto_finance.go:70` | `TotalDue` | `float64` | **MONEY (C，瞬时)** | 应付总额 |
| 26 | `model/entity/transaction_record.go:14` | `TransactionRecord.Amount` | `float64` | **MONEY (C，落库)** | 遗留 `transaction_records`；实体自陈「已弃用」（`:21-24`） |
| 27 | `model/entity/transaction.go:16` | `Transaction.Amount` | `float64` | **MONEY (C，落库)** | 主力收支表 `transactions`，**h6b 目标** |
| 28 | `model/entity/transaction.go:28` | `CreateTransactionInput.Amount` | `float64` | **MONEY (C，入参)** | 客户端直收，无精度/上限校验 |
| 29 | `model/entity/transaction.go:36` | `UpdateTransactionInput.Amount` | `*float64` | **MONEY (C，入参)** | 同上（指针，可空） |
| 30 | `model/entity/short_video.go:26` | `AspectRatio` | `float64` | NOT-MONEY | 视频宽高比；曾被 `align_column_drift:148` 从 `double precision` 改成 `numeric` |
| 31 | `model/entity/used_car_order.go:138` | `WysUsedCarOrder.Amount` | `float64` | **MONEY (C，落库)** | 二手车成交价，列 `numeric(14,2)` 且**被放宽为可 NULL** |
| 32 | `model/entity/used_car_order.go:180` | `UsedCarOrderDTO.Amount` | `float64` | **MONEY (C，读模型)** | 同值转发（`:201`）。同行另有 `AmountLabel string`（`:172`）是「回收/寄售」**文案**，NOT-MONEY，不计入本条 |
| 33 | `model/entity/analytics_record.go:25` | `MetricRevenue` | `float64` | NOT-MONEY（**但 money-adjacent**） | 投放营收**统计指标**，非账户资金：不可花、不可退、无借贷平衡约束。列已被放宽为裸 `numeric`（`align_column_drift:125`） |
| 34 | `model/entity/analytics_record.go:26` | `MetricCost` | `float64` | NOT-MONEY（**但 money-adjacent**） | 同上，投放成本指标 |
| 35 | `model/entity/analytics_record.go:27` | `MetricROI` | `float64` | NOT-MONEY | 投入产出**比值** |
| 36 | `model/entity/analytics_record.go:28` | `MetricBounceRate` | `float64` | NOT-MONEY | 跳出率（百分比） |
| 37 | `model/entity/analytics_record.go:30` | `ScoreQuality` | `float64` | NOT-MONEY | 质量评分 |
| 38 | `model/entity/analytics_record.go:31` | `ScoreRisk` | `float64` | NOT-MONEY | 风险评分 |

小计：38 条中 **MONEY 24 条**、**NOT-MONEY 14 条**（24 + 14 = 38，与 grep 条数吻合）。
MONEY 侧细分：**B 口径 5 条**（#3/#4/#5/#6/#7）、**C 落库 3 条**（#26 `transaction_records` / #27 `transactions` / #31 `wys_used_car_order`）、
**C 入参 2 条**（#28/#29）、**C 读模型 1 条**（#32）、**C 瞬时（DTO 派生，不落库）13 条**（#10/#12/#13/#15/#16/#17/#18/#19/#21/#22/#23/#24/#25，全在 auto_finance）。
#33/#34 是本表最容易被误判的两条：它们语义是钱、性质是统计，**不纳入资金迁移**，理由已写明（无账户、无借贷平衡、不参与结算）。

### 2.2 mandated grep 的**盲区**：6 条金额字符串被漏掉

`float64|string.*numeric` 这个 pattern 要求同一行出现 `numeric`，因此**所有没有 gorm `numeric` tag 的
decimal-as-string 读模型都扫不出来**。补齐如下（这 6 条**必须**纳入清单，否则「完备」是假的）：

| file:line | 字段 | 类型 | 判定 | 说明 |
|-----------|------|------|------|------|
| `model/entity/mall.go:179` | `MallShelfItem.Price` | `string` | **MONEY (B)** | 货架列表读模型 |
| `model/entity/mall.go:190` | `MallSKUOffer.Price` | `string` | **MONEY (B)** | 买家规格报价读模型 |
| `model/entity/mall.go:346` | `MallOrderListItem.Amount` | `string` | **MONEY (B)** | 订单列表行（Brief 提到的 :346 就是这条，mandated grep 抓不到） |
| `model/entity/membership.go:64` | `MembershipPlan.Price` | `string` | **MONEY (B)** | 由 `FormatFenToYuan(PriceFen)` 派生（`:90`）—— **A→B 换算点** |
| `model/entity/cash_wallet.go:65` | `WalletSummary.Balance` | `string` | **MONEY (B)** | 由 `FormatFenToYuan` 派生 —— **A→B 换算点**；与 `BalanceFen`（`:66`）同响应体**双口径并存** |
| `model/entity/cash_wallet.go:78` | `WalletRechargeResult.Balance` | `string` | **MONEY (B)** | 同上（`:79`/`:80` 又带 fen），双口径并存 |

→ **验证命令要修订**：真实金额字段数 = 38 + 6 = **44**（+ §2.3 的分口径 int64）。
现命令 `grep ... | wc -l` 与「清单条数一致」这条验收项，**在修订 pattern 前不可能成立**。记为 Partial 的原因之一。

### 2.3 口径 A（整数分 / 积分）的 `int64` 清单

`grep -rnE 'int64' model/entity/` = **135** 行；与 §2.1 的 38 行相加 = 173，正是
`grep -rnE 'float64|string.*numeric|int64' model/entity/ | wc -l` 的 173。三分类计数（有重叠，故用残差法）：

| 分类 | 计数方式 | 命中 |
|------|----------|------|
| ID 主外键 | 字段名以 `ID`/`id` 结尾 | 56 |
| 计数/统计 | `Count`/`Total`/`Num`/`Heat`/`Streak`/`PV`/`UV`/`Click`/`Convert`/`Stats`/审核四格等 | 28 |
| 时间 | `*At`/`Time`/`Window`/`Second` | 9 |
| 金额/积分 | `fen`/`price`/`amount`/`balance`/`delta`/`points`/`reward` | 45 |
| **残差（不属于以上任何一类）** | 四条 pattern 全部取反 | **0** |

残差为 0 → 135 行 `int64` 无遗漏。其中 45 行「金额/积分」里真正是资金口径 A 的字段：

| file:line | 字段 | DB 列类型（查 migrations） |
|-----------|------|-----------------------------|
| `model/entity/payment.go:53` | `WysPaymentRecord.AmountFen` | `bigint NOT NULL` + `CHECK (amount_fen > 0)`（`20261104160000_payment_ledger.up.sql:8,21`） |
| `model/entity/payment.go:82` | `LocalAmountFen` | `bigint NOT NULL DEFAULT 0`（同文件 `:47`） |
| `model/entity/payment.go:84` | `ChannelAmountFen` | `bigint NOT NULL DEFAULT 0`（同文件 `:49`） |
| `model/entity/membership.go:48` | `WysMembershipOrder.AmountFen` | `bigint` **NOT NULL 已被放宽**（建表 `20260923140000_membership_subscribe.up.sql:20` 带 `CHECK (amount_fen > 0)`；`20260924110000_align_column_drift.up.sql:66` `DROP NOT NULL`）→ **Go 是 `int64`，DB 可为 NULL** |
| `model/entity/membership.go:63` | `MembershipPlan.PriceFen` | 无表（代码内目录 `MembershipCatalog()`） |
| `model/entity/cash_wallet.go:30` | `WysCashWallet.BalanceFen` | `bigint NOT NULL DEFAULT 0 CHECK (balance_fen >= 0)`（`20260923095000_cash_wallet.up.sql:4`） |
| `model/entity/cash_wallet.go:40,41` | `WysCashLedger.DeltaFen` / `BalanceFen` | `bigint NOT NULL`（同文件 `:11,12`，**无 CHECK** → 流水列可负） |
| `model/entity/cash_wallet.go:66,79,80` | 读模型里的 `BalanceFen`/`DeltaFen` | 不落库 |
| `model/entity/cash_wallet.go:23,24` | `WalletRechargeMinFen=1` / `MaxFen=5000000` | 常量（0.01 ~ 50000.00 元） |
| `model/entity/auto_finance.go:21` | `OneTimeFeeFen` | `bigint DEFAULT 0 NOT NULL`（`align_gorm_only:129`） |
| `model/entity/auto_finance.go:23` | `SubsidyValue` | `bigint DEFAULT 0 NOT NULL`（`:131`）—— **单位随 `subsidy_type` 变：bps 或 fen** |
| `model/entity/auto_finance.go:24` | `CompulsoryInsuranceFen` | `bigint DEFAULT 0 NOT NULL`（`:132`） |
| `model/entity/mall.go:180,191,211,256,279,282` | `PricePoints`/`TotalPoints`/`LinePoints` | `bigint DEFAULT 0 NOT NULL`（`align_column_drift:14-17`；`wys_mall_sku.price_points` 原由内嵌 SQL 建） |
| `model/entity/points.go:33,43,44,57,68,92,95,103,111,117,118` | 积分 `Balance`/`Delta`/`Points`/`Reward` | `bigint`：wallet `balance DEFAULT 0 NOT NULL CHECK (balance >= 0)`（`align_gorm_only:142,144`）、ledger `delta`/`balance` **无 CHECK**（`:150,151`） |

**积分不是钱**：`points` 是独立整数记账单位，全仓**查不到任何积分↔元的汇率常量或换算函数**（见 §4.3 负结论）。
本 Slice 只登记其存在，不纳入金额迁移。

---

## 3. entity 声明 vs DB 实际列类型（分歧表）

Brief 要求「查 `migrations/*.sql`，不要只信 entity tag」。本仓**已有前例**，且前例本身就是 `align_column_drift`
这个迁移存在的理由（其头注 `:1-10` 自陈：「差异全是『迁移声明比实际更严/更窄』…34 列类型偏窄（integer→bigint、
**double precision→numeric**、text/character→varchar）」）。

| 表.列 | entity 声明 | 迁移声明 | **最终有效列型** | 分歧影响 |
|--------|-------------|----------|-------------------|----------|
| `transactions.amount` | `float64` `gorm:"not null"`（`transaction.go:16`） | 裸 `numeric NOT NULL`（`align_gorm_only:40`） | **`numeric`（精度无界）** | ⚠ **最严重**：DB 接受 >2 位小数脏值并原样存；Go 读回 `float64` 即失真；`gorm not null` 与迁移的 `NOT NULL` 巧合一致，但**类型从未一致** |
| `transaction_records.amount` | `float64`（`transaction_record.go:14`） | 裸 `numeric NOT NULL`（`:53`） | **`numeric`（无界）** | 同上；且该表 `user_id bigint` 与 UUID 口径混用、表名无 `wys_` 前缀，实体注释自陈已弃用（`:21-24`） |
| `wys_used_car_order.amount` | `float64`（`used_car_order.go:138`） | `numeric(14,2) NOT NULL`（`20260922221000_used_car_order.up.sql:15`）→ `DROP NOT NULL`（`align_column_drift:103`） | **`numeric(14,2)` 可空** | ⚠ Go 侧 `float64` 非指针 → **NULL 与 0 不可区分**；`numeric(14,2)` 上限 9,999,999,999.99 元，远超 float64 的精确整数域（2^53）之上仍可表达，但读回必经二进制浮点 |
| `wys_membership_order.amount_fen` | `int64` 非指针（`membership.go:48`） | `bigint NOT NULL CHECK(>0)`（`membership_subscribe.up.sql:20`）→ `DROP NOT NULL`（`align_column_drift:66`） | **`bigint` 可空，CHECK 仍在** | ⚠ NULL→0，而 `CHECK (amount_fen > 0)` 又不允许 0 落库 → **可空性与非正数约束互相矛盾**，脏数据只能以 NULL 形态存在 |
| `wys_cash_ledger.delta_fen` | `int64`（`cash_wallet.go:40`） | `bigint NOT NULL`，**无 CHECK**（`cash_wallet.up.sql:11`） | `bigint` | 一致；符号即收支方向，无约束（设计如此） |
| `wys_cash_wallet.balance_fen` | `int64`（`:30`） | `bigint NOT NULL DEFAULT 0 CHECK (>=0)`（`:4`） | `bigint` | 一致 |
| `wys_mall_sku.price` | `string` `numeric(10,2)`（`mall.go:210`） | `numeric(10,2) NOT NULL` + `CHECK (price >= 0)`（`mall_schema.up.sql:51,60`） | 一致 | tag 与列吻合 |
| `wys_mall_order.amount` / `_item.price` / `_item.line_amount` / `wys_mall_payment.amount` / `wys_mall_refund.amount` | `string` `numeric(10,2)`（前 4 条有 tag） | 全部 `numeric(10,2) NOT NULL`（`mall_schema.up.sql:101,132,134,161,191`）+ 各自 `CHECK`（`:110,170,199`） | 一致 | **`line_amount` 无 CHECK**（只在 order/payment/refund 三级有） |
| `wys_payment_records.amount_fen` | `int64`（`payment.go:53`） | `bigint NOT NULL` + `chk_..._amount CHECK (>0)`（`payment_ledger.up.sql:8,21`） | 一致 | 一致（用命名约束而非列内 CHECK） |
| `analytics_records.metric_revenue/metric_cost` | `float64`（`analytics_record.go:25,26`） | `DOUBLE PRECISION`（建表 `:20,21`）→ `TYPE numeric`（`align_column_drift:125,124`） | **`numeric`** | **前例本体**：迁移与实体双向都错过；`metric_roi/bounce_rate/score_*` 同（`:123,126,127,128`） |
| `wys_mall_product.cover_aspect` | `float64` `numeric(4,2)` | `numeric(4,2) NOT NULL DEFAULT 1.00` + `CHECK 0.50..2.00`（`mall_schema.up.sql:26,33`） | 一致 | 非钱 |
| `wys_short_videos.aspect_ratio` | `float64`（`short_video.go:26`） | `double precision`（`20260922170000_short_video.up.sql:9`）→ `TYPE numeric`（`align_column_drift:148`） | **`numeric`** | 非钱，但证明「同一漂移模式」不限于金额列 |

**没有找到迁移的（仅 AutoMulate 或内嵌 SQL 建表）**：
- `wys_points_*` / `wys_check_ins` / `wys_task_claims` 早期**只由内嵌 SQL** `repository/postgres/sql/points_schema.sql` 建立，
  直到 `align_gorm_only_schema.up.sql:139-140` 注释才被补进迁移链（「以下 4 张此前只由内嵌 SQL 建立」）。
- 结论：**「列类型以 entity tag 为准」在本仓历史上至少错过三回**，故 §5 的核查必须打到真库。

---

## 4. 跨界换算点清单（本 Slice 首要产出）

约定：`谁做 Round` = 单位转换在哪个域、哪个函数发生；`丢精度方向` = 精度在往哪一侧流失。

### 4.1 `mall`（B：decimal 元）→ `payment`（A：整数分）— **1 处，安全**

| file:line | 表达式 | 谁做 Round | 丢精度方向 |
|-----------|--------|-------------|-------------|
| `app/internal/mall/service.go:553` | `fen, err := entity.ParseYuanToFen(order.Amount)`，随后 `payment.PrepayInput{AmountFen: fen}`（`:563`） | **mall 侧**（调用方），经 `entity.ParseYuanToFen` | **无损**：`big.Rat` × 100 后 `!r.IsInt()` 直接 `return 0, err`（`model/entity/cash_money.go:17-20`）→ **>2 位小数报错，绝不四舍五入** |
| `app/internal/mall/service.go:554` | `if err != nil \|\| fen <= 0 { return nil, ErrInvalidPrice }` | mall | 负/零被拒；与 `wys_payment_records` 的 `CHECK (amount_fen > 0)` 同向 |

补充：`mall` 自己还有一张**本地模拟支付单** `wys_mall_payment.amount numeric(10,2)`（`mall.go:308`）与真实渠道流水
`wys_payment_records.amount_fen bigint`（`payment.go:53`）**并存**。同一笔商城交易因此有两个金额真相源（元 vs 分）。
本 Slice 判定：**属「双台账」而非「换算错误」**，收敛口径需在 h6b 之后单独立 Slice（记 Deferred）。

### 4.2 `membership`（A）↔ `payment`（A）— **0 处换算（负结论）**

两侧同为 `int64` 分，`PriceFen` **直通** `AmountFen`，全程无乘除：

- `app/internal/membership/service.go:183`、`:201`、`:214`、`:253`、`:386`、`:444` — `AmountFen: plan.PriceFen`
- `app/internal/membership/service.go:285`、`:334` — `AmountFen: order.AmountFen`
- 核销比对：`app/internal/payment/repo_postgres.go:102` `if rec.AmountFen != amountFen { return ErrAmountMismatch }`
- 对账差异：`app/internal/payment/reconcile.go:53`（`LocalAmountFen = rec.AmountFen`）、`:73`（`ChannelAmountFen = fact.AmountFen`）、
  `:75` `if fact.AmountFen != rec.AmountFen` → `PaymentDiffAmountMismatch`

→ **本边界不需要迁移，也不该被顺手改**。唯一残留风险是 §3 里 `amount_fen` 被放宽成 NULL。

**目录价是 A→B 的唯一派生点**（对外展示）：`model/entity/membership.go:90`
`Price: FormatFenToYuan(fen)`，输入是硬编码分常量（`:74-79`：3000 / 15000 / 28000 / 4800 / 24000 / 39800）。
`FormatFenToYuan` 用整数 `fen/100` + `fen%100`（`cash_money.go:30-32`）→ **精确，无 float**。

### 4.3 `wallet`/`points` ↔ `mall` — **2 处换算（均安全）+ 1 条负结论**

| file:line | 方向 | 表达式 | 谁做 Round | 丢精度 |
|-----------|------|--------|-------------|--------|
| `app/internal/mall/service.go:707` | mall(B) → wallet(A) | `fen, err := entity.ParseYuanToFen(order.Amount)` → `u.walletSvc.Debit(ctx, buyerUserID, fen, CashReasonMallPay, ref)`（`:711`） | **mall 侧** | **无损**（同上，>2dp 报错） |
| `app/internal/mall/service.go:756` | mall(B) → wallet(A)（退款） | `fen, err := entity.ParseYuanToFen(row.Amount)` → `u.walletSvc.Credit(...CashReasonMallRefund, fen)`（`:764`） | **mall 侧** | **无损**；退款**复用订单原值**，不重算 → 无「退款比实付多/少一分」问题 |
| `app/internal/wallet/service.go:115` | 客户端(B) → wallet(A) | `fen, err := entity.ParseYuanToFen(in.Amount)`，再夹到 `[WalletRechargeMinFen, MaxFen]`（`:116`） | wallet | **无损**，但有 **上限 50000.00 元** |
| `app/internal/wallet/service.go:36`、`:143` | wallet(A) → 对外(B) | `Balance: entity.FormatFenToYuan(fen)` | wallet | **无损**（整数除法） |

**负结论（重要，避免 h6b 之后有人去「补汇率」）**：
`points` 与元之间**不存在任何换算**。积分是 SKU 上的**平行价格**：
`wys_mall_sku.price numeric(10,2)` 与 `price_points bigint` 两列并存（`mall.go:210,211`），
`MallSKUPaymentMode(priceCNY, pricePoints)` 按「谁非零」派生 CNY / Points / Mixed 三态（`model/entity/mall.go:88-100`）。
结算时两笔**各自全额扣减**、互不折算：`app/internal/mall/service.go:699`（扣积分 `order.TotalPoints`）
与 `:707-711`（扣钱包 `fen`）在**同一事务**内并列执行。

→ 注意由此产生的**语义坑**（不是精度坑）：`service.go:672-679` 里 `needsCNY` 优先，
`Mixed` 模式下若走人民币渠道，渠道参数被强制成 `MallPayPoints`（`:679`）的分支**只在 `needsCNY` 为假时生效**（`:678` `} else if needsPoints {`）；
Mixed 且 `needsCNY` 为真时**积分照扣**（`:699`）。即 Mixed 订单 = 现金 + 积分**双重计费**，而非「积分抵扣部分现金」。
本 Slice 只登记，不改（改属业务语义决策，需人批）。

**证据链（2026-11-06 逐行复核，供人直接核对；Mixed 语义是否算资损需人判定）**：

1. Mixed 的定义：`model/entity/mall.go:88-100` —— `priceCNY` 非零**且** `pricePoints>0` → `MallPayModeMixed`（`:97`，常量 `:44` = 3）。
2. 下单侧两笔金额**各按全额独立累加**，没有任何「积分折让冲减现金」的公式：
   `app/internal/mall/service.go:342-349`（`total` ← `big.Rat` 累加 CNY；`totalPoints += sku.PricePoints * qty`）
   → `:380-382`（`Amount: formatMoney(total)`、`TotalPoints: totalPoints`）。
3. 结算侧两笔**并列全额扣减**，同一事务：`:672-673`（`needsCNY` / `needsPoints` 同时为真）、
   `:695-701`（积分 `Debit order.TotalPoints`）、`:703-713`（钱包 `Debit fen` ← `ParseYuanToFen(order.Amount)`）。
   `:678` 的「渠道强制改成积分」分支只在 `needsCNY` 为假时进入，Mixed 进不去 → **积分不会替代现金**。
4. 走第三方渠道时同样照扣积分，且现金由渠道收：`SettleChannelPaid`（`service.go:585`）的事务内
   `:608-616`（`locked.TotalPoints > 0` → `pointsSvc.Debit`）；现金已在 payment 侧按 `amount_fen`
   与渠道报文核对（`app/internal/payment/repo_postgres.go:102` `if rec.AmountFen != amountFen`）后才走到这一步。
   `:582-584` 的注释「混合单（积分 + 人民币）的积分在这一步才扣」是**作者自述：Mixed 就是要同时收现金与积分**。
   `:551`/`:675-676` `ErrNeedCNYChannel` 的文案「混合或人民币订单需使用人民币支付渠道」也说明**设计意图就是 Mixed 要用现金渠道**，即积分+现金双收，不是「或」。
5. 可达性（不是理论路径）：运营建 SKU 时 `price` 与 `price_points` **都是入参**
   （`app/internal/mall/handler.go:76,92` → `service.go:118,127` 只校验 `>= 0`，**不校验不能同时非零**）；
   种子数据里就有现成的 Mixed SKU：`app/internal/mall/repo_seed.go:84`
   `{"SEED-PTS-MIX", "积分加价礼包", "9.90", 50}` —— 标题写「加价」，与实现一致；单测也把 Mixed 当作合法态
   （`app/internal/mall/service_test.go:36`）。

**是否构成资损 —— 本 Slice 判定：取决于产品口径，现状不自洽，需人裁决。**
- **若口径是「现金 + 积分组合购买」= 设计意图**：`:380-382` 与 `:695-713` 自洽（两笔都应收），
  `repo_seed.go:84`「积分加价礼包」正是这个语义 → **不构成资损**，只是 §4.3 的「无汇率」结论的另一种表述。
- **若口径是「积分抵扣部分现金」**：用户为同一件商品付了两份，用户侧受损（平台侧多收），
  退款 `:769-773` 会把两笔都原额退回，故不是平台亏钱而是**过度收费 / 客诉与合规风险**。
- **现状没有第三种自洽读法**：既不存在抵扣率（§4.3 负结论），也不存在「积分只抵扣一部分」的代码，
  因此**任一文档/口径若写「积分可抵扣现金」而代码仍全额双收，就是真实资损**。
- 我不能替人定这个口径，也不掌握线上 Mixed SKU 的实际成交数据（**未查库，无取证**），
  故登记为 **Deferred**，标为需人裁决；裁决前**不建议** h6b 动这段金额计算（金额与积分口径改动会同时影响 `:380` 与 `:699/:711`）。

### 4.4 渠道边界（微信/支付宝报文 ↔ 口径 A）

| file:line | 渠道字段 | 表达式 | 谁做 Round | 丢精度 |
|-----------|----------|--------|-------------|--------|
| `app/internal/payment/verify.go:60` | 微信 v2 `total_fee` | `strconv.ParseInt(vals["total_fee"], 10, 64)` | payment | **无换算**：微信 v2 报文本身就是分 |
| `app/internal/payment/channel.go:129` | 支付宝 notify `total_amount` | `fen, err := entity.ParseYuanToFen(resp.TotalAmount)` | payment | **无损**（B→A，>2dp 报错）；支付宝报文是元字符串 |
| `app/internal/payment/verify.go:118` | 支付宝 form 回调 `total_amount` | `fen, err := entity.ParseYuanToFen(yuan)` | payment | 同上 |
| `app/internal/payment/service.go:285` | 下单方向：A → 支付宝「元」字符串 | `amountYuan := fmt.Sprintf("%.2f", float64(amountFen)/100.0)` | payment | ⚠ **唯一一处刻意经过 `float64` 的资金出站点**：`int64` 分 → `float64` 元 → `"%.2f"`。对分口径数值范围（本仓 ≤ 1e10 分）在 2^53 内不产生误差，但**模式上是 float 参与资金**，且 `%` 格式化会**四舍五入** |

→ §4.4 是「A 是真相、渠道各自表达」的边界；`service.go:285` 建议在后续 Slice 换成 `FormatFenToYuan`（纯整数），
但**不属本轮 ONLY**，记 Deferred。

### 4.5 `auto_finance`（A 落库 ↔ C 对外）— **风险最高的一组，因为「算」而不是「存」**

落库纯 A（`bigint` 分 + BPS），**运算也全程在 `int64` 分**（这点值得肯定），只在**进/出 DTO 时**过 float：

| file:line | 表达式 | 方向 | 谁做 Round | 丢精度 |
|-----------|--------|------|-------------|--------|
| `app/internal/purchase_calculator/service.go:255` | `yuanToFen(y float64) int64 { return int64(math.Round(y * 100)) }` | 入参 C→A | calculator（`math.Round` 半远离零） | ⚠ 客户端传 `12345.675` → `1234568` 分，**静默改钱**，无校验无告警 |
| `app/internal/purchase_calculator/service.go:259` | `fenToYuan(f int64) float64 { return float64(f) / 100 }` | 出参 A→C | 无 | float 表达，展示层二次格式化可能再进位 |
| `app/internal/purchase_calculator/service.go:263` | `roundMoney2(v) = math.Round(v*100)/100` | 比例类 | calculator | 非资金 |
| `app/internal/purchase_calculator/service.go:76` | `taxableFen := divRound(bareFen*100, 113)`（÷1.13） | 纯 A | calculator | **整数四舍五入**（`divRound` `:267-275`），口径明确 |
| `app/internal/purchase_calculator/service.go:84` | `taxFen := divRound(taxableFen, 10)` | 纯 A | calculator | 同上 |
| `app/internal/purchase_calculator/service.go:281` | `P := float64(principalFen) / 100` → 等额本息 `math.Pow` → `yuanToFen(m)`（`:288`） | A→C→A | calculator | ⚠ **真精度风险点**：月供经 `float64` 指数运算后一次性 `math.Round` 回分 |
| `app/internal/purchase_calculator/service.go:157` | `totalRepayFen := monthlyFen * int64(in.TermMonths)` | 纯 A | — | **注意后果**：利息由「月供 × 期数 − 本金」倒推（`:158`），**末期 1 分以内的舍入被吸收进「利息合计」**，因此 `TotalRepayment` 与逐期实际现金流可能差 <1 分/单。属**报价**（不落库、不扣款），危害限于展示 |
| `model/entity/auto_finance.go:23` | `SubsidyValue int64 // bps 或 fen` | — | — | ⚠ **一列两义**：`subsidy_type=1` 时是 BPS、`=2` 时是分（`FinanceSubsidyRateCut`/`FinanceSubsidyAmountFen` `:9-11`）。消费点 `service.go:143`（`rateBPS -= int(prod.SubsidyValue)`）与 `:149`（`cut := prod.SubsidyValue` 直接减本金）。**误读即错 100 倍** |
| `app/internal/purchase_calculator/service.go:215,217,224` | `fenToYuan(...)` 组装 `FinanceProductDTO` | A→C | 无 | 展示层 |

### 4.6 汇总：谁做 Round

| 边界 | 换算点数 | 执行方 | 舍入策略 | 判定 |
|------|----------|--------|----------|------|
| mall(B) → payment(A) | 1（`mall/service.go:553`） | **mall** | **拒绝**（`ParseYuanToFen` 非整分即 err） | ✅ 安全 |
| membership(A) → payment(A) | **0** | — | 直通 | ✅ 无需迁移 |
| wallet(A) ↔ mall(B) | 2（`707` 扣 / `756` 退） | **mall** | 拒绝；退款复用原值 | ✅ 安全 |
| wallet(A) 对外(B) / 入参(B) | 3（`wallet/service.go:36,115,143`） | wallet | 拒绝（+ 上下限夹逼） | ✅ 安全 |
| 目录价 membership(A) → 展示(B) | 1（`membership.go:90`） | entity | `FormatFenToYuan` 整数除法 | ✅ 精确 |
| 渠道报文 ↔ payment(A) | 3（`verify.go:60`、`channel.go:129`、`verify.go:118`）+1 出站（`service.go:285`） | payment | 入站拒绝；**出站经 float + `%.2f` 四舍五入** | ⚠ 出站待收 |
| auto_finance 入/出 DTO | 2 核心（`service.go:255,259`） | calculator | **`math.Round` 静默进位** | ⚠ 待收 |
| auto_finance 月供公式 | 1（`service.go:281-287`） | calculator | float 指数 + 单次 Round | ⚠ 展示层可容忍，落库不可 |
| mall 改价入口 | 1（`mall/service.go:106` `normalizeMoney`） | mall | **`big.Rat.FloatString(2)` 静默四舍五入** | ⚠ **唯一真正吃掉精度且无信号** |
| transaction(C) | **0** | **没有人 Round** | 无 | ❌ 最脏：客户端 float 原样落裸 `numeric` |

`mall` 的**下单聚合**是仓内最严谨的资金运算：`big.Rat` 全程精确（`mall/service.go:303` `total := big.NewRat(0,1)`、
`:342` 解析单价、`:346` `Mul(unit, qty)`、`:347` `Add`），落库/展示时才 `FloatString(2)`（`:364` `LineAmount`、`:380` `Amount`）。
→ **给 h6b 的示范**：新链路照 `big.Rat` 或整数分写，别照 transaction 的 float 直传写。

---

## 5. 精度事故核查 SQL（§5.1 已由 coordinator 只读补跑；§5.2/§5.3 待人执行）

前提：只跑 `SELECT`，**禁止 DDL/DML**；执行方式：`./scripts/load-env.sh psql -h 127.0.0.1 -U postgres -d <库名> -f check.sql`，
或 docker 栈 `docker exec my_go_study_postgres psql -U postgres -d <库名> -At -c "..."`。
Docker 起来之前，本仓沙箱内 `127.0.0.1:5432` 未监听，worker 一条都没跑过。

**执行状态（2026-11-06，如实登记）**：

| 组 | 状态 | 执行者 | 结果 |
|----|------|--------|------|
| §5.1 (1)(3)(4) | **已跑**（`transactions` / `transaction_records` 脏值、`wys_used_car_order` NULL） | **coordinator**（只读，非本 worker） | `0` / `0` / `0` —— **仅覆盖本机 `my_go_study`**，其它环境/生产库需各自跑 |
| §5.1 (2) 样本查询 | 未跑 | — | 前三条计数为 0，样本自然为空，但没实际执行 |
| §5.2 numeric 侧 | **未跑** | — | B 口径是否已失真：无结论 |
| §5.3 跨口径 5 条（A–E） | **未跑** | — | 账本求和不变式：无结论 |

### 5.1 float 侧（C 口径）：值是否已丢失 2 位精度

```sql
-- (1) transactions：裸 numeric 可容纳任意小数位 → 找脏值
SELECT count(*) AS dirty_rows
FROM   transactions
WHERE  amount IS NOT NULL
  AND  amount <> round(amount, 2);

-- (2) 样本（人核对用；确认是客户端 float 累加残值还是人工脏写）
SELECT id, user_id, type, category, amount::text, date, created_at
FROM   transactions
WHERE  amount IS NOT NULL AND amount <> round(amount, 2)
ORDER  BY id DESC
LIMIT  50;

-- (3) 遗留表同理
SELECT count(*) AS dirty_rows
FROM   transaction_records
WHERE  amount IS NOT NULL AND amount <> round(amount, 2);

-- (4) used_car：列有 scale=2，DB 侧不会脏；但要看 NULL（Go 非指针 float64 → NULL 读成 0）
SELECT count(*) AS null_amount_rows
FROM   wys_used_car_order
WHERE  amount IS NULL;
```

回填门槛（h6b 验收第 3 条引用同一判据）：**第 (1) 条必须为 0 行**才允许 `amount → amount_fen` 回填；
非 0 就停下来上报行数与样本，**不许 `trunc` 悄悄吃掉钱**。
→ **本机口径下此门槛已满足**：coordinator 2026-11-06 只读补跑第 (1) 条 = **0 行**（执行者非本 worker）；
换环境/上生产前必须重跑，本结论不外推。

### 5.2 numeric 侧（B 口径）：`numeric(10,2)` 是否被绕过

```sql
-- mall 五处金额列：*100 后非整即异常（正常应恒为 0 行；scale=2 由 DB 保证）
SELECT 'wys_mall_sku.price' AS col, count(*) AS bad
FROM   wys_mall_sku
WHERE  price * 100 <> trunc(price * 100)
UNION ALL
SELECT 'wys_mall_order.amount', count(*)
FROM   wys_mall_order
WHERE  amount * 100 <> trunc(amount * 100)
UNION ALL
SELECT 'wys_mall_order_item.line_amount', count(*)
FROM   wys_mall_order_item
WHERE  line_amount * 100 <> trunc(line_amount * 100)
UNION ALL
SELECT 'wys_mall_payment.amount', count(*)
FROM   wys_mall_payment
WHERE  amount * 100 <> trunc(amount * 100)
UNION ALL
SELECT 'wys_mall_refund.amount', count(*)
FROM   wys_mall_refund
WHERE  amount * 100 <> trunc(amount * 100);
```

### 5.3 跨口径一致性（真正会暴露事故的一组）

```sql
-- (A) 商城订单金额(B) 换算成分，与渠道流水(A) 的 amount_fen 是否恒等
--     非 0 行 = 存在「元/分」两套真相已开始分叉
SELECT count(*) AS mall_payment_drift
FROM   wys_mall_payment p
JOIN   wys_mall_order   o ON o.order_id = p.order_id
JOIN   wys_payment_records r
       ON r.ref_type = 'mall_order'
      AND r.ref_id   = o.order_id::text
WHERE  r.amount_fen IS DISTINCT FROM (o.amount * 100)::bigint;

-- (B) 钱包余额 = 流水累加？（分口径内部自洽性，A 侧最重要的不变式）
SELECT count(*) AS wallet_ledger_drift
FROM (
  SELECT w.user_id,
         w.balance_fen,
         coalesce(sum(l.delta_fen), 0) AS ledger_sum
  FROM   wys_cash_wallet w
  LEFT   JOIN wys_cash_ledger l ON l.user_id = w.user_id
  GROUP  BY w.user_id, w.balance_fen
) t
WHERE  balance_fen <> ledger_sum;

-- (C) 积分同上
SELECT count(*) AS points_ledger_drift
FROM (
  SELECT w.user_id, w.balance, coalesce(sum(l.delta), 0) AS ledger_sum
  FROM   wys_points_wallet w
  LEFT   JOIN wys_points_ledger l ON l.user_id = w.user_id
  GROUP  BY w.user_id, w.balance
) t
WHERE  balance <> ledger_sum;

-- (D) 订单合计 vs 行小计之和（B 侧内部）
SELECT count(*) AS order_line_drift
FROM   wys_mall_order o
JOIN (
  SELECT order_id, sum(line_amount) AS lines_sum
  FROM   wys_mall_order_item
  GROUP  BY order_id
) i ON i.order_id = o.order_id
WHERE  o.amount <> i.lines_sum;

-- (E) membership：amount_fen 被放宽成 NULL（§3 矛盾点），到底有多少存量
SELECT count(*) AS membership_null_amount_fen
FROM   wys_membership_order
WHERE  amount_fen IS NULL;
```

**执行结论**：`Partial` —— PostgreSQL 未运行，5.1/5.2/5.3 **全部未跑**。
`docs/acceptance-records/2026-11-06-harden-h6-money-unit-audit.md` 里记为 Deferred，不称已核查。

---

## 6. 迁移方案（单一口径目标态）

### 6.1 目标口径推荐

**推荐：内部与存储统一到整数分（A），decimal-as-string（B）保留为对外表达，淘汰 float64（C）。**

理由（按证据）：
1. A 侧已有**最完整的约束**（`CHECK (amount_fen > 0)`、`balance_fen >= 0`）且**与渠道对账天然同单位**
   （微信 v2 `total_fee` 直接是分，`verify.go:60`）；
2. `ParseYuanToFen`/`FormatFenToYuan` 已经提供**无损且拒绝越界**的 A↔B 双向门（`cash_money.go:10,25`）——
   本仓**已经有正确的基础设施**，缺的是让 C 侧也走这道门；
3. B 不宜做进程内类型（Go 无原生 decimal，`string` 不能参与运算，`mall` 之所以能算准是因为额外用了 `big.Rat`）；
4. 引入第三方 `decimal` 库（shopspring 等）会新增依赖，与 AGENTS「勿盲跑 `go mod tidy` / 升级依赖需单独一次改动 + 全量 `make test`」
   的硬约束冲突（AGENTS.md 全局约束 9）→ **不推荐**，除非人明确批准。

→ 即：**目标态 = 存储/计算用 A（`int64` 分），对外 JSON 继续用 B（元字符串），彻底移除 C。**

### 6.2 五步法（以 `transactions` 为例，其余链路复用同形）

> 每步都必须是**可单独回滚的一次发布**；步骤之间至少留一个「读旧写新」的观察期。
> 迁移一律落 `migrations/`（`parity-p0-1` 已定其为 schema 单一真相），**不得靠 AutoMigrate 建列**。

| 步 | 动作 | SQL 草案 | 回滚点 | 停机评估 |
|----|------|----------|--------|----------|
| **1 加列** | 加 `amount_fen bigint`，**先不加 NOT NULL/CHECK**（避免存量不满足直接失败） | `ALTER TABLE transactions ADD COLUMN IF NOT EXISTS amount_fen bigint;`<br>`CREATE INDEX IF NOT EXISTS idx_transactions_amount_fen ON transactions (user_id, amount_fen);` | 独立可回滚：`ALTER TABLE transactions DROP COLUMN amount_fen;`（`.down.sql` 对映） | **无停机**；PG `ADD COLUMN` 无默认值 → catalog-only，不重写表 |
| **2 回填** | **前置硬门槛**：§5.1(1) 必须 0 行。分批回填，`trunc` 禁用 | `UPDATE transactions SET amount_fen = round(amount * 100)::bigint WHERE id BETWEEN $1 AND $2 AND amount_fen IS NULL;`（按 id 分页，每批 5k，循环至 0 行） | 回填是 `NULL→值`，可反复重跑（幂等 `WHERE amount_fen IS NULL`）；全量失败直接 `SET amount_fen = NULL` 重做 | **无停机**，但要避开高峰；行数 × 批次决定时长，先在 scratch 量时间 |
| **3 双写** | 写路径同时落 `amount`（旧）与 `amount_fen`（新）；读仍走旧列 | Go 侧：`repo_postgres.go:90`（Create）与 `:124-125`（Update）两个入口都补 `amount_fen` | 关掉双写即回退到步 2 末态；旧列未被破坏所以**读不受影响** | **无停机**；一次代码发布 |
| **4 切读** | 读路径改「以 `amount_fen` 为准，换算回元对外」；对外 JSON **字段名与单位不变** | 无 SQL；Go 侧 entity/DTO 出口 | 单 flag 可切回旧列（观察期内保留）；出问题回退发布即可 | **无停机**；建议灰度（先 lab 后 lan） |
| **5 收紧 + 删列** | 先 `NOT NULL` + `CHECK`，观察后再删旧列 | `ALTER TABLE transactions ALTER COLUMN amount_fen SET NOT NULL;`<br>`ALTER TABLE transactions ADD CONSTRAINT chk_transactions_amount_fen CHECK (amount_fen >= 0);`<br>（观察 ≥1 周后）`ALTER TABLE transactions DROP COLUMN amount;` | 约束可 `DROP CONSTRAINT`；**删列不可逆** → 必须单独一次变更、且事先做 `pg_dump` 单表备份 | 约束校验会**全表扫**（大表长事务，需评估 `lock_timeout`）；删列放最后一个发布窗口 |

**步 5 的约束方向要先核实数据**：h6b 已写「先核实现网是否已有负 `amount`，有则约束降级并在 record 说明为什么」——
`transactions` 用 `type` 列表达收支方向（`transaction.go:14`），故 `>= 0` 是**目标**而非现状；核查 SQL：

```sql
SELECT count(*) AS negative_amount_rows FROM transactions WHERE amount < 0;
```

### 6.3 其余链路各自的最短路径（本 Slice 只给方向，不落地）

| 链路 | 建议 | 为什么不能照抄 transaction 方案 |
|------|------|-------------------------------|
| `wys_used_car_order.amount` | 同样五步；但**先决定 NULL 语义**（`align_column_drift:103` 放宽了 NOT NULL） | Go 是非指针 `float64`，NULL 与 0 不可区分 → 迁移时必须先选 `NULL` 还是 `0`，属业务决策 |
| `transaction_records`（遗留） | 建议**不迁移**，走 `Removed`/`Deferred` | 实体注释自陈已弃用（`transaction_record.go:21-24`）；给它做五步等于给死表做手术。若仍有读方，先确认再删 |
| `mall`（B） | **不动**，只把 `normalizeMoney`（`mall/service.go:106`）的静默舍入改成显式报错 | B 已是精确口径，迁移收益 < 风险；真正要修的是那一处四舍五入 |
| `auto_finance` | 落库已是 A，**只需把 DTO 从 `float64` 元改为 `string` 元**（或补 `*_fen` 字段并存） | 无存储迁移，纯契约变更 → 必须与 Flutter 同步发版 |
| `payment` / `membership` / `wallet` / `points` | **已是目标态，不动** | §4.2/§4.3 已证无换算损失 |

---

## 7. Supabase 路径不可控（必须在方案里回答）

**事实**：`auth.provider=supabase` 时，收支写入打到**云端表**，本仓 `migrations/` 管不到它。

- `app/internal/transaction/repo_supabase.go:137` — `payload["amount"] = input.Amount`（`float64`）→ `Insert`
- `app/internal/transaction/repo_supabase.go:174` — `payload["amount"] = *input.Amount` → `Update`
- 二者都走 `client.WithUserToken(accessToken)`（`:128`/`:159`），受云端 **RLS** 管辖，非本仓 DDL。

**回答（本 Slice 的结论）**：**`provider=supabase` 路径本轮不做迁移，并显式记为 `Degraded`。**

1. 加列/回填/约束在云端需要 Supabase 侧 migration，属另一套发布流程与另一套权限；
2. 代码上必须做到**本地路径生效、Supabase 路径行为不变**（同一实体不能一边有 `amount_fen` 一边没有）；
3. 由此产生**已知不对称**：同一用户在 `provider=local` 下金额精确到分、在 `provider=supabase` 下仍可存 >2dp 脏值。
   → 必须登记到 `docs/placeholder-inventory.md`（本 Slice 已追加，见该文件「金额三套口径」条目），
     并在 h6b 验收里留痕（h6b Brief 验收第 5 条正是这条）。
4. 「该路径不做迁移」是**合法结论**而非遗漏 —— 前提是说出来的，不是让它在切 provider 时静默漂移。

**残留风险**：若哪天把默认 provider 从 local 切回 supabase（`docs/local-auth-postgres.md` 支持回切），
`amount_fen` 在 Supabase 侧不存在会**直接写失败**。→ 解除条件：Supabase 云端表补列，或代码显式禁止该组合。

---

## 8. 连带影响清单

| 影响面 | 证据 | 改单位时的动作 | 归属 |
|--------|------|----------------|------|
| **PostgREST / RLS** | `repo_supabase.go:128,159` `WithUserToken`；AGENTS 全局约束「禁止 Admin 绕过 RLS」 | 云端加列须同步更新 RLS policy 与 PostgREST schema 缓存（`NOTIFY pgrst`）；**不得**为省事改用 service_role | Supabase 侧，Deferred（§7） |
| **Elasticsearch mapping** | `app/internal/transaction/repo_es.go:48` `"amount": { "type": "double" }`；写入 `:93`（`doc["amount"] = t.Amount`）；reindex 脚本 `scripts/reindex_transactions_es.sh:15` 同样 `double` | ES 的 `double` **不能原地改 type**（mapping 不可变）→ 换单位**必然 reindex**；若要精确建议 `scaled_float`(`scaling_factor: 100`) 或 `long`(分) | 单独 Slice（h6b **不改** ES，见其「不做」第 3 条） |
| **搜索语义** | `docs/elasticsearch-transactions.md` | 若 `amount` 改分，用户按「元」搜金额会 100 倍偏 | 同上，需与 mapping 变更同批 |
| **Flutter `TransactionApi`** | `my_ai_project/features/home/lib/home/model/transaction_model.dart:22,39`（`(json['amount'] as num?)?.toDouble()` / `final double amount`） | **对外 JSON 保持「元」则 Flutter 零改动**（h6b 的推荐口径即此）；若改 `amount_fen` 则跨仓同步发版 | 跨仓 Brief |
| **Flutter 其余金额契约**（证明客户端已同时承受三套口径） | `double`：`used_car_order_models.dart:120,148`、`purchase_calculator_models.dart:65,71`、`transaction_list_item.dart:118`（`formatAmount(double)`）<br>`String`：`mall_order.dart:68,79,139,156`（amount）、`:21`（lineAmount）、`wallet_api.dart:106`（recharge `String amount`）<br>`int` 分：`wallet_models.dart:12,21,65,66,74,109,110,115`（`balance_fen`/`delta_fen`）、`membership_models.dart:40`（`price_fen`） | 说明「统一口径」在客户端**同样是破坏性变更**，且 wallet/membership 已按分契约、mall 已按元字符串契约 → **服务端单方收敛会打破其中一侧** | 跨仓 Brief |
| **Realtime / Push 载荷** | 交易变更经 WS Hub 广播（`docs/realtime-websocket.md`） | 若 payload 内金额表达变化，在线端需同步 | 与 Flutter 同批 |
| **`wys_mall_refund` 僵尸表** | 迁移里有建表 + `CHECK (amount > 0)`（`mall_schema.up.sql:184-199`），实体**只有表名常量**（`mall.go:63`）、**无 struct、无写入方** | 实际退款走 `walletSvc.Credit`（`mall/service.go:756-764`），不落该表 → 审计金额时**别去这张表找退款**，会得出「零退款」的错误结论 | 建议单独立 Slice 清理（`Removed` 候选） |

---

## 9. h6b 输入契约（供 [`harden-h6b-transaction-money-unit`](../plans/slices/harden-h6b-transaction-money-unit.md) 直接引用）

1. **目标口径**：`transactions` 存储与内部计算统一到**整数分 `int64`（`amount_fen bigint`）**，
   采用 §6.2 五步；步 2 回填的**前置硬门槛**是 §5.1(1) 返回 0 行，非 0 即停并上报，禁止 `trunc`。
   → **该门槛在本机 `my_go_study` 已由 coordinator 只读取证为 0 行**（2026-11-06，执行者非本 worker）；
   h6b 若换库/上生产需重跑，不外推。
2. **对外 JSON 是否变**：**不变**。`amount` 字段名与「元」的含义保持原样
   （`app/internal/delivery/http/dto/request/transaction_request.go:8,17` 与
   `app/internal/delivery/http/dto/response/transaction.go:16` 三处 float 契约本轮**不动**）。
   → 理由：与 Flutter `TransactionApi` 同步发版超出本轮（§8），且「存储收敛、契约后移」可让每步独立回滚。
   → **本条即 h6b Brief「不做」第 1 条的正式推荐**（默认倾向已在本 Slice 确认证据）。
3. **约束方向**：`CHECK (amount_fen >= 0)`，负向由 `type` 列表达（`transaction.go:14`）。
   **执行前先跑** §6.2 末尾的 `negative_amount_rows` 核查；若现网已有负值，降级为无 CHECK 并在 record 说明原因。
4. **示范代码**：新增金额运算请照 `mall/service.go:303-347`（`big.Rat` 精确聚合）或整数分 + `divRound`
   （`purchase_calculator/service.go:267-275`），**不要**照 `repo_postgres.go:90`（float 直传落裸 numeric）。
5. **不要顺手改的**：`membership`（§4.2 无换算）、`wallet`/`points`（§4.3 已安全且无汇率）、
   `payment`（口径 A 是本方案的目标态本身）、`mall`（B 是精确口径；唯一真 bug 是 `normalizeMoney` 静舍入，另立 Slice）。
6. **必须留痕的不对称**：Supabase 路径不迁移（§7）、ES mapping 仍是 `double`（§8）→ 两条都进 `placeholder-inventory`。

---

## 10. 本 Slice 未做 / Deferred

| 项 | 状态 | 说明 |
|----|------|------|
| §5.1 float 侧核查 SQL | **已取证（本机 0 行）** | worker 沙箱 PG 不可达未跑；**coordinator** 2026-11-06 只读补跑（零 DDL/DML）→ `transactions` 0 / `transaction_records` 0 / `wys_used_car_order` NULL 0。**只覆盖本机 `my_go_study`**，其它环境需各自重跑 |
| §5.2 numeric 侧 + §5.3 跨口径求和 SQL | **Deferred** | 仍未执行 → B/A 口径「现有数据是否已失真」无结论。禁止据此自称「已核查」或「无事故」 |
| 验证命令「清单条数 = 38」 | **Partial** | 38 是 mandated pattern 的上界，但 pattern 本身漏 6 条金额字符串（§2.2）→ 需修订 pattern 才能作为门禁；本文按 44 + int64 清单交付 |
| Flutter 契约的**运行时**验证 | **Partial** | 只做了静态 `file:line` 取证（§8）。`my_ai_project` 在**本仓工作区之外**（本 worktree 的 `../../my_ai_project` 不存在，实际在 `/Users/qpy/Desktop/github/my_ai_project`）→ 未跑 `dart analyze`，未看页面渲染 |
| `normalizeMoney` 静默舍入的修复 | **Deferred** | 属「顺手加个校验」，Brief「不做」明令禁止；需单独立 Slice |
| `payment/service.go:285` 出站 float | **Deferred** | 同上，改 `FormatFenToYuan` 是另一条改动 |
| `wys_mall_refund` 僵尸表清理 | **Deferred** | §8；`Removed` 候选，需先确认无外部读方 |
| `subsidy_value` 一列两义改造 | **Deferred** | §4.5；拆两列或加类型包装，属 `auto_finance` 专项 |
| Mixed 支付模式「现金 + 积分双重计费」语义 | **Deferred** | §4.3 证据链（5 步）+ 资损判断：**取决于产品口径，需人裁决**；口径 A（组合购买）则代码自洽非资损，口径 B（积分抵扣）则用户被双收。**未查线上 Mixed SKU 数量与成交**，故不升级为事故。**h6b 与任何 Slice 都不得顺手改**（h9 未批） |
| 任何代码 / 迁移 / DTO 改动 | **本轮 ONLY 不含** | 落地在 h6b 及后续 Slice |

---

## 附：本文取证用的命令（可复现）

```bash
cd my_go_study
grep -rnE 'float64|string.*numeric' model/entity/ | wc -l          # 38 → §2.1
grep -rnE 'int64' model/entity/ | wc -l                            # 135；38+135=173 → §2.3
grep -rnE '\b(Amount|Price|Balance|Fee|Cost|Total|LineAmount|SubsidyAmountCut)\b[[:space:]]+string' model/entity/   # 盲区 6 → §2.2
grep -rnE 'int64' model/entity/ | grep -viE '<§2.3 的四类 pattern 全部取反>'   # 残差 0 → §2.3
grep -rn "ParseYuanToFen\|FormatFenToYuan" --include="*.go" app/ core/ common/ model/   # 换算门 → §4
grep -rniE "(_fen|amount|price|balance|delta)[^,]*" migrations/*.up.sql | grep -iE "numeric|bigint|double"  # 列型 → §3
grep -rnE "Round|Trunc|decimal|ParseFloat" --include="*.go" app/internal/transaction/   # 0 命中 → §1 结论 3
```
