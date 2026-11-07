# Acceptance Record — harden-h4-push-route-authz

- 日期：2026-11-05
- Slice：`plans/slices/harden-h4-push-route-authz.md`
- 主交付物：两处「客户端可指定任意推送目标」的越权收敛（仅改 controller），各配负例/对照单测
- 前置自证：`pwd=/Users/qpy/orca/workspaces/my_go_study/harden-h4`、`HEAD=e288553`（满足）

## status: **Migrated**（两处越权收敛）；跨用户群发本身 = **Deferred**（另立 Slice）

## 越权点与修法

| 入口 | 旧行为 | 新行为 |
|------|--------|--------|
| `jpush_controller.go:91-93` | `targetUser = body.UserID`，空才回落本人；`body.Alias` / `body.RegistrationIDs` 原样透传 → 任意 user_id / alias / registration_id 都能下发 | `user_id` 非空且 != 会话 `user.ID` → **403 + warn**；`alias` / `registration_ids` 非空一律**忽略 + warn**；`SendInput` 固定 `UserID=user.ID, Alias=user.ID, RegistrationIDs=nil` |
| `realtime_controller.go:109-112` | `targetUserID = req.UserID`，空才回落本人 → 任意 userId 推送 | `userId` 非空且 != 会话 `user.ID` → **403 + warn**（`BackendError` 自动映射 `CodeForbidden`）；`PushToUser` 固定 `UserID=user.ID` |

- 自身路径不回归：`user_id`/`userId` 留空或填自己 → 200，usecase 照常调用（对照测试锁死）。
- warn 走 `common/logger.L()`（全局 zap，不改 `main.go`、不注入 logger），结构化字段
  `session_user` / `requested_user` / `ignored_alias` / `ignored_registration_ids`；实测三条路径均输出 warn 日志。
- 取舍说明：Brief 口径「一律以会话用户登记的 device/alias 为准，非本人登记则忽略」。解析「是否本人登记」
  需把 `PushDeviceRepository` 注进 controller（要改 `main.go`，黑名单且与 h8 冲突），故本片**一律丢弃**
  客户端 `alias`/`registration_ids` 并把目标锁定 `user.ID` 别名（与 `RegisterDevice` 默认别名一致），
  即「本人之外的目标一律忽略」，不可绕过。若将来要放行「本人自填 registration_id」，需另立 Slice
  注入 device repo 做归属校验。

## 代码改动（全部在白名单内）

| 文件 | 改动 |
|------|------|
| `app/internal/delivery/http/controller/jpush_controller.go` | `Send` 加 user_id 越权 403；忽略 alias/registration_ids + warn；目标锁定本人；函数注释与 import |
| `app/internal/delivery/http/controller/realtime_controller.go` | `Push` 加 userId 越权 403 + warn；目标锁定本人；函数注释与 import |
| `app/internal/delivery/http/controller/jpush_controller_test.go`（新） | 4 例：他人 user_id（含带 alias 的用例）→ 403 且 spy 计数 0；留空/填自己 → 200 且 spy 计数 1；alias+registration_ids 忽略，极光受众只剩本人 |
| `app/internal/delivery/http/controller/realtime_controller_test.go`（新） | 3 例：他人 userId → 403 且 spy 计数 0；留空/填自己 → 200 且 spy 计数 1、入队 UserID=本人 |
| `docs/placeholder-inventory.md` | 状态速览追加 #22「跨用户推送 = Deferred（需 access 注入，另立 Slice）」 |

未动：`app/cmd/api/main.go`、`router/router.go`、`**/go.mod`、`**/go.sum`（黑名单/h7/h8 独占）；
未 `git add` 任何未跟踪构建前置（go.work / go.work.sum / thunder/logs/logs.go / bin/migrate）。

### 「usecase 未被调用」如何断言

两处 controller 持有的是**具体 usecase 指针**（非接口），为不引入生产侧接口重构 / typed-nil 陷阱，
测试用真实 usecase + 可观测依赖作 spy：

- JPush：`jpush.Client` 注入 `recordingTransport`（`http.RoundTripper`）。`usecase.Send` 一旦被调用，
  在已配置 client 下必然打到该 transport 并记一次 `calls`；`calls==0` 即证明 `Send` 未被调用。
  同时断言 403（本配置下 `Send` 成功只会返回 200，不可能产出 403），双证据。
  捕获的下发体再断言 `audience.alias==["user-1"]` 且无 `registration_id`。
- Realtime：异步模式（`Queue.Enabled=true` → `UseAsyncPush()`）下 `usecase.PushToUser` 必然调用
  `PushEnqueuer.EnqueueRealtimePush`；`recordingEnqueuer.calls==0` 即证明未被调用、未入队。

## 验证（go1.26.4）

按要求用 `/usr/local/go/bin/go`（默认 `go` 是 1.27.3，配 x/net v0.54.0 会让 `go build all` 红，
非本片 bug）；全程 `GOTOOLCHAIN=local`。

```
$ /usr/local/go/bin/go version
go version go1.26.4 darwin/amd64

$ GOTOOLCHAIN=local /usr/local/go/bin/go -C app test ./internal/delivery/http/... -count=1
ok  app/internal/delivery/http/controller        （11 例全 PASS，含 7 例新增）
ok  app/internal/delivery/http/dto/response
ok  app/internal/delivery/http/handler
ok  app/internal/delivery/http/middleware
ok  app/internal/delivery/http/router

$ GOTOOLCHAIN=local /usr/local/go/bin/go -C app test ./internal/usecase/ -run 'Realtime|JPush' -count=1
ok  app/internal/usecase

$ GOTOOLCHAIN=local /usr/local/go/bin/go build all
（无输出，exit 0）
```

## 人工冒烟（未跑，口径说明）

`./scripts/push_notify_user.sh <他人userId>` 预期**返回 403**：脚本用 `curl_json` 且 `set -e`，
非 2xx 会以「错误: realtime push HTTP 403」退出 1，这正是加固后的预期（目标 userId 与操作员
会话用户不一致）。脚本硬编码的是**他人** userId（非操作员自身），无需改断言口径；本片未改该脚本
（不在白名单）。本轮**未实跑**：需 API 已起（HTTP 8081）+ Supabase service role，且 DB 只用 scratch、
业务库只读；该链路的等价性由上面负例/对照单测覆盖。

## 已知残留 / 未验证项

| 项 | 状态 | 说明 |
|----|------|------|
| 跨用户群发（合法管理员路径） | **Deferred** | 需把 `access.Service` 注进 controller → 改 `main.go`（h8 独占、与本片冲突），另立 Slice；已登记 `placeholder-inventory.md` #22 |
| 实机 HTTP 冒烟 | 未跑 | 见上；负例/对照单测覆盖逻辑 |
| 客户端自填本人 registration_id 的放行 | 未做 | 需注入 device repo 校验归属；本片按 Brief 一律忽略 |
| 限流 | 不做 | Brief 明确 P2 范围 |

## harness

- post ok? 本片只改白名单文件、不改权限模型 / 不改 main.go / 不改 Flutter；负例断言 usecase 未被调用；
  `go build all` 绿（go1.26.4）。
- 未 commit / 未 push / 未 `rm -rf`；DB 仅 scratch、业务库只读。
