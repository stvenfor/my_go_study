# 验收记录 — harden-h6b-transaction-money-unit

- Slice：`plans/slices/harden-h6b-transaction-money-unit.md`（权威合同，已完整读）
- 执行者：worker `term_dc38c95a-dd8d-47c7-81c1-e3ab89fc9355`（分支 `maoqian/harden-h6`，基线 HEAD `e288553`）
- 日期：2026-11-06（第二次执行；第一次因迁移文件跑不起来被打回）
- **status：`Degraded`**
- note：本轮**只覆盖 `transactions` 的本地 Postgres 路径**。`provider=supabase` 路径（`repo_supabase.go`）与
  ES 索引（`repo_es.go`）按 Brief 黑名单**刻意未动** → 双 provider 金额口径不一致（残留 23）；
  旧列 `transactions.amount` 未删（Brief 明令不删），双写尚未做过漂移核对之外的发布动作；
  **业务库 `my_go_study` 未被本迁移改过**（只做过 SELECT，见 §4）。

---

## 0. 一句话结论

`transactions` 的存储口径已收敛到整数分（`amount_fen bigint NOT NULL CHECK (>=0)`，由 `migrations/` 落地、AutoMigrate 不参与建列），
入参超过两位小数**改成拒绝式**（不再静默舍入），对外 JSON 字段名与单位一字未改；
但 Supabase/ES 路径未收敛、旧列未删、亚分被拒后 HTTP 状态码是 502 而非 400 → 记 `Degraded`，不记 `Migrated`。

---

## 1. 错误更正（coordinator 打回的两条缺陷，先写这条）

第一次交付的 `migrations/20261106170000_transaction_amount_fen.up.sql` **根本跑不起来**，coordinator 机检复现成立，我复跑同样成立：

| # | 缺陷 | 后果（当时） | 本轮修法 | 复验 |
|---|------|--------------|----------|------|
| 1 | DO 块里变量名用了 `strict` —— PL/pgSQL **保留字** | `ERROR: syntax error at or near "strict"`，整个 `.up.sql` 在这一行就中断，**回填 UPDATE / NOT NULL / CHECK 从未执行** | 变量改名 `n_blocking` / `n_strict` / `n_negative` | `grep -Ec '(^\|[^a-z_])(strict\|verbose)([^a-z_]\|$)' /tmp/h6b_do_block.sql` → **命中 0** |
| 2 | `bad` 从未在 `DECLARE` 里声明 | 就算改掉 `strict`，下一行仍会炸 | `DECLARE` 补齐三个变量，全部显式声明 | DO 块单独 `psql -f -` 直接返回 `DO`（不再报变量错） |

**必须作废的旧结论**（不许保留）：第一轮 record 里写的
「p06b_it 实测两行脏值…放过/拦住」**不是**来自含 `strict`/`bad` 的那版文件 —— 那版文件在任何库上都只产出语法错误。
当时的数字来自我另外手跑的分解 SELECT（把 WHERE 条件单独贴进 psql），属于「证明了判据、但没证明交付物」。
本轮所有数字**均由修好的 `.up.sql` 文件本体产出**，命令与原始输出见 §3、§4。

顺带纠正一处我自己写错的判据解释（第一版注释里写反了）：
`1e-9` 容差**放行的正是** IEEE754 表示残值（`0.30000000000000004` → `amount*100=30.00000000000000400`，差 `4e-15`，round 回 30 分，一分钱没丢），
**拦下的才是**真·亚分金额（`1.005` → `100.500`，差 `0.5`，round 会凭空多吃半分）。
Brief 验收第 3 条用的是 `1e-9` 版，所以放行条件用它；`amount <> round(amount,2)`（h6 那条更严口径）只作为附带计数打印，便于人工核对。

---

## 2. Brief 9 条验收逐条

