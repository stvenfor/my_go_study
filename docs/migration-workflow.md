# 迁移工作流（schema 单一真相）

> 对应任务 [`plans/slices/parity-p0-1-migration-gate.md`](../plans/slices/parity-p0-1-migration-gate.md)。
> 一句话：**进程启动不再建表，`migrations/` 是 schema 的唯一真相；库版本落后就拒绝启动。**

## 1. 为什么改

改之前有四套机制各建一部分 schema，每次启动全跑一遍：

| 机制 | 入口 | 问题 |
|------|------|------|
| GORM `AutoMigrate` | `app/cmd/api/main.go` 的 `autoMigrate`（13 实体白名单） | 启动期 DDL；两实例并发撞 `DDL 锁` / 42701 |
| 内嵌 SQL | `postgres.ConsolidateLocalUsers` + 各域 `EnsureSchema`（8 处） | 同上，且迁移目录看不全 |
| 域内兜底建表 | `EnsurePushDeviceSchema` / `EnsureImSchema` / `EnsureSearchIndexes` | 失败只 `Warn`，缺表要到运行期才暴露 |
| `migrations/` | `make migrate-up` | **从没端到端跑通过**：空库跑到 `20260923170000` 就断（trgm 索引建在没人建过的表上），且有重复版本号 |

## 2. 现在的规则

配置三件套（`configs/config.yaml` → `database.*`，同名 env 覆盖）：

| 键 | env | 默认 | 作用 |
|----|-----|------|------|
| `auto_migrate` | `DATABASE_AUTO_MIGRATE` | `false` | 是否允许进程启动期执行 DDL |
| `schema_version_check` | `DATABASE_SCHEMA_VERSION_CHECK` | `true` | 启动时比对库内版本与 `migrations/` 最高版本 |
| `migrations_dir` | `DATABASE_MIGRATIONS_DIR` | `migrations` | 迁移目录（镜像里是 `/app/migrations`） |

`app/internal/schemaboot.Boot()` 是唯一入口，行为矩阵：

| 情况 | `auto_migrate=true`（本机逃生门） | `auto_migrate=false`（lan / prod） |
|------|--------------------------------|----------------------------------|
| 正常 | 建表 → 无版本表则跳过校验 | 零 DDL → 版本一致才启动 |
| 库版本落后 | 报错退出 | **报错退出**（`/health` 连不上） |
| `dirty=true` | 报错退出 | 报错退出 |
| 无 `schema_migrations` + 有 `users` | 跳过校验 | `Warn` 放行（提示 `make migrate-baseline`） |
| 无 `schema_migrations` + 空库 | 建表 | **报错退出** |

`APP_ENV=lan|prod|production` 由 `common/config.NoDDLAppEnv` **强制** `auto_migrate=false`，
写 yaml、写 env 都开不开——防止「改个 APP_ENV 就把建表路径带回来」。

## 3. 日常操作

```bash
make migrate-status          # 库内版本 + 目录最新（CLI 落在 tmp/migrate，版本 pin v4.15.2）
make migrate-up              # 应用全部未执行迁移
make migrate-down            # 回滚一个版本（每个 up 都必须配 down）
make migrate-baseline        # 历史 AutoMigrate 库首次接入：声明「已是最新」，不重放 SQL
make migrate-force VERSION=20260923167500   # 迁移中断留下 dirty 时重置版本号
```

DSN 一律取自 `.env` / `.env.local` 的 `DATABASE_*`（不再硬编码 `postgres:postgres`）。
`golang-migrate` 的 CLI **必须带 tag 构建**（`-tags "postgres pgx"`），否则报
`unknown driver postgres`；Makefile 与 `docker/Dockerfile` 都已带上。

### compose（`make lan-up` / `make docker-up`）

一次性 `migrate` 服务在 `app`/`worker` 之前跑完并退出，`app` 用
`depends_on: {migrate: service_completed_successfully}` 等它：

```bash
docker compose -f docker/docker-compose.yml logs migrate   # 看迁移日志
MIGRATE_BASELINE=off make lan-up    # 拒绝自动基线旧库（未版本化的非空库会直接失败）

# 只想验证迁移服务、不动业务库：把库名指到一个 scratch 库
docker exec my_go_study_postgres psql -U postgres -d postgres -c "CREATE DATABASE migrate_smoke;"
DATABASE_DBNAME=migrate_smoke ./scripts/docker-compose.sh run --rm -e DATABASE_DBNAME=migrate_smoke migrate
```

