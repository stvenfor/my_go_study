# Program Plan — 2026-11-05 后端工程地基加固（backend-hardening）

> 输入：2026-11-04 问题盘点（本文件 §背景），非 `xhs-parity` 功能补齐，而是**把已交付的 P0 变成可持续交付**
> 编排合同：[`.harness/rules/`](../.harness/rules/)（slice-contract · status-enums · no-silent-accept · dual-truth · module-boundary · auth-token-hygiene）
> **状态：待人批 Brief。批后按 §执行顺序 派发。**

## 背景（取证结论，全部有 file:line）

| # | 事实 | 证据 |
|---|------|------|
| 1 | ~~两个**已装配进主程序**的包完全不在 git 里~~ **已解除（2026-11-05 提交 `3f0963e`）**：`rag`/`authz`/`txdb`/`schemaboot`/`model/entity/payment.go`/`migrations`/`plans`/`docs` 均已 tracked；`go build all` 在 HEAD 通过。剩余杂物只有一项：`app/worker`（39MB 二进制，未被 ignore） | `git ls-files` 命中；`git status --short` 未跟踪 73→6；`go build all` 6.6s 绿 |
| 2 | 依赖漂移（已随 `3f0963e` 进入 HEAD）：`core` gorm **v1.31.2**、`go 1.25.0`、新增 casbin/gorm-adapter；`app` 声明 gorm **v1.25.12** 却**实际编到 v1.31.2**（workspace MVS 覆盖，声明与构建不一致）；`go` 指令三套（`core` 1.25.0 / `app` 1.24.6 / `common`+`model` 1.22.0）—— 违反 AGENTS.md ✱ | `core/go.mod:3,6,7,15`、`app/go.mod:3,6,20`、`go -C app list -m gorm.io/gorm` → `v1.31.2`（而 `app/go.mod:20` 写 `v1.25.12`） |
| 3 | 5 条真库集成用例**没有任何执行入口**：`grep -rn 'tags=integration' Makefile scripts/ .github` → 0 命中；无 CI、无 lint、无 `make vet` | `Makefile:54-60`（`test` 只跑 `go test ./...`） |
| 4 | 越权：`POST /api/v1/push/send` 与 `POST /api/v1/realtime/push` 都信任 body 里的 `user_id` 覆盖会话 userID，无权限校验（同一形状两处） | `app/internal/delivery/http/controller/jpush_controller.go:91-93`、`app/internal/delivery/http/controller/realtime_controller.go:109-112` |
| 5 | 静默吞错实测 5 处（4 处 `_, _ =` + 1 处 `_ = err`），最危险的是「删去重键失败 → 该用户该时段推送永久跳过」 | `delivery/worker/handler.go:216`、`usecase/realtime_push_usecase.go:168`、`access/service.go:102`、`usecase/profile_usecase.go:82`、`usecase/im_group_usecase.go:163`（`_ = err`） |
| 6 | 金额口径是**三套**（更正：不是两套）：①整数分 `int64`（`payment.go:53,82,84`、`membership.go:48,63`）②decimal-as-string `Amount string` + `numeric(10,2)`（`mall.go:255,281,308,346`）③`float64`（`transaction.go:16,28,36`、`transaction_record.go:14`、`used_car_order.go:138,180`、`auto_finance.go:52,60-70`）。且 `transactions.amount` / `transaction_records.amount` 在 DB 侧是**裸 `numeric`（无精度无标度）**，能收 float 累加残值；全链路**没有任何一处 `Round(_,2)`** | `migrations/20260923167500_align_gorm_only_schema.up.sql:35-58`、`app/internal/delivery/http/dto/request/transaction_request.go:8,17` → `handler.go:171,232` → `repo_postgres.go:90,124-125` / `repo_supabase.go:137,174` / `repo_es.go:93`、`app/internal/used_car/handler.go:94` |
| 7 | 环境名口径不一致：`configs/config.prod.yaml` **存在且正确**，但 `APP_ENV=production` 找不到 `config.production.yaml` 时只 `log.Printf` 就**静默回退基础 `config.yaml`（`mode: debug`、`sslmode: disable`）**；代码侧 `IsLabAppEnv`/`NoDDLAppEnv` 却同时认 `prod` 和 `production`。`/metrics` 无鉴权且 `lan-up` 把它映射到局域网 | `common/config/config.go:844-852`、`configs/config.yaml:5,22`、`common/config/config.go:342-361`、`app/internal/delivery/http/router/router.go:65-66` |
| 8 | harness 机检 `agent-pre/agent-post/agent-check-boundary` 在 Makefile 里 0 命中（program R4 未解） | `grep Makefile` |
| 9 | P0-2 留的 7 条残留：F1 Flutter 未接 `status=pending`（渠道支付在 App 上等于坏）、F2 membership 无 `RegisterSettler`、F3 对账缺 `local_missing`、F4 商城 IAP 假成功、F5 `unifiedorder` 响应未验签、F6 真实 T+1 未实弹、F7 核销后不通知 | `docs/acceptance-records/2026-11-04-parity-p0-2-payment-notify-ledger.md` §七 |
| 10 | 客服 RAG 管线 **5 个文件零测试**（chunker / embedding 适配 / ES 向量存储 / indexer / retriever），而它已挂进 `/api/v1/cs-agent/chat` 的检索路径 | `ls app/internal/rag/`（无 `_test.go`）、`app/internal/usecase/cs_agent_usecase.go` |