| # | Brief 验收条目 | 结果 | 证据（本文内编号） |
|---|----------------|------|--------------------|
| 1 | 走 `migrations/` 新增一对 up/down，编号不冲突 | ✅ | `ls migrations/ \| tail -4` + `重复编号数=0`（§3-D 组 C5） |
| 2 | 新列 `amount_fen bigint NOT NULL CHECK (amount_fen >= 0)`；负数口径先核实 | ✅（负数 0 行，且**主动拦**：库里真有负数时 RAISE） | `\d transactions`（§3-B1）、业务库 `negative=0`（§4-F1） |
| 3 | 回填前置硬门槛 `abs(amount*100-round(amount*100))>1e-9` 必须 0 行 | ✅（三态实测：干净放行 / 亚分中止 / 负数中止） | §3-A、§3-D |
| 4 | 双写 + 读以分为准 + 单测「0.1+0.2 记两笔后聚合等于 0.30」 | ✅ | `money_fen_test.go`、`TestMoneyUnitCreateReadAggregate`、HTTP 侧「两笔分之和=30 元=0.30」（§3-L1-7） |
| 5 | Degraded 留痕 | ✅ | `docs/placeholder-inventory.md` 行 22/23/24/25/26 + 详解段 |
| 6 | 真库 scratch 用例（非 sqlite/mock） | ✅ 9 个用例全 PASS，库是我自己的 `p06b_it` | §3-G2 |
| 7 | 两种库形态启动冒烟（NoDDL 语义） | ✅ 两种都过，并量化了二者形状差异 | §3-L1 / §3-L2 |
| 8 | `go -C app test ./internal/transaction/ -count=1`、`go build all`、`make test-integration` | 测试 ✅；`go build all` **因环境工具链在 HEAD 就红**（非本轮引入，见 §5.1）；`make test-integration` **该 target 在本仓 Makefile 不存在** | §3-G、§3-J |
| 9 | 状态词只用五态 | ✅ `Degraded` + note（第一轮我用过 `Partial`，不合法，已在 §6 更正） | 本记录头部 |

---

## 3. 重做要求 1)–6) 的真实输出

### 组 A —— DO 块单独跑，三态（要求 1 + 2a/2b/2c）

库：scratch `p06b_check`（最小表 `id/user_id/amount`，迁移前形状，无 `amount_fen`）。
提取命令与 coordinator 的复现命令同形：

```bash
awk '/^DO \$\$/,/END \$\$;/' migrations/20261106170000_transaction_amount_fen.up.sql > /tmp/h6b_do_block.sql
docker exec -i my_go_study_postgres psql -U postgres -d p06b_check -f - < /tmp/h6b_do_block.sql
```

真实输出：

```text
行数=      34
DECLARE
  n_blocking bigint;
  n_strict   bigint;
  n_negative bigint;
BEGIN
保留字命中数=0  bad 命中数=0

=== 状态 a) 12.34（干净）+ 0.30000000000000004（IEEE754 残值）===
DO                                    ← 放行

=== 状态 b) 插一行 1.005 ===
INSERT 0 1
psql:<stdin>:34: ERROR:  回填门槛未过：1 行金额换成分会改变数值（真·亚分脏值）。附带计数：更严口径 amount <> round(amount, 2) 下有 2 行（含这 1 行），
其余属 IEEE754 表示残值，round 回分不丢钱。禁止用 round/trunc 吃掉钱 —— 先人工核对样本：
SELECT id, user_id, amount FROM transactions WHERE ... abs(amount*100 - round(amount*100)) > 1e-9 ORDER BY id DESC LIMIT 50;
处理完再重跑本迁移（中止已把 schema_migrations 置 dirty，先 make migrate-force VERSION=20261104160000 复位）。
CONTEXT:  PL/pgSQL function inline_code_block line 18 at RAISE

=== 状态 c) 改插一行 -1 ===
INSERT 0 1
psql:<stdin>:34: ERROR:  transactions 有 1 行负数 amount，与 CHECK (amount_fen >= 0) 冲突（负数应由 type 表达）。
CONTEXT:  PL/pgSQL function inline_code_block line 30 at RAISE

=== 判据的真实数值（解释 b/c 为何不同）===
1 amount=12.34                  amount*100=1234.00             diff=0.00                 strict_hit=false
2 amount=0.30000000000000004    amount*100=30.00000000000000400 diff=0.00000000000000400 strict_hit=true
```

