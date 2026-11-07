# Acceptance Record — harden-h10-unique-index-name-alignment

- 日期：2026-11-06（Brief 落盘日；实测执行 2026-11-07）
- Slice：`plans/slices/harden-h10-unique-index-name-alignment.md`
- 工作树：`/Users/qpy/orca/workspaces/my_go_study/harden-h10` @ `e288553`
- 本轮落地：**路线 (B)**（entity tag `uniqueIndex`→`unique`，零 DDL、零迁移、零新增 .down.sql）
- 路线 (A) / 破坏性 `DROP CONSTRAINT`：**未做**，人批 **未发生**，已另发 `ask` 等 coordinator 转人批

## status: **Degraded**

note（Degraded 必填）：

1. 路线 (B) 已落地并实测通过：#1/#2 两处 `auto_migrate=true` 启动 `grep -c 42704 = 0`（两种库形态各跑一遍）、
   每列仅一条等价唯一对象、build/定向测试绿、迁移链 `make migrate-down`→`make migrate-up` 真跑过。
2. (B) **不满足原目标「两侧同名」**：唯一对象仍叫 `<table>_<column>_key`（DDL 内联 `UNIQUE` 的 PG 自动名），
   entity `unique` tag 不带显式名，没有建立 `uk_*` 这类单一规范名。按 Brief (B) 款需把目标改写成
   「零 42704 + 零重复索引」并写明取舍 —— Brief 正文禁改，故此处书面记录取舍，**不擅自改 Brief 正文**。
3. 达成「两侧同名」的路线 (A) 需要对存量库执行 `DROP CONSTRAINT`（破坏性 DDL），**人批未发生**，
   本 worker 无批准权、也未替人做批准决定；未碰任何存量库对象，业务库 `my_go_study` 全程只读 SELECT。
   走 A 还是 B 的决定已通过单独的 `ask` 上报，待 coordinator 转人批 —— 该部分 `Deferred` 到批复后收口。

## 缺陷机理（源码级，逐行核过）

出口不是「索引名对不上」，是 GORM `MigrateColumnUnique` 的 `unique && !field.Unique` 分支：

- `gorm@v1.31.2/migrator/migrator.go:599-612` 与 `gorm@v1.25.12/migrator/migrator.go:580-598`：
  `unique, ok := columnType.Unique()`；`constraint := NamingStrategy.UniqueName(table, column)` = `uni_<table>_<column>`；
  `if unique && !field.Unique { DropConstraint(value, constraint) }` ← 42704 出口。
- `gorm.io/driver/postgres@v1.6.0/migrator.go:545-578`：`columnType.Unique()` 只读
  `information_schema.table_constraints`（`constraint_type IN ('PRIMARY KEY','UNIQUE')` 且单列）。
  列内联 `UNIQUE` → `<table>_<column>_key` 约束 → `unique=true`；`CREATE UNIQUE INDEX` 无 pg_constraint 行 → `unique=false`。
- `schema/field.go:130`：`field.Unique` 只由 `unique` tag 决定；`uniqueIndex` 只写 `field.UniqueIndex`（`schema/index.go:136`）。
- `schema/naming.go:89-96`（两版）：`IndexName`=`idx_*`、`UniqueName`=`uni_*`。
- `AutoMigrate`（`migrator.go:122+`）在 `ReorderModels` 循环里 `return err` 即退出 —— **fail-fast 跨表遮蔽**。

版本相关性能给的最强表述：**两版相关源码逐字相同（1.25.12 migrator.go:580-598 / 1.31.2 :599-612、两版 schema/naming.go:89-96），未做运行时对照启动**。
（构建实际 MVS 解析到 gorm v1.31.2；app/go.mod 声明 v1.25.12、core/go.mod v1.31.2，属 h1/h2 锁区，未动。）

## 实测（贴原始行 / 原始输出）

scratch 库仅用 `p10_it`；HTTP 端口 8081（主仓 :8080 未抢）；业务库 `my_go_study` 全程只读。
runner：`/tmp/h10_run.sh`（Go 进程吞 SIGALRM，故用后台 PID + `kill -9` 收口，非 `perl alarm`）。

### 基线复现（HEAD / BASE 二进制，未改 entity，form ①）

`grep -c 42704` = **2**；原始启动行：

```
2026/11/07 00:49:48  ERROR: constraint "uni_wys_im_store_group_group_id" of relation "wys_im_store_group" does not exist (SQLSTATE 42704)
[0.908ms] [rows:0] ALTER TABLE "wys_im_store_group" DROP CONSTRAINT "uni_wys_im_store_group_group_id"
2026-11-07T00:49:48.778+0800 WARN  schemaboot/boot.go:48 域内建表兜底失败，功能可能受限 {"step": "融云 IM 表", "error": "ERROR: constraint \"uni_wys_im_store_group_group_id\" of relation \"wys_im_store_group\" does not exist (SQLSTATE 42704)"}
```