镜像侧依赖（缺一个就会在 `migrate` 服务里炸，均已实测）：`/app/migrate`（`-tags "postgres pgx"`）、
`/app/migrations`、`/usr/bin/psql`（判空库 vs 旧库）、`/bin/bash`（脚本用 `[[ ]]`，busybox sh 跑不动）。

脚本 [`scripts/migrate-entrypoint.sh`](../scripts/migrate-entrypoint.sh) 判库三分支：
无版本表 + 无 `users` → 空库直接 `up`；无版本表 + 有 `users` → 基线化到最新再 `up`
（判据与 `schemaboot` 一致）；`dirty` → 非 0 退出，让 app 起不来而不是带病运行。

## 4. 加一张表 / 加一个索引

1. **写迁移，别改实体就完事**：新增 `migrations/<14 位时间戳>_<snake_case_name>.up.sql` + `.down.sql`。
   - 时间戳必须**唯一且大于**目录里现有最大值（`make migrate-status` 会显示）；
     重复版本号是硬错误——golang-migrate 只记一个号，重复会让后一个文件在
     「库恰好停在该版本」时被**永久跳过**（本仓库 `20260923160000` 撞过）。
   - 新业务表用 `wys_` 前缀（`docs/local-auth-postgres.md` §3）。
   - 全用 `CREATE TABLE IF NOT EXISTS` / `CREATE INDEX IF NOT EXISTS`：
     现网库里可能已经有历史 AutoMigrate 建的同名对象。
   - 依赖别人的表 → 版本号必须排在被依赖者之后（社区 trgm 索引就踩过这个）。
   - 用到 `gin_trgm_ops` 之类扩展，先 `CREATE EXTENSION IF NOT EXISTS pg_trgm;`。
2. **本地验证**：
   ```bash
   createdb p01_check                      # 或 docker exec my_go_study_postgres createdb p01_check
   DATABASE_DBNAME=p01_check make migrate-up
   DATABASE_DBNAME=p01_check make migrate-down
   dropdb p01_check
   ```
3. **实体侧**：GORM 实体照常写（`AutoMigrate` 仍在本机逃生门里用），但**不要**指望它上线建表。
4. 启动路径不需要改：`schemaboot` 会读目录里的新文件，版本落后就拒启。

## 5. 本机逃生门（不想跑迁移）

`APP_ENV=dev` + `configs/config.dev.yaml` 里 `auto_migrate: true`（已配好），
`make run` 空库也能起来。注意两点：

- 一旦库里出现过 `schema_migrations`，本机也会被版本校验拦住 —— 这是有意的
  （防止「本机跑的是老 schema，线上是新 schema」）。
- 逃生门只对本机有效：`lan` / `prod` 代码层强制关闭。

## 6. 已知残留（诚实清单）

| 残留 | 为什么留着 | 怎么收 |
|------|-----------|--------|
| 域内 `EnsureSeed` 里还有 `CREATE TABLE IF NOT EXISTS wys_home_todo_packing_demo`（`app/internal/home/repo_seed.go`） | 种子链路属 P0-5，域文件不在本片白名单 | 该表已由 `migrations/20260922160000_home_todo` 建，种子改成纯数据即可 |
| `app/internal/repository/postgres/{push_device_repo,im_repo}.go` 的 `db.AutoMigrate` | 是 `EnsureXxxSchema` 的实现体，调用点已被 `schemaboot` 收口 | 无需再动 |
| 66 处列 **默认值** 差异（库 vs 迁移） | 审计只对齐了表/列/类型/可空性；默认值差异均为增量方向，不影响读写 | 见 `migrations/20260924110000_align_column_drift.up.sql` 头注释 |
| 迁移目录里 9 个文件带 `-- +goose Up` 注释 | 历史从 goose 文档抄来；golang-migrate 当普通 SQL 执行，无 `+goose Down` 段所以不会误解 | 下次碰这些文件时顺手删注释 |

## 7. 相关

- 验收记录：[`docs/acceptance-records/2026-11-03-parity-p0-1-migration-gate.md`](./acceptance-records/2026-11-03-parity-p0-1-migration-gate.md)
- 密钥与 compose 必填变量：[`docs/startup-guide.md`](./startup-guide.md) §13
- 命令索引：[`docs/startup-guide.md`](./startup-guide.md)（`make migrate-*`）
