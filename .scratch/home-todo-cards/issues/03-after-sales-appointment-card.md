# 03 — 售后预约 + 聚合中卡 + Flutter 只读列表

**What to build:** 当前店存在待处理且预约日 ≥ 今天（上海自然日）的售后预约时，首页聚合出现「售后预约」中卡；店管可打开只读列表，与 count 一致。过期未处理不进首页 count。本票不做预约写流与装箱滑动。

**Blocked by:** 01 — 入店申请 + 首页聚合（仅 partner_pending）+ Flutter 大卡与确认/拒绝

**Status:** done

- [x] 售后预约持久化（`wys_`）；待处理 + 预约日 ≥ 今天才计入 count
- [x] 有 `store_admin` 且 count>0 时聚合出现 `after_sales_appointment`；排序在 `follow_up_customer` 之后、小卡之前
- [x] SessionAuth 只读列表 API 与 count 口径一致
- [x] 域 usecase（含固定上海日界）与 HomeTodo 接入有自动化测试
- [x] Flutter：中卡展示 + 只读列表页；种子可联调