`diff=4e-15` 那行 = 放行（round 回 30 分不丢钱）；`1.005` 那行 `diff=0.5` = 拦住。两条口径（`1e-9` vs `<>`）在真实数据上确实分叉，
所以 record 里两个数字都要报，放行条件只用 Brief 那条。

### 组 D —— 完整迁移链三态 + 中止不留列 + down 真跑（要求 2 + 3）

库：scratch `p06b_it`。前一次版本 `20261104160000_payment_ledger`，本次 `20261106170000_transaction_amount_fen`。

```text
### D0 组 C 留下的 dirty 态（本身是一条运维发现）
version=20261106170000 dirty=true
up 会拒绝重放：error: Dirty database version 20261106170000. Fix and force version.
force 到 20261104160000（不执行任何 SQL）→ force exit=0
version=20261104160000 dirty=false
cols=id,user_id,type,category,amount,date,note,created_at,updated_at

### D1 状态 a) 全干净 → up 成功
20261106170000/u transaction_amount_fen (22.827203ms)
migrate exit=0
version=20261106170000 dirty=false
cols=...,amount_fen
CHECK=transactions_amount_fen_nonneg / 1

### D2 状态 b) 先 down 1 回迁移前，插一行 1.005 → up 中止
20261106170000/d transaction_amount_fen (19.019429ms)   version=20261104160000 dirty=false
INSERT 0 1
error: migration failed: 回填门槛未过：1 行金额换成分会改变数值（真·亚分脏值）。...
migrate exit=1
--- 中止后 ---
cols=id,user_id,type,category,amount,date,note,created_at,updated_at   ← **amount_fen 不存在**
CHECK=无 CHECK / 0
version=20261106170000 dirty=true
残留脏行数=1

### D3 状态 c) 负数 -1 → 同样中止
error: migration failed: transactions 有 1 行负数 amount ...
migrate exit=1
cols=...（无 amount_fen）      CHECK=无 CHECK / 0      version=20261106170000 dirty=true

### D4 清理后复位重放
20261106170000/u transaction_amount_fen (28.393586ms)  migrate exit=0
CHECK=transactions_amount_fen_nonneg / 1

### D5 老式「只写 amount 不写 amount_fen」的写入方在迁移库上
ERROR:  null value in column "amount_fen" of relation "transactions" violates not-null constraint
DETAIL:  Failing row contains (17, null, expense, GATE, 5.00, 2026-01-01, null, null, null, null).

### D6 down 1 真跑（要求 3）
rows=0 sum_amount=0
20261106170000/d transaction_amount_fen (20.430104ms)   migrate exit=0
version=20261104160000 dirty=false
cols=id,user_id,type,category,amount,date,note,created_at,updated_at
CHECK=无 CHECK / 0

### D7 再 up（可重放）+ 漂移核对
20261106170000/u transaction_amount_fen (18.063435ms)   migrate exit=0
drift_rows=0
```

三条承诺的落点：
- 「中止后 `amount_fen` 列本身也不残留」—— **成立**，D2/D3 的 `cols=` 与 `CHECK=0` 就是证据（golang-migrate 把一个 `.up.sql` 包在一个事务里，`RAISE EXCEPTION` 触发整体回滚）。
- 「回填只用 round 抹平表示残值」—— **成立**，真·亚分在 D2 就被拦住，round 那行根本到不了它。
- 「处理完直接重跑本迁移」—— **原表述不完整**，中止会把版本行留在 `dirty=true`，必须先
  `make migrate-force VERSION=20261104160000` 再 `make migrate-up`。本轮把这句写进了迁移文件的注释与异常文案（见 D0）。

