# Slice — harden-h6b-transaction-money-unit

> 由 [`harden-h6-money-unit-audit`](./harden-h6-money-unit-audit.md) 派生。人已批「方案 + **直接落地 transaction 一条链路**」（2026-11-06）。

## 落地前必须知道的取证（别按「只是改个类型」的假设开工）

| 事实 | 证据 |
|------|------|
| `transactions.amount` 是**裸 `numeric`（无精度无标度）** —— 不是我原先说的 float-only。PG 会照收 `0.30000000000000004` 这种 float 累加残值 | `migrations/20260923167500_align_gorm_only_schema.up.sql:35-45`（`amount numeric NOT NULL`） |
| 遗留表 `transaction_records.amount` 同样是裸 `numeric`，且 `user_id bigint`（与 UUID 口径混用）、表名无 `wys_` 前缀 | 同文件 `:48-58`、`model/entity/transaction_record.go:21-24`（注释自陈「已弃用」） |
| 金额从客户端一路裸传到三种存储：`transaction_request.go:8,17` → `handler.go:171,232` → `repo_postgres.go:90,124-125` / `repo_supabase.go:137,174` / `repo_es.go:93` | 全链路**没有任何一处做 `Round(_,2)`** |
| Supabase/PostgREST 那条路写的是**云端表**，本仓 `migrations/` 管不到 | `repo_supabase.go:137,174`；`docs/supabase-integration.md` |

→ 所以本 Slice 的**真正目标不是「float 换成分」**，而是「让 `transactions` 的存储侧不再能容纳 >2 位小数的脏值，并且累加口径唯一」。

## Slice Brief
- SOURCE_MODULE: `model/entity/transaction.go`、`app/internal/transaction/`、`migrations/`
- TARGET_MODULE: 同上
- Source entry: 见上表
- Target entry: `transactions` 有整数分列 + `CHECK` 约束；写路径双写；读路径以分为准；**对外 JSON 契约不变**（仍 `amount` 元）
- 本轮 ONLY: **只迁 `transactions` 一条链路**（`transaction_records` 遗留表、`used_car`、`auto_finance`、`mall` 一律不动）
- 不做:
  - **不改对外 JSON 字段名与单位**（改成 `amount_fen` 会连带 Flutter `TransactionApi` 同步发版，超出本轮；本轮存储收敛，契约后移）
  - 不动 `repo_supabase.go`（云端表不受本仓迁移管辖）→ 但必须把它记成**已知不对称**，见验收第 5 条
  - 不改 ES mapping（`repo_es.go:93` 继续写元），避免 reindex + 客户端二次改动
  - 不删 `amount numeric` 旧列（删列属另一轮，且需先确无双写漂移）
- 验收:
  - [ ] 新迁移走 `migrations/`（**不得靠 AutoMigrate 建列** —— `parity-p0-1` 已把 `migrations/` 定为 schema 单一真相），含 `.down.sql`，编号不与既有文件冲突
  - [ ] `amount_fen bigint NOT NULL CHECK (amount_fen >= 0)`（负数口径由 `type` 表达；先核实现网是否已有负 `amount`，有则约束降级并在 record 说明为什么）
  - [ ] **回填不丢值的硬门槛**：回填前先跑 `SELECT count(*) FROM transactions WHERE abs(amount*100 - round(amount*100)) > 1e-9` → 必须 **0 行**才允许回填；非 0 就**停下来上报行数与样本**，不许 `trunc` 悄悄吃掉钱
  - [ ] 写路径双写（`repo_postgres.go:90,124-125`），读路径以 `amount_fen` 换算回元；配单测断言「0.1+0.2 记两笔后聚合结果等于 `0.30`，不是 `0.30000000000000004`」
  - [ ] **已知不对称要留痕**：`provider=supabase` 时 `amount_fen` 不存在 → 代码必须在本地路径生效、Supabase 路径行为不变，并在 `docs/placeholder-inventory.md` 追加一条（状态 `Degraded`，写明「双 provider 金额口径不一致」）
  - [ ] 真库证明（scratch 库，**绝不连业务库 `my_go_study`**）：`//go:build integration` 用例覆盖「建表 → 写入 → 读回 → 聚合」，并断言 `amount_fen` 与旧列换算一致
  - [ ] 两种库形态启动冒烟（同 h2 口径）：全新迁移链 + 已迁移库再启动，`APP_ENV` 走 `NoDDLAppEnv` 分支不报错
  - [ ] `go -C app test ./internal/transaction/ -count=1` 绿、`go build all` 绿、`make test-integration`（h3 产物）绿
  - [ ] 状态词：**Migrated 仅限本地 PG 路径**；整体写 `Partial` 并列出「Supabase 路径 / ES / 旧列删除 / used_car+auto_finance」四项残留
