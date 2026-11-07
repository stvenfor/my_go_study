# Program Plan — 2026-11-05 后端工程地基加固（backend-hardening）

> 输入：2026-11-04 问题盘点（本文件 §背景），非 `xhs-parity` 功能补齐，而是**把已交付的 P0 变成可持续交付**
> 编排合同：[`.harness/rules/`](../.harness/rules/)（slice-contract · status-enums · no-silent-accept · dual-truth · module-boundary · auth-token-hygiene）
> **状态：待人批 Brief。批后按 §执行顺序 派发。**

## 背景（取证结论，全部有 file:line）

| # | 事实 | 证据 |
|---|------|------|
| 1 | ~~两个**已装配进主程序**的包完全不在 git 里~~ 包本体已随 `3f0963e` 入库（`rag`/`authz`/`txdb`/`schemaboot`/`model/entity/payment.go`/`migrations`/`plans`/`docs` 均已 tracked）；**但「`go build all` 在 HEAD 通过」这条我于 2026-11-06 收回** —— 它只在「本机存在未跟踪 `go.work`」的前提下成立，见第 11 条与 R1。剩余杂物只有一项：`app/worker`（39MB 二进制，未被 ignore） | `git ls-files` 命中；`git status --short` 未跟踪 73→6；`go build all` 6.6s 绿 |
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

11. **【2026-11-06 新取证：R1 未解除的现行实例】** 纯跟踪文件构树编不过，两处根因都在 `.gitignore`：
    - `.gitignore:11` 裸 `logs/`（「# Logs」段）连带忽略源码包 `thunder/logs/logs.go`（4KB 单文件；被 `thunder/req/param.go:17`、`thunder/midd/cache.go:21`、`thunder/upload/aliyun_oss.go:21` import）
    - `.gitignore:33-34` 忽略 `go.work` / `go.work.sum` → 干净树没有 workspace 文件：`go build all` = `"all" matched no packages`，`go -C app build ./cmd/api` = `updates to go.mod needed`
    - 复现命令（coordinator 亲跑，非推断）：`git archive HEAD | tar -x -C <tmp>` + 手写 `go.work`（`use ./common ./core ./model ./app ./thunder`）→ `go build all` **exit 1**；同一命令在主工作区（有未跟踪 go.work）**exit 0**
    - 处置归属：`thunder/logs` → h1；**`go.work` 是否入库 = 人批决策门**（Go 官方对「库作者」建议忽略 go.work，但本仓是「一应用 + 4 模块 + 本地 replace thunder」，不入库则 CI/新克隆必挂）

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

> **2026-11-06 人批三则（coordinator 逐条转述，原文见会话）**：
> - **G-new-1 `go.work` → 入库**：`.gitignore:33-34` 放行 `go.work`（`go.work.sum` 同口径处理并写明理由）。验收升级为：**纯跟踪文件构树、不手写 go.work** → `go build all` **exit 0**。归 **h1** 执行（改 `.gitignore` 属其白名单；`git add go.work` 由**人**执行）。
> - **G-new-2 商城 Mixed 支付口径 = 「积分抵扣部分现金」** → 当时定性为缺陷（**此定性后被 G-new-3 更正，见下**）。⚠ 缺一条拆分规则 → 新开 Slice **h9**，h6b 不吞这条；**h9 未批规则前不得动工**（此约束已于 G-new-3 解除）。
> - **G-new-3（2026-11-06 17:0x）h9 抵扣规则 = 口径②「兑换率折算」**：`price` 语义重写为「**抵扣后应付现金**」，`price_points` = 抵扣所需积分，二者按兑换率相加才是展示原价。
>   - **更正 G-new-2 的缺陷定性**：② 下 `mall/service.go:672-700` 的「收全额 `order.Amount` + 扣全额 `order.TotalPoints`」正是正确写法（`price` 已是扣完的应付数），**不是同一笔价款收两遍**。Brief 已据此重写并加硬约束：**支付/结算主路径（`service.go:320-390`、`:660-730`）本轮不许动**，改成「先算抵扣再收现金」= 二次打折。
>   - 真实缺陷重新落在「口径无定义处」：兑换率在代码中不存在、`docs/` 58 篇零商城文档、`CreateSKUInput`（`:97-132`）对店员录入零口径约束 → 店员把原价填进 `price` 的那一行才会双重计费，且那是**数据**问题。
>   - 存量实测（coordinator 只读 SQL）：Mixed SKU **1 行**（`sku_id=37`，`price=9.90`，`price_points=50`）、Mixed 订单 **0 行** → 影响面比原判定小得多；`sku 37` 是「已按②定价」还是「原价误填」只有商家知道，列为**人裁定项**，默认零迁移不动数据。
>   - coordinator 采纳的两条默认（人可推翻）：率 = **100 积分 / 1 元**（人原话「如 100 积分=1 元」是举例；此率下 1 积分 = 1 分，整除无余数）；率**进常量不进配置**。
> - **G3（原 h3 未决门）CI = GitHub Actions，且 integration 先 allow-fail**（`continue-on-error`），build/vet/lint/单测 + `check-tracked-imports.sh` 为阻塞项。


1. ~~h1 的 `git add` 与提交~~ → 已由人完成（`3f0963e` 一并入库代码 + `plans/` + `docs/`）；h1 剩余决策只有一条：**`app/worker`（39MB 未 ignore 的二进制）只忽略还是删除**。
2. **h1 杂物归属（已收敛）**：`NOTES.md`、`learning-records/`、`reference/`、`assets/`、`data/` 实测**已随 `66ece97` 入库**（tracked 11 项）→ 只剩「保持在库（默认）」vs「`git rm --cached` 移出」。
3. ~~h2 方向~~ → **已批（2026-11-06）：接受 gorm `1.31.2`，把 `app/go.mod:20` 从 `v1.25.12` 回写为 `v1.31.2`**（这是「把既成事实写进声明」，HEAD 本来就在编 1.31.2）。仍要求 h2 用两种库形态把 R2 证伪或坐实。
4. ~~h6 是否落地~~ → **已批：方案 + 直接落地 `transactions` 一条链路**，拆成 h6（只读方案）→ h6b（落地）。边界：对外 JSON 单位不变、Supabase 与 ES 路径本轮不动、旧列不删。

剩余**唯一未决门**：h3 的 CI 落点（GitHub Actions vs 本地 `make` + githooks）、以及 integration 是否设为阻塞门禁。

## 风险与对策

| ID | 风险 | 影响 | 对策 |
|----|------|------|------|
| R1 | ~~import 的包本体不在库 → HEAD 编不过~~ → **2026-11-06 收回「已解除」：这一类缺陷未解除，只是触发点从 import 换到「被 ignore 的源码目录 + workspace 文件本身」**。实证：`git archive HEAD \| tar -x` 的纯跟踪树里补上 go.work 后 `go build all` → `no required module provides package github.com/mszlu521/thunder/logs`；`git ls-files thunder/logs` = **0 条**；`git check-ignore -v thunder/logs` → `.gitignore:11` 的裸 `logs/`（本意忽略日志产物，实际匹配任意深度的**源码包** `thunder/logs`） | 新克隆 / CI / Orca worktree 从 HEAD 检出后**编不过**，本机却绿 —— 典型的「只在作者机器上成立」 | h1 范围扩两条：① 收窄 `logs/` 规则使 `thunder/logs/logs.go` 可跟踪（worker 改 `.gitignore`，`git add` 由人执行）② `check-tracked-imports.sh` 以 `thunder/logs` 为回归锚点 |
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
