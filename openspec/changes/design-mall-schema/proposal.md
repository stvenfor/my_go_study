## Why

本地库已有 `users` 与 `wys_store`，没有商品、库存、订单和支付。实体与虚拟若拆成两套商品表，价格、上下架和订单行会各写一遍。先共用目录，支付渠道写死枚举，任意渠道点支付都在本库直接记成功（不调渠道 SDK），把主流程跑通。

## What Changes

- 新增门店商城：类目、SPU、SKU（价格与实体库存在 SKU）、虚拟兑换码池、购物车、订单与明细快照、订单状态日志、支付、退款、审计
- 金额用 `numeric(10,2)`。不建物理外键。类目/商品/SKU 软删除
- `payment_channel`：`1` 支付宝、`2` 微信、`3` 苹果内购、`4` 华为内购。订单、支付、退款共用该枚举
- 买家点支付：只要渠道合法且订单待支付，本库写入成功支付并扣库存/发码，**不调用**任何渠道接口
- 店内写目录沿用 `store_admin` / `store_staff`，新增能力码 `mall.catalog.write`
- 不做：真实渠道验签、物流承运商、优惠券、评价、秒杀、分表

## Capabilities

### New Capabilities

- `mall-commerce`: 门店商城目录（实体/虚拟共用 SPU/SKU）、购物车、下单、本地模拟支付（四渠道均可成功落库）

### Modified Capabilities

- （无既有 main specs）

## Impact

- **DB**：本地 `wys_mall_*`；逻辑关联 `users.user_id`、`wys_store.store_id`；无物理外键
- **Go**：迁移、entity、仓储、用例、SessionAuth HTTP；仅 `auth.provider=local`
- **Flutter**：本变更不改客户端
- **权限**：`mall.catalog.write` 绑到现有店内角色
