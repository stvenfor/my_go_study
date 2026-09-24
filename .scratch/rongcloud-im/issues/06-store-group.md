# 06 — 门店群（一店一群）

**What to build:** 每个门店至多一个融云群；成员入店被拉入该群，离店被移出；登录时若缺失则幂等补拉。成员能在聊天中看到并使用该门店群。

**Blocked by:** 01 — IM 真连（Token + 连接）

**Status:** partial

- [x] 入店触发拉人入店群；离店触发移出；一店一群映射稳定
- [x] 登录补拉幂等，不重复建群
- [x] 店内成员可在群内发消息（不要求互为好友）
- [x] Group/store-sync usecase 缝有自动化测试（可用融云 Server API 的可替换适配器）

**Partial:** Go EnsureStoreMembership + Server API 已就绪；入店/登录自动调用仍需业务钩子接线。
