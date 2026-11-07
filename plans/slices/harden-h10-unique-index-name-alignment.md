# Slice — harden-h10-unique-index-name-alignment

> 来源：h2 依赖定版执行时撞到的 `SQLSTATE 42704`（融云 IM 表）。`schemaboot.BestEffort` 只 Warn 不阻断，所以它**一直静默存在**。
> 发现者：h2 worker（CodeBuddy，dispatch `ctx_114b42132a21`）。**注意**：它 record 里写的「人已批」不成立 —— 它从未发 ask；本 Slice 的成立与派发均由 coordinator 决定，人批的是 Brief。
> ⚠ **coordinator 自我更正（2026-11-06，派发前自查）**：本 Brief 的**第一版机理与站点表有两处是错的**，已由源码 + `pg_indexes` 实证推翻，现文即为更正版：
> ① 第一版断言「裸 `uniqueIndex` → GORM 期望的名字是 `uni_<table>_<column>`」——**错**。裸 `uniqueIndex` 生成的是**索引** `idx_<table>_<column>`；`uni_*` 是**另一条代码路径**（`MigrateColumnUnique`）里的**约束名**，见下节。这正是 auth 站被我误判为缺陷的原因。
> ② 第一版把「与 gorm 版本无关」降级为「未证」，现在**给出了静态证据**（1.25.12 与 1.31.2 相关源码逐字相同，行号见下），但**仍不等同于运行时对照实测**，措辞按「静态同源」写，不许写成「已实测版本无关」。

## 缺陷机理（源码级，已逐条落到文件行号）

触发 42704 的**不是**「索引名对不上」，而是 GORM 这一段：

```
gorm@v1.31.2/migrator/migrator.go:599-612  func (m Migrator) MigrateColumnUnique(...)
    unique, ok := columnType.Unique()          // 库里该列「当前有没有 UNIQUE 约束」
    constraint := m.DB.NamingStrategy.UniqueName(stmt.Table, field.DBName)   // = uni_<table>_<column>
    if unique && !field.Unique { return ...DropConstraint(value, constraint) }   // ← 42704 出口
    if !unique && field.Unique { return ...CreateConstraint(value, constraint) }
```
`gorm@v1.25.12/migrator/migrator.go:580-598` 为**同一函数同一逻辑**（逐字比对一致）；`schema/naming.go:89-96` 两版同为 `idx_*` / `uni_*` 规则 —— 这是本 Slice 唯一能给出的「版本相关性」证据：**静态同源，未做运行时对照**。

三个前提必须同时成立才会报 42704：

1. **列侧有真正的 UNIQUE 约束**。`columnType.Unique()` 只认 `information_schema.table_constraints`（`driver/postgres@v1.6.0/migrator.go:545-578`，`constraint_type IN ('PRIMARY KEY','UNIQUE')` 且该约束只覆盖 1 列）。PG 的 `CREATE UNIQUE INDEX` **不产生 pg_constraint 行** → 查不到 → `unique=false`；而迁移里的**列内联 `UNIQUE`** 产生 `<table>_<column>_key` 约束 → `unique=true`。
2. **entity 没写 `unique` tag**。`field.Unique` 只由 `unique` tag 决定（`schema/field.go:130`），`uniqueIndex` 只写 `field.UniqueIndex` / index 的 `CLASS=UNIQUE`（`schema/index.go:136`）→ `field.Unique=false`。
3. 于是 `unique=true && field.Unique=false` → GORM 去 `ALTER TABLE … DROP CONSTRAINT uni_<table>_<column>`，而 PG 那个约束叫 `<table>_<column>_key` → **`SQLSTATE 42704`**。

**另外：IM 的 DDL 有两个真相源，改一个不够。** `migrations/20260924100000_rongcloud_im.up.sql:41,48` 与 `app/internal/repository/postgres/sql/im_schema.sql:41,48`（`im_repo.go:16-17` 用 `//go:embed` 引入，`EnsureImSchema` 先执行它再 `AutoMigrate`）是**逐字重复的两份内联 `UNIQUE`**。空库从零建时实际生效的是 embed 那份。