> 第 10 条**未排入本 Program 的任何 Slice**：它属于「功能正确性」而非「地基」，建议并入 `xhs-parity` P1 观测/测试线或另立 `harden-h9-rag-tests`。列在这里是为了不被静默丢弃（`.harness/rules/no-silent-accept.md`）。

## Milestone

| ID | Name | Goal | Out of scope |
|----|------|------|--------------|
| H1 | 仓库真实性（h1） | `git ls-files` = 能编译运行的代码；杂物不进库；一次干净检出 = `go build all` 通过 | 不改任何业务逻辑 |
| H2 | 依赖与门禁（h2–h3） | 依赖版本口径明确；CI + `make test-integration/lint/vet` 真跑，能挡回归 | 不做发布流水线、不做多环境镜像 |
| H3 | 安全与留痕（h4–h5） | 越权路径清零；`_, _ =` 静默失败变「留痕 + 指标 + 可重试」 | 不引入 outbox（P2-8） |
| H4 | 口径与可上线（h6–h6b–h7） | 金额口径盘点 + **`transactions` 一条链路落地到整数分**；环境名不再静默回退、`/metrics` 不外泄 | 不做 `used_car`/`auto_finance`/`mall` 的口径迁移；不改对外 JSON 单位 |
| H5 | P0 残留收口（h8） | 回调能主动发会员权益；Flutter 侧契约有明确跨仓 Brief | F3 渠道账单下载（另立 P2） |

## Slice 队列

| 顺序 | Slice | 一句话 | 依赖 | 可并行 |
|------|-------|--------|------|--------|
| 1 | [`harden-h1-repo-hygiene`](./slices/harden-h1-repo-hygiene.md) | import 覆盖机检 + 二进制杂物 ignore + 干净 worktree 复证（原「抢救入库」前提已被 `3f0963e` 解除，范围已收窄） | — | 是（不再是一切的前置） |
| 2 | [`harden-h2-dependency-pinning`](./slices/harden-h2-dependency-pinning.md) | gorm/eino/go 指令版本口径决策 + 全量验证（注意：漂移已随 `3f0963e` **进入 HEAD**，不再是待提交 diff） | 无（h1 不再是前置） | 否（改 go.mod，需独占） |
| 3 | [`harden-h3-ci-and-test-entrypoints`](./slices/harden-h3-ci-and-test-entrypoints.md) | `make test-integration/lint/vet` + CI 门禁 + harness 机检落地 | h1,h2 | 否（CI 会锁依赖选择） |
| 4 | [`harden-h4-push-route-authz`](./slices/harden-h4-push-route-authz.md) | **两处**推送入口不再信任客户端 `user_id`（`jpush_controller.go:91-93` + `realtime_controller.go:109-112`） | h3 | 与 h5/h6/h7/h8 并行 |
| 5 | [`harden-h5-silent-error-sweep`](./slices/harden-h5-silent-error-sweep.md) | 5 处吞错（4×`_, _ =` + 1×`_ = err`）留痕 + 指标 | h3 | 与 h4/h6/h7/h8 并行 |
| 6 | [`harden-h6-money-unit-audit`](./slices/harden-h6-money-unit-audit.md) | 金额**三套口径**影响面 + 跨界换算点 + 迁移方案（**只读 + 方案**） | **无**（2026-11-06 放宽：只读审计不依赖 CI，验证命令只有 grep/SELECT） | 与 h1/h2 同批起步 |
| 6b | [`harden-h6b-transaction-money-unit`](./slices/harden-h6b-transaction-money-unit.md) | **落地**：`transactions` 存储收敛到整数分 + 回填不丢值硬门槛（人已批：本轮只迁 transaction 一条链路，JSON 契约不变） | h6 | 与 h7/h8 并行 |
| 7 | [`harden-h7-prod-bootability`](./slices/harden-h7-prod-bootability.md) | 环境名不再静默回退 debug 配置 + `/metrics` 收口 | h3 | 与 h4/h5/h6/h8 并行 |
| 8 | [`harden-h8-membership-settler`](./slices/harden-h8-membership-settler.md) | F2：`RegisterSettler(membership_order)`，回调主动发权益 | h3 | **是**（取证更正：`main.go:246` 已注入 `paymentSvc`，注册可在 `membership.NewService` 包内完成，**零 main.go 改动**，与 h4/h5/h7 可并行） |

## 冲突矩阵（并行派发前必读）

