# Program Plan — 2026-11-05 后端工程地基加固（backend-hardening）

> 输入：2026-11-04 问题盘点（本文件 §背景），非 `xhs-parity` 功能补齐，而是**把已交付的 P0 变成可持续交付**
> 编排合同：[`.harness/rules/`](../.harness/rules/)（slice-contract · status-enums · no-silent-accept · dual-truth · module-boundary · auth-token-hygiene）
> **状态：待人批 Brief。批后按 §执行顺序 派发。**

## 背景（取证结论，全部有 file:line）

| # | 事实 | 证据 |
|---|------|------|
| 1 | 两个**已装配进主程序**的包完全不在 git 里：`app/internal/rag/`（零测试）、`core/authz/`（casbin），另有 `app/internal/{txdb,schemaboot}`、`model/entity/payment.go`、`migrations/*.sql` 7 个、`plans/`、`docs/` 全部未跟踪 | `git status --short` 未跟踪 73 项；`app/cmd/api/main.go:54,61,655-714`、`app/internal/access/service.go:19` |
| 2 | 依赖漂移违反 AGENTS.md ✱：`core/go.mod` gorm **1.25.12→1.31.2**、`go 1.22→1.25`、新增 casbin；`app/go.mod` eino **0.6.0→0.9.21**、`go 1.24.6` | `git diff core/go.mod app/go.mod` |
| 3 | 5 条真库集成用例**没有任何执行入口**：`grep -rn 'tags=integration' Makefile scripts/ .github` → 0 命中；无 CI、无 lint、无 `make vet` | `Makefile:54-60`（`test` 只跑 `go test ./...`） |
| 4 | 越权：`POST /api/v1/push/send` 信任 `body.UserID` 覆盖会话 userID，无 admin 校验 | `app/internal/delivery/http/controller/jpush_controller.go:91-93` |
| 5 | 静默吞错仍有 5 处，最危险的是「删去重键失败 → 该用户该时段推送永久跳过」 | `delivery/worker/handler.go:216`、`usecase/realtime_push_usecase.go:168`、`access/service.go:102`、`usecase/profile_usecase.go:82`、`usecase/im_group_usecase.go:163` |
| 6 | 金额两套口径：payment/wallet/mall 是整数分，`transaction`/`used_car`/`auto_finance` 实体仍是 `float64`，且 `used_car` handler 直收客户端 float | `model/entity/transaction.go:16,28,36`、`used_car_order.go:138,180`、`auto_finance.go:52-70`、`app/internal/used_car/handler.go:94` |
| 7 | `APP_ENV=production` 起不来（无 `configs/config.production.yaml`）；`/metrics` 无鉴权且 `lan-up` 把它映射到局域网 | `ls configs/`、`app/internal/delivery/http/router/router.go:65-66` |
| 8 | harness 机检 `agent-pre/agent-post/agent-check-boundary` 在 Makefile 里 0 命中（program R4 未解） | `grep Makefile` |
| 9 | P0-2 留的 7 条残留：F1 Flutter 未接 `status=pending`（渠道支付在 App 上等于坏）、F2 membership 无 `RegisterSettler`、F3 对账缺 `local_missing`、F4 商城 IAP 假成功、F5 `unifiedorder` 响应未验签、F6 真实 T+1 未实弹、F7 核销后不通知 | `docs/acceptance-records/2026-11-04-parity-p0-2-payment-notify-ledger.md` §七 |

## Milestone

| ID | Name | Goal | Out of scope |
|----|------|------|--------------|
| H1 | 仓库真实性（h1） | `git ls-files` = 能编译运行的代码；杂物不进库；一次干净检出 = `go build all` 通过 | 不改任何业务逻辑 |
| H2 | 依赖与门禁（h2–h3） | 依赖版本口径明确；CI + `make test-integration/lint/vet` 真跑，能挡回归 | 不做发布流水线、不做多环境镜像 |
| H3 | 安全与留痕（h4–h5） | 越权路径清零；`_, _ =` 静默失败变「留痕 + 指标 + 可重试」 | 不引入 outbox（P2-8） |
| H4 | 口径与可上线（h6–h7） | 金额单位有单一口径与迁移方案；`APP_ENV=production` 真能起、`/metrics` 不外泄 | 不在本轮做完 float→分 全量迁移 |
| H5 | P0 残留收口（h8） | 回调能主动发会员权益；Flutter 侧契约有明确跨仓 Brief | F3 渠道账单下载（另立 P2） |

## Slice 队列

| 顺序 | Slice | 一句话 | 依赖 | 可并行 |
|------|-------|--------|------|--------|
| 1 | [`harden-h1-repo-hygiene`](./slices/harden-h1-repo-hygiene.md) | 未跟踪的已装配包入库 + 杂物 ignore + 干净检出可编译证明 | — | 否（一切的前置） |
| 2 | [`harden-h2-dependency-pinning`](./slices/harden-h2-dependency-pinning.md) | gorm/eino/go 指令版本口径决策 + 全量验证 | h1 | 否（改 go.mod，需独占） |
| 3 | [`harden-h3-ci-and-test-entrypoints`](./slices/harden-h3-ci-and-test-entrypoints.md) | `make test-integration/lint/vet` + CI 门禁 + harness 机检落地 | h1,h2 | 否（CI 会锁依赖选择） |
| 4 | [`harden-h4-push-route-authz`](./slices/harden-h4-push-route-authz.md) | 推送路由不再信任客户端 user_id | h3 | 与 h5/h6/h7 并行 |
| 5 | [`harden-h5-silent-error-sweep`](./slices/harden-h5-silent-error-sweep.md) | 5 处 `_, _ =` 留痕 + 指标 | h3 | 与 h4/h6/h7 并行 |
| 6 | [`harden-h6-money-unit-audit`](./slices/harden-h6-money-unit-audit.md) | 金额 float→分 影响面与迁移方案（**只读 + 方案**） | h3 | 与 h4/h5/h7 并行 |
| 7 | [`harden-h7-prod-bootability`](./slices/harden-h7-prod-bootability.md) | `config.production.yaml` + `/metrics` 收口 | h3 | 与 h4/h5/h6 并行 |
| 8 | [`harden-h8-membership-settler`](./slices/harden-h8-membership-settler.md) | F2：`RegisterSettler(membership_order)`，回调主动发权益 | h1,h3 | 否（动 `cmd/api/main.go`） |

## 冲突矩阵（并行派发前必读）

| 热文件 | 涉及 Slice | 结论 |
|--------|-----------|------|
| `app/cmd/api/main.go` | h1（只 add）、h8（装配 RegisterSettler） | **h8 必须晚于 h1 提交**；本轮只有 h8 改它 |
| `go.mod` / `go.sum`（core, app） | h2 独占 | h2 期间不许其它 Slice 加依赖 |
| `Makefile` | h3（新增 target）、h1（只读） | h3 独占 |
| `.github/workflows/` | h3 独占 | — |
| `app/internal/delivery/http/router/router.go` | h7（`/metrics` 收口） | 只有 h7 |
| `app/internal/delivery/worker/handler.go` | h5 | 只有 h5 |
| `app/internal/{payment,membership}` | h8 | 只有 h8 |
| `docs/placeholder-inventory.md` | h4/h5/h6/h7/h8 各自登记条目 | **追加式改动**，按顺序合，勿整文件重写 |

## 决策门（需人批，编排不得自决）

1. **h1 的 `git add` 与提交**：规则是不自动 add/commit。h1 只产出清单 + `.gitignore` 改动 + 待批命令；人批准后执行。
2. **h1 杂物归属**：`NOTES.md`、`learning-records/`、`reference/`、`assets/`、`data/` 是 `/teach` 学习工作区产物 → 入库 or ignore or 移出仓库？三选一。
3. **h2 方向**：gorm 回 pin `1.25.12`（保守，本轮资金链路已在 1.25 语义下验证过索引命名行为）vs 正式接受 `1.31.2`（需全量回归 + 单列一次提交说明）。
4. **h6 是否落地**：float→分 涉及存量数据迁移与 Flutter 契约，默认只做方案，落地另批。

## 风险与对策

| ID | 风险 | 影响 | 对策 |
|----|------|------|------|
| R1 | 任何人一次 `git commit -am`（不含 `-u`）就会把 `main.go` 里对 `rag`/`authz` 的 import 提交、包本体不在库 → **HEAD 立刻编不过** | 灾难性，且当前 159 个改动都在这个窗口里 | h1 第一优先；h1 完成前**禁止任何提交** |
| R2 | h2 若选「接受 1.31.2」，GORM AutoMigrate 的索引/约束命名与 `uniqueIndex` 行为差异可能悄悄改变 `payment.EnsureSchema` | 已迁移库启动即报错（本轮已踩过 `uni_*/uk_*` 一次） | h2 验收强制：全新迁移库 + 已迁移库两种形态都跑启动冒烟 |
| R3 | h3 的 CI 一旦把 `go test ./...` 设为门禁，`-tags=integration` 默认不跑 → 真库用例继续裸奔 | P0-1/2/3 的回归不可见 | h3 必须在 CI 里显式起 PG service 并跑 integration job |
| R4 | h5 把静默失败改成「记 error + 指标」时，可能让原本 best-effort 的路径变成硬失败（消息重投、请求 500） | 行为变更超出加固范围 | h5 明确：默认**降级留痕**，只有会丢数的路径才升级为失败；每条改动列出「改前/改后可观测差异」 |
| R5 | h8 让回调直接发会员权益后，与客户端回跳的 `ConfirmBuyout` 形成两条发放路径 | 双发权益 | 复用现有 `ExtendEntitlement` 幂等语义 + 加并发真库用例（同 order 两条链） |

## 出口条件（Program Done when）

- `git ls-files` 覆盖所有被 import 的包；干净 worktree 检出 `go build all` 通过（人证 + 命令）
- CI 至少一次真绿，且**故意制造一次失败并观察到被拦**（lint、单测、integration 各一次）
- `git grep -E '_, _ = ' -- app/internal | grep -v _test` → 0（或逐条豁免登记）
- 越权：`POST /api/v1/push/send` 负例测试（跨 userID 被拒）落地
- `APP_ENV=production` 起得来；`/metrics` 从局域网口不可访问（人证）
- F2 从 `Degraded` 变 `Migrated`，或给出明确 `Deferred` + 理由
- 8 份 acceptance-record 落盘，状态词只用 `Migrated/Degraded/Placeholder/Deferred/Partial`
