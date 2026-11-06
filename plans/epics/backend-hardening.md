# Epic — backend-hardening（工程地基加固）

> Program: [`plans/2026-11-05-backend-hardening-program.md`](../2026-11-05-backend-hardening-program.md)
> 差距来源: 2026-11-04 问题盘点（Program §背景，9 条全部带 file:line）
> Acceptance 口径: 只认 `AGENTS.md` + 本 Epic/各 Slice Brief + `docs/acceptance-records/`（`.harness/rules/dual-truth.md`）
> **状态：待人批 Brief。**（原「批前禁止 commit」前置已失效：人于 2026-11-05 提交 `3f0963e`，把代码包 + `plans/` + `docs/` 一并入库，HEAD `go build all` 实测通过。）

## 差距表

| 能力 | 现状（证据） | 目标 | Slice |
|------|--------------|------|-------|
| 仓库真实性 | **原前提已解除**（`3f0963e` 把 `rag`/`authz`/`txdb`/`schemaboot`/`payment.go`/`migrations`/`plans`/`docs` 全部入库，HEAD `go build all` 通过）。剩余：`app/worker`（39MB 二进制）未被 `.gitignore` 覆盖；「import 全覆盖」缺机检 | 脚本证明「每个被 import 的工作区包都 tracked」；编译产物不入库；干净 worktree 复证 | H1 |
| 依赖口径 | `core/go.mod:15` gorm **v1.31.2** + `:3` go 1.25.0 + `:6,7` casbin/gorm-adapter；`app/go.mod:20` 仍声明 gorm **v1.25.12**，但 `go -C app list -m gorm.io/gorm` 实测输出 **v1.31.2**（workspace MVS 用 core 的高版本覆盖 app 的声明）；`app/go.mod:6` eino v0.9.21；`go` 指令三套（1.25.0/1.24.6/1.22.0）。漂移已随 `3f0963e` 进入 HEAD | 声明与实际一致、版本口径明确、一次性独立提交、全量回归；`go mod tidy` 不再隐式升级（AGENTS.md ✱） | H2 |
| 测试入口 | 5 条 `-tags=integration` 用例无任何执行入口（`grep -rn 'tags=integration' Makefile scripts/ .github` → 0）；无 CI、无 lint、无 vet；harness 机检 `make agent-pre` 在 Makefile 0 命中 | `make test-integration/lint/vet` 可跑；CI 起 PG 真跑 integration；故意造失败能被拦 | H3 |
| 接口鉴权 | `POST /api/v1/push/send`（`jpush_controller.go:91-93`）与 `POST /api/v1/realtime/push`（`realtime_controller.go:109-112`）都信任 body `user_id` 覆盖会话 userID，任何登录用户可给任意用户发通知 | 客户端只能给**自己**推送；跨用户需显式权限位 + 两处负例测试 | H4 |
| 错误可观测 | 4 处 `_, _ =` 吞错（`worker/handler.go:216`、`realtime_push_usecase.go:168`、`access/service.go:102`、`profile_usecase.go:82`）+ `im_group_usecase.go:163` 的 `_ = err` | 每条降级都有日志+指标；会丢数的路径升级为失败 | H5 |
| 金额口径 | **三套**并存：整数分 `int64`（`payment.go:53,82,84`、`membership.go:48,63`）、decimal-as-string（`mall.go:255,281,308,346` + DB `numeric(10,2)`）、`float64`（`transaction.go:16,28,36`、`transaction_record.go:14`、`used_car_order.go:138,180`、`auto_finance.go:52,60-70`）。`transactions.amount` 在 DB 侧是**裸 `numeric`（无精度）**，全链路零 `Round(_,2)` | 口径盘点 + 跨界换算点清单；**`transactions` 一条链路落地整数分**（JSON 单位不变） | H6 · H6b |
| 可上线性 | `configs/config.prod.yaml` 存在，但 `APP_ENV=production` 找不到同名文件时**静默回退 `config.yaml`（`mode: debug`、`sslmode: disable`）**（`common/config/config.go:844-852`）；`/metrics` 无鉴权且 `lan-up` 映射到局域网（`router.go:65-66`） | 环境名不再静默回退；`/metrics` 只在本机口，Prometheus 抓取不受影响 | H7 |
| P0 残留 | F2：`membership_order` 无 `RegisterSettler`（`payment/settle.go:103-116` 走 `settled_no_settler`），渠道买会员要靠客户端回跳 `ConfirmBuyout` 才发权益 | 回调核销即主动发权益，双链幂等 | H8 |

