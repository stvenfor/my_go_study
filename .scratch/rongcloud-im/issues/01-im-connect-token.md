# 01 — IM 真连（Token + 连接）

**What to build:** 已登录用户能从 BFF 拿到绑定业务 UUID 的 IM Token，Flutter 关闭 Mock 后用 IMLib 连上国内融云；登出断开。可用融云 IM 调试页验收「已连接」。本票不要求发出真实聊天消息。

**Blocked by:** None — can start immediately

**Status:** done

- [x] Go：SessionAuth 下签发 IM Token；返回 userId = 业务 UUID；App Secret 仅服务端；未配置时错误明确
- [x] Flutter：真实 App Key + 拉 Token + IMLib 连接成功；登出断开；真实模式不再使用 `im_u_*` 映射
- [x] IM Session usecase 缝有自动化测试（已登录成功 / 未登录拒绝 / 缺 Secret 失败）
- [x] 调试入口能看出 useMockIm=false 且连接态为已连接（或等价可见状态）
