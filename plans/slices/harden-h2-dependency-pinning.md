# Slice — harden-h2-dependency-pinning

## Slice Brief
- SOURCE_MODULE: `core`（`core/go.mod`）、`app`（`app/go.mod`）
- TARGET_MODULE: 同上 + `go.work` / `go.work.sum` 口径
- Source entry: `core/go.mod:15 gorm.io/gorm v1.31.2`、`core/go.mod:6,7 casbin v2.21.0 + gorm-adapter v3.1.0`、`core/go.mod:3 go 1.25.0`、`app/go.mod:20 gorm.io/gorm v1.25.12`、`app/go.mod:6 eino v0.9.21`、`app/go.mod:3 go 1.24.6`、`common/go.mod:3`/`model/go.mod:3 go 1.22.0`
- Target entry: 单一 gorm 版本口径 + 明确写明的 eino 口径；`go list -m` 结果与 `go.mod` 声明一致（不再出现「声明 1.25.12、实际编 1.31.2」）
- 前置事实更正：漂移**已随 `3f0963e` 进入 HEAD**（`git status` 里 `go.mod` 已无待提交改动），且 HEAD `go build all` 通过 —— 也就是说当前是「已在用的既成事实」，不是「还没验的改动」。**若选回 pin `1.25.12`，那是回退，必须重新跑全套验证；若选接受 `1.31.2`，重点是补齐声明一致性**。
- 本轮 ONLY: 只改 `app`/`core` 的 `go.mod`/`go.sum` 与版本口径说明；把「声明版本 == 构建版本」做成可复核事实
- **决策门 3 已拍板（2026-11-06，人批）**：**接受 gorm v1.31.2，把 `app/go.mod:20` 从 `v1.25.12` 回写为 `v1.31.2`**，不回 pin。含义：
  - 这是「把既成事实写进声明」，不是升级 —— HEAD 本来就在编 1.31.2
  - 因此风险 R2（AutoMigrate 索引/约束命名差异）**已经发生过了**，不是将要发生。你的活是**把它证伪或坐实**：`payment.EnsureSchema` 与 `migrations/` 在全新库 + 已迁移库两种形态下都要跑通
  - `casbin/gorm-adapter v3.1.0` 与 1.31 的兼容性一并验（`core/authz` 起不来就是它）
- 不做: 不改任何 `.go`（真需要改代码就说明超出本 Slice，停下问人）；不动 `go.work`；不升级其它依赖；不顺手做 h3 的 CI
- 验收:
  - [ ] **先取证再落地**：`go -C app list -m gorm.io/gorm` 现输出 `v1.31.2`，而 `app/go.mod:20` 写的是 `v1.25.12`（workspace MVS 用 core 的高版本覆盖了 app 的声明）。按人已批方向：把 `app/go.mod` 回写为 `v1.31.2`，消除不一致
  - [ ] 版本口径落盘：每个模块 `go.mod` 显式写 gorm/eino/casbin 版本，不留「靠 tidy 隐式拉最新」的口子（AGENTS.md ✱）
  - [ ] `go` 指令口径统一（当前 1.25.0 / 1.24.6 / 1.22.0 三套），并说明为什么选它
  - [ ] 全量回归：`go build all` + `go -C app test ./... -count=1` + integration 五例（`-tags=integration`，scratch 库，绝不连业务库 `my_go_study`）
  - [ ] **两种库形态启动冒烟**（风险 R2）：a) 全新迁移库从空建起；b) 已按旧迁移建过的库再启动 —— 都要成功，尤其 `payment.EnsureSchema` 的 `uni_wys_payment_records_out_trade_no`（本轮已踩过 SQLSTATE 42704）
  - [ ] 独立一次提交（只含 `go.mod`/`go.sum`），commit message 写明「本仓曾发生 tidy 隐式升级」的来龙去脉（等你说提交时才提交）
- 文件白名单:
  - `app/go.mod`
  - `core/go.mod`
  - `app/go.sum`
  - `core/go.sum`
  - `docs/acceptance-records/2026-11-05-harden-h2-dependency-pinning.md`
  - `plans/slices/harden-h2-dependency-pinning.md`（仅 Context Card）
- 文件黑名单:
  - 任何 `.go` 业务代码（改依赖不该改代码；真要改代码说明超出本 Slice，先停下问人）
  - `go.work`（改 workspace 会牵动所有模块，需单独决策）
  - `thunder/`
  - `.env*`
- 验证命令:
  - `go build all`
  - `go -C app list -m gorm.io/gorm github.com/cloudwego/eino && go -C core list -m gorm.io/gorm github.com/casbin/casbin/v2`
  - `GOPROXY=https://goproxy.cn,direct go -C app test ./... -count=1`
  - `GOPROXY=https://goproxy.cn,direct go -C app test -tags=integration ./internal/{payment,mall,membership}/ -run 'MoneyTx|Notify|Tx|Reconcile' -count=1`
- 证据: `docs/acceptance-records/2026-11-05-harden-h2-dependency-pinning.md`

## 决策门（2026-11-06 已批，不再阻塞）
- ~~gorm 方向~~ → **已批：接受 `1.31.2`，回写 `app/go.mod`**。
- 仍待你上报再定：casbin + gorm-adapter 是否留在 `core`（`core/authz` 已 tracked；本 Slice 默认**保留**，只在兼容性取证失败时提出移除建议）。

## Context Card — harden-h2-dependency-pinning
- 已完成:
- 未做/Deferred:
- 关键文件:
- harness: post ok?
- 下一 Slice 建议:
- 已知坑:
