# Acceptance Record — parity-p0-1-migration-gate

- 实施日期：2026-11-03　验收日期：2026-11-04（跨天：casbin 闸门补修 + 实机闸门重跑在 11-04）
- Slice：`plans/slices/parity-p0-1-migration-gate.md`
- Epic：`plans/epics/xhs-parity-p0-correctness.md` §P0-1
- 主交付物：`app/internal/schemaboot/`（单一 DDL 闸门 + 版本校验）+ `migrations/` 补建对齐（25 up / 25 down）
  + `make migrate-*` 五条命令 + compose 一次性 `migrate` 服务 + `docs/migration-workflow.md`

## status: **Migrated**

note（口径拆开说，避免「能编译 = 完成」）：

1. **三条重点验收全部实机通过**（`/tmp/p01-gates.sh`，二进制 `bin/api` 构建于 2026-11-04 11:11，含 casbin 闸门修复）：
   ①生产启动零 DDL（schema 指纹前后一致 + 日志零 DDL 关键字）②版本落后必拒启（非 0 退出、`/health` 连不上）
   ③两实例并发启动无 DDL 竞态。
2. **本轮额外补了一个真漏洞**（G8）：`casbin/gorm-adapter v3.1.0` 在 `auto_migrate=false` 的进程里仍会自己
   `CREATE TABLE casbin_rule`（日志逐字命中 `adapter.go:283`），等于借第三方库的手绕过闸门。
   已在 `app/cmd/api/main.go` 用 `schemaboot.HasTable` 挡住，并加 G8 专项门禁复验。
3. **compose 侧走通到「真跑一次性 `migrate` 服务」**（G10，用 `docker compose run` + scratch 库，不碰你的
   `my_go_study` 业务库）；没做的是整栈 `make lan-up` 起 app/worker —— 那会对 **dev 库 `my_go_study` 真放 DDL**，
   属破坏性动作，等你点头再做。

## 实机闸门（每条都是本机 macOS + 本机 Docker PostgreSQL 跑出来的）

环境：`my_go_study_postgres`（pg16）+ `my_go_study_redis`；scratch 库 `p01_prod` / `p01_empty` / `p01_old` /
`p01_blank` / `p01_legacy` / `p01_fresh`；角色 `gateuser`（强口令，非 `postgres:postgres`）；API 端口 8081/8082/8083
（`:8080` 留给用户的其它服务，未动）。

| 门 | 场景 | 期望 | 实测 |
|---|------|------|------|
| **G1** | `APP_ENV=prod` + `DATABASE_AUTO_MIGRATE=true`（被 prod 策略强制关回 false）打全迁移库 `p01_prod` | 起来且零 DDL | `http=200`；`启动期 DDL 已关闭` + `schema 版本校验通过 {version: 20260924110000, migrations: 25}`；日志 DDL 关键字命中 **0**；启动前后指纹逐字节相同：`cols=fbba74710f2751194a54aa8aec6f62d2 idx=08324876c0c291e73192e06ae63fea4e tables=61 seqs=33` |
| **G2** | 同库 `force 20260923160000`（人为落后 5 个迁移） | 非 0 退出 + 点名缺失迁移 | `exit=1`、`http=000`；错误串：`schema 版本落后：库内 20260923160000，迁移目录最高 20260924110000；缺失 5 个迁移：align_column_drift, align_gorm_only_schema, community_search_trgm, new_car_follow_log, rongcloud_im；执行 make migrate-up` |
| **G3** | `UPDATE schema_migrations SET dirty=true` | 拒启并给修法 | `exit=1`、`http=000`；`上一次迁移中断（schema_migrations.dirty=true, version=20260924110000）… make migrate-force 重置版本再启动` |
| **G4** | 空库 + 闸门关 | 拒启 | `exit=1`、`http=000`；`数据库未初始化：缺少 schema_migrations 且无业务表；先执行 make migrate-up（compose 环境由一次性 migrate 服务负责）` |
| **G5** | 历史 AutoMigrate 库（有 `users`、无版本表） | Warn 放行（不误伤存量库） | `http=200`；`库由历史 AutoMigrate 建立，尚无 schema_migrations 版本记录 {"baseline_now": "make migrate-baseline", "then": "make migrate-up"}` |
| **G6** | 两实例并发（8081+8082，同一库） | 无 DDL 竞态 | `A=200 B=200`；`42701/already exists/deadlock/duplicate` 命中 `0/0`；并发后指纹与 G1 完全一致（`tables=61`） |
| **G7** | `APP_ENV=dev` + 空库 `p01_blank`（本机逃生门） | 照常建表并服务 | `启动期 DDL 已启用（database.auto_migrate=true） {"app_env": "dev"}` + `本地 AutoMigrate 路径…跳过版本校验`；`/health=200`；建表 **59** 张 |
| **G8** | `p01_old`（有 `users`、**无** `casbin_rule`）+ 闸门关 | casbin 适配器不得建表 | `http=200`；`casbin_rule 不存在且启动期 DDL 已关闭：权限判定保持原有 SQL 逻辑 {"create_via": "make migrate-up"}`；启动后 `to_regclass('public.casbin_rule')` 仍为**空**、`tables=1`、日志 `CREATE TABLE` 命中 **0** |
| **G9** | 空库经 `scripts/migrate-entrypoint.sh` 推平后，prod 进程接入 | 迁移→启动全链路可用 | `entrypoint exit=0`、版本 `20260924110000`、`p01_empty` **61** 张表；prod 启动 `http=200`、`migrations: 25`、DDL 关键字 **0** |

### `migrate-entrypoint.sh` 四分支（compose 里 `migrate` 服务的全部逻辑）

| 分支 | 库状态 | 实测 |
|------|--------|------|
| 增量 | 有版本表 | G9（`up` 走完 25 对迁移） |
| 空库 | 无版本 + 无 `users` | `has_users=f` → 直接 `up`（同上） |
| 基线 | 无版本 + 有 `users` | `MIGRATE_BASELINE=auto` → `基线化到 20260924110000（不重放 DDL）`，库仍 2 张表、`dirty=false`；`MIGRATE_BASELINE=off` → **exit 1** 并提示 `make migrate-baseline` |
| dirty | `dirty=true` | `库处于 dirty（20260924110000 (dirty)）… make migrate-force VERSION=<n>`，非 0 退出 |

> 本机跑这个脚本需要 `psql`（compose 镜像里已 `apk add postgresql16-client`）。我用 `/tmp/psqlshim/psql`
> 转发进容器复现，**顺手测出并修掉两个真 bug**：
> - `echo "… $latest（…）"` 在 macOS `/bin/bash` 3.2 下把全角括号的多字节首字节吃进变量名，
>   `set -u` 直接 `unbound variable`。同类四处已改为 `${var}` 并 `bash -n` 复验
>   （`migrate-entrypoint.sh` 两处、`push_notify_user.sh`、`rabbitmq-ensure-dlq.sh` 各一处）。
> - **运行时镜像没装 `bash`**：`docker run --entrypoint /app/scripts/migrate-entrypoint.sh` 报
>   `env: can't execute 'bash': No such file or directory`（脚本用了 `[[ ]]`/`pipefail`，busybox sh 跑不动），
>   整条 `make lan-up` 会卡在 `migrate` 服务。已加进 `docker/Dockerfile:61` 的 `apk add` 并重建复验（G10）。

### G10 compose 一次性 `migrate` 服务真跑（`my_go_study-app:latest`，2026-11-04 重建）

```
$ docker exec my_go_study_postgres psql -U postgres -d postgres -c "CREATE DATABASE p01_compose2;"
$ DATABASE_DBNAME=p01_compose2 ./scripts/docker-compose.sh run --rm -e DATABASE_DBNAME=p01_compose2 migrate
...
20260924110000/u align_column_drift (699.214097ms)
[migrate] 完成，当前版本：20260924110000
→ p01_compose2: v20260924110000, dirty=false, tables=61
```

镜像内资产清点：`/app/migrate`（11 MB，带 postgres/pgx tag）、`/app/migrations` 50 个文件、
`/app/scripts/migrate-entrypoint.sh`（可执行）、`/usr/bin/psql` 均在。
`docker compose config` 渲染的依赖图：`app.depends_on.migrate = {condition: service_completed_successfully}`、
服务顺序 `elasticsearch postgres migrate rabbitmq redis app worker …`。**未**起整栈（原因见开头 note 3）。

## 命令与测试

- `go build all` → **exit 0**（11-04 11:2x；11-03 那次因另一会话正在改 `thunder/go.mod`（eino v0.6.0→v0.9.21 等）
  短暂报 `unknown revision`，等其落定后复跑通过。`thunder/` 不在本片白名单，我未改。）
- `go -C app build ./...`（含 `cmd/api` + `cmd/worker`）→ 通过；`go -C app vet ./cmd/... ./internal/schemaboot/...` → 通过
- `go -C app test ./internal/schemaboot/... -count=1` → `ok 0.801s`（7 个用例：迁移目录加载/去重/缺 down 报错、
  `NamesAfter`、闸门关时零步骤、Required 失败即止、BestEffort 失败继续）
- `go -C common test ./config/... -count=1` → `ok 0.385s`（含 `TestLoadProdKeepsSchemaGateClosed`：
  prod 下即使 `DATABASE_AUTO_MIGRATE=true` 也被强制关回 false）
- `make migrate-status` → `目录最新：20260924110000` + 对本机 dev 库（历史 AutoMigrate、无版本表）给出
  `库内无版本记录：… make migrate-baseline`。**注意**：我没对用户的 `my_go_study` 库跑 `make migrate-up`
  （那是真实 DDL，需你点头），只在 scratch 库上验了同一命令链路。
- `make check-secrets` → `✅ 通过`（仅 2 条 lab 占位提示：`.env:DATABASE_PASSWORD` / `.env:JWT_SECRET`）

## 逐条勾验收（§P0-1 的 8 条）

- [x] `go build all` + `go -C app build ./cmd/api` + `./cmd/worker` 通过
- [x] prod 启动零 DDL：日志关键字 0 命中 + **schema 指纹前后逐字节一致**（比 `pg_stat_activity` 更强的证据）
- [x] 版本落后 → 非 0 退出 + 点名 5 个缺失迁移 + `/health` 拒连（G2）
- [x] 空库一条命令到位：`migrate-entrypoint.sh`（等价 `make migrate-up`）→ 61 表 → prod `/health` 200（G9）
- [x] 双实例并发无 DDL 竞态（G6）
- [x] `migrations/` 25 up ↔ 25 down 全配对；唯一点名：`20260924110000_align_column_drift.down.sql` 是
      `SELECT 1;` 占位（列默认值回滚有损，故意不做，见 `docs/migration-workflow.md` 已知残留）
- [x] `make check-secrets` 通过
- [x] 本记录落盘 + slice Context Card 回填

## 已知残留（Deferred，均已在文档点名）

| 残留 | 状态 | 说明 |
|------|------|------|
| 整栈 `make lan-up`（app/worker 真起在 Docker 里） | **Partial** | 一次性 `migrate` 服务已真跑（G10）、依赖图渲染正确；整栈会把 `migrate up` 打在**你的 dev 库 `my_go_study`**（该库现状是历史 AutoMigrate 建的、无版本表 → 走基线化分支），属破坏性动作，等你确认后再跑 |
| `home.EnsurePackingDemo` 内嵌 `CREATE TABLE` | **Deferred** | 仍走 BestEffort（Warn 继续），demo 表；不在生产链路。归 P2 清单 |
| 仓储层 `db.AutoMigrate` 实现（如 casbin 适配器） | **Degraded** | 适配器已用 `HasTable` 挡住启动期建表；其它 `repository/*` 里的 `AutoMigrate` 调用无人调用时不执行，但代码仍在。审计口径见 `docs/project-detail.md` §6.3 |
| `make deps-up` 里 `PASSWORD 'postgres'` | **Deferred** | 仅本机首次起 PG 的 dev 引导；`migrate-*` 的 DSN 已全部改从 `.env` 取（P0-7 记的那条残留本片收口） |
| 66 处列默认值漂移（`align_column_drift` 的 down 不回滚） | **Deferred** | up 已把 146 条 ALTER 打平；回滚有损故占位 |
| `IsLabAppEnv("lan")=true` → lan 仍跑种子 | **Degraded** + note | 闸门只关 DDL，不改种子语义（`NoDDLAppEnv` 与 `IsLabAppEnv` 是两个正交开关）；lan 是联调环境，保留种子是刻意的 |
| 9 个迁移文件带 `-- +goose` 注释 | **Removed**(不适用) | 无 goose 运行器，注释纯历史遗留，不影响执行；清理属 cosmetic，未做 |