- 文件白名单:
  - `migrations/`（新增一对 up/down）
  - `model/entity/transaction.go`
  - `app/internal/transaction/repo_postgres.go`
  - `app/internal/transaction/service.go`
  - `app/internal/transaction/*_test.go`
  - `app/internal/schemaboot/`（仅当 schema 版本清单需要同步）
  - `docs/placeholder-inventory.md`（追加式）
  - `docs/acceptance-records/2026-11-06-harden-h6b-transaction-money-unit.md`
  - `plans/slices/harden-h6b-transaction-money-unit.md`（仅 Context Card）
- 文件黑名单:
  - `app/internal/transaction/repo_supabase.go`、`repo_es.go`（本轮刻意不动，动了要在 Brief 先改口径再执行）
  - `app/internal/delivery/http/dto/request/transaction_request.go`（JSON 契约不变）
  - `app/internal/{used_car,auto_finance,mall,wallet,points,payment,membership}/**`
  - `app/cmd/api/main.go`、`**/go.mod`、`**/go.sum`
  - `../../my_ai_project/**`（Flutter 改动走跨仓 Brief，不自行落地）
- 验证命令:
  - `make migrate-up` / `make migrate-status` / `make migrate-down`（`Makefile:86-104` 已 pin `migrate` v4.15.2；先在 scratch 库跑，历史 AutoMigrate 库需 `make migrate-baseline`）
  - `GOPROXY=https://goproxy.cn,direct go -C app test -tags=integration ./internal/transaction/ -run MoneyUnit -count=1`
  - `go -C app test ./internal/transaction/ -count=1`
  - `go build all`
- 证据: `docs/acceptance-records/2026-11-06-harden-h6b-transaction-money-unit.md`

## Context Card — harden-h6b-transaction-money-unit
- 已完成:
  - 存储口径收敛落地：`migrations/20261106170000_transaction_amount_fen.{up,down}.sql`（加列 → **回填前硬门槛** → 回填 → NOT NULL → `CHECK (amount_fen >= 0)`），
    AutoMigrate 不参与建列（`make migrate-*` 是单一真相，parity-p0-1）
  - 硬门槛三态**实测**：干净（含 IEEE754 残值 4e-15）放行 / `1.005`（diff 0.5）RAISE 中止 / 负数 RAISE 中止；
    中止后 `amount_fen` 列与 CHECK **均不残留**（整文件一事务），证据是 `cols=` 列表与 `\d` 计数
  - entity 口径分离：`model/entity/transaction.go:19-46`（`Amount float64` 对外 + `AmountFen *int64 json:"-"` 存储 +
    `AmountFenFromYuan` **拒绝式** / `YuanFromAmountFen`）
  - 仓储双写 + 读以分为准：`app/internal/transaction/repo_postgres.go:15-34`（`ErrAmountPrecision` / `amountFenOrErr` / `applyAmountFromFen`）、
    `:82,101,171,183`（List/Get/Update 三处读归一）、`:112-122`（Create 双写）、`:157`（Update 双写）
  - 测试：`money_fen_test.go`（拒绝/接受表 + `0.1+0.2` 聚合=0.30 + nil 回落）与
    `money_fen_integration_test.go`（真库 9 例，迁移形状断言不过就 Fatal）
  - 两种库形态启动冒烟（端口 8081）：迁移库 `null=NO`+CHECK+`schema 版本校验通过 {version:20261106170000, migrations:27}`；
    AutoMigrate 库 `null=YES`+无 CHECK+无 `schema_migrations`，且 `amount_fen=NULL` 的行**读回 88.5 而非 0**
  - 业务库 `my_go_study` 只读重跑门槛（0 行 → 空洞为真，已明确标注边界）+ 漂移 SQL 在其上报 `column "amount_fen" does not exist`（未污染的证据）
  - inventory 登记 #25（两种库形态形状差异）与 #26（亚分被拒后 502 而非 400），并更新 #22/#23 措辞
  - 对外契约零变更实测：列表 JSON 键集与 `amount` 值不变，响应体不含 `amount_fen`
- 未做/Deferred:
  - `provider=supabase`（`repo_supabase.go`）与 ES（`repo_es.go`）路径按黑名单未动 → 双 provider 口径不一致（#23）
  - 旧列 `transactions.amount` 未删（Brief 明令），切读后的**删列**属后续轮
  - `handler.go:176/:202` 错误映射未改（黑名单）→ 亚分拒绝返回 **502**（#26）
  - used_car / auto_finance / mall（含 #24 `normalizeMoney`、h6 残留 11 Mixed 双计费）本轮一律未动
  - `make migrate-up|status|down` 未在业务库执行（红线 3：那里只允许 SELECT）；等价命令以 pinned `bin/migrate` 连 scratch 库跑全
  - `make test-integration` 不存在（`make -n` 报 No rule），以 `-tags=integration -p 1` 定向命令替代
