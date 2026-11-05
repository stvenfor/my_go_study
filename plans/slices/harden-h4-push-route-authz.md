# Slice — harden-h4-push-route-authz

## Slice Brief
- SOURCE_MODULE: `app`（`internal/delivery/http/controller`）
- TARGET_MODULE: 同上
- Source entry: **同一形状两处**，不只一处 —— `jpush_controller.go:91-93`（`targetUser := body.UserID; if "" { user.ID }`）与 `realtime_controller.go:109-112`（`targetUserID := req.UserID; if "" { user.ID }`）。两者都只过了 SessionAuth，任何登录用户都能给**任意 userID / 任意 alias / 任意 registration_id** 发通知
- Target entry: 客户端只能向**自己**下发；`body.UserID`/`req.UserID` 与会话 userID 不一致即 403；跨用户下发只保留一条明确的受控路径
- 本轮 ONLY: 修这两处目标覆盖 + 各自的负例测试；不动 `access`/`authz` 权限模型，不动 `cmd/api/main.go`
- 不做:
  - 不在本 Slice 引入「管理员可群发」新能力（需要把 `access.Service` 注进 controller → 要改 `main.go`，与 h8 冲突；列为 Deferred 并给出跨 Slice 建议）
  - 不改 SessionAuth 语义、不改 realtime 协议、不改 Flutter
  - 不顺手做限流（那是 P2 的事）
- 验收:
  - [ ] `POST /api/v1/push/send` 带他人 `user_id` → **403**，且不入队（负例测试断言 usecase 未被调用）
  - [ ] `POST /api/v1/realtime/push` 带他人 `user_id` → **403**（同上）
  - [ ] `alias` / `registration_ids` 也不能由客户端任意指定目标：本 Slice 采用「一律以会话用户登记的 device/alias 为准，客户端传的 `alias`/`registration_ids` 若非本人登记则忽略并留 warn」，测试覆盖
  - [ ] 自身路径不回归：`user_id` 留空、或填自己 ID → 200，行为与改前一致（对照测试）
  - [ ] `go -C app test ./internal/delivery/http/... -count=1` 绿；`go build all` 绿
  - [ ] 人在 `docs/placeholder-inventory.md` 追加「跨用户推送 = Deferred（需 access 注入，另立 Slice）」一行
- 文件白名单:
  - `app/internal/delivery/http/controller/jpush_controller.go`
  - `app/internal/delivery/http/controller/realtime_controller.go`
  - `app/internal/delivery/http/controller/*_test.go`（新增负例/对照测试）
  - `app/internal/delivery/http/dto/request/`（仅当需要收紧 DTO 校验）
  - `docs/placeholder-inventory.md`（追加式）
  - `docs/acceptance-records/2026-11-05-harden-h4-push-route-authz.md`
  - `plans/slices/harden-h4-push-route-authz.md`（仅 Context Card）
- 文件黑名单:
  - `app/cmd/api/main.go`（h8 独占）
  - `app/internal/delivery/http/router/router.go`（h7 独占）
  - `app/internal/{payment,membership}/**`（h8 独占）
  - `**/go.mod`、`**/go.sum`
- 验证命令:
  - `go -C app test ./internal/delivery/http/... -count=1`
  - `go -C app test ./internal/usecase/ -run 'Realtime|JPush' -count=1`
  - `go build all`
  - 人工冒烟（API 已起）：`./scripts/push_notify_user.sh <他人userID>` 应返回 403（脚本若硬编码自身则改断言口径并在 record 说明）
- 证据: `docs/acceptance-records/2026-11-05-harden-h4-push-route-authz.md`

## Context Card — harden-h4-push-route-authz
- 已完成:
- 未做/Deferred:
- 关键文件:
- harness: post ok?
- 下一 Slice 建议:
- 已知坑:
