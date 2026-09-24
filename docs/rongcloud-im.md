# 融云 IM 联调与配置

ADR：Go `docs/adr/0017`、`0018`；Flutter `docs/adr/0012`。Spec：`.scratch/rongcloud-im/spec.md`。

## 凭据

| 项 | 落点 |
|----|------|
| App Key | Flutter：`--dart-define=RONG_APP_KEY=...` 或 `EnvConfig.rongAppKey` |
| App Secret | Go：`.env` / `.env.local` → `RONGCLOUD_APP_SECRET`（勿进客户端） |
| API 域名 | 国内默认 `https://api.rong-api.com`；可选 `RONGCLOUD_API_BASE_URL` |

未配置或 App Key 含 `PLACEHOLDER` 时，Flutter 仍走 Mock Engine（`USE_MOCK_IM=true` 可强制 Mock）。

## 主要 API（SessionAuth）

- `POST /api/v1/im/session` → `{ user_id, token, expires_in_seconds }`（user_id = 业务 UUID）
- `GET /api/v1/im/users/search?q=`
- `GET /api/v1/im/friends` · `POST /api/v1/im/friends/requests` · `POST .../requests/:id/respond`
- `GET /api/v1/im/private/admission?peer_user_id=`
- `POST /api/v1/im/groups/free` · invite/kick/quit/dismiss
- `POST /api/v1/im/groups/store/sync` `{ store_id }`
- `POST /api/v1/im/messages/backup` `{ events: [...] }`

## 边界

- 聊天消息客户端直连融云；不经 Realtime WS / SSE。
- AI 小石头仍走 SSE（Flutter ADR 0005）。
- 离线推送（融云推送 vs JPush）单独立项，本程序不挡主线。

## 验收清单（最低）

1. Go：配置 Secret 后，已登录调用 `/im/session` 拿到 token；缺配置返回「融云未配置」。
2. Flutter：填入真实 App Key 后调试页 `useMockIm=false`，连接成功；登出断开。
3. 两台已登录账号（已知 UUID）可单聊文本（好友准入在 04 之后强制）。
4. 好友搜索手机号 → 申请 → 同意 → 可开单聊；非好友 admission=false。
5. 备份 flush 后业务库有记录且 message_uid 幂等。
6. 门店 sync / 自由群群主权限按 usecase 测试覆盖。

## 跑测

```bash
# Go
cd my_go_study
go test ./pkg/rongcloud/ ./internal/usecase/ -count=1 -run 'ImSession|ImFriend|ImGroup|ImBackup|GetToken'
go build -o /dev/null ./cmd/api

# Flutter
cd my_ai_project/components/rongcloud_im && flutter pub get && dart analyze
```