| 热文件 | 涉及 Slice | 结论 |
|--------|-----------|------|
| `app/cmd/api/main.go` | **本轮无人需要改**（h8 取证更正：注册点在 `membership.NewService` 包内） | 任何 Slice 若声称「必须改 main.go」→ 停下问人 |
| `go.mod` / `go.sum`（core, app） | h2 独占 | h2 期间不许其它 Slice 加依赖 |
| `Makefile` | h3（新增 target）、h1（只读） | h3 独占 |
| `.github/workflows/` | h3 独占 | — |
| `app/internal/delivery/http/router/router.go` | h7（`/metrics` 收口） | 只有 h7 |
| `app/internal/delivery/worker/handler.go` | h5 | 只有 h5 |
| `app/internal/{payment,membership}` | h8 | 只有 h8 |
| `app/internal/transaction/**`、`model/entity/transaction.go`、`migrations/` | h6b 独占（h6 只读） | 只有 h6b 写；h6 若发现要改代码 = 越界 |
| `docs/placeholder-inventory.md` | h4/h5/h6/h6b/h7/h8 各自登记条目 | **追加式改动**，按顺序合，勿整文件重写 |

## 决策门（需人批，编排不得自决）

1. ~~h1 的 `git add` 与提交~~ → 已由人完成（`3f0963e` 一并入库代码 + `plans/` + `docs/`）；h1 剩余决策只有一条：**`app/worker`（39MB 未 ignore 的二进制）只忽略还是删除**。
2. **h1 杂物归属（已收敛）**：`NOTES.md`、`learning-records/`、`reference/`、`assets/`、`data/` 实测**已随 `66ece97` 入库**（tracked 11 项）→ 只剩「保持在库（默认）」vs「`git rm --cached` 移出」。
3. ~~h2 方向~~ → **已批（2026-11-06）：接受 gorm `1.31.2`，把 `app/go.mod:20` 从 `v1.25.12` 回写为 `v1.31.2`**（这是「把既成事实写进声明」，HEAD 本来就在编 1.31.2）。仍要求 h2 用两种库形态把 R2 证伪或坐实。
4. ~~h6 是否落地~~ → **已批：方案 + 直接落地 `transactions` 一条链路**，拆成 h6（只读方案）→ h6b（落地）。边界：对外 JSON 单位不变、Supabase 与 ES 路径本轮不动、旧列不删。

剩余**唯一未决门**：h3 的 CI 落点（GitHub Actions vs 本地 `make` + githooks）、以及 integration 是否设为阻塞门禁。

## 风险与对策

| ID | 风险 | 影响 | 对策 |
|----|------|------|------|
| R1 | ~~一次 `git commit -am` 就会提交 `main.go` 里对 `rag`/`authz` 的 import 而包本体不在库 → HEAD 编不过~~ **已解除**：`3f0963e` 把包与文档一并入库，HEAD `go build all` 实测通过（2026-11-05） | — | h1 只做机检复核，防复发（脚本 + 大文件规则） |
| R2 | h2 若选「接受 1.31.2」，GORM AutoMigrate 的索引/约束命名与 `uniqueIndex` 行为差异可能悄悄改变 `payment.EnsureSchema` | 已迁移库启动即报错（本轮已踩过 `uni_*/uk_*` 一次） | h2 验收强制：全新迁移库 + 已迁移库两种形态都跑启动冒烟 |
| R3 | h3 的 CI 一旦把 `go test ./...` 设为门禁，`-tags=integration` 默认不跑 → 真库用例继续裸奔 | P0-1/2/3 的回归不可见 | h3 必须在 CI 里显式起 PG service 并跑 integration job |
| R4 | h5 把静默失败改成「记 error + 指标」时，可能让原本 best-effort 的路径变成硬失败（消息重投、请求 500） | 行为变更超出加固范围 | h5 明确：默认**降级留痕**，只有会丢数的路径才升级为失败；每条改动列出「改前/改后可观测差异」 |
| R5 | h8 让回调直接发会员权益后，与客户端回跳的 `ConfirmBuyout` 形成两条发放路径 | 双发权益 | 复用现有 `ExtendEntitlement` 幂等语义 + 加并发真库用例（同 order 两条链） |

## 出口条件（Program Done when）

- `git ls-files` 覆盖所有被 import 的包 —— 由 `scripts/check-tracked-imports.sh` 机检输出「缺失 = 0」证明（原前提已由 `3f0963e` 达成，本条改为防复发）；干净 worktree 从 HEAD 检出 `go build all` 通过
- CI 至少一次真绿，且**故意制造一次失败并观察到被拦**（lint、单测、integration 各一次）
- `git grep -nE '_, _ = |^\s*_ = err' -- app/internal | grep -v _test` → 0（或逐条豁免登记）
- 越权：`POST /api/v1/push/send` 与 `POST /api/v1/realtime/push` 跨 userID 负例测试均被拒
- `APP_ENV=prod|production` 都能拿到 prod 覆盖（不再静默回退 debug 基础配置）；`/metrics` 从局域网口不可访问（人证），Prometheus 本机抓取不受影响
- F2 从 `Degraded` 变 `Migrated`，或给出明确 `Deferred` + 理由
- 9 份 acceptance-record 落盘（h1–h8 + h6b），状态词只用 `Migrated/Degraded/Placeholder/Deferred/Partial`
