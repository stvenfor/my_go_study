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
- 未做/Deferred:
- 关键文件:
- harness: post ok?
- 下一 Slice 建议:
- 已知坑:
