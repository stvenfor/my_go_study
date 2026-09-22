## Context

See proposal.md — Why. Mall schema and write path already ship (`wys_mall_order*`, create / get / pay / cancel). Flutter「我的」有「我的订单」快捷入口但 toast「开发中」；`features/mall` 只有货架与立即购买闭环。本设计只补买家读列表 + App 列表/详情展示，不改表结构。

约束：OpenSpec apply 根在 `my_go_study`；Flutter 改动在兄弟仓库 `my_ai_project`，任务里单独标明。

## Goals / Non-Goals

**Goals:**

- `GET /api/v1/mall/orders` 买家分页列表 + 可选 `status`
- 列表行载荷够画卡片（含行快照），详情复用现有 GetOrder
- Flutter：Mine 入口 → 列表（状态 Tab）→ 详情；待支付可支付/取消
- 视觉对齐 `MallTheme` / `MineTheme`（灰底白卡、Vercel 字重、红价）

**Non-Goals:**

- 新订单表或改 status 枚举
- 店员/门店侧订单管理、退款、物流、售后
- 真实支付 SDK（继续本地 mock pay）
- 购物车结算页重构

## Decisions

### 1. 表：复用，不重建

- **选择**：继续用 `wys_mall_order` / `wys_mall_order_item` / payment / log。
- **理由**：表与索引 `ix_wys_mall_order_buyer (buyer_user_id, created_at DESC)` 已覆盖列表；再建一套会分裂真相源。
- **备选**：新建「展示用订单投影表」——拒绝，YAGNI。

### 2. API：列表新路由；详情不动

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/mall/orders` | **新增**；`page`/`size`/`status?`；`SuccessList` |
| GET | `/api/v1/mall/orders/:order_id` | 既有详情 |
| POST | `.../pay` · `.../cancel` | 既有；UI 接线 |

- Gin 注意：`GET /orders` 必须注册在 `GET /orders/:order_id` **之前**（或同组无歧义），避免 `orders` 被当成 id。
- 列表项建议结构（与详情字段同源，避免客户端两套模型）：

```json
{
  "order_id": 1,
  "order_no": "...",
  "store_id": 1,
  "status": 0,
  "amount": "19.90",
  "created_at": "...",
  "items": [ { "product_title": "...", "cover_url": "...", "qty": 1, "line_amount": "19.90", "kind": 0 } ]
}
```

- `status` 查询：省略=全部；合法 `0–4`；非法 → 400。
- **备选**：列表只返回 header、客户端再拉 items——多一轮 RTT，列表卡片需要封面，故列表一次带 items。

### 3. 分层落地（Go）

沿现有 mall 链路：`MallRepository.ListOrdersByBuyer` → `MallUsecase.ListOrders` → `MallController.ListOrders` → `registerMallRoutes`。SessionAuth + accountGate 与其它 mall 路由相同；`buyer_user_id` 只取会话，不信任 query。

### 4. Flutter UI

入口：`MineQuickServiceData` id=`order` → 登录后 `Get.toNamed(RoutePath.mallOrders)`（名称可微调）。

| 屏 | 职责 |
|----|------|
| 订单列表 | 顶 Tab：全部 / 待支付 / 已支付 / 已取消（`status` 映射；已履约可并入「已支付」Tab 或单独——默认「已支付」含 `1|2`，取消含 `3`，关闭 `4` 归「已取消」或「全部」可见） |
| 订单卡片 | 左封面（首行）、标题、数量、右状态文案 + 金额；点击进详情 |
| 订单详情 | 状态条、商品行列表、金额、收货信息（实体有则显示）、虚拟码/内容链（已支付且有则显示）、底栏：待支付「取消 / 去支付」 |

Token：复用 `MallTheme`（accent `#0070F3`、price `#EE0000`、background `#F5F5F5`）；空态/加载对齐地址列表页模式。

支付：详情「去支付」走既有 mock 渠道路径（与商品详情 `buy` 相同渠道枚举），成功后刷新详情；列表待支付可次要入口「去支付」进详情再付（少做列表内弹层）。

包归属：订单页放在 `features/mall`（已有 API/主题），Mine 只负责导航，避免 settings 依赖商城细节。

### 5. 文档

在 `docs/local-auth-postgres.md` 商城段补一行列表 API；不必新长大文档。

## Risks / Trade-offs

- [列表带全量 items 体积] → 买家订单行数通常很少；若日后膨胀再加「首行摘要」字段。
- [Flutter 跨仓，本 OpenSpec apply 根不含 `my_ai_project`] → tasks 分 Go / Flutter 两段；Go 可先合入，Flutter 同会话或另仓实施。
- [Tab「已支付」是否含履约] → 默认合并 `1+2`，避免过多空 Tab；记录在 Open Questions 若产品要拆。

## Migration Plan

1. 部署 Go：仅加读接口，无 migration。
2. 发 Flutter：接入口与页；旧客户端忽略新路由无影响。
3. 回滚：去掉列表路由与 Flutter 页即可；表无变更。

## Open Questions

- 列表 Tab 是否单独展示「已履约」：默认否（并入已支付）；若产品要拆，只改 Flutter Tab，API 已支持单 status。