## Slice backlog（有序）

| # | ID | ONLY 摘要 | 依赖 | Brief 已落盘 |
|---|-----|-----------|------|--------------|
| 1 | `harden-h1-repo-hygiene` | import 覆盖机检 + 二进制杂物 ignore + 干净 worktree 复检 | 无 | ☐ 待批 |
| 2 | `harden-h2-dependency-pinning` | gorm/eino/go 指令版本口径落地 + 两种库形态启动冒烟 | 无 | ☐ 待批 |
| 3 | `harden-h3-ci-and-test-entrypoints` | `make test-integration/lint/vet` + CI（含 PG integration job）+ harness 机检 | h1,h2 | ☐ 待批 |
| 4 | `harden-h4-push-route-authz` | **两处**推送入口不再信任客户端 user_id + 负例测试 | h3 | ☐ 待批 |
| 5 | `harden-h5-silent-error-sweep` | 5 处吞错留痕/指标/升级 + 清单落盘 | h3 | ☐ 待批 |
| 6 | `harden-h6-money-unit-audit` | 三套口径影响面 + 换算点 + 迁移方案（只读） | **无**（放宽：不依赖 CI） | ☐ 已批，2026-11-06 派发 |
| 6b | `harden-h6b-transaction-money-unit` | **落地** `transactions` → 整数分（含回填不丢值硬门槛） | h6 | ☐ 已批范围，待派发 |
| 7 | `harden-h7-prod-bootability` | 环境名不再静默回退 + `/metrics` 收口 | h3 | ☐ 待批 |
| 8 | `harden-h8-membership-settler` | `RegisterSettler(membership_order)` + 双链幂等真库用例 | h3 | ☐ 待批 |

> 并行（`3f0963e` 之后）：h1、h2 可同时起步；h3 完成后 h4/h5/h6/h7/h8 五条**可同批并行**（文件锁互不重叠）；h6b 串行在 h6 之后，可与 h7/h8 并行。

## 文件锁（并行派发前必读）

| Slice | 主写 | 勿抢 |
|-------|------|------|
| H1 | `.gitignore`、`scripts/`（新增机检） | **不改任何业务逻辑**；不碰 `go.mod`；不删除 `app/worker`（只 ignore，删除请人处置） |
| H2 | `app/go.mod`、`core/go.mod`、对应 `go.sum` | h2 期间其它 Slice 不许加依赖 |
| H3 | `Makefile`、`.github/workflows/`、`scripts/` | 不改业务代码；不改测试断言语义 |
| H4 | `app/internal/delivery/http/controller/jpush_controller.go`、`realtime_controller.go`、其测试 | 勿改 SessionAuth 全局语义 |
| H5 | `delivery/worker/handler.go`、`usecase/{realtime_push,profile,im_group}_usecase.go`、`access/service.go`、`common/metrics/` | 勿动 `payment/`（那是 H8） |
| H6 | 只读 + `docs/`（方案文档） | **禁止改实体与迁移**（落地是 H6b） |
| H6b | `app/internal/transaction/`、`model/entity/transaction.go`、`migrations/` | 勿动 `repo_supabase.go`/`repo_es.go`/DTO；勿删旧列；勿碰其它金额域 |
| H7 | `configs/`、`docker/docker-compose*.yml`、`router/router.go`（仅 `/metrics` 段）、`common/config/config.go`（仅环境名解析） | 勿顺手做鉴权重构 |
| H8 | `app/internal/membership/**`（含 `NewService` 内自注册） | **零 `cmd/api/main.go` 改动**；勿改 `payment` notify 验签逻辑 |
| 全部 | `docs/placeholder-inventory.md` | **追加式**登记，勿整文件重写 |

## 决策门（需人批，编排不得自决）

1. H1 的 `git add`/commit 范围与执行时机。
2. H1 杂物归属：`NOTES.md`、`learning-records/`、`reference/`、`assets/`、`data/`（`/teach` 产物）→ 入库 / ignore / 移出仓库。
3. ~~H2 方向~~ → **已批（2026-11-06）：接受 `1.31.2`，回写 `app/go.mod`**。
4. ~~H6 是否落地~~ → **已批：方案 + 落地 `transactions` 一条链路**（新增 H6b）。

**唯一未决门**：H3 的 CI 落点（GitHub Actions vs 本地 `make` + githooks）、integration 是否设为阻塞门禁。
