# Epic — backend-hardening（工程地基加固）

> Program: [`plans/2026-11-05-backend-hardening-program.md`](../2026-11-05-backend-hardening-program.md)
> 差距来源: 2026-11-04 问题盘点（Program §背景，9 条全部带 file:line）
> Acceptance 口径: 只认 `AGENTS.md` + 本 Epic/各 Slice Brief + `docs/acceptance-records/`（`.harness/rules/dual-truth.md`）
> **状态：待人批 Brief；批前禁止任何 commit/push（风险 R1）。**

## 差距表

| 能力 | 现状（证据） | 目标 | Slice |
|------|--------------|------|-------|
| 仓库真实性 | `app/internal/rag/`、`core/authz/`、`app/internal/{txdb,schemaboot}`、`model/entity/payment.go`、`migrations/*.sql`、`plans/`、`docs/` 均未跟踪（`git status --short` 73 项）；而 `app/cmd/api/main.go:54,61` 已 import 进主程序 | `git ls-files` 覆盖所有被 import 的包；杂物不入库；干净检出 `go build all` 通过 | H1 |
| 依赖口径 | `core/go.mod` gorm 1.25.12→1.31.2、`go 1.22→1.25`、新增 casbin；`app/go.mod` eino 0.6.0→0.9.21 —— 违反 AGENTS.md ✱ | 版本口径明确、一次性提交、全量回归；`go mod tidy` 不再隐式升级 | H2 |
| 测试入口 | 5 条 `-tags=integration` 用例无任何执行入口（`grep -rn 'tags=integration' Makefile scripts/ .github` → 0）；无 CI、无 lint、无 vet；harness 机检 `make agent-pre` 在 Makefile 0 命中 | `make test-integration/lint/vet` 可跑；CI 起 PG 真跑 integration；故意造失败能被拦 | H3 |
| 接口鉴权 | `POST /api/v1/push/send` 信任 `body.UserID` 覆盖会话 userID（`jpush_controller.go:91-93`），无 admin 校验 | 客户端只能给**自己**推送；跨用户需显式权限位 + 负例测试 | H4 |
| 错误可观测 | 4 处 `_, _ =` 吞错（`worker/handler.go:216`、`realtime_push_usecase.go:168`、`access/service.go:102`、`profile_usecase.go:82`）+ `im_group_usecase.go:163` 吞迁移错 | 每条降级都有日志+指标；会丢数的路径升级为失败 | H5 |
| 金额口径 | payment/wallet/mall 整数分；`transaction.go:16,28,36`、`used_car_order.go:138,180`、`auto_finance.go:52-70` 仍 `float64`，`used_car/handler.go:94` 直收客户端 float | 单一口径 + 可执行迁移方案（本轮只出方案） | H6 |
| 可上线性 | `APP_ENV=production` 无 `configs/config.production.yaml` 起不来；`/metrics` 无鉴权且 `lan-up` 映射到局域网（`router.go:65-66`） | production 配置可用且不含弱默认值；`/metrics` 只在本机口 | H7 |
| P0 残留 | F2：`membership_order` 无 `RegisterSettler`（`payment/settle.go:103-116` 走 `settled_no_settler`），渠道买会员要靠客户端回跳 `ConfirmBuyout` 才发权益 | 回调核销即主动发权益，双链幂等 | H8 |

## Slice backlog（有序）

| # | ID | ONLY 摘要 | 依赖 | Brief 已落盘 |
|---|-----|-----------|------|--------------|
| 1 | `harden-h1-repo-hygiene` | 已装配未跟踪包入库（含待批命令）+ `.gitignore` 收口 + 干净检出编译证明 | 无 | ☐ 待批 |
| 2 | `harden-h2-dependency-pinning` | gorm/eino/go 指令版本口径落地 + 两种库形态启动冒烟 | h1 | ☐ 待批 |
| 3 | `harden-h3-ci-and-test-entrypoints` | `make test-integration/lint/vet` + CI（含 PG integration job）+ harness 机检 | h1,h2 | ☐ 待批 |
| 4 | `harden-h4-push-route-authz` | 推送路由不信任客户端 user_id + 负例测试 | h3 | ☐ 待批 |
| 5 | `harden-h5-silent-error-sweep` | 5 处吞错留痕/指标/升级 + 清单落盘 | h3 | ☐ 待批 |
| 6 | `harden-h6-money-unit-audit` | float→分 影响面与迁移方案（只读） | h3 | ☐ 待批 |
| 7 | `harden-h7-prod-bootability` | `config.production.yaml` + `/metrics` 收口 | h3 | ☐ 待批 |
| 8 | `harden-h8-membership-settler` | `RegisterSettler(membership_order)` + 双链幂等真库用例 | h1,h3 | ☐ 待批 |

> 并行：h4/h5/h6/h7 四条互不冲突可同批派发；h1→h2→h3 串行；h8 必须晚于 h1 提交。

## 文件锁（并行派发前必读）

| Slice | 主写 | 勿抢 |
|-------|------|------|
| H1 | `.gitignore`、`git add` 范围（无代码改动） | 不改任何业务逻辑；不碰 `go.mod` |
| H2 | `app/go.mod`、`core/go.mod`、对应 `go.sum` | h2 期间其它 Slice 不许加依赖 |
| H3 | `Makefile`、`.github/workflows/`、`scripts/` | 不改业务代码；不改测试断言语义 |
| H4 | `app/internal/delivery/http/controller/jpush_controller.go`、其 router/测试 | 勿改 SessionAuth 全局语义 |
| H5 | `delivery/worker/handler.go`、`usecase/{realtime_push,profile,im_group}_usecase.go`、`access/service.go`、`common/metrics/` | 勿动 `payment/`（那是 H8） |
| H6 | 只读 + `docs/`（方案文档） | **禁止改实体与迁移** |
| H7 | `configs/`、`docker/docker-compose*.yml`、`router/router.go`（仅 `/metrics` 段） | 勿顺手做鉴权重构 |
| H8 | `app/cmd/api/main.go`、`app/internal/membership/`、`app/internal/payment/`（仅注册处） | 勿改 `payment` notify 验签逻辑 |
| 全部 | `docs/placeholder-inventory.md` | **追加式**登记，勿整文件重写 |

## 决策门（需人批，编排不得自决）

1. H1 的 `git add`/commit 范围与执行时机。
2. H1 杂物归属：`NOTES.md`、`learning-records/`、`reference/`、`assets/`、`data/`（`/teach` 产物）→ 入库 / ignore / 移出仓库。
3. H2 方向：gorm 回 pin `1.25.12` vs 正式接受 `1.31.2`。
4. H6 是否落地 float→分（默认只做方案）。
