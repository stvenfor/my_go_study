# Acceptance Record — harden-h2-dependency-pinning

- 日期：2026-11-06（Slice 文件日期 2026-11-05）
- Slice：`plans/slices/harden-h2-dependency-pinning.md`
- 人已批方向：**接受 gorm v1.31.2**，把 `app/go.mod` 的声明从 `v1.25.12` 回写为 `v1.31.2`（Brief 决策门 3，2026-11-06 人批；把既成事实写进声明，不回 pin）
- **现场更正（2026-11-06 开工前情报核对）**：崩溃 codex worker 遗留的 `go 1.24.6 → 1.25.0` **无人批准、且无技术必要，已回退到 `1.24.6`**。三条决策理由与取证见 §1。
- 主交付物：`app/go.mod` / `app/go.sum` 的 gorm 声明对齐 + 两种库形态启动取证
- 未提交：按合同 commit/push 由人执行；本轮零 `.go` 改动

## status: **Degraded**

note（不静默接受，逐条说清哪里没满）：

1. gorm 声明对齐、版本口径落盘、全量回归、`payment.EnsureSchema` 两种库形态冒烟**均已完成并通过**（§3/§4/§5）。
2. **`go` 指令「四模块统一口径」未达成**：按 §1 三条理由，`app` 保持 `1.24.6`，`core` / `go.work` / `thunder` 为 HEAD 既有的 `1.25.0`，`common` / `model` 为 `1.22.0`（白名单外）。统一到任一版本都要动白名单外文件或重走抬版本，**本轮不擅自执行**（§6 残留 1）。
3. `auto_migrate=true` 的 3 次启动各有 1 条**融云 IM 表** `uni_wys_im_store_group_group_id` 的 `SQLSTATE 42704`（`schemaboot.BestEffort` 只 Warn、不阻断启动）。故「启动日志零 42704」这条口径本轮**不成立**；它与 payment 旧 bug 同类（均为裸 `uniqueIndex` 命名不一致），是否在 gorm `v1.25.12` 下同样触发**未做对照实测**（见 §6.1）。修它要改 `.go`，按 Brief 属越界，已登记残留（§6 残留 2）。

**本轮决定（2026-11-06，coordinator 下达）**：残留 1 → 按 §1 三条理由**保持 `1.24.6`、不动 go 指令行**；残留 2 → **coordinator 决定：另开 Slice h10，待 Brief 落盘后派发**（本轮零 `.go` 改动、不落地）。

> **措辞更正（依据 `.harness/rules/no-silent-accept.md`）**：Run 的 mailbox 中本 worker **从未发出 `ask`**，也不存在任何关于 IM 索引决定的人批记录。此前版本把对话层问答写成「人已批（2026-11-06 追问）」属把未发生的批准写进交付文档，已全部撤除；残留 2 改按 coordinator 决定表述。未发生的批准宁可留空，不许代答。

---

## 1. `go` 指令决策：保持 `1.24.6`，不动 go 指令行

### 决策：选 (a) 保持 `1.24.6` —— 三条理由（2026-11-06 定）

1. **workspace 级下限已经是 1.25.0，模块再写 1.25.0 只是重复断言**。`go.work` 入库已拍板（授权转 h1，由 h1 改 `.gitignore:33` 放行）；`go.work` 自身声明 `go 1.25.0`，**workspace 级下限已经是 1.25.0**。`workspace 的 go 指令 ≥ 模块的 go 指令` 是合法组合，`app` 的 `1.24.6` 与之共存无冲突（下方取证表实测混版构建通过）。把 `app` 抬到 1.25.0 不改变任何构建事实。
2. **最小改动原则**。h2 的实质是「把 gorm 既成事实写进声明」，已由 `v1.25.12 → v1.31.2` 完成；再抬 go 指令会把「声明」和「工具链行为」两件事塞进同一个 commit，扩大 review 面且无收益。
3. **抬 toolchain 会放大不可复现的生成物风险，不能留给下一任猜**。`make proto`（`Makefile:17-25`）依赖本机 `protoc` + `protoc-gen-go` + `protoc-gen-go-grpc`（插件按 `@latest` 用 `go install` 装）；本机无 `buf`、无网时插件装不上，stubs **不能重生成**。本仓已存在生成器 / 声明漂移：`app/go.mod:19` 声明 `google.golang.org/protobuf v1.36.6`，而 `app/api/gen/go/analytics/v1/analytics.pb.go:3` 头部是 `protoc-gen-go v1.35.2`（同目录 `analytics_grpc.pb.go:3-4` 为 `protoc-gen-go-grpc v1.5.1` / `protoc v7.36.1`）。改 go 指令若牵动工具链，这类漂移无法在本机重生成验证 → 不主动扩大工具链面。

> **证据核对（不代答）**：现场情报给出的 `app/protoc-gen-go-version.txt`（写 `1.36.6`）在本 worktree **不存在**；stubs 中亦无 `1.36.10` 字样（全仓仅 `app/go.mod:19` 有 `1.36.6`）。理由 ③ 的**风险判断**成立，已改用本仓可核实的实际证据落到具体文件 / 行号；对不上的文件名与版本号未照抄。

### 取证（本机 macOS，`go version go1.26.4`，`GOTOOLCHAIN=auto`）

| 证据 | 命令 | 结果 |
|------|------|------|
| 依赖最低 Go 版本 | `go -C app list -m -json gorm.io/gorm`（及 eino） | `GoVersion 1.18` / `1.18` |
| 同上（core 侧） | `go -C core list -m -json casbin/v2`、`gorm-adapter/v3` | `GoVersion 1.13` / `1.14` |
| workspace 各模块 go 指令 | `go -C app list -m -f '{{.Path}} {{.GoVersion}}'` | `common 1.22.0` · `core 1.25.0` · `model 1.22.0` · `app 1.24.6` · `thunder 1.25.0` |
| 全工作区构建 | `go build all` | exit 0 |
| 只读构建完整性 | `go -C app build -mod=readonly ./...` | exit 0（go.mod/go.sum 无需 tidy 补齐） |
| core 构建（casbin 兼容） | `go -C core build ./...` | exit 0 |
| 全量单测 | `GOPROXY=... go -C app test ./... -count=1` | exit 0，30 个 `ok` 包，无 FAIL |

**结论：没有任何依赖要求 `go >= 1.25.0`（最高仅 1.18）；workspace 中 1.22.0 / 1.24.6 / 1.25.0 混版共存可正常构建、测试全绿。** 这既证伪了「toolchain 必须抬」，也直接支撑理由 ①：下限由 `go.work` 的 1.25.0 提供，模块保持 1.24.6 不改变构建事实。

### 为什么否掉崩溃 worker 的 `1.25.0` 理由
上一版草稿（本文件被覆写前）写的理由是：「`go.work` 与 `thunder/go.mod` 都是 1.25.0，workspace 的 go 指令是最低门槛，app 依赖链也只能往上对齐」。**这不构成硬要求**：

- `go.work` 的 `go 1.25.0` 决定的是 **workspace 所需工具链下限**；每个模块 `go.mod` 的 `go` 指令只是**该模块自身代码的最低语言版本**，二者不必相等，Go 也未要求 workspace 内各模块 go 指令一致。
- 反证就是上表：把 `app` 放回 `1.24.6` 后，混版 workspace 的 `go build all` / `-mod=readonly` / 全量单测全部通过。
- `core` / `go.work` / `thunder` 的 `1.25.0` 是 **HEAD 已跟踪内容**，不在本次未提交 diff 内；把它们当作「app 必须抬」的依据等于把既成事实反过来追认一次未批准的改动。

### 口径说明（对应 Brief 验收第 3 条「说明为什么选它」）
本轮口径 = **各模块保持 HEAD 既有 go 指令**：`app` 保持 `1.24.6`（§1 三条理由），`core`/`go.work`/`thunder` 保持 `1.25.0`（HEAD 既有、且属黑名单/白名单外的既有事实），`common`/`model` 保持 `1.22.0`（白名单外）。「四模块统一」需单独决策，本 Slice 不擅自执行 → 该验收项记 `Deferred`。

---

## 2. 改了什么（只动白名单内文件）

| 文件 | 改动 |
|------|------|
| `app/go.mod` | `gorm.io/gorm v1.25.12` → `v1.31.2`（Brief 决策门 3 人批，保留崩溃 worker 的既成事实）；`go` 指令按 §1 三条理由从 worker 留下的 `1.25.0` **回退为 HEAD 的 `1.24.6`**（回退后该行与 HEAD 相同，故 `git diff` 里只剩 gorm 一行） |
| `app/go.sum` | `gorm.io/gorm v1.25.12` 两行 → `v1.31.2` 两行（hash 与 `core/go.sum` 一致，保留） |
| `docs/acceptance-records/2026-11-05-harden-h2-dependency-pinning.md` | 本记录（重写：纠正 go 指令决策、换成实跑证据、改掉非法的 `Partial` 状态、撤除未发生的「人已批」措辞、补 §6.1 IM 证据包） |
| `plans/slices/harden-h2-dependency-pinning.md` | 仅填 Context Card（重写，去掉 `go 1.25.0` 的错误陈述与「人已批」措辞） |

`core/go.mod` / `core/go.sum` **无需改动**：本就显式写 `gorm.io/gorm v1.31.2`、`casbin v2.21.0`、`gorm-adapter v3.1.0`、`go 1.25.0`。
未动：任何 `.go`、`go.work`、`thunder/`、`.env*`、`.gitignore`（黑名单 / 非本片范围）。`git diff --name-only | grep '\.go$'` 为空。

> 环境说明（均非交付物）：`go.work` / `go.work.sum` 目前仍被 `.gitignore:33-34` 忽略、`thunder/logs/` 与 `bin/migrate` 未入库，是崩溃 worker 从主检出 `/Users/qpy/Desktop/github/my_go_study` 复制进本 worktree 的，目的是让 `go build all` 与迁移链可跑。本轮**未改动**它们，仅使用。`go.work` 入库授权已转 h1（改 `.gitignore:33` 放行），属 h1 范围。
> `mktemp -d` 建的临时目录 `/tmp/h2verify.*`、`/tmp/h2smoke.*` 按现场约定未清理；§6.1 引用的启动日志在 `/tmp/h2smoke.ezRsWU/`。

---

## 3. 验收逐条

| # | 验收项 | status | 证据 |
|---|--------|--------|------|
| 1 | 先取证再落地：声明 1.25.12 ↔ MVS 实际 1.31.2 的不一致消除 | **Migrated** | 改前 `go -C app list -m gorm.io/gorm` → `v1.31.2`（声明仍 `v1.25.12`）；改后 `app/go.mod` 写 `v1.31.2` 且 `list -m` 仍 `v1.31.2`；`go -C app list -m gorm.io/gorm` = `gorm.io/gorm v1.31.2` |
| 2 | 版本口径落盘：gorm/eino/casbin 显式写明 | **Migrated** | `go -C app list -m gorm.io/gorm github.com/cloudwego/eino` → `v1.31.2` / `v0.9.21`；`go -C core list -m gorm.io/gorm github.com/casbin/casbin/v2 github.com/casbin/gorm-adapter/v3` → `v1.31.2` / `v2.21.0` / `v3.1.0`，与各自 go.mod 声明一致（无「靠 tidy 隐式拉最新」口子） |
| 3 | `go` 指令口径统一 | **Deferred** | 见 §1 口径说明与三条理由：各模块保持 HEAD 值，`app` 保持 `1.24.6`；四模块统一未在白名单内、本轮不擅自执行（common/model 1.22.0、core 1.25.0 均未动） |
| 4 | 全量回归 | **Migrated** | `go build all` exit 0；`go -C app test ./... -count=1` exit 0（30 ok 包）；`go -C core test ./... -count=1` exit 0（jpush/queue/rongcloud ok，authz 等无测试文件）；`go -C app build -mod=readonly ./...` exit 0；集成五例见 §5 |
| 5 | 两种库形态启动冒烟（风险 R2） | **Migrated**（payment）· **Degraded**（全局 42704，见 §6.1） | §4：26 条迁移建库 + 二次 `no change` + 3 次启动、空库 AutoMigrate 从零建 + 2 次启动，`/health` 全 ok；`uk_wys_payment_records_out_trade_no` 两种形态均存在；**payment 的历史 `SQLSTATE 42704` 未复现**；各次启动 `Casbin Enforcer 已就绪` |
| 6 | 独立一次提交（只含 go.mod/go.sum） | **Deferred** | 按合同「不要 git add/commit/push（由人执行）」。当前待提交：`app/go.mod` + `app/go.sum` + 本记录 + Slice Context Card；建议 message 写明「本仓曾发生 tidy 隐式升级 gorm 1.25.12→1.31.2，本次把声明对齐既成事实；go 指令按 §1 三条理由保持 1.24.6」 |
| 7 | casbin + gorm-adapter 是否留在 core | **Migrated**（保留） | `go -C core build ./...` / `test ./...` 通过；§4 五次启动均打印 `Casbin Enforcer 已就绪`（gorm-adapter v3.1.0 在 gorm v1.31.2 下构造 / `LoadPolicy` 正常）→ 兼容性取证通过，按 Slice 默认**保留在 core**，不提移除建议 |

---

## 4. 两种库形态启动冒烟（原始证据，本轮实跑）

环境：`APP_ENV=dev`、`SERVER_PORT=8081`、`GRPC_ENABLED=false`、`QUEUE_ENABLED=false`、`AUTH_PROVIDER=local`；DB 为 scratch 库 **`p03_it`**（`P03_PG_DBNAME=p03_it`，脚本内对 `my_go_study`/`postgres`/`template*` 有拒绝防呆），**全程未连业务库 `my_go_study`**；`psql` 走 `docker exec my_go_study_postgres`；二进制 `go -C app build -o <tmp>/api ./cmd/api`。

### A) 全新空库 → `migrations/` 迁移链建起 → 多次启动

```text
[db] 重建空库: p03_it
[migrate:p03_it] 26 条迁移全部 up（末条 20261104160000/u payment_ledger）
[migrate:p03_it] 第二次 up → no change            # 已迁移库再跑：幂等
[idx:p03_it] idx_wys_payment_records_ref
[idx:p03_it] idx_wys_payment_records_status_created
[idx:p03_it] idx_wys_payment_records_user_created
[idx:p03_it] uk_wys_payment_records_out_trade_no   # 迁移建的是 uk_ 索引，非表级约束
[idx:p03_it] wys_payment_records_pkey

[boot:A1_migrated_ddl_off]  auto_migrate=false health_ok=1 {"status":"ok"} 42704_lines=0
  INFO schemaboot 启动期 DDL 已关闭：schema 由发布流程 migrations/ 管理
  INFO schemaboot schema 版本校验通过 {"version": 20261104160000, "migrations": 26}
  INFO api/main.go:234 Casbin Enforcer 已就绪

[boot:A2_migrated_ddl_on]   auto_migrate=true  health_ok=1 {"status":"ok"} 42704_lines=2（仅 IM，见下）
  INFO schemaboot 启动期 DDL 已启用（database.auto_migrate=true）  # EnsureSchema 在已迁移表上重跑
  INFO schemaboot schema 版本校验通过 {"version": 20261104160000, "migrations": 26}
  INFO api/main.go:234 Casbin Enforcer 已就绪
  → payment 的 uni_wys_payment_records_out_trade_no 未报错；uk_ 索引仍在

[boot:A3_restart]           auto_migrate=false health_ok=1 {"status":"ok"} 42704_lines=0
  INFO schemaboot schema 版本校验通过
  INFO api/main.go:234 Casbin Enforcer 已就绪
[idx:p03_it] uk_wys_payment_records_out_trade_no   # 重跑后索引未丢失/未改名
```

### B) 全新空库 → AutoMigrate(lab) 从零建起 → 再启动

```text
[db] 重建空库: p03_it
[boot:B1_automigrate_first]  auto_migrate=true health_ok=1 {"status":"ok"} 42704_lines=2（仅 IM）
  INFO schemaboot 启动期 DDL 已启用
  INFO api/main.go:234 Casbin Enforcer 已就绪
[idx:p03_it] uk_wys_payment_records_out_trade_no   # GORM AutoMigrate 建出的名字与迁移一致
[idx:p03_it] wys_payment_records_pkey

[boot:B2_automigrate_second] auto_migrate=true health_ok=1 {"status":"ok"} 42704_lines=2（仅 IM）
  INFO schemaboot 启动期 DDL 已启用
  INFO api/main.go:234 Casbin Enforcer 已就绪
[idx:p03_it] uk_wys_payment_records_out_trade_no   # 第二次启动仍不报 payment 42704
```

### R2 结论
- `payment.EnsureSchema` 的历史 `uni_wys_payment_records_out_trade_no` / `SQLSTATE 42704` **已不存在**：entity 的 `uniqueIndex:uk_wys_payment_records_out_trade_no`（`model/entity/payment.go:50`）与迁移的 `CREATE UNIQUE INDEX uk_wys_payment_records_out_trade_no`（`migrations/20261104160000_payment_ledger.up.sql`）同名；**全新迁移库、迁移后再启动、AutoMigrate 从零建、再启动** 四种情形都不触发。gorm v1.31.2 下未出现索引/约束命名行为回退。
- **casbin / gorm-adapter**：5 次启动全部 `Casbin Enforcer 已就绪`，`casbin_rule` 由迁移建表，v3.1.0 + gorm v1.31.2 组合可用（决策门「是否留在 core」→ 保留）。

---

## 5. 全量回归原始输出（本轮实跑）

```text
$ go build all                                   → exit 0（无输出）
$ go -C app build -mod=readonly ./...            → exit 0
$ go -C core build ./...                         → exit 0
$ go -C app list -m gorm.io/gorm github.com/cloudwego/eino
  gorm.io/gorm v1.31.2
  github.com/cloudwego/eino v0.9.21
$ go -C core list -m gorm.io/gorm github.com/casbin/casbin/v2 github.com/casbin/gorm-adapter/v3
  gorm.io/gorm v1.31.2 / v2.21.0 / v3.1.0
$ GOPROXY=https://goproxy.cn,direct go -C app test ./... -count=1   → exit 0，30 个 ok 包
$ GOPROXY=https://goproxy.cn,direct go -C core test ./... -count=1   → exit 0
```

集成五例（`-tags=integration`，`-p 1` 串行，scratch 库 `p02_it` / `p03_it`，绝不连 `my_go_study`）：

```text
$ DATABASE_HOST=127.0.0.1 DATABASE_PORT=5432 DATABASE_USER=postgres \
  DATABASE_PASSWORD=<.env> DATABASE_DBNAME=p03_it \
  P02_PG_DBNAME=p02_it P03_PG_DBNAME=p03_it GOPROXY=https://goproxy.cn,direct \
  go -C app test -p 1 -tags=integration ./internal/payment/ ./internal/mall/ ./internal/membership/ \
    -run 'MoneyTx|Notify|Tx|Reconcile' -count=1 -v        → IT_EXIT=0
--- PASS: TestReconcileWritesDiffsAndLateSettles / TestReconcileRerunDoesNotDuplicateDiffs
--- PASS: TestPostgresConcurrentNotifySettlesOnce / TestPostgresReconcileDiffUnique
--- PASS: TestWechat* / TestAlipay*（13 例）
ok  app/internal/payment    1.927s
--- PASS: TestChannelNotifySettlesMallOrder / TestChannelNotifyTamperedAmountKeepsUnpaid
--- PASS: TestMoneyTxRollbackOnOrderFailure / TestMoneyTxConcurrentPayDebtsOnce / TestMoneyTxCancelRefundsAtomically
ok  app/internal/mall      1.919s
--- PASS: TestMembershipBalanceTxRollsBackDebit / TestMembershipBalanceTxConcurrentDebtsOnce
ok  app/internal/membership 1.147s
```

**测试隔离坑（如实登记）**：首轮把三个包全指向 `p03_it` 且 `go test` 默认并行跑包时，`app/internal/membership` 两例 FAIL —— `relation "wys_cash_wallet" does not exist (42P01)` 与 `duplicate key ... pg_type_typname_nsp_index (23505)`，是 mall 的 `t.Cleanup` 删表与 membership 的 AutoMigrate 并发撞车（`| tail` 还吞了非零退出码）。加 `-p 1` 串行、并让 payment 走 `p02_it` 后全部 PASS。**属测试隔离问题，非 gorm 回归，无需改 `.go`。**

---

## 6. 未验证项 / 残留清单（诚实口径）

| # | 残留 | status | 说明 / 建议 |
|---|------|--------|-------------|
| 1 | `go` 指令未四模块统一（`app 1.24.6` / `core·go.work·thunder 1.25.0` / `common·model 1.22.0`） | **Deferred** | **本轮决定（§1 三条理由）：保持现状** —— 各模块维持 HEAD 既有值，`app` 不动 go 指令行，不追加改动、不重走版本审批 |
| 2 | 融云 IM 表 `constraint "uni_wys_im_store_group_group_id" does not exist (SQLSTATE 42704)`：A2 / B1 / B2 三次 `auto_migrate=true` 启动各 1 条，`schemaboot.BestEffort` 只 Warn、`/health` 不受影响 | **Degraded** + note | 根因、行号、三次原始报错整行、v1.25.12 对照结论见 **§6.1 证据包**；修法（entity 显式命名 `uk_` + 迁移改 `CREATE UNIQUE INDEX`）需改 `.go`，按 Brief 越界 → **coordinator 决定：另开 Slice h10，待 Brief 落盘后派发**；本轮零 `.go` 改动、不落地 |
| 3 | 首轮集成并行 FAIL 的复现/隔离 | **Deferred** | 已用 `-p 1` 规避；若要让三包共享一个库并行跑，需改测试 helper 的库隔离（`.go`），不在本 Slice 范围 |
| 4 | 历史 AutoMigrate 形态的旧库（P0-1 之前）未重放 | **Deferred** | 本轮「已迁移库」用迁移链建库 + 二次 up/重跑覆盖；真正的旧 AutoMigrate 库基线化属 P0-1 流程，不在本 Slice |
| 5 | `go mod tidy -diff` 未作验收依据 | **Deferred** | 上一版草稿提过 `sum.golang.org` 超时；本轮以 `-mod=readonly build ./...` + 全量测试通过证明 go.mod/go.sum 完整 |
| 6 | 独立提交未做 | **Deferred** | 按合同由人执行 |
| 7 | 本 worktree 依赖主检出的 `go.work`/`go.work.sum`/`thunder/logs`/`bin/migrate` 才能跑 `go build all` 与迁移 | **Deferred** | `go.work` 入库授权已转 h1；`thunder/logs`/`bin/migrate` 是否入库建议 h1 一并决定 |
| 8 | `p03_it` 现留有形态 B 的 AutoMigrate 表；崩溃 worker 的 `h2_a_fresh`/`h2_b_lab` scratch 库亦未清理 | **Deferred** | 按现场约定临时产物不清理；均为 scratch 库，`my_go_study` 未动。需要空库时重建即可 |

### 6.1 IM 42704 证据包（供 Slice h10 直接引用）

**根因：两份 schema 资产对同一约束叫不同名字（静态命名不一致）**

- entity 侧：`model/entity/im.go:67` —
  `GroupID string \`json:"group_id" gorm:"column:group_id;size:128;uniqueIndex;not null"\``
  裸 `uniqueIndex` 未显式命名 → GORM 默认生成 `uni_wys_im_store_group_group_id`。
  *同类第 2 处*：`model/entity/im.go:77` `MessageUID` 亦为裸 `uniqueIndex`，而 `migrations/20260924100000_rongcloud_im.up.sql:48` 用 `message_uid text NOT NULL UNIQUE` —— h10 需与 `wys_im_message_backup` 一并核对。
- 迁移侧：`migrations/20260924100000_rongcloud_im.up.sql:41` — `  group_id   text NOT NULL UNIQUE,`（`:39` 为 `CREATE TABLE IF NOT EXISTS wys_im_store_group (`）。该行**未写约束名**，PG 按默认规则自动命名为 `wys_im_store_group_group_id_key`。本机实测该命名规则（scratch 库 `p03_it` 临时表 `_h2_improbe`）：`SELECT conname FROM pg_constraint` → `_h2_improbe_group_id_key`，即 `<table>_<column>_key`。GORM AutoMigrate 要 DROP 的却是 `uni_wys_im_store_group_group_id` → `SQLSTATE 42704`。
- 已修好的同类对照：`model/entity/payment.go:50` 显式写 `uniqueIndex:uk_wys_payment_records_out_trade_no`，`migrations/20261104160000_payment_ledger.up.sql` 用 `CREATE UNIQUE INDEX ... uk_wys_payment_records_out_trade_no`，两份资产同名 → §4 四种形态均不触发 42704。

**三次启动原始报错整行**（去终端 ANSI 转义，制表符保留；日志目录 `/tmp/h2smoke.ezRsWU/`，按现场约定未清理）

```text
# A2  /tmp/h2smoke.ezRsWU/api_A2_migrated_ddl_on.log
5566:2026/11/06 16:29:36  ERROR: constraint "uni_wys_im_store_group_group_id" of relation "wys_im_store_group" does not exist (SQLSTATE 42704)
5568:2026-11-06T16:29:36.059+0800	WARN	schemaboot/boot.go:48	域内建表兜底失败，功能可能受限	{"step": "融云 IM 表", "error": "ERROR: constraint \"uni_wys_im_store_group_group_id\" of relation \"wys_im_store_group\" does not exist (SQLSTATE 42704)"}

# B1  /tmp/h2smoke.ezRsWU/api_B1_automigrate_first.log
3704:2026/11/06 16:29:39  ERROR: constraint "uni_wys_im_store_group_group_id" of relation "wys_im_store_group" does not exist (SQLSTATE 42704)
3706:2026-11-06T16:29:39.989+0800	WARN	schemaboot/boot.go:48	域内建表兜底失败，功能可能受限	{"step": "融云 IM 表", "error": "ERROR: constraint \"uni_wys_im_store_group_group_id\" of relation \"wys_im_store_group\" does not exist (SQLSTATE 42704)"}

# B2  /tmp/h2smoke.ezRsWU/api_B2_automigrate_second.log
2889:2026/11/06 16:29:41  ERROR: constraint "uni_wys_im_store_group_group_id" of relation "wys_im_store_group" does not exist (SQLSTATE 42704)
2891:2026-11-06T16:29:41.618+0800	WARN	schemaboot/boot.go:48	域内建表兜底失败，功能可能受限	{"step": "融云 IM 表", "error": "ERROR: constraint \"uni_wys_im_store_group_group_id\" of relation \"wys_im_store_group\" does not exist (SQLSTATE 42704)"}
```

