## Why

门店商城已有订单表与创建/详情/支付/取消接口，但缺少买家「我的订单」列表；Flutter「我的」页快捷入口仍是「开发中」占位。用户下单后无法回看历史订单，列表与详情闭环断裂。

## What Changes

- **复用**既有 `wys_mall_order` / `wys_mall_order_item`（及支付、履约字段），**不新建**另一套订单表
- 新增买家订单列表 API：`GET /api/v1/mall/orders`（分页 + 可选状态筛选），仅返回当前登录买家的订单
- 详情继续用既有 `GET /api/v1/mall/orders/:order_id`；待支付订单在详情/列表可走既有 pay/cancel
- Flutter：接通「我的」→「我的订单」入口；订单列表页（状态 Tab）与订单详情页，对齐现有 `MallTheme` / Mine 视觉
- 文档补充列表契约（`docs/local-auth-postgres.md` 或短 API 说明）

## Capabilities

### New Capabilities

- `mall-buyer-orders`: 买家侧「我的订单」列表与详情可读契约（含状态筛选、分页、跨用户隔离；待支付可支付/取消）

### Modified Capabilities

- （无。主库 `openspec/specs/` 尚未归档 `mall-commerce`；本变更以新能力描述买家订单视图，不改写目录/购物车/下单写路径）

## Impact

- **Go BFF**：`mall` repository / usecase / controller / router；列表响应形状对齐既有 `SuccessList`（`list` + `pagination`）
- **表**：无 schema 变更（已有 `ix_wys_mall_order_buyer (buyer_user_id, created_at DESC)`）
- **Flutter**（兄弟仓库 `my_ai_project`）：`features/mall` 增订单 API/页；`MineQuickServiceData` 的 `order` 入口从 toast 改为路由；`RoutePath` 增订单列表/详情
- **鉴权**：与现有商城一致——仅 `auth.provider=local` + SessionAuth；按 `buyer_user_id` 隔离
- **非目标**：退款、物流、店员订单管理、真实支付渠道路由、改价/售后