只报 #1（`uni_wys_im_store_group_group_id`），#2 未出现 —— 证实 fail-fast 遮蔽。

### #2 遮蔽解除实测（PARTIAL 二进制：只修 #1，form ①）

`grep -c 42704` = **2**，但约束名换成 #2 —— 证明 #2 是真实潜伏缺陷、只修 #1 等于把报错挪位置：

```
2026/11/07 01:15:49  ERROR: constraint "uni_wys_im_message_backup_message_uid" of relation "wys_im_message_backup" does not exist (SQLSTATE 42704)
[1.315ms] [rows:0] ALTER TABLE "wys_im_message_backup" DROP CONSTRAINT "uni_wys_im_message_backup_message_uid"
2026-11-07T01:15:49.321+0800 WARN  schemaboot/boot.go:48 域内建表兜底失败，功能可能受限 {"step": "融云 IM 表", "error": "ERROR: constraint \"uni_wys_im_message_backup_message_uid\" of relation \"wys_im_message_backup\" does not exist (SQLSTATE 42704)"}
```

### 路线 (B) 零 42704 复证（两种库形态各跑）

| 形态 | 构造 | `grep -c 42704` | `启动期 DDL 已启用` |
|------|------|------------------|----------------------|
| ① 迁移建库 | `make migrate-up` 全新 p10_it → `(B)` 二进制 `auto_migrate=true` 启动 | **0** | 1 |
| ② 空库从零 | 空 p10_it，无 migrate，直接 `(B)` 二进制 `auto_migrate=true` 从零建 | **0** | 1 |

（exit=1 的 `grep -c` 仅代表 0 命中，是期望值。）两形态均跑过完整 DDL 链（marker=1），
不是「health 绿但没跑到 DDL」。#1 修好后 #2 不再新增 42704 —— 两形态总 `42704` 计数均为 0。

### 零重复索引（贴原始输出）

form ① 精确 indexdef：

```
CREATE UNIQUE INDEX wys_im_message_backup_message_uid_key ON public.wys_im_message_backup USING btree (message_uid)
CREATE UNIQUE INDEX wys_im_message_backup_pkey ON public.wys_im_message_backup USING btree (id)
CREATE UNIQUE INDEX wys_im_store_group_group_id_key ON public.wys_im_store_group USING btree (group_id)
CREATE UNIQUE INDEX wys_im_store_group_pkey ON public.wys_im_store_group USING btree (store_id)
```

`wys_im_store_group.group_id` 与 `wys_im_message_backup.message_uid` 各仅一条等价唯一对象（`_key`），
无 `idx_*` / `uni_*` 重复。form ② 同样（贴自 runner）：

```
wys_im_message_backup.idx_wys_im_message_backup_target_id
wys_im_message_backup.idx_wys_im_message_backup_user_id
wys_im_message_backup.ix_wys_im_message_backup_user
wys_im_message_backup.wys_im_message_backup_message_uid_key
wys_im_message_backup.wys_im_message_backup_pkey
wys_im_store_group.wys_im_store_group_group_id_key
wys_im_store_group.wys_im_store_group_pkey
```

> 说明：`idx_wys_im_message_backup_target_id` / `_user_id` 是其它列（`target_id`/`user_id`）上 `index` tag 的普通索引，
> 与 `group_id`/`message_uid` 的唯一对象无关；它们在 (B) 下才出现，是因为列阶段不再 fail-fast、索引阶段得以完整跑到。
> 另：form ② 也得到 `_key` 而非 `uni_*`，因为 `EnsureImSchema`（`im_repo.go:24`）先 `db.Exec(imSchemaSQL)` 建表，
> AutoMigrate 走列迁移路径而非 `CreateTable` —— 这是实测结果，与 Brief 推论中「从零建会是 uni_*」的假设不同，据此记录、未改 Brief。

### 存量库升级/回滚路径（贴原始输出）

路线 (B) **零新增迁移、无新增 .down.sql**，故「新增迁移 .down.sql 往返」对本轮 N/A；
仍按 Brief 字面在 scratch 上真跑 `make migrate-down` → `make migrate-up` 以证迁移链可逆。
`load-env.sh` 会把 `DATABASE_DBNAME` 重置为业务库，故临时 `.env.local` 写 `DATABASE_DBNAME=p10_it`
（gitignored，跑前先 `RESOLVED_DB=p10_it` 校验，跑完删除）：