- 关键文件:
  - `migrations/20261106170000_transaction_amount_fen.up.sql`（门槛在回填之前，含运维复位说明）/ `.down.sql`
  - `model/entity/transaction.go`、`app/internal/transaction/repo_postgres.go`
  - `app/internal/transaction/money_fen_test.go`、`money_fen_integration_test.go`
  - `docs/acceptance-records/2026-11-06-harden-h6b-transaction-money-unit.md`（全部实测输出 + 第一版缺陷的错误更正）
  - `docs/money-unit-audit.md`（h6 输入契约）、`docs/placeholder-inventory.md` #22–#26
- harness: post ok? **未运行** —— 本仓 `make agent-post` / `check-brief` 机检脚本尚未落地（`.harness/rules/slice-contract.md` 标注「若已落地」）；
  依 `.harness/rules/no-silent-accept.md` 以「验证命令 + acceptance-record + Context Card」人工核对收口。
  状态词 **`Degraded` + note**（`.harness/rules/status-enums.md` 五态；第一轮误用的 `Partial` 已作废）。
  验证命令结果：`go -C app test ./internal/transaction/ -count=1` → `ok 1.489s`；
  `-tags=integration -p 1 … -run MoneyUnit` → 9/9 PASS `ok 1.265s`；`go build all` → **exit 1（HEAD 既有环境问题，见已知坑）**。
  未 commit / 未 push。
- 下一 Slice 建议:
  1. **h6b-2：切读收口 + 状态码修正** —— handler 用 `errors.Is(err, ErrAmountPrecision)` → 400（#26），
     并加「本地已迁移但 provider=supabase」的启动期拒绝或云端补列（#23）；两步都需要把 handler 纳进白名单
  2. **h9（mall 规则）**：`normalizeMoney` 改拒绝式（#24）+ Mixed 双计费裁决（h6 残留 11，**待人裁决后才动码**）
  3. **依赖修复（h2/h1 名下）**：`grpc 1.71.0 × x/net 0.54.0 × go1.27.3` 的 `http2.TrailerPrefix`；
     `.gitignore` 的 `logs/` 误伤 `thunder/logs/`、`go.work` 被忽略导致 worktree 不可构建
  4. used_car / auto_finance 金额口径按 §6.2 五步同样收敛（每条一个 Slice，勿合并）
- 已知坑:
  - **保留字**：PL/pgSQL 里 `strict`/`verbose` 不能当变量名；第一版因此从未跑起来 → record §1 的错误更正
  - **中止会留 dirty**：门槛 RAISE 后整文件回滚（列/约束不残留），但 `schema_migrations` 停在 `dirty=true`，
    重跑前必须 `make migrate-force VERSION=20261104160000`；已写进迁移文件注释与异常文案
  - **`make migrate-baseline` 对我的新迁移是陷阱**：它 force 到 `MIGRATE_LATEST`（= `20261106170000`），
    会把「未执行」标记成「已应用」→ 历史 AutoMigrate 库应 force 到**前一条** `20261104160000` 再 `migrate-up`
  - **NOT NULL 是破坏性变更**：只写旧列的写入方在迁移库上直接报错（`null value in column "amount_fen"`），这是刻意的响保护，发布前需排查这类写入方
  - **`go build all` 在 HEAD 就红**（非本轮引入）：`grpc v1.71.0` 要求 `x/net v0.34.0`，工作区 MVS 选到 `v0.54.0`，
    其 `http2/server.go` 带 `!(go1.27 && !http2legacy)` → go1.27.3 下 `http2.TrailerPrefix` 被编译掉。
    实测 `GOTOOLCHAIN=local`（1.26.4）与 `-tags http2legacy` 均 exit 0；只影响 `delivery/grpc{,/interceptor}`，改动的包本身全绿。
    修它要动 `go.mod`/`go.sum`（h2）或 Makefile（h1），本轮黑名单内未动
  - 本机 `go mod tidy`/`go get` 禁用（`proxy.golang.org` 不通）；`GOFLAGS=-mod=mod` 会**改坏** `app/go.mod`（第一轮踩过，已 `git checkout --` 还原）
  - zsh 下 `source-env.sh` 变量取不到，环境相关命令一律 `bash -c`；`DATABASE_DBNAME` 默认 `my_go_study`，跑 scratch 必须在 source **之后**覆盖
  - 本机业务库 `transactions` 为 0 行 → 门槛「0 行」在本机是空洞为真，判别力只在 scratch 库上验证
