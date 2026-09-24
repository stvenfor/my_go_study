# 02 — 单聊文本 + 会话列表

**What to build:** 两名已登录、已知对方 UUID 的 lab 用户，能在聊天 Tab 发收文本消息；会话列表来自融云 SDK，不再依赖 Mock 种子会话。本票**不**强制好友准入（准入见 04）。

**Blocked by:** 01 — IM 真连（Token + 连接）

**Status:** partial

- [ ] 双方 IM 已连接时可发收文本；消息出现在对方会话中
- [ ] 会话列表反映真实单聊会话（非 Mock 固定三人）
- [ ] Flutter 聊天详情/列表走 SDK，不再写 Mock store 作为真相源
- [ ] 与 Realtime / AI SSE 无耦合

**Partial:** BFF/会话准入与连接已就绪；聊天列表/发收仍可走 Mock store，真 SDK 会话同步待补齐（需真机 App Key）。
