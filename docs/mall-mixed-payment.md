# 商城组合支付（现金 + 积分）口径

> 口径来源：人批「口径②」（2026-11-06，dispatch `task_eeb8ddbf228a`）；决策记录为
> `docs/adr/0019-mall-mixed-payment-exchange-rate.md`（**部分取代** ADR-0008）。该记录由 coordinator 代笔人批决定，
> worker 未改 `docs/adr/**`；本 worktree 内的 0019 文件系 coordinator 在验收时补入，故链接在本树可解析。
> 结算规则继承 [`docs/adr/0008-mall-points-and-mixed-payment.md`](./adr/0008-mall-points-and-mixed-payment.md)，
> 定价 / 展示模型以 0019 为准（见 §1）。
> 适用表：`wys_mall_sku` / `wys_mall_order` / `wys_mall_order_item`（实体见 `model/entity/mall.go`）。
> 本轮交付状态见 [`docs/acceptance-records/2026-11-06-harden-h9-mall-mixed-payment-deduction.md`](./acceptance-records/2026-11-06-harden-h9-mall-mixed-payment-deduction.md)（`Degraded`，跨仓另一半未做）。

## 1. 两个价格字段的语义（口径②）

| 字段 | 类型 | 语义 | 不是什么 |
|------|------|------|----------|
| `WysMallSKU.Price` | `string` + DB `numeric(10,2)`（元） | **抵扣后应付现金**：买家在扣掉积分抵扣之后，实际要支付的现金金额 | **不是原价**，不是「积分抵扣前的标价」 |
| `WysMallSKU.PricePoints` | `int64` | **抵扣所需积分**：为拿到上述抵扣而必须消耗的积分数量 | 不是「可抵扣金额」，不是现金 |

一句话：`price` 是「还要付多少钱」，`price_points` 是「还要交多少积分」；两者相加（把积分折成现金后）才是商品的**展示原价**。

> **与 ADR-0008 的关系要分开看**（按 `docs/adr/0019-mall-mixed-payment-exchange-rate.md`，0019 **部分取代** 0008）：
> - **继承 0008**：结算两腿并列、互不净额 —— 收全额 `order.Amount`（现金）+ 扣全额 `order.TotalPoints`（积分），
>   `PointsValueFen` 严禁进入结算路径。0008 的 "mixed = pay the full CNY price and the full points price together,
>   no sliding points-as-discount" 在**结算层**完全保留。
> - **取代 0008 的 `Rejected: "points offset N yuan"`**：口径②接受的正是「积分抵扣部分现金」这个模型，
>   但**接受发生在商家定价那一刻**（他把抵扣后的应付现金直接写进 `price`），**不发生在下单 / 结算那一刻**；
>   因此兑换率折算只用于派生「展示原价」，不改结算。
> - 0008 拒绝理由的前半句（harder ledger / partial-pay edge cases）**仍然有效**；后半句前提
>   （*before the points wallet exists*）**已过期** —— 积分台账 `app/internal/points` 早已存在
>   （`Credit` / `Debit` 带 reason 与 ref）。
>
> 一句话：`price` 与 `price_points` 是两笔**并列**、互不净额的支付义务，不是「在下单时用积分去抵掉一部分 `price`」。

## 2. 兑换率与唯一折算入口

- 默认率：`entity.MallPointsPerYuan = 100`，即 **100 积分 = 1 元 = 100 分**，故 **1 积分 = 1 分**，
  默认率下整除、无余数。
- 全仓唯一积分→现金折算入口：`entity.PointsValueFen(points int64) int64`
  （`model/entity/mall.go:119`），返回该批积分对应的**现金价值（分）**。
  - 取整方向：整数除法向零截断（正数即向下取整），只保留整分。
  - 余数归属：不足 1 分的亚分价值直接丢弃，归平台/商家，**不找零、不累计、也不折算进应付现金**；
    应付现金只由 `Price` 决定。默认率下永不产生余数，截断语义仅在改率后生效。
  - 该函数**只用于派生「展示原价」**，严禁进入结算扣减（见 §4）。

## 3. 原价算式与真实算例

```
展示原价（分） = ParseYuanToFen(Price) + PointsValueFen(PricePoints)
按数量展开   ： 展示原价（分） = [ParseYuanToFen(Price) + PointsValueFen(PricePoints)] × qty
等价地        ： 实付现金（分） + PointsValueFen(实扣积分) == 展示原价（分）
```

