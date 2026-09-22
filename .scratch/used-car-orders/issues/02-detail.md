# 02 — 详情（API + Flutter 详情）

**What to build:** 销售点进一条本人业务单，看到客户、完整车况、金额（随类型标签）、状态、只读驳回原因/星级、可选图。

**Blocked by:** 01 — 摘要 + 列表（API + Flutter 列表）

**Status:** done

- [x] `GET /api/v1/used-car-orders/:order_id`（仅本人×当前店）
- [x] 非本人/跨店返回不可见错误
- [x] Flutter 详情页美化并对接真详情（非交易详情文案）
