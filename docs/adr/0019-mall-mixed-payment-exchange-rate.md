# 商城组合支付口径 = 兑换率折算（部分取代 ADR-0008）

Status: 2026-11-06 人批（口径选项②）。本 ADR **部分取代** [`0008-mall-points-and-mixed-payment.md`](./0008-mall-points-and-mixed-payment.md)：结算规则全部保留，被取代的只有 0008 里「Rejected: points offset N yuan」那一条对**定价与展示模型**的否决。

## 决定

每个 SKU 的两个定价字段语义固定为：

- `wys_mall_sku.price`（`numeric(10,2)`，元）= **抵扣后应付现金**。买家实际要付的现金，**不是原价**。
- `wys_mall_sku.price_points`（`int64`）= **为拿到上述抵扣而必须消耗的积分数量**。
- 兑换率 = **100 积分折 1 元现金**，落在常量 `entity.MallPointsPerYuan`（`model/entity/mall.go`），**不进配置文件**；唯一折算入口 `entity.PointsValueFen(points) int64`（分）。
- **展示原价（分）= `ParseYuanToFen(price) + PointsValueFen(price_points)`**。这是给人看的数，不参与结算。

例：`price=9.90` + `price_points=50` ⇒ 顾客付 ¥9.90 与 50 积分，折合原价 ¥10.40。

## 从 ADR-0008 保留下来、明确不许改的部分

1. **两腿并列、互不净额**：结算仍然「收全额 `order.Amount`（现金）+ 扣全额 `order.TotalPoints`（积分）」。`PointsValueFen` **严禁进入结算路径** —— 用它去净额抵扣现金腿等于把积分价值算两次，是二次打折 / 反向资损。
2. 一单只允许同一种 `payment_mode`；积分在 pay 动作扣、失败或履约前取消全额恢复；组合单需要现金腿与积分腿都成功。

口径②因此**不产生任何行为变化**，它只改变「库里那两个数字该怎么读、录入端该怎么填、前端该怎么展示」。

## 0008 拒绝的东西现在被接受到哪一步

0008 拒绝「points offset N yuan」的理由是 *harder ledger + partial-pay edge cases **before the points wallet exists***。前半句仍然成立（所以结算规则照抄保留），后半句的前提已过期：积分台账早已存在（`app/internal/points`，`Credit/Debit` 带 reason 与 ref）。于是本期接受的是**定价与展示层**的「积分抵扣部分现金」口径，**拒绝的仍然是结算层的净额抵扣**。两者不矛盾：抵扣发生在商家定价那一刻（他把抵扣后的应付现金直接写进 `price`），不发生在下单那一刻。

## 后果与代价

- **存量数字含义变了**：同一条 `price` 从「原价」改读作「抵扣后应付」。已入库的唯一一行组合价 SKU（`sku_id=37`，`price=9.90`，`price_points=50`）到底是「已按②定价」还是「把原价误填进 price」，**只有商家能定性** → 本轮零迁移、零 UPDATE，列为人裁定项（Slice h9 残留）。
- **改兑换率 = 数据变更**：调 `MallPointsPerYuan` 会让所有存量 `price` 与商家意图的原价错配，必须同步重刷存量价并回归不变量测试；这是它不进配置的原因（配置漂移会静默改变存量价语义）。
- **本轮顾客可见价格零变化**：后端没有在 offer/detail 出参里新增「折合原价 / 已抵 ¥x」字段（加字段即改 JSON 契约）。要让买家真看到抵扣感，需要一份跨仓 Brief（Flutter 商城页 + 店员录入端的语义校验）。
- `CreateSKUInput` 未加语义校验，录入正确性目前只由文档与代码注释约束 → 残留项。

## 关联

- Slice：`plans/slices/harden-h9-mall-mixed-payment-deduction.md`（口径②落地）
- 落地文档：`docs/mall-mixed-payment.md`
- 证据：`docs/acceptance-records/2026-11-06-harden-h9-mall-mixed-payment-deduction.md`