每次 `auto_migrate=true` 启动命中 2 行 = 1 条 GORM `ERROR` + 1 条 `schemaboot/boot.go:48` `WARN`（`BestEffort` 语义，不阻断启动）。`A1`/`A3`（`auto_migrate=false`，DDL 关）各 `42704_lines=0`：报错只在 EnsureSchema 实际执行时出现。

**「为什么与 gorm 版本无关」—— 对照实测结论：未测**

- **未在 `gorm.io/gorm v1.25.12` 下做对照实测**。原因：本 workspace 的 MVS 由 `core/go.mod` 的 `v1.31.2` 决定，仅把 `app/go.mod` 声明改回 1.25.12 实际仍编 1.31.2；要真测需整仓降版本（改 core / go.work 等，超出 h2 范围且会破坏交付）。因此**不能坐实「与 gorm 版本无关」**；此前 status note / 残留表里的该绝对断言已收敛为「同类、均为裸 uniqueIndex 命名不一致」。
- 可确证的只有静态事实：报错是 entity tag 名与迁移约束名两份**静态资产**不一致触发的 `DROP CONSTRAINT ... does not exist`。h10 若需坐实版本无关，应在 1.25.12 环境下补一次对照启动（或证明 GORM 默认 `uni_<table>_<column>` 命名在 1.25.12 / 1.31.2 两版一致）。

---

## 7. 与上一版草稿的关系（避免误读）

本文件在崩溃 worker 手上已存在一版，其中：把 `go` 指令选成 `1.25.0` 并写了「workspace 最低门槛」理由（§1 已证伪并回退）；冒烟用的是 `h2_a_fresh`/`h2_b_lab` 两个 scratch 库、声称 `Partial`（`status-enums` 无此状态，非法）。本版**全部重写**：纠正 go 指令决策、改用 `p03_it` 并重跑全部命令取证、状态改为合法枚举 `Degraded` + note；并撤除把对话层问答误记为「人已批」的措辞。Slice 的 Context Card 同样重写。

## 8. 复现命令

```bash
cd /Users/qpy/orca/workspaces/my_go_study/harden-h2
go build all
go -C app build -mod=readonly ./...
go -C app list -m gorm.io/gorm github.com/cloudwego/eino
go -C core list -m gorm.io/gorm github.com/casbin/casbin/v2 github.com/casbin/gorm-adapter/v3
GOPROXY=https://goproxy.cn,direct go -C app test ./... -count=1
GOPROXY=https://goproxy.cn,direct go -C core test ./... -count=1

DATABASE_HOST=127.0.0.1 DATABASE_PORT=5432 DATABASE_USER=postgres \
DATABASE_PASSWORD=<.env:DATABASE_PASSWORD> DATABASE_DBNAME=p03_it \
P02_PG_DBNAME=p02_it P03_PG_DBNAME=p03_it GOPROXY=https://goproxy.cn,direct \
go -C app test -p 1 -tags=integration ./internal/payment/ ./internal/mall/ ./internal/membership/ \
  -run 'MoneyTx|Notify|Tx|Reconcile' -count=1 -v

# 两种库形态冒烟（脚本 /tmp/h2smoke.ezRsWU/smoke.sh，DB 固定 scratch p03_it）
P03_PG_DBNAME=p03_it /tmp/h2smoke.ezRsWU/smoke.sh /tmp/h2smoke.ezRsWU/api /tmp/h2smoke.ezRsWU <db_password>

# h10 复现 IM 42704 原始行（去 ANSI）
sed -E 's/\x1b\[[0-9;]*m//g' /tmp/h2smoke.ezRsWU/api_A2_migrated_ddl_on.log | grep -n '42704'
```