```
==== make migrate-down (down 1) ====
20261104160000/d payment_ledger (15.047856ms)
DOWN_EXIT=0
==== version after down ====
20260924110000|f
==== make migrate-up ====
20261104160000/u payment_ledger (15.409528ms)
UP_EXIT=0
==== version after up ====
20261104160000|f
```

业务库 `my_go_study`（只读复核，`schema_migrations` 前后均无变更；IM 约束只读）：

```
chk_wys_im_message_backup_dir
wys_im_message_backup_message_uid_key
wys_im_message_backup_pkey
wys_im_store_group_group_id_key
wys_im_store_group_pkey
```

无任何 `uni_*` / `uk_*` 对象；**未执行任何 `DROP CONSTRAINT`**，未碰任何存量库对象。

### 构建 / 测试

- `go -C app build ./cmd/api`（Brief 字面命令无 `-o`）：在 HEAD 因 `app/api/` 已存在为目录而撞名；
  加 `-o /tmp/h10_notag` 后暴露**既有构建断裂**：`go1.27.3` + x/net `v0.54.0`
  （由 `gorm.io/driver/sqlserver` → x/crypto `v0.52.0` 带入，属 h2 依赖定版）使 `http2/server.go`
  被 build tag `!(go1.27 && !http2legacy)` 排除，grpc v1.71.0 的 `http2.TrailerPrefix` 未定义。
  该断裂与本 Slice 改动无关，`-tags http2legacy`（不碰 go.mod/go.sum/go.work）可过。
- `go -C app build -tags http2legacy -o /tmp/h10_B ./cmd/api` → `BUILD_OK`。
- `go -C app test ./internal/repository/postgres/ -count=1` → `ok  app/internal/repository/postgres  0.592s`（grpc 不在该包依赖图，无需 tag）。
  postgres 包内仅 `profile_repo_test.go`（纯单测，连库），无针对 `EnsureImSchema` 的测试。

## 代码改动

| 文件 | 改动 |
|------|------|
| `model/entity/im.go:67` | `uniqueIndex` → `unique`（#1 `WysImStoreGroup.GroupID`） |
| `model/entity/im.go:77` | `uniqueIndex` → `unique`（#2 `WysImMessageBackup.MessageUID`） |

`git diff --stat`：`model/entity/im.go | 4 ++--`（2 insertions, 2 deletions）。未动：
`auth_user.go` / `community.go`（Brief 已证非缺陷，黑名单）、`payment.go`（模板）、两份 DDL 真相源
（`migrations/20260924100000_rongcloud_im.up.sql`、`app/internal/repository/postgres/sql/im_schema.sql` —— 走 (A) 才需同改）、
`main.go` / `go.mod` / `go.sum` / `go.work` / `schemaboot` / `delivery/worker` / `.gitignore` / `scripts/*`（各锁区）。

## 路线 (A) 需要的 DROP CONSTRAINT（仅列清单，未执行、人批未发生）

若人批走 (A)（两侧同名 / payment 模板），存量库需（**破坏性**）：

```sql
ALTER TABLE wys_im_store_group    DROP CONSTRAINT wys_im_store_group_group_id_key;
ALTER TABLE wys_im_message_backup DROP CONSTRAINT wys_im_message_backup_message_uid_key;
```

并同改两份 DDL 真相源为 `CREATE UNIQUE INDEX IF NOT EXISTS uk_wys_im_store_group_group_id` /
`uk_wys_im_message_backup_message_uid`、entity 改显式 `uniqueIndex:uk_*`、新增迁移承载上述 DDL 且
`.down.sql` 能建回 `_key`。新增迁移若含 PL/pgSQL DO 块，需避开 h6b 的坑（`strict` 保留字、
`SELECT ... INTO` 未声明变量、DO 块崩导致其后回填整段不执行），并用
`awk '/^DO \$\$/,/END \$\$;/' ... | docker exec -i ... psql -f -` 逐字复现验证 —— 本轮 (B) 无新增迁移，未触发。

**路线 (C)（`RENAME CONSTRAINT ... → uni_*`）为陷阱，禁止采用**：改完 `unique && !field.Unique` 仍成立，
那条 `DROP CONSTRAINT` 会执行成功、把唯一性静默删掉。本轮未采用。

## 残留 / 待办

- 走 A 还是 B、以及 A 的 `DROP CONSTRAINT` 人批：**未发生**；已发单独 `ask` 等 coordinator 转人批。
- 「两侧同名」原目标在 (B) 下未达（见 status note）。
- `schemaboot.BestEffort` 仍只 Warn（h5 锁区），本轮未改其吞错语义。
- 版本相关性仅按静态同源口径书写：两版相关源码逐字相同、未做运行时对照启动（行号与原文见「缺陷机理」节）。