其中 `ParseYuanToFen` 见 `model/entity/cash_money.go:10`（元字符串 → 整数分，避免 float）。

**真实算例 `sku_id=37`**（存量行，见 §7，本轮零迁移）：

| 项 | 值 | 分 |
|----|----|----|
| `price`（抵扣后应付现金） | 9.90 元 | 990 |
| `price_points`（抵扣所需积分） | 50 积分 | `PointsValueFen(50)` = 50 |
| **展示原价（折合）** | **10.40 元** | **1040** |

买家实付：`9.90 元现金 + 50 积分`，合计价值 `10.40 元`。若 `qty=3`：实付 `29.70 元 + 150 积分`，
展示原价 `31.20 元`（3120 分）。此算例由 `TestMixedPaymentDeductionInvariants` 与
`TestPointsValueFen` 锁死（`app/internal/mall/service_mixed_payment_test.go`）。

## 4. 支付 / 结算 / 退款口径（最要紧：防反向资损）

现行写法在口径②下**就是正确的**，不得改动：

- 下单 `CreateOrder`（`app/internal/mall/service.go:293`）：现金腿 `Price×qty` 与积分腿
  `PricePoints×qty` **各自独立累加**；`order.Amount` 只来自 `Price`，`order.TotalPoints` 只来自 `PricePoints`。
- 支付 `settleLocal`（`app/internal/mall/service.go:665`）：收**全额** `order.Amount`（现金）
  \+ 扣**全额** `order.TotalPoints`（积分）。渠道支付 `SettleChannelPaid` 同样按 `TotalPoints` 全额扣。
- 退款 `CancelOrder`（`app/internal/mall/service.go:744`）：按 `row.Amount` 全额退现金、
  按 `row.TotalPoints` 全额退积分，与实付 / 实扣一一对应。

**禁止「先算抵扣再收现金」**：把 `price` 当成原价、再减一次 `PointsValueFen(price_points)` 去收现金，
等于把积分的抵扣价值用了两次 —— 二次打折 / 反向资损（商家少收 `PointsValueFen(TotalPoints)`）。
`entity.PointsValueFen` 绝不参与结算扣减。

不变量（由 `app/internal/mall/service_mixed_payment_test.go` 断言）：

1. 实付现金额 == `price` 分数 × `qty`；
2. 实扣积分 == `price_points` × `qty`；
3. 实付现金分 + `PointsValueFen(实扣积分)` == 展示原价分；
4. 退款对称：退现金 == 实付现金，退积分 == 实扣积分；
5. 不存在同一路径既收现金又扣同一份价值：现金腿与积分腿是展示原价的两块**互不重叠**的组成部分。
   若现金腿 == 展示原价，则同一份积分价值既以现金收、又以积分扣 = 双重计费。

## 5. 店员录入约定（CreateSKUInput）

`app/internal/mall/service.go:92` 的 `CreateSKUInput` 只做「金额格式 + 负数」校验，
**不做口径语义校验** —— 店员把原价填进 `price` 时，那一行才会变成双重计费。录入时须遵守：

- `Price` 填**抵扣后应付现金**，不是原价；`PricePoints` 填获得该抵扣所需的积分。
- 想表达「原价 10.40、用 50 积分抵 0.50 元」→ 录 `Price="9.90"`、`PricePoints=50`。
  **不要**录 `Price="10.40"`、`PricePoints=50`：那样现金按原价收、积分又扣一份价值，双重计费。
- 纯现金：`PricePoints=0`；纯积分：`Price="0.00"`；组合：两者都 > 0。
- 录入自校：`entity.ParseYuanToFen(Price) + entity.PointsValueFen(PricePoints) == 期望原价分`。
- 一笔订单只允许同一种 `payment_mode` 的 SKU（`ErrPaymentModeMixed`，见 §6）。

> 残留（Deferred）：`CreateSKU` 未加「原价/抵扣」语义校验，本轮只补文档与代码注释，
> 未改校验逻辑（超出本轮 ONLY 范围）。

## 6. 与 payment_mode 的派生关系

`entity.MallSKUPaymentMode(priceCNY, pricePoints)`（`model/entity/mall.go:94`）由两个字段**派生**，不另存列：