D5 是本轮最需要人知情的**行为变更**：`amount_fen NOT NULL` 之后，任何「只写旧列」的写入方（手写 SQL、
其它语言服务、未来的 `repo_supabase` 回写）在迁移库上会直接报错，而不是静默少写一列。这是刻意选的响保护，
但它是**破坏性变更**，发布前要先确认没有这类写入方。

### 组 E —— 回填真实数值与漂移（要求 6 的一部分）

```text
### E0 迁移前两行参照数据
18 amount=88.50               amount*100=8850.00
19 amount=0.30000000000000004 amount*100=30.00000000000000400

### E1 up → 回填
18 amount=88.50               -> amount_fen=8850  回读元=88.50
19 amount=0.30000000000000004 -> amount_fen=30    回读元=0.30

### E2 漂移核对 SQL（要求 6）
SQL: select count(*) from transactions where amount_fen is distinct from round(amount*100)::bigint;
drift_rows=0
total=2 gate_blocking(1e-9)=0 gate_strict(<>)=1 negative=0

### E3 CHECK 在写入侧真拦得住
ERROR:  new row for relation "transactions" violates check constraint "transactions_amount_fen_nonneg"
DETAIL:  Failing row contains (18, null, expense, DEMO, 88.50, 2026-01-01, null, null, null, -1).
```

### 组 G —— 测试（要求 5）

```text
=== G1 单元测试（Brief 验证命令原句）===
go -C app test ./internal/transaction/ -count=1
ok  	app/internal/transaction	1.489s

=== G2 集成测试（scratch 库 p06b_it / p06b_am，-p 1 防并行抢库）===
GOPROXY=https://goproxy.cn,direct go -C app test -tags=integration -p 1 ./internal/transaction/ -run MoneyUnit -count=1 -v
--- PASS: TestMoneyUnitCreateReadAggregate (0.05s)
--- PASS: TestMoneyUnitRejectsSubCent (0.04s)
--- PASS: TestMoneyUnitUpdateDualWrites (0.05s)
--- PASS: TestMoneyUnitCheckConstraintRejectsNegative (0.05s)
--- PASS: TestMoneyUnitAutoMigrateShapeFallsBack (0.10s)
--- PASS: TestMoneyUnitYuanToFenAcceptsTwoDecimals (0.00s)
--- PASS: TestMoneyUnitYuanToFenRejectsSubCentAndSign (0.00s)
--- PASS: TestMoneyUnitAggregateIsExact (0.00s)
--- PASS: TestMoneyUnitFenToYuanRoundTrip (0.01s)
PASS
ok  	app/internal/transaction	1.265s
```

集成用例连的是**迁移后的真实库形状**：`openMoneyFenDB` 会断言 `bigint + is_nullable='NO' + CHECK + 旧列仍在`，
断言不过就 `t.Fatal`（不是 Skip）—— 这样「库里根本没迁移」不会被伪装成绿灯。

### 组 L —— 两种库形态启动冒烟（要求 4，端口 8081）

**形态 ①：迁移建好的库 + `auto_migrate=false`（NoDDL 语义）**

```text
INFO schemaboot/boot.go:52  启动期 DDL 已关闭：schema 由发布流程 migrations/ 管理  {"app_env": "dev", "how_to_apply": "make migrate-up"}
INFO schemaboot/boot.go:90  schema 版本校验通过                                   {"version": 20261106170000, "migrations": 27}
INFO api/main.go:545        HTTP 服务启动                                        {"port": 8081, "env": "dev"}
/health → {"status":"ok"}
库形态：amount_fen null=NO / CHECK=transactions_amount_fen_nonneg

1) POST 88.50  → http=201  {"id":4,...,"amount":88.5,...}   库内 4 amount=88.5 fen=8850
2) POST 1.005  → http=502  {"error":"金额最多两位小数: 1.005","code":50000}     L1BAD 入库行数=0
3) POST -1     → http=502  {"error":"金额最多两位小数: -1","code":50000}
4) GET  列表   → http=200  首条键= ['amount','category','createdAt','date','id','note','type','updatedAt','userId']
                          amount 值= [88.5]      响应含 amount_fen? False     ← 对外契约一字未改
5) PUT  0.15   → http=200  amount=0.15 fen=15                                ← 双写
6) PUT  1.005  → http=502  amount=0.15 fen=15（未被改动）                      ← 拒绝且不留痕
7) 两笔 0.10 + 0.20 → 分之和=30  元=0.30                                        ← Brief 验收第 4 条
8) 漂移 → drift_rows=0 / total=3 gate_blocking=0 gate_strict=0 negative=0
```

