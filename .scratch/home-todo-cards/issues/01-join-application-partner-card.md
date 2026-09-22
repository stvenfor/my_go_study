# 01 — 入店申请 + 首页聚合（仅 partner_pending）+ Flutter 大卡与确认/拒绝

**What to build:** 用户可对某店提交入店申请；当前店下有 `member.write` 且存在 pending 申请时，首页待办卡聚合返回一张「新伙伴待确认」大卡（服务端只下发 type，不下发 size）；Flutter 用聚合替换该 mock，以大卡展示并可进入列表确认（成为门店成员）或拒绝。本票不要求中/小卡、装箱滑动或多域混排。

**Blocked by:** None — can start immediately

**Status:** done

- [x] 入店申请持久化（`wys_`）；状态 pending/approved/rejected；SessionAuth 下可申请、按当前店列出 pending、确认、拒绝
- [x] 确认后申请人成为当前店门店成员；拒绝后不再计为 pending；重复确认/已是成员行为安全（不重复成员行）
- [x] `GET /api/v1/home/todo-cards`：无当前店或 count=0 或无 `member.write` 时不出现 `partner_pending`；有则带 title/subtitle/action_label/action_route/count；排序位居首位；不下发 `size`/`pages`
- [x] 域 usecase 与 HomeTodo usecase（至少 partner 路径）有自动化测试锁住上述行为
- [x] Flutter：首页该卡走聚合；大卡展示；列表页可确认/拒绝并刷新；LAN 可种子至少一条 pending 便于联调
