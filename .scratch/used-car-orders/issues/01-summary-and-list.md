# 01 — 摘要 + 列表（API + Flutter 列表）

**What to build:** 销售打开首页「二手车」能看到顶栏摘要（人设+四格）与本人当前店业务单分页列表（可按 kind/status 筛），数据来自新表种子，不再是收支账本。

**Blocked by:** None — can start immediately.

**Status:** done

- [x] `wys_used_car_order` 表 + 种子（含三种 kind、四态）
- [x] `GET /api/v1/used-car-orders/summary` 与 `GET /api/v1/used-car-orders`（本人×当前店；kind/status 筛选；分页）
- [x] Usecase 单测锁住统计与筛选口径
- [x] Flutter 二手车列表改接新 API，UI 展示类型/车型/金额标签/状态（非收入支出）