**形态 ②：空库 + `auto_migrate=true`（AutoMigrate 自己建表）**

```text
INFO schemaboot/boot.go:40   启动期 DDL 已启用（database.auto_migrate=true）  {"app_env": "dev"}
INFO schemaboot/boot.go:101  本地 AutoMigrate 路径：schema 未纳入迁移版本，跳过版本校验
INFO api/main.go:545         HTTP 服务启动                                    {"port": 8081, "env": "dev"}

amount_fen type=bigint null=YES
CHECK 数=0
schema_migrations 存在=0

POST 88.50 → http=201，库内 1 amount=88.5 fen=8850          ← 写入路径与形态 ① 一致
把 fen 手工置 NULL（模拟未回填/历史行）：1 amount=88.5 fen=NULL
GET /api/v1/transactions/1 → http=200 {"id":1,...,"amount":88.5,...}   ← **读回 88.5，不是 0**
POST 1.005 → http=502 {"error":"金额最多两位小数: 1.005","code":50000}   L2BAD 入库行数=0
```

要求 4 要的那句「AutoMigrate 形态会不会让读路径静默读 NULL 回落旧列」—— **答案是「会回落，但回落是设计而非事故，且不会读成 0 元」**：
`AmountFen` 是 `*int64` 且带 `json:"-"`，`applyAmountFromFen` 在 `nil` 时**原样返回**，`amount` 保留旧列值；
证据是形态 ② 的 GET（`88.5` 而非 `0.0`）与 `TestMoneyUnitAutoMigrateShapeFallsBack`。
真正的差异在**写入侧约束**：AutoMigrate 库里既没 NOT NULL 也没 CHECK，只有 Go 侧的拒绝式换算拦着 —— 这条已登记为残留 25。

---

## 4. 业务库 `my_go_study`：我自己重跑的只读核查（红线 1 + 红线 3）

Brief/coordinator 要求「自己重跑硬门槛，不许引用别人的绿灯」。跑的是**门槛原句**，只读，零 DDL/DML：

```bash
docker exec my_go_study_postgres psql -U postgres -d my_go_study -At -c "
select '业务库 transactions 行数='||count(*)
     ||', 更严门槛(<>)命中='||count(*) filter (where amount is not null and amount<>round(amount,2))
     ||', Brief 门槛(1e-9)命中='||count(*) filter (where amount is not null and abs(amount*100-round(amount*100))>1e-9)
     ||', 负数='||count(*) filter (where amount<0) from transactions;"
```

```text
业务库 transactions 行数=0, 更严门槛(<>)命中=0, Brief 门槛(1e-9)命中=0, 负数=0
transaction_records 更严门槛命中=0
wys_used_car_order amount IS NULL=0 / 总=5
业务库 amount_fen 列数=0
业务库 schema_migrations：空表
```

**必须诚实标注的边界**：本机业务库 `transactions` 是 **0 行**，所以「门槛 0 行」在这里是**空洞为真**（vacuous），
它不能证明任何真实数据干净 —— 门槛的判别力是在 §3-A/§3-D 的 scratch 库上用 `1.005`/`-1`/残值三行**打出来**的。
生产库/其它环境**必须各自重跑**这条 SQL 才允许回填。

要求 6 的漂移 SQL 在业务库上的真实输出是**报错**，而这恰恰是本迁移未污染业务库的证据：

