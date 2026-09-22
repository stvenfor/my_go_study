# 02 — 待跟进客户 + 聚合中卡 + Flutter 中卡与只读列表

**What to build:** 当前店存在「下次跟进时间已到期（含此刻）」的客户时，首页聚合在权限允许下多出「待跟进客户」中卡；店管可打开只读列表，列表条数与卡上 count 一致。本票不做跟进完成写操作，也不做装箱左右滑（交给 05）。

**Blocked by:** 01 — 入店申请 + 首页聚合（仅 partner_pending）+ Flutter 大卡与确认/拒绝

**Status:** done

- [x] 门店客户（或等价）持久化，含 `next_follow_up_at`；当前店 count = 已到期未清条目
- [x] 有 `store_admin`（首期映射）且 count>0 时聚合出现 `follow_up_customer`；排序在 `partner_pending` 之后、其余中/小卡之前；无权限或 0 则省略
- [x] SessionAuth 只读列表 API：仅当前店、过滤口径与 count 一致
- [x] 域 usecase（count/列表口径）与 HomeTodo 接入有自动化测试
- [x] Flutter：中卡占整行两格展示；CTA 进只读列表；种子数据可联调