**推论（务必实测坐实，别只修 42704 就收）**：42704 抛在**列阶段**，`AutoMigrate` 在第一个 error 处即返回（`db.AutoMigrate(...)` fail-fast），所以
- 同一次调用里**排在其后的表根本没被检查**；
- 修好 #1 之后，#2 才会露出来（h2 五次启动只报 1 个 IM 约束名、库里 `wys_im_message_backup` 只有 `_key` 没有 GORM 版索引名，就是这个遮蔽的现场证据 —— 见「站点实测」）；
- 且列阶段通过后，GORM 的**索引阶段**会给裸 `uniqueIndex` 补建 `idx_<table>_<column>`，与已有 `_key` **并存成重复索引**（payment 侧实测只有一条 `uk_*`，因为两侧同名，见下）。

## 站点实测（coordinator 亲跑，2026-11-06；证据可复现）

全量扫描 `grep -rn 'uniqueIndex' model/entity/*.go` → 8 处（IM 2 / auth 1 / community 1 / payment 4）。逐个定性：

| # | entity | 迁移/建表侧 | 实际后果 | 证据 |
|---|---|---|---|---|
| 1 | `model/entity/im.go:67` `WysImStoreGroup.GroupID` `…;uniqueIndex;not null` | **两份 DDL 都写内联 `UNIQUE`**：`migrations/20260924100000_rongcloud_im.up.sql:41` 与 `app/internal/repository/postgres/sql/im_schema.sql:41`（后者由 `im_repo.go:16-17` embed、`:24` 先执行） | **今天真报 42704** | h2 `p03_it` 五次启动日志里唯一出现的约束名就是 `uni_wys_im_store_group_group_id`（A2/B1/B2 各 2 行 = 1 ERROR + 1 WARN）；库里 `pg_constraint` 有 `wys_im_store_group_group_id_key` |
| 2 | `model/entity/im.go:77` `WysImMessageBackup.MessageUID` `…;uniqueIndex;not null` | 同上两文件的 `:48`（`message_uid text NOT NULL UNIQUE`） | **同源缺陷，当前被 #1 遮蔽**（`im_repo.go:32→33` 顺序 fail-fast，永远轮不到它） | 库里 `wys_im_message_backup` 只有 `_key`，**没有** GORM 版索引名 → 索引阶段也没跑到它；修 #1 后它必须一起修，否则只是把报错往下挪一行 |
| 3 | `model/entity/auth_user.go:12` `AuthRefreshToken.TokenHash` `…;uniqueIndex;not null` | `migrations/20260923167500_align_gorm_only_schema.up.sql:18` `CREATE UNIQUE INDEX IF NOT EXISTS idx_auth_refresh_tokens_token_hash` | **不是缺陷（第一版判错，已更正）** | 裸 `uniqueIndex` 的 GORM 索引名 = `idx_auth_refresh_tokens_token_hash`，**与迁移给的名字逐字相同**；库里 `pg_indexes` 只有 `idx_*` 一条，无重复；`CREATE UNIQUE INDEX` 无 pg_constraint 行 → `unique=false` → 两个分支都不进；h2 全部日志零 `uni_auth_*` |
| 4 | `model/entity/community.go:19` `Name` `…;uniqueIndex` | 无迁移（`grep wys_community migrations/*.sql` 空） | **不是缺陷（第一版判定正确）** | 唯一真相源是 GORM 自己 → 两侧必然同名。**不许顺手改** |
| — | `model/entity/payment.go:50,77,79,80` 显式 `uniqueIndex:uk_*` | `migrations/20261104160000_payment_ledger.up.sql:26` 显式 `CREATE UNIQUE INDEX uk_*` | **正确模板，别动** | `p03_it` 实测 `pg_indexes`：`uk_wys_payment_records_out_trade_no` / `uk_wys_payment_reconcile_diff` 各 1 条，无 `_key`、无重复；h2 四种形态零 42704 |

结论：**真缺陷 2 处（#1 已报、#2 潜伏），假阳性 2 处（#3、#4），模板 1 套（payment）**。第一版写的「4 条，3 真 1 假」是错的。

## 候选路线（含一条陷阱；必须实测选路，不许只按本 Brief 推断就落地）

