# 04 — 首页待办「二手车待审」

**What to build:** 店管在首页待办看到独立小卡「二手车待审」，计数为当前店全部待审业务单；与「订单待审核」并存。

**Blocked by:** 01 — 摘要 + 列表（API + Flutter 列表）

**Status:** done

- [x] `GET /api/v1/home/todo-cards` 在 `store_admin` 且全店待审 count>0 时下发 `used_car_pending_review`
- [x] 无权限或 count=0 不出现；不并入 `order_pending_review`
- [x] Flutter 待办卡类型映射为小卡并带可用 action_route
