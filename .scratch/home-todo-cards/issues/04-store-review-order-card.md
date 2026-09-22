# 04 — 店务审核单 + 聚合小卡 + Flutter 只读列表

**What to build:** 当前店存在待审核的店务审核单（与商城买家订单分离）时，首页聚合出现「订单待审核」小卡；店管可打开只读列表，与 count 一致。本票不扩展 `wys_mall_order`，不做审单写流与装箱滑动。

**Blocked by:** 01 — 入店申请 + 首页聚合（仅 partner_pending）+ Flutter 大卡与确认/拒绝

**Status:** done

- [x] 店务审核单持久化（`wys_`）；待审状态计入当前店 count；禁止复用商城订单表承载此域
- [x] 有 `store_admin` 且 count>0 时聚合出现 `order_pending_review`；排序在所有中卡之后
- [x] SessionAuth 只读列表 API 与 count 口径一致
- [x] 域 usecase 与 HomeTodo 接入有自动化测试
- [x] Flutter：小卡展示 + 只读列表页；种子可联调