- **(A) payment 模板**（推荐起点，与我最初的判断一致）：entity 改显式 `uniqueIndex:uk_<table>_<column>`，迁移侧把列内联 `UNIQUE` 换成独立 `CREATE UNIQUE INDEX IF NOT EXISTS uk_*`。
  - 代价：**存量库必须先 `ALTER TABLE … DROP CONSTRAINT <table>_<column>_key`**（否则 `unique` 仍为 true → 42704 照旧，且 `_key` 与 `uk_*` 双索引并存）。`DROP CONSTRAINT` 属破坏性 DDL，**需人单独批**，`.down.sql` 要能把它建回去。
  - 收益：两种库形态两侧同名、`pg_indexes` 只一条 —— 与 payment 完全一致，可机检。
- **(B) 零 DDL**：entity tag 由 `uniqueIndex` 改为 `unique` → `field.Unique=true` → 两个分支都不进 → 42704 归零，不动迁移、不动存量库。
  - 代价：两种库形态**名字仍然不同**（迁移库 `<table>_<column>_key`；AutoMigrate 从零建的库 `uni_<table>_<column>`，因为 GORM CreateTable 用 `UniqueName` 内联约束）。本 Brief 的 TARGET_MODULE「两侧同名」这条**不满足** → 选它就得把目标改成「零 42704 + 零重复索引」并写清为什么放弃同名。
- **(C) 陷阱，禁止采用**：把 PG 约束 `RENAME CONSTRAINT <table>_<column>_key` 改成 `uni_<table>_<column>`（即第一版 Brief 里的路线 B）。改完 `unique=true && field.Unique=false` 仍成立，但那句 `DROP CONSTRAINT` **会执行成功 → 唯一性被静默删除**，比报错严重得多。第一版 Brief 把这条列为候选是**我的错误**，现列为禁止项；你若认为它可用，必须先给出反证并回报。

## Slice Brief

- SOURCE_MODULE: `model/entity/im.go`、`app/internal/repository/postgres/im_repo.go`（`//go:embed` + `EnsureImSchema`）、`app/internal/repository/postgres/sql/im_schema.sql`、`migrations/20260924100000_rongcloud_im.{up,down}.sql`
- TARGET_MODULE: #1/#2 两处 `auto_migrate=true` 启动**零 42704**，且 `pg_indexes` 不出现同一列两条等价唯一索引；若走 (A) 则两侧同名（原目标），若走 (B) 按上面的口径改写目标并写明取舍
- Source entry: `model/entity/im.go:67,77`、`app/internal/repository/postgres/im_repo.go:16-17,24,27-34`、`sql/im_schema.sql:41,48`、`migrations/20260924100000_rongcloud_im.up.sql:39-49`
- Target entry: 路线 (A) → entity `uniqueIndex:uk_wys_im_store_group_group_id` / `uk_wys_im_message_backup_message_uid` + 迁移 `CREATE UNIQUE INDEX IF NOT EXISTS` + 一次新增迁移做 `DROP CONSTRAINT`；路线 (B) → 仅 entity tag 改 `unique`
- **本轮 ONLY**: 只修 #1/#2 两处唯一索引口径（含必要的新增迁移）。不动业务逻辑、不动金额口径、不动 h2 的 gorm 声明。
- 不做:
  - **不改 `auth_user.go:12`、不改 `community.go:19`**（上表已证不是缺陷；改了就是越界。第一版 Brief 要求「单独处理 #3」的指令**作废**，代之以这条「不许动」）
  - 不动 `payment`（模板，已绿）
  - 不采用路线 (C)
  - 不删任何既有约束**除非**人明确批了「h10 允许对 IM 两列 `DROP CONSTRAINT`」；未批 → 只落 (B) 或只落「迁移文本改新库、存量库留人裁定」，并在 record 里写 `Degraded`
  - 不改 `schemaboot.BestEffort` 的吞错语义（h5 锁区；若 h5 已升级为硬失败，两侧口径撞一次再写进 record）
  - 不引入 AutoMigrate 之外的隐式建表：新索引走 `migrations/` + `make migrate-up`（pinned migrate v4.15.2，`Makefile:66-71`/`:86-104`）
