# Slice — parity-p0-1-migration-gate

## Slice Brief
- SOURCE_MODULE: `app/cmd/api/main.go` 启动建表链（AutoMigrate + EnsureSchema + 内嵌 SQL）
- TARGET_MODULE: `migrations/` 单一真相 + `app/internal/schemaboot/` 版本校验
- Source entry: `main.go:107`（AutoMigrate 调用）、`main.go:637-651`（13 实体白名单）、8 处 `EnsureSchema`、`main.go:634`（`ConsolidateLocalUsers` 内嵌 SQL）、`main.go:185`（失败仅 Warn）
- Target entry: `make migrate-up`（`Makefile:68-72` 已有）→ API 启动只读 `schema_migrations` 并校验
- 本轮 ONLY:
  - 新配置 `database.auto_migrate`（默认 false）、`database.schema_version_check`（默认 true），按 `config.go:647-716` 的 yaml+BindEnv+默认值三件套
  - 把四类 DDL 入口收口到单一开关（建议 `app/internal/schemaboot/`），`APP_ENV=production|lan` 强制关闭
  - 版本缺失 → 返回 error 退出（替换 Warn 继续）
  - schema 对齐审计 + 幂等补建迁移（`IF NOT EXISTS` 全用上）
  - 补 `migrations/20260924100000_rongcloud_im.down.sql`
  - compose 一次性 `migrate` 服务 + API `depends_on: service_completed_successfully`
  - 文档：迁移工作流（加表/加索引/本地逃生门）
- 不做:
  - 不引入 goose/flyway（继续用现有 `migrate` CLI）
  - 不改任何业务 SQL 查询与实体字段语义
  - 不顺手做 P2-6 的 outbox / P2-5 的索引重构（只补「因对齐审计发现的缺失索引」）
  - 不动 4S 店务域的业务逻辑
- 验收: 见 [`plans/epics/xhs-parity-p0-correctness.md`](../epics/xhs-parity-p0-correctness.md) §P0-1 全部 8 条勾完；重点三条：①生产启动零 DDL ②版本落后必拒绝启动（`/health` 连不上）③**两实例并发启动无 DDL 竞态**
- 文件白名单:
  - app/cmd/api/main.go
  - app/cmd/worker/main.go
  - app/internal/schemaboot/
  - common/config/config.go
  - configs/
  - migrations/
  - docker/docker-compose.yml
  - docker/docker-compose*.yml
  - Makefile
  - scripts/
  - docs/
  - .env.example
  - .env.lan.example
- 文件黑名单:
  - app/internal/usecase/
  - app/internal/repository/
  - model/
  - core/queue/
  - thunder/
- 验证命令:
  - `go build all`
  - `go -C app build ./cmd/api && go -C app build ./cmd/worker`
  - `go -C app test ./internal/schemaboot/...`
  - `make migrate-up && make run` → `curl -sS http://127.0.0.1:8080/health`
  - `APP_ENV=production make run`（应无 DDL 日志；库版本落后时应退出）
  - 双实例并发：`go -C app run ./cmd/api` ×2（不同端口）+ `migrate up` 各跑一次
  - `make check-secrets`
- 证据: docs/acceptance-records/2026-11-<date>-parity-p0-1-migration-gate.md

## Context Card — parity-p0-1-migration-gate

- status: **Migrated**（进程侧闸门）/ **Partial**（compose 全栈 `make lan-up` 端到端未跑，见 acceptance-record 残留表）
- 证据：[`docs/acceptance-records/2026-11-04-parity-p0-1-migration-gate.md`](../../docs/acceptance-records/2026-11-04-parity-p0-1-migration-gate.md)
- 已完成:
  - `app/internal/schemaboot/`：`Boot()` 单一入口（Required 失败即止 / BestEffort Warn 继续 / 闸门关零步骤 /
    dirty・版本落后・空库・无版本历史库四类判定）+ `HasTable()`；7 个单测
  - `common/config`：`database.auto_migrate`(默认 false)、`schema_version_check`(默认 true)、`migrations_dir`；
    `NoDDLAppEnv = production|prod|lan` 强制关；4 个单测
  - `app/cmd/api/main.go`：8 处内联 DDL 块 + `autoMigrate` 收进 `schemaboot`；casbin 构造加表存在性闸门
  - `app/cmd/worker/main.go`：同一套闸门；`EnsurePushDeviceSchema` 与 jpush 装配解耦（不再「建表成功才装配」）
  - `migrations/`：25 up ↔ 25 down；补 `rongcloud_im.down`、`20260923167500_align_gorm_only_schema`（14 表）、
    `20260924110000_align_column_drift`（146 ALTER）
  - `Makefile`：`migrate-up/down/status/baseline/force`（CLI pin v4.15.2 + `-tags "postgres pgx"`，DSN 取自 `.env`）
  - `docker/Dockerfile` + `docker/docker-compose.yml`：一次性 `migrate` 服务，`app` 用
    `depends_on: service_completed_successfully`；`scripts/migrate-entrypoint.sh` 四分支
  - 文档：`docs/migration-workflow.md` 新建；`startup-guide` §11/§12/§14；`project-detail` §4/§6.3/§6.4/M13-M20/H7
- 未做/Deferred:
  - `make lan-up` 全栈起停 + 容器内 `migrate` 服务真跑（本机 `:8080` 被占，只 build 了镜像）
  - `home.EnsurePackingDemo` 内嵌 DDL 仍在 BestEffort 里（demo 表，未迁走）
  - 66 处列默认值漂移的**回滚**（`align_column_drift.down` 是 `SELECT 1;` 占位）
- 关键文件:
  - `app/internal/schemaboot/boot.go`（闸门语义唯一来源）
  - `app/cmd/api/main.go:121-140`（Boot 调用）、`main.go:216-230`（casbin 闸门）
  - `scripts/migrate-entrypoint.sh`（compose 侧同一判据：`to_regclass('public.users')`）
  - `docs/migration-workflow.md`（加表/加索引/逃生门流程）
- harness: post ok? 「验证命令」全部人工执行完毕（`go build all` exit 0、两套单测 ok、G1–G9 实机）；
  `.harness` 的 `make agent-pre/agent-post` 仍未落地（Program R4）
- 下一 Slice 建议: `parity-p0-3-money-tx-atomic`（P0-2 依赖它）
- 已知坑:
  - workspace 根无 `go.mod`，`go build ./...` 失效 → 用 `go build all` / `go -C app`（AGENTS.md 约束 8）
  - 缺依赖时**勿跑 `go mod tidy`**，会拉最新版（viper/zap/gorm 实测被升过，AGENTS.md 约束 9）
  - 历史上四套机制各建了一部分 schema，空库能起 ≠ 老库能过校验（R1）
  - `wys_` 前缀规范：新表必须遵守（AGENTS.md 表命名）
  - **`echo "… $var（…）"` 会把全角括号首字节吃进变量名**：macOS `/bin/bash` 3.2 + `set -u` 直接
    `unbound variable`。中文输出里的变量一律写 `${var}`（本片实测踩到并修了 4 处）
  - 第三方库会自己建表（casbin `gorm-adapter` 实测在闸门关的进程里 `CREATE TABLE casbin_rule`）：
    闸门要盖到「装配第三方组件」这一层，不能只盯自家 `EnsureSchema`
  - `migrate` CLI v4.15.2 **没有** `baseline` / `status` 子命令：基线用 `force <version>`，
    状态靠 `version` + shell 兜底
  - compose 的 `JWT_SECRET` 兜底默认值已由 **P0-7 移除**（改为 `${JWT_SECRET:?}` 必填），此处口径以 P0-7 为准