| `Price` | `PricePoints` | `payment_mode` | 含义 |
|---------|---------------|----------------|------|
| > 0 | = 0 | `MallPayModeCNY`(1) | 纯现金 |
| = 0 | > 0 | `MallPayModePoints`(2) | 纯积分 |
| > 0 | > 0 | `MallPayModeMixed`(3) | 组合支付（本文件的主题） |
| = 0 | = 0 | `MallPayModeCNY`(1) | 兜底（异常录入，应避免） |

- `payment_mode` 是**商品层的定价形态**；`MallPayPoints`(5) 等是**支付渠道编号**，两者不同概念，勿混用。
- 组合订单要求所有行同 mode，混搭在下单层即被拒（`ErrPaymentModeMixed`）。
- `price_points` 的现金价值只影响「展示原价」的派生，**不影响** `payment_mode` 的判定逻辑
  （判定只看两字段是否为零）。

## 7. 改率须重刷存量价的风险声明

`MallPointsPerYuan` 是编译期常量，也是全仓折算的唯一口径来源。改动它会改变
`PointsValueFen` 的结果，从而改变**所有** SKU 的「展示原价」派生值；但存量
`wys_mall_sku.price`（抵扣后应付现金）与 `wys_mall_sku.price_points`（抵扣所需积分）
是按**旧率**由商家逐行议定 / 录入的，改率后二者不再自洽：要么
`price + PointsValueFen(price_points)` 不再等于商家意图的原价，要么积分抵扣价值与现金腿错配。

改率必须同时：

1. 全量盘点：`SELECT sku_id, price, price_points FROM wys_mall_sku WHERE price::numeric>0 AND price_points>0;`
   （业务库**只读**）；
2. 按新率重算并**重刷**存量 `price` / `price_points` —— 逐行由商家裁定，程序无权代定；
3. 同步 Flutter 端展示口径、对账口径与店员录入培训材料；
4. 回归本文件 §4 的不变量测试（尤其 `TestPointsValueFen` 里对 `MallPointsPerYuan==100` 的守卫）。

**严禁只改常量不刷数据**。

**存量 `sku_id=37` 待人裁定**：该行 `price=9.90`、`price_points=50` 属于「已按口径②正确定价」
还是「把原价误填进 price」只有商家知道；执行方与人（本轮）均无权定性。本轮**默认零迁移、不动数据**，
不 `UPDATE` 该行，作为待人裁定项写进 acceptance-record 残留。

## 8. 代码与测试索引

| 位置 | 作用 |
|------|------|
| `model/entity/mall.go:54` | `MallPointsPerYuan = 100` 常量 |
| `model/entity/mall.go:119` | `PointsValueFen` 唯一折算入口 |
| `model/entity/mall.go:94` | `MallSKUPaymentMode` 派生 payment_mode |
| `app/internal/mall/service.go:92` | `CreateSKUInput` 店员录入（本轮仅补口径注释） |
| `app/internal/mall/service.go:293` | `CreateOrder` 现金/积分腿独立累加 |
| `app/internal/mall/service.go:665` | `settleLocal` 全额收现金 + 全额扣积分 |
| `app/internal/mall/service.go:744` | `CancelOrder` 退款对称 |
| `app/internal/mall/service_mixed_payment_test.go` | 五条不变量 + `sku_id=37` 算例 |

## 9. 本轮之后买家可见价格零变化（硬事实，不许被「口径已落地」盖过去）

- `grep -rn "PointsValueFen" --include='*.go' app/internal/mall model/entity/mall.go` 在**生产代码零调用点**：
  只有 `model/entity/mall.go` 的定义 / 常量、`app/internal/mall/service.go` 的口径注释、以及测试文件。
  这是口径②的**正确结果**（结算本来就不需要折算），但也意味着：
  **本轮没有任何接口输出「折合原价 / 已抵 ¥0.50」**，`MallSKUOffer` / `MallProductDetail` 出参未新增任何字段，
  买家侧看到的数字与改口径前**完全一致**。
- 要让抵扣对顾客可感知，必须另出**跨仓 Brief**：改 offer / detail 出参加「折合原价 / 已抵 ¥x」
  （加字段 = 改 JSON 契约），并同步 Flutter 商城页展示与店员录入端语义校验。
- 本条是整片 `Degraded` 的成因之一，已写进 acceptance-record 残留；**不得**用「口径已落地 / 折算入口已补」盖过去。