- 验收:
  - [ ] **零 42704 复证（硬指标）**：两种库形态各跑 —— ①`make migrate-up` 建全新库后 `auto_migrate=true` 启动 ②空库直接 `auto_migrate=true` 从零建；**贴原始启动行**并给 `grep -c 42704` = 0。不许用 health 绿代替（`BestEffort` 只 Warn）
  - [ ] **#2 遮蔽解除的实测**：证明修完 #1 之后 `wys_im_message_backup` 不再新增 42704（若你只修 #1 就交 → 记 `Degraded` 并写明「#2 会在下次启动暴露」这一条已被本次实测证实/证伪）
  - [ ] **零重复索引**：`pg_indexes` 里 `wys_im_store_group.group_id`、`wys_im_message_backup.message_uid` 各只允许一条等价唯一对象（贴查询原始输出）
  - [ ] **路线选择写理由**：(A) 还是 (B)；(A) 的 `DROP CONSTRAINT` 是否已拿到人批；未拿到就跑 (B) 或 `Degraded`，不许擅自 DROP
  - [ ] 存量库升级/回滚路径：新增迁移（若有）`.down.sql` **必须真跑过** `make migrate-down` 再 `make migrate-up`
  - [ ] 只连 scratch 库（`P03_PG_DBNAME`，本 Slice 用新名 `p10_it`）；业务库 `my_go_study` **只允许只读 SELECT**，禁止任何 DDL/DML
  - [ ] 状态词只用 `.harness/rules/status-enums.md:5` 五态（`Migrated/Degraded/Placeholder/Deferred/Removed`）；「与 gorm 版本无关」只能写成「两版相关源码逐字相同（行号已给），未做运行时对照实测」
- 文件白名单:
  - `model/entity/im.go`
  - `migrations/20260924100000_rongcloud_im.{up,down}.sql`、新建一次 `migrations/2026*align_unique_index_names*.{up,down}.sql`
  - `app/internal/repository/postgres/im_repo.go`、`app/internal/repository/postgres/sql/im_schema.sql`（**IM DDL 的第二真相源，走 (A) 必改**）
  - `docs/acceptance-records/2026-11-06-harden-h10-unique-index-name-alignment.md`
  - `docs/placeholder-inventory.md`（追加式）、`plans/slices/harden-h10-unique-index-name-alignment.md`（**仅 Context Card**；Brief 正文不许改，有异议 `ask`）
- 文件黑名单:
  - `model/entity/auth_user.go`、`model/entity/community.go`、`model/entity/payment.go`
  - `app/cmd/api/main.go`、`go.mod` / `go.sum` / `go.work`（h1/h2 锁区）
  - `app/internal/payment/**`、`app/internal/mall/**`、`app/internal/transaction/**`（h8/h9/h6b 锁区）
  - `app/internal/schemaboot/**`、`app/internal/delivery/worker/**`（h5 锁区）
  - `.gitignore`、`scripts/**`（h1 锁区）
- 验证命令:
  - `./scripts/load-env.sh make migrate-up` / `make migrate-down`（scratch 库）
  - `go -C app build ./cmd/api`
  - `go -C app test ./internal/repository/postgres/ -count=1`（有测试则跑；无测试记录「无测试」并写进残留）
  - 启动日志计数：`(cd app && go run ./cmd/api) 2>&1 | grep -c 42704` → 期望 0（端口 **8081**，不得与主仓实例抢 :8080）
  - 唯一对象计数：`docker exec my_go_study_postgres psql -U postgres -d <scratch> -Atc "select indexname from pg_indexes where tablename in ('wys_im_store_group','wys_im_message_backup') order by 1;"`
  - 只读复核业务库（**不许改**）：`… -d my_go_study -Atc "select conname from pg_constraint where conrelid::regclass::text in ('wys_im_store_group','wys_im_message_backup');"`

## 交给本 Slice 的他人踩坑（coordinator 转交，别重复交学费）

h6b 在 `.up.sql` 的 PL/pgSQL DO 块上被同一坑打两次（coordinator 已复现）：
1. `strict` 是 PG 保留字，**不能当变量名**（`syntax error at or near "strict"`）。
2. `SELECT … INTO bad` 里 `bad` **没声明** → `"bad" is not a known variable`；而 DO 块一崩，整个 `.up.sql` 在它后面的回填全部不执行。

