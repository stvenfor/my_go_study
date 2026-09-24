# 融云 IM 联调与配置

ADR：Go `docs/adr/0017`、`0018`；Flutter `docs/adr/0012`。Spec：`.scratch/rongcloud-im/spec.md`。

## 凭据

| 项 | 落点 |
|----|------|
| App Key | Flutter：`--dart-define=RONG_APP_KEY=...` 或 `EnvConfig.rongAppKey`（测试/预发默认 lab Key） |
| App Secret | Go：`.env` / `.env.local` → `RONGCLOUD_APP_SECRET`（勿进客户端） |
| API 域名 | 国内默认 `https://api.rong-api.com`；可选 `RONGCLOUD_API_BASE_URL` |

未配置或 App Key 含 `PLACEHOLDER`、或 `USE_MOCK_IM=true` 时，Flutter 走 Mock Engine。测试/预发默认真连；线上仍需配置真实 App Key。

`make lan-up`（Docker）需在 `.env.local` 写 `RONGCLOUD_APP_KEY` / `RONGCLOUD_APP_SECRET`，由 compose 注入容器；日志应出现「融云 IM 服务端已配置」。

## 主要 API（SessionAuth）

- `POST /api/v1/im/session` → `{ user_id, token, expires_in_seconds }`（user_id = 业务 UUID；签发时把本地昵称/http(s) 头像同步到融云 getToken）
- `GET /api/v1/im/users/profile?user_ids=` → `{ items: [{ user_id, display_name, avatar_url, ... }] }`（会话列表昵称/头像）
- `GET /api/v1/im/users/search?q=`
- `GET /api/v1/im/friends` · `GET /api/v1/im/friends/requests`（收件箱）· `POST /api/v1/im/friends/requests` · `POST .../requests/:id/respond`
- `GET /api/v1/im/private/admission?peer_user_id=`
- `POST /api/v1/im/groups/free` · invite/kick/quit/dismiss
- `POST /api/v1/im/groups/store/sync` `{ store_id }`
- `POST /api/v1/im/messages/backup` `{ events: [...] }`

## 边界

- 聊天消息客户端直连融云；不经 Realtime WS / SSE。
- 头像/昵称以业务 `users` 表为准；BFF 批量下发；`data:` 头像不同步到融云（仅 http/https）。
- AI 小石头仍走 SSE（Flutter ADR 0005）。
- 离线推送（融云推送 vs JPush）单独立项，本程序不挡主线。

## 验收清单（最低）

1. Go：配置 Secret 后，已登录调用 `/im/session` 拿到 token；缺配置返回「融云未配置」。
2. Flutter：调试页 `useMockIm=false`，连接成功；登出断开。
3. 会话列表标题/头像来自 `/im/users/profile`（非 picsum Mock）。
4. 两台已登录账号（已知 UUID）可单聊文本/图片/语音；会话列表来自 SDK（非 Mock 种子）。
5. 好友搜索手机号 → 申请 → 同意 → 可开单聊；非好友 admission=false。
6. 备份 flush 后业务库有记录且 message_uid 幂等。
7. 门店：入店/切店/登录补拉自动 Ensure；离店 Remove；自由群：好友页「建群」+ 群聊「…」邀请/退群/解散。

## 跑测

```bash
# Go
cd my_go_study
go test ./pkg/rongcloud/ ./internal/usecase/ -count=1 -run 'ImSession|ImFriend|ImGroup|ImBackup|GetToken'
go build -o /dev/null ./cmd/api

# Flutter
cd my_ai_project
dart analyze components/rongcloud_im/lib commons/core/lib/env features/chat/lib/chat/repository
```