```text
select count(*) from transactions where amount_fen is distinct from round(amount*100)::bigint;
ERROR:  column "amount_fen" does not exist
```

**业务库落地本迁移的正确顺序（本轮我没做，也不该由我做）**：该库 `schema_migrations` 是空表（历史 AutoMigrate 库），
直接 `make migrate-up` 会**从头重放整条 27 个迁移**（对已有表会炸）；
而 `make migrate-baseline` 是 `force $(MIGRATE_LATEST)` —— `MIGRATE_LATEST` 现在算出来是**我的 `20261106170000`**，
所以对它 baseline 会把我的迁移**标记成已应用却根本没执行**（`amount_fen` 不存在，Go 侧写入随即撞 NOT NULL）。
建议人执行的是：`force 20261104160000`（我这条的前一条）→ `make migrate-up` → 再看 §3-E2 那条漂移 SQL。
这条 Makefile 与「最新迁移」的相互作用是我这条新迁移引入的运维风险，值得单独确认（`Makefile:71`、`Makefile:98-104`）。

---

## 5. 环境/工具链发现（不是本轮改动引入，但挡住或影响验收）

### 5.1 `go build all` 在本机默认工具链下于 HEAD 就红

```text
J1 go build all（默认 go1.27.3）              exit=1
   # google.golang.org/grpc/internal/transport
   handler_server.go:271:18: undefined: http2.TrailerPrefix
J2 GOTOOLCHAIN=local /usr/local/go/bin(1.26.4) go build all   exit=0
J3 go build -tags http2legacy all                            exit=0
```

因果链（三条都实测）：`grpc v1.71.0` 要求 `x/net v0.34.0`，工作区 MVS 实际选到 **`x/net v0.54.0`**
（`go -C app mod graph`：`golang.org/x/crypto@v0.52.0 golang.org/x/net@v0.54.0`）；
而 `x/net@v0.54.0/http2/server.go` 带 `//go:build !(go1.27 && !http2legacy)`，在 go1.27.3 下 `http2.TrailerPrefix` 被编译掉 → grpc 编译失败。
受影响的只有 `app/internal/delivery/grpc{,/interceptor}`（`go -C app test ./internal/...` 里唯一的两个 FAIL 就是它们的 build failed），
**我改过的包全绿**（`ok app/internal/transaction`，`go -C app build ./internal/transaction/ ./internal/schemaboot/` 与 `go -C model build ./entity/` 通过）。
修法属于 `go.mod`/`go.sum`（h2 的黑名单位）与 Makefile（h1 的）：要么升 grpc、要么钉 `x/net`、要么给构建加 `-tags http2legacy`。
**我没有**动这些文件（本会话早期一次 `GOFLAGS=-mod=mod` 误改 `app/go.mod`/`go.sum` 已 `git checkout --` 还原，现 `git status` 无该二文件）。

### 5.2 `make test-integration` 不存在

```text
$ make -n test-integration
make: *** No rule to make target `test-integration'.  Stop.
```
Brief 验收第 8 条列了这条命令，但本仓 Makefile 无该 target（`Makefile:54` 只有 `test:`）。
等价替代 = §3-G2 的定向 `-tags=integration` 命令；这条差异要么是 Brief 笔误，要么该 target 属其它 Slice 的交付，**留给人裁决**。

### 5.3 worktree 缺两个 gitignore 掉的构建必需项

- `go.work` 被 `.gitignore:33` 忽略 → worktree 里没有它，工作区模式失效，`go build all` / `go -C app ...` 全不可用。
  本轮在 worktree 内本地补了 `go.work`（未跟踪、不入库）。
- `.gitignore:11` 的 `logs/` 连带忽略了**源码目录** `thunder/logs/` → `go build all` 看不到 thunder。
  本轮从主检出 `cp -R` 回来才通。修 `.gitignore` 是 h1 的事，我没碰。
- 默认 DSN：`make migrate-up` 走 `.env` 的 `DATABASE_DBNAME=my_go_study`（`Makefile:84-86`）。
  红线 3 禁止我往业务库发 DDL，所以本轮**没有**跑 `make migrate-up/down/status`，
  而是用同一 `bin/migrate`（pinned v4.15.2，`Makefile:69-78`）显式连自己的 scratch 库；
  上面 §3-D/E 的所有 `migrate` 输出都是这么产生的。`make migrate-down` 同理未跑（它会 down 业务库最新一条，即我这条，但业务库本就没迁移）。

---

## 6. 残留登记（`docs/placeholder-inventory.md`）

| 条目 | status | 本轮动作 |
|------|--------|----------|
| 22 金额三套口径并存 | `Degraded` | 概览行从「待排查」更新为「h6b 已落 `transactions` 本地路径」；used_car / auto_finance / mall 未动 |
| 23 `provider=supabase` 路径不迁移（双 provider 不一致） | `Degraded` | Brief 验收第 5 条要求的留痕，`repo_supabase.go` 一字未改 |
| 24 mall 改价 `normalizeMoney` 静默四舍五入 | `Degraded` | **本轮没碰**（h9 未批、`mall/**` 在黑名单），也**没**顺手修 |
| 25 两种库形态 `amount_fen` 形状不同 | `Degraded` | 新增，详解见 inventory §25（本轮实测：迁移=NOT NULL+CHECK，AutoMigrate=可空无约束，读路径回落已验证） |
| 26 亚分被拒后返回 502 而非 400 | `Degraded` | 新增，详解见 inventory §26（`handler.go:176`/`:202` 在黑名单，刻意不改） |

状态词更正：上一轮 h6 记录里我用了 `Partial`，`.harness/rules/status-enums.md` 五态里没有它，本文与后续一律用 `Degraded` + note。

---

## 7. 需要人裁决 / 转人

1. **h6 残留 11「Mixed = 现金 + 积分双重计费」**：`mall/**` 是 h9 未批区，我在 h6b 里**没动也没修**。
   证据链与「不确定 —— 取决于产品口径」的判断写在 h6 记录（`docs/acceptance-records/2026-11-06-harden-h6-money-unit-audit.md`）新增段，需人裁决。
2. **业务库落地顺序**（§4 末）：`force 20261104160000` → `migrate-up`，不要用 `make migrate-baseline`（会把本迁移标记为已应用）。
3. **§3-D5 的破坏性行为变更**：`amount_fen NOT NULL` 会让「只写旧列」的写入方直接报错。发布前请确认没有这类写入方（含未来 `repo_supabase` 回写）。
4. **`go build all` 红**（§5.1）：h2 的 `go.mod`/`go.sum` 或 h1 的 Makefile 需决定修法。

---

## 8. 白名单/黑名单遵守（`git status --porcelain` 原样输出）

```text
 M app/internal/transaction/repo_postgres.go
 M docs/placeholder-inventory.md
 M model/entity/transaction.go
 M plans/slices/harden-h6-money-unit-audit.md
?? app/internal/transaction/money_fen_integration_test.go
?? app/internal/transaction/money_fen_test.go
?? docs/acceptance-records/2026-11-06-harden-h6-money-unit-audit.md
?? docs/money-unit-audit.md
?? migrations/20261106170000_transaction_amount_fen.down.sql
?? migrations/20261106170000_transaction_amount_fen.up.sql
```
（加上本记录与 `plans/slices/harden-h6b-transaction-money-unit.md` 的 Context Card 改动。）

黑名单零命中：`repo_supabase.go`、`repo_es.go`、`transaction_request.go`、`app/cmd/api/main.go`、
`app/internal/{used_car,auto_finance,mall,wallet,points,payment,membership}/**`、`**/go.mod`、`**/go.sum`、
`.gitignore`、`scripts/`、`model/entity/im.go`、`model/entity/auth_user.go`、Flutter 仓 —— 均未改。
未 commit / 未 push（提交由人执行）。