复现命令（scratch 库，逐字可跑）：
```
awk '/^DO \$\$/,/END \$\$;/' migrations/<你的>.up.sql | docker exec -i my_go_study_postgres psql -U postgres -d <scratch> -f -
```
另：别人正在改的资产**别共用**——`transaction` 与它的 migrations 属 h6b，scratch 库各用各的名字（本 Slice 只用 `p10_it`）。

## Context Card — harden-h10-unique-index-name-alignment

- status: **Degraded**（B 已落地实测通过；原目标「两侧同名」未达，路线 (A) 的 `DROP CONSTRAINT` 人批未发生、已发 ask 等转人批）
- 已完成: 缺陷机理源码级定位（三个前提 + fail-fast 遮蔽推论）；8 处 `uniqueIndex` 全量定性（真 2 / 假 2 / 模板 1 套）；路线 (A)(B) 与陷阱 (C) 的代价核算；**落地路线 (B)**（`model/entity/im.go:67,77` `uniqueIndex`→`unique`，零 DDL/零迁移）；实测两种库形态 `grep -c 42704=0`、每列仅一条 `_key` 唯一对象；#2 遮蔽解除实测证实（PARTIAL 只修 #1 → 42704 挪到 `uni_wys_im_message_backup_message_uid`）；`make migrate-down`→`make migrate-up` 真跑过（scratch p10_it）；业务库 `my_go_study` 全程只读；build（`-tags http2legacy`）+ 定向测试绿。证据见 `docs/acceptance-records/2026-11-06-harden-h10-unique-index-name-alignment.md`
- 未做/Deferred: 路线 (A) 与「两侧同名」原目标 —— 待人批（`DROP CONSTRAINT` 人批**未发生**，本 worker 无批准权、未执行、未碰任何存量库对象）；走 A 还是 B 的决定已通过单独 `ask` 上报，待 coordinator 转人批；未动两份 DDL 真相源（走 A 才需同改）；`schemaboot.BestEffort` 吞错语义未改（h5 锁区）
- 关键文件: `model/entity/im.go:67,77`、`app/internal/repository/postgres/im_repo.go:16-17,24,27-34`、`app/internal/repository/postgres/sql/im_schema.sql:41,48`、`migrations/20260924100000_rongcloud_im.up.sql:41,48`、`model/entity/payment.go:50`（模板）
- harness: post ok? — 编译/定向测试 OK；`go -C app build ./cmd/api`（字面命令）在 HEAD 因 go1.27.3 + x/net v0.54.0（h2 依赖定版带入）断在 grpc `http2.TrailerPrefix`，`-tags http2legacy` 可过、未动 go.mod/go.sum/go.work；验收硬指标按 `grep -c 42704=0`（两形态）计，不靠 /health
- 下一 Slice 建议: 与 h1 的 `check-tracked-imports.sh` 构成「结构性缺陷机检」两件套；本 Slice 收口后建议把「同列唯一对象 >1」与「`pg_constraint` 有 `_key` 而 entity 用 `uniqueIndex`」写成一条机检脚本挂 h3 门禁
- 已知坑: ①`BestEffort` 只 Warn，验收必须 grep 计数而非看 health；②`AutoMigrate` fail-fast 会遮蔽后一张表，只修 #1 等于把报错挪位置（本轮实测证实）；③路线 (C) 会静默删唯一性；④h5 可能同时把该吞错点升级为硬失败，口径要撞一次；⑤h6b PL/pgSQL 坑：`strict` 是保留字、`SELECT…INTO` 变量须声明、DO 块崩会跳过其后回填（走 A 新增迁移时适用，本轮 B 无新增迁移）；⑥`load-env.sh` 会把 `DATABASE_DBNAME` 重置回业务库 `my_go_study`，`make migrate-*` 必须先用 `.env.local`（gitignored）指向 scratch 并校验 RESOLVED_DB
- 版本口径: 两版相关源码逐字相同（1.25.12 migrator.go:580-598 / 1.31.2 :599-612、两版 schema/naming.go:89-96），**未做运行时对照启动**
