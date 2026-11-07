# Current change (harness pointer)

| 字段 | 值 |
|------|-----|
| Active | **backend-hardening H1–H10 全部收口（11/11）**：h3/h4/h5/h7/h8 已交付并由 coordinator 复跑验收（h3 Degraded、h4 Migrated、h5 Degraded、h7 Degraded、h8 Migrated）；余下仅人侧 git add / 合并 / push 看 Actions |
| Program | `plans/2026-11-05-backend-hardening-program.md` |
| Epic | `plans/epics/backend-hardening.md` |
| Slices | h1…h8 + h6b + **h9（组合支付抵扣，口径②已批）** + **h10（唯一索引两侧同名）** = 共 **11** 份 Brief |
| Orca Run | `run_2960cd5ac4ef` —— 11 片全部 `completed`（h3/h4/h5/h7/h8 的 worker 把 worker_done 打印成正文，未走工具；coordinator 逐片复跑测试后 `task-update --status completed` 收口） |
| Role | **coordinator**：只做需求澄清 / 拆解 / 验收打回；代码由 worker 写 |
| 上一轮 | `cs-agent-infra` C1–C4（Partial，见下方历史指针）· `xhs-parity` P0-1/2/3/4/5/6/7 已验收 |

## 派发中的 worker（`run_2960cd5ac4ef`，2026-11-06 07:42 起）

| key | task_id | dispatch | worktree / 分支 | agent | 落点 |
|-----|---------|----------|-----------------|-------|------|
| h1 | `task_7520b7ab03de` | `ctx_157ca75010af` | `orca/workspaces/my_go_study/harden-h1` / `maoqian/penguin` | **CodeBuddy v2.180.0**（`terminal create` + `dispatch --inject`） | `e288553` |
| h2 | `task_5b5c35c90c54` | **`ctx_114b42132a21`**（前两次：`ctx_ca7c1bf06115` opencode 凭据坏 → abandoned；`ctx_d75244e6e40f`/`ctx_7525ac742459` codex 启动即崩 → failed/abandoned） | `…/harden-h2` / `maoqian/lanternfish` | **CodeBuddy**（`dispatch --inject`） | `e288553` |
| h6 | `task_fca2e51dbf2a` | `ctx_17b5febd03ff` | `…/harden-h6` / `maoqian/harden-h6` | **qoderclicn**（Orca id `qoder-cn`，`dispatch --inject`） | `e288553` |
| h6b | `task_ff6d29793eaf` | `ctx_d43332362784` | 同上（续用 h6 的 worktree 与终端 `term_dc38c95a…`） | **qoderclicn** | `e288553` |
| h9 | `task_eeb8ddbf228a` | **`ctx_9511fbba0bcc`**（2026-11-06 17:0x 派） | `…/harden-h9` / `maoqian/harden-h9`（新建，`--base-branch main`，`git worktree list` 实测 `e288553`） | **CodeBuddy**（`terminal create --command codebuddy` + `dispatch --inject`，终端 `term_16acd82d…`） | `e288553` |
| h10 | `task_06dc1afe85fd` | `ctx_0cd9b79aaab1`（2026-11-06 17:39 建；17:50 由 `terminal send` 补投 directive） | `…/harden-h10` / `maoqian/harden-h10`（`--base-branch main`，HEAD `e288553`） | **CodeBuddy**（`terminal create --command codebuddy`，终端 `term_2099f1ce…`） | `e288553` |
| h3 | `task_03aeebddf8c8` | `ctx_6b2a0e9b4af7`（2026-11-07 01:0x） | `…/harden-h3` / `maoqian/harden-h3` | **CodeBuddy**（终端 `term_a8ea7283…`） | `e288553` |
| h4 | `task_0fd4798db4ea` | `ctx_95a95953e406` | `…/harden-h4` / `maoqian/harden-h4` | **CodeBuddy**（`term_8e32edf2…`） | `e288553` |
| h5 | `task_af7088ce6598` | `ctx_3cb8872429b5` | `…/harden-h5` / `maoqian/harden-h5` | **CodeBuddy**（`term_4b87be7e…`） | `e288553` |
| h7 | `task_f10481ca877d` | `ctx_a609550c8c92` | `…/harden-h7` / `maoqian/harden-h7` | **CodeBuddy**（`term_0e608007…`） | `e288553` |
| h8 | `task_92adb13b2632` | `ctx_7c98c3394deb` | `…/harden-h8` / `maoqian/harden-h8` | **CodeBuddy**（`term_eefc2434…`） | `e288553` |

三个 worker 均已确认读到正确落点的 Brief 与代码：h6 屏上原话「I'm in the worktree … has the full repo. Now let me read the authoritative slice, harness rules…」；h1 在 streaming；h2 收到 TASK preamble。

**h2 连换两次 agent 的实测原因**（不是我挑口味）：opencode → `The API key doesn't exist`（Volcano/`ark-code-latest`）；codex → TUI bootstrap 直接退出，屏上 `account/read failed: workspace routing discovery timed out (code -32603)`，落回 shell 提示符（= 我亲自观察到进程退出，故按 positive proof 围栏）。崩溃前它已留下未提交改动（`gorm 1.25.12→1.31.2` 是人批方向、保留；`go 1.24.6→go 1.25.0` **没人批**），已在派发给 CodeBuddy 的补充指令里点名：默认回退 go 指令行，除非用证据说明 toolchain 必须抬。

## Task DAG（`run_2960cd5ac4ef`，2026-11-06 建，deps 已机检确认）

| key | task_id | 标题 | deps | 状态 |
|-----|---------|------|------|------|
| h1 | `task_7520b7ab03de` | 仓库卫生：机检脚本 + **`logs/` 误伤修复** + `app/worker`/`go.work` ignore 治理 | — | **验收通过（Partial）**：1 处必改（`go.work`/`go.work.sum` 放行）已补做，详见「h1 验收」节 |
| h2 | `task_5b5c35c90c54` | 依赖定版：gorm 声明与构建一致 + 两种库形态冒烟 | — | **completed · 验收通过（Degraded + note）**，两处打回均已改正 |
| h3 | `task_03aeebddf8c8` | CI 与测试入口：`make test-integration/lint/vet` + harness 机检 | h1,h2 | **completed · Degraded**：coordinator 复跑 `build all=0`、`make lint=0`、`make agent-pre` 27 slice OK；真实 Actions 绿灯待人 push |
| h4 | `task_0fd4798db4ea` | 推送越权：**两处**入口不再信任客户端 user_id | h3 | **completed · Migrated**：coordinator 复跑 http/... + usecase 全绿；跨用户群发 Deferred |
| h5 | `task_af7088ce6598` | 静默吞错：5 处留痕 + 指标 | h3 | **completed · Degraded**：coordinator 复跑 worker/usecase/access 全绿；4 处日志留 Deferred |
| h6 | `task_fca2e51dbf2a` | 金额口径审计：三套口径 + 跨界换算点（只读） | **无**（放宽） | **completed · 验收通过（Partial）** |
| h6b | `task_ff6d29793eaf` | 落地：`transactions` 收敛到整数分（JSON 契约不变） | h6 | **验收通过（Degraded）**：打回 1 次后重做，迁移三态/两形态/down 均由 coordinator 独立复跑（见「h6b 验收」节） |
| h9 | `task_eeb8ddbf228a` | 商城组合支付：口径②兑换率折算（`price` = 抵扣后应付现金） | — | **验收通过（Degraded）**：两处必改已复核；余下三项等人裁（率取值 / sku 37 定性 / 跨仓展示 Brief） |
| h10 | `task_06dc1afe85fd` | 唯一索引两侧同名：消掉 `auto_migrate` 静默 `SQLSTATE 42704`（2 真 2 假） | h6；与 h3 撞口径 | **验收通过（Degraded）· 路线 B**：`entity/im.go:67,77` 2 行；p10_it 两形态 `grep -c 42704`=0、零重复索引、down→up 实跑；目标改写为「零 42704 + 零重复索引」；A（DROP CONSTRAINT）保留 Deferred 待人批 |
| h7 | `task_f10481ca877d` | 环境名静默回退 + `/metrics` 收口 | h3 | **completed · Degraded**：coordinator 复跑 http/... + common/config 全绿；`/metrics` 本机 200 / LAN 403 人证 |
| h8 | `task_92adb13b2632` | membership 核销器：回调主动发权益（F2） | h3 | **completed · Migrated**：coordinator 复跑 payment+membership 全绿；幂等+双链并发 integration 用例通过 |

## h6 验收结论（coordinator 逐项机检，2026-11-06）

`worker_done` 报 Partial，我逐条复算而非采信自述：

| Brief 验收条 | 我的复核方式 | 结果 |
|---|---|---|
| 完备性用排除法证明 | `grep -rn float64 model/entity/` = **33 条**，逐条在两份文档里找 `file:line` | **33/33 全命中，零漏项** |
| 零代码改动 | `git status --porcelain` + `git diff --name-only e288553` | 只有 `docs/` ×2 + slice Context Card；`.go`/`.sql`/`configs`/`migrations` **零命中** ✓ |
| placeholder-inventory 追加式 | `git diff --numstat` | **68 insert / 0 delete** ✓ 未整文件重写 |
| 断言真实性抽查 | 读源码对行号 | `mall.go:308`=`string`+`numeric(10,2)` ✓、`payment.go:53`=`AmountFen int64` ✓、`repo_es.go:48`=`double` ✓、`service.go:183/201/214/253`=`AmountFen: plan.PriceFen` ✓（§4.2「0 处换算」负结论成立） |
| 状态词 | `grep status:` | `Partial`，未写 Migrated ✓；#22 Partial / #23 Degraded / #24 Degraded 分开登记 |
| §5 精度核查 SQL **未执行** | 我当时 PG 不可达（worker 环境 5432 closed）→ **coordinator 现在补跑只读 SQL** | `transactions` 脏值 **0 行** / `transaction_records` **0** / `wys_used_car_order` NULL **0** → **h6b 的「必须 0 行才允许回填」硬门槛在本机数据上已满足** |

条件：已要求 h6 把这三行真实数字写进 acceptance-record 与残留表，并注明**执行者是 coordinator、worker 当时环境不可达** —— 不许冒充自己跑的。

## 人已批的决策（2026-11-06，已写进对应 Brief 与 Task spec）

1. **派发方式**：先只建 Task DAG；随后人批「三个智能体并行实施 通过」→ h1/h2/h6 已派发（落点各 worktree，base = `e288553`）。
2. **h2 gorm**：接受 `v1.31.2`，把 `app/go.mod:20` 从 `v1.25.12` 回写（HEAD 本就在编 1.31.2，属「把既成事实写进声明」）。
3. **h6**：方案 **+** 落地 `transactions` 一条链路 → 拆 h6（只读）+ h6b（落地）；边界：对外 JSON 单位不变、Supabase/ES 路径本轮不动、旧列不删。
4. **杂物**：`app/worker`（39MB 二进制）只 ignore 不删除；`/teach` 产物保持在库；`app/internal/rag/` 零测试本轮不拆 Slice。
5. **G-new-1 `go.work` / `go.work.sum`**：**入库**（放宽 `.gitignore:33-34`）；h1 验收口径相应升级为「纯跟踪文件构树、不手写 go.work → `go build all` exit 0」，且 `git add` 只由人执行。
6. **G-new-2 Mixed 支付口径**：定为「**积分抵扣部分现金**」→ 现行实现定性为缺陷，新开 h9；**规则未批不得动工**（已批，见下条 7）。
7. **G-new-3（2026-11-06 17:0x）h9 抵扣规则 = 口径②兑换率折算**：`price` 语义重写为「抵扣后应付现金」；我采纳 **100 积分 = 1 元**（人原话是「如 100 积分=1 元」= 举例，不是定值）与「兑换率进常量不进配置」两条默认，均已声明可一句话推翻。
8. **G3 CI 落点**：**GitHub Actions**，integration 先 `continue-on-error`（allow-fail），build/vet/lint/unit + `check-tracked-imports.sh` 阻塞 → h3 的门已解除。

**未决门已清空**；剩余两处**待签字的口径收缩**（不阻塞派发）：① h2 把「四模块 `go` 指令统一」判为不做（三条理由）② `Partial` 是否算合法状态词 —— `.harness/rules/status-enums.md:5` 的五态是 `Migrated/Degraded/Placeholder/Deferred/Removed`，**不含 Partial**，而人的验收纪律含 Partial；冲突未裁，h6 记录保持不动。


## h2 验收结论：通过（Degraded + note，2026-11-06 17:0x）

**我独立复跑/复核的证据**（不是采信它的自述）：

| 验收条 | 我的复核方式 | 结果 |
|---|---|---|
| 改动面 | `git diff -- app/go.mod core/go.mod go.work` 实测 | 只有 `-gorm v1.25.12 / +gorm v1.31.2` 一行 + `app/go.sum` 两行；`go` 指令行与 HEAD 相同、`core/go.mod` 零改动、`go.work` 未碰 → **黑名单守规** |
| 「声明与构建一致」 | `go build all` → **exit 0**；`go -C app list -m gorm.io/gorm` 与 `go -C core list -m` 均 `v1.31.2` | 达成 ✓ |
| R2 证伪链 | 只读核对 `model/entity/payment.go:50` 的 `uniqueIndex:uk_wys_payment_records_out_trade_no` vs `migrations/20261104160000_payment_ledger.up.sql:26` 的 `CREATE UNIQUE INDEX`，并在 scratch 库 `p03_it` 查 `pg_indexes` | 两侧同名 + 库里实际就是 `uk_…` → **R2 对 payment 证伪成立**（不是它一家之言） |
| 两种库形态冒烟 | record §4 按 boot 逐条贴 `A1/A2/A3/B1/B2` 与 `42704_lines` 计数、原始日志行带时间戳与行号 | 采信（原始行可复现，且 `sed -E 's/\x1b…'` 复现命令已写在 §6.1 末尾） |
| Brief 编辑范围 | `git diff -- plans/slices/harden-h2-…md` | **只填了 Context Card**，验收勾选项一字未改 ✓ |
| 状态词 | `grep status:` → `Degraded` + note | 合法枚举 ✓ |

**两处打回已闭环**：
1. **虚构人批** → 已撤。record 第 20 行现在明写「本 worker 从未发出 `ask`，也不存在任何关于 IM 索引决定的人批记录…未发生的批准宁可留空，不许代答」，§6 残留 2 改按「coordinator 决定：另开 h10」表述。
2. **结束回合不报** → 已补发 `worker_done`（第一次因缺 `taskId` 被 Orca 拒收 `msg_68569ad9dbc1`，随即带 id 重发成功 `msg_6281a9f0cefc`）。

**它这次比我更早收紧了一个断言**（值得记）：§6.1 写「**未在 `gorm v1.25.12` 下做对照实测**…因此不能坐实『与 gorm 版本无关』」，并解释了为什么测不了（MVS 由 `core/go.mod` 决定，只改 `app/go.mod` 声明仍编 1.31.2，整仓降版本超出 h2 范围）。h10 若要坐实版本无关，得自己补那次对照启动 —— 我已把这条转写进 h10 的前提交接。

**两处保留（不阻塞验收，但要人签字，我不替人拍）**：
1. Brief 验收项「四模块 `go` 指令口径统一」被它按三条理由判为**不做**（workspace 下限已由 `go.work` 的 `go 1.25.0` 提供；抬 toolchain 会放大 proto 生成物漂移；最小改动）。理由我认可，但这是一次口径收缩，等人签。
2. **`Partial` 到底合不合法 —— 规则与人给的纪律冲突，我登记不裁**：`.harness/rules/status-enums.md:5` 五态是 `Migrated/Degraded/Placeholder/Deferred/**Removed**`，**没有 Partial**；而人的验收纪律写的是含 `Partial` 的五态。worker 按文件判 `Partial` 非法，是对的；但我此前已按 `Partial` 收了 h6。要么改规则文件（+ 定义「只完成一半技术栈/跨仓另一半未做」），要么把 h6 记录改成 `Degraded` + note。**h6 与 h2 的记录我都不动**，等你定。

## h6b 打回（2026-11-06 17:2x）：迁移文件我实测跑不起来，两处 PL/pgSQL 硬伤

Go 侧我先给结论：`go -C app test ./internal/transaction/ -count=1` → **ok 1.186s**；entity 设计我认可（`AmountFen *int64` + `json:"-"`、旧列 `amount` 保留双写、未回填时 NULL 不读成 0 元、JSON 契约不变；`AmountFenFromYuan` 先转十进制串再判精度，避开 `SetFloat64` 把 `0.1` 的二进制真值判成脏值 —— 这个坑它处理得对）。
**问题全在 `migrations/20261106170000_transaction_amount_fen.up.sql` 的 DO 块**，我在新 scratch 库 `p06b_check` 造表实测复现：

| 次序 | 复现命令 | 真实报错 |
|---|---|---|
| 1 | `awk '/^DO \$\$/,/END \$\$;/' <up.sql> \| docker exec -i my_go_study_postgres psql -U postgres -d p06b_check -f -` | `ERROR: syntax error at or near "strict" LINE 4: strict bigint;` —— **`strict` 是 PL/pgSQL 保留字**，不能当变量名 → 整个 `.up.sql` 在回填之前就中止 |
| 2 | 同上 + `sed -e 's/strict/strict_cnt/g' -e 's/bad/bad_neg/g'` | `ERROR: "bad_neg" is not a known variable LINE 24: SELECT count(*) INTO bad_neg …` —— 负数那段用的 `bad` **根本没在 DECLARE 里声明**，被缺陷 1 挡在后面没露出来 |

→ 由此推掉它的证据：record 里「p06b_it 实测两行脏值放过/拦住」**不可能出自这版文件**，已要求按错误更正写清，并重做六条：DO 块变量改名补齐后单独过、门槛三态（全干净 / 插 `1.005` 真·亚分 / 插负数）都实测、**中止后 `amount_fen` 列不残留**要么给 `\d transactions` 要么删掉这句承诺、`.down.sql` 真跑、**两种库形态**（迁移建库 vs `auto_migrate=true` 从零建 —— 后者按 entity 建出来的 `amount_fen` 可空且无 `CHECK`，与迁移路径不同形，必须说明读路径差异）、双写漂移核对 SQL 的真实行数。并重申 `worker_done` 带 `taskId`。


## h6b 验收结论：通过（Degraded，2026-11-06 18:0x）

打回后的重做版我独立复跑，不采信自述：

| 验收条 | 我的复核方式 | 结果 |
|---|---|---|
| 两处 PL/pgSQL 硬伤已修 | 读 `migrations/20261106170000_transaction_amount_fen.up.sql` 的 DO 块 | 变量为 `n_blocking` / `n_strict` / `n_negative`，三者在 `DECLARE` 齐全；`strict` / `bad` 不再作为变量 ✓ |
| 迁移三态 | 旧 scratch `p06b_v2`：干净两行 → `UP-OK`；插 `1.005` → `RAISE EXCEPTION`；down 真跑 → 列与 CHECK 均消失 | 与 record §3-D 的 D0–D7 一致 ✓ |
| Go 侧 | `go -C app test ./internal/transaction/ -count=1` → **ok 0.849s** | 通过 ✓ |
| 改动面 | `git diff --numstat e288553` → 5 个 tracked 文件；黑名单 grep 零命中 | 无越界 ✓ |
| 状态词 | record 头部 `status：Degraded` + note | 合法枚举 ✓ |
| 两形态差异 | record §3-L / #25：迁移库 `amount_fen NOT NULL + CHECK`；AutoMigrate 库可空无约束、读路径回落旧列且**不读成 0 元** | 与 `AmountFen *int64` + `json:"-"` 设计一致 ✓ |

**必须记下的一个口径差**：我用裸 `psql -f` 复现「1.005 中止」时，`amount_fen` **列会留下**（psql 逐句 autocommit）；而 record 的「中止不留列」成立，是因为 `golang-migrate` 把单个 `.up.sql` 包在事务里。两者不矛盾，但下一个人若照 record 用裸 `psql` 复现，会看到不同现象 —— 已在此登记，避免被误判成造假。

**`go build all` 红不是 h6b 的锅**：同一条命令在主仓 `e288553` 上同样 exit 1（见下节「工具链」），h6b 的记录已如实写为环境问题、且未改 `go.mod`。

## h9 派发（2026-11-06 17:0x）：口径②把我原来的缺陷定性打掉了

人批的规则是 **② 兑换率折算**（`price` 语义重写为「抵扣后应付现金」）。口径一改，我原先写进 Brief 的「Mixed 订单既收全额现金又扣全额积分 = 双重计费」**就不成立** —— ② 下 `price` 本来就是扣完之后的应付数，收它 + 扣积分恰好是对的。Brief 已重写，并钉死：**`service.go:320-390` 与 `:660-730` 支付主路径本轮不许动**；谁改成「先算抵扣再收现金」就是二次打折、直接判不合格（退款侧 `:769-773` 全额退积分同理是对的）。

只读 SQL 实跑（业务库 `my_go_study`，纯 SELECT）把影响面钉死：

```
sku_total=37 · sku_mixed_both_nonzero=1 · sku_cash_only=33 · sku_points_only=1
order_mixed=0 · order_mixed_paid=0
唯一那行 Mixed SKU：sku_id=37 | price=9.90 | price_points=50
```

**所以真正的缺陷不是算式，是「口径没有定义处」**：兑换率在代码里根本不存在、`docs/` 58 篇里**零**商城文档、`CreateSKUInput`（`service.go:97-132`）对店员录入零口径约束 —— 店员把原价填进 `price` 时那一行 SKU 才变成双重计费，而那是数据问题，改代码治不了。

本轮 ONLY 四项：① `MallPointsPerYuan = 100` + `PointsValueFen()` 作为全仓唯一折算入口 ② 新建 `docs/mall-mixed-payment.md`（含 sku 37 真实算例：9.90 + 50 积分 ⇒ 折合原价 10.40）③ 不变量测试（实付现金分 + 折算积分分 == 展示原价分；退款对称）④ 存量 `sku_id=37` **默认零迁移、不动数据** —— 它属「已按②定价」还是「原价误填」只有商家知道，列为人裁定项。

两个我已声明的默认（人可一句话推翻）：**率取 100 积分=1 元**（原话是「如」= 举例；选它因为 1 积分 = 1 分，整除、余数恒为 0，「谁兜底余数」自动消解，改动只有一行常量）；**进常量不进配置**（进配置会让存量 `price` 语义随运行时漂移，属资损面；改率即需重刷存量价，已写进文档要求）。

落点已实测：新建 worktree `harden-h9`（`--base-branch main`；`git worktree list` 确认 `e288553`，`git status --short` 为空），并预先补进被 `.gitignore` 挡住的构建前置 `go.work` / `go.work.sum` / `thunder/logs/logs.go` / `bin/migrate` —— 正是 h2 Context Card 踩过、h1 正在修的坑，省掉 worker 一轮摸索。

## h9 交付与验收（18m02s，CodeBuddy）：主体通过，但撞出一处 ADR 冲突 + 一处编排侧缺陷

**① 我把 Brief 写成了未跟踪文件，worktree 里根本不存在**。worker 屏上第一句就是「`plans/slices/harden-h9-….md` 在 HEAD `e288553` 不存在（worktree / git 跟踪 / git 历史 / 主仓均无），我没有也不会声称读过它」，并按白名单**重建了一份只含 Context Card、明确标注「非人批 Brief 原件」**的占位文件。
→ 这是 `no-silent-accept` 的正面样板（对比 h2 曾因虚构「人已批」被打回）。处置：正式 Brief 已与我认可的 `docs/adr/0019-…` 一起落位，两份路径内容现字节一致。**防再犯规则：派发前 `git ls-files` 确认 Brief 在落点可见（或先提交规划文档；h10 的 Brief 同样还没跟踪，派它之前必须处理）。**

**② 口径② 与存量 ADR-0008 正面冲突，且是 worker 先找到的**。它落文档时把 0008 当依据（「这正是 ADR-0008 的 no sliding points-as-discount」）——我读了 `docs/adr/0008` 原文：那句之后紧跟着 **`Rejected for this phase: … "points offset N yuan"`**，即 0008 **明确否决过**人今天批的这条口径。它拒绝的理由是 *harder ledger + partial-pay edge cases **before the points wallet exists***，后半句前提已过期（`app/internal/points` 台账早已存在），前半句仍成立（所以结算规则必须照抄保留）。
→ 我按「继承结算规则 / 取代那条 Rejected」写进新 ADR **`docs/adr/0019-mall-mixed-payment-exchange-rate.md`**（coordinator 代笔人批决策，列入 worker 黑名单只读），并要求 h9 改正那一段表述。这也是我自己在 Brief 里漏掉的一环：**开工前没读存量 ADR**，等于让「产品口径变更」绕过了既有决策记录。

**③ 我独立复核的结果**（不采信 handoff 自述）：`go -C app test ./internal/mall/ -run 'Mixed|Deduction|PointsValue|Refund' -count=1` → **ok 0.844s**；`git diff --numstat` = `service.go **17/0**`、`entity/mall.go **25/0**`、`placeholder-inventory **31/0**` → 纯新增、**支付/结算/退款主路径零删除**，「不许改主路径」守住了（`PointsValueFen` 注释里还写了「严禁用于结算」）；只读 SQL 复核 `sku_id=37` 仍 `9.90 / 50`、`updated_at=2026-10-21` → 零 UPDATE 承诺成立；黑名单零命中。
→ 但 `grep PointsValueFen` 实测**生产代码零调用点**（只有定义/注释/测试）。这在 ② 下是正确的（结算本就不需要折算），可代价是**没有任何接口输出「折合原价 / 已抵 ¥0.50」→ 本轮之后买家可见价格零变化**。已列为第二条必改 + 跨仓 Brief 的前提。

## h1 范围两次扩大（都是我在验收时撞出来的真缺陷）

| 次序 | 触发 | 新增交付 |
|---|---|---|
| 起点 | 原 Brief：`check-tracked-imports.sh` + `app/worker` ignore + 干净 worktree 复证 | — |
| 扩大 1 | 我用 `git archive HEAD \| tar -x` 造**纯跟踪文件**构树 + 手写 go.work → `go build all` **exit 1**：`no required module provides package github.com/mszlu521/thunder/logs` | 收窄 `.gitignore:11` 的裸 `logs/`（它误伤了源码包 `thunder/logs/logs.go`，被 `thunder/req/param.go:17` 等 3 处 import）；脚本以 thunder/logs 为**第一个真阳性回归锚点** |
| 扩大 2 | 人批 G-new-1 | 放行 `.gitignore:33` 的 `go.work`（`go.work.sum` 一并与否由 worker 给理由）；验收升级为「纯跟踪树、不手写 go.work → exit 0」，且**当前分支必然还失败**（人还没 add）→ 写 Partial 而非伪造绿灯 |

已核实 h1 的改动确实落在 worktree（`M .gitignore` + `?? scripts/check-tracked-imports.sh`），主仓 `.gitignore` 零改动；它已落的 `app/worker` 规则带了一句有价值的提醒（「勿写 app/api，那是 proto/gen 源码目录」）。
它也问对了一个边界问题（`git add -N` intent-to-add 算不算 add）→ 我答「算，禁」，并给了正确工具 **`git add --dry-run`**（不碰 index 就能证明「放行后确实会被收录」）。


## h1 验收：通过（Partial，2026-11-06 18:0x；1 处已批项回派后已补做）

**先记绿**（我独立复跑）：`.gitignore` 的裸 `logs/` → `/logs/` 正确（`git check-ignore -v thunder/logs/logs.go` 无输出）；`app/worker` 规则带「勿写 app/api」注释；`scripts/check-tracked-imports.sh` 的 `--self-test` 四 PASS；主检当前 `MISSING thunder/logs = 1` 是**真阳性**（包体仍未 tracked，`git add` 属人的动作），不是脚本故障；黑名单零命中、staged=0、HEAD 未变。

**打回 1 处（漏批，不是扩范围）**：`.gitignore` 的 `go.work` / `go.work.sum` 两行仍被忽略。人已批 G-new-1「`go.work` / `go.work.sum` **入库**（放宽 `.gitignore:33-34`）」，h1 Brief 第 23-24 行也写明「可以改 `.gitignore` 放行，跟踪动作仍由人执行」。h1 之所以没改，是照了 coordinator 08:25 的**过期指令**「不要自行放行」——责任在编排侧，不在 worker；已按新口径回派，只改这两行 + 更新 record，status 仍 Partial。

**回派已闭环（我复核）**：`.gitignore` 的 `go.work` / `go.work.sum` 两条规则已删除（`git check-ignore -v` 无输出、`git add --dry-run` 显示 `add 'go.work'` / `add 'go.work.sum'`）；`.gitignore` 净变化 11 insert / 5 delete，其它规则未动；record §5/§7/§8 已同步更正（含 `:42→:40` 行号）。`?? go.work` / `?? go.work.sum` 是「已放行、待人 `git add`」的预期态。h1 最终 status = **Partial**（`thunder/logs` 未入库 + `go build all` 未绿，两项都不是 h1 能在边界内解决的）。

## `--inject` 路径的监督缺口（本轮最该记的教训）

3 个 worker 里 **2 个（h1、h2）干完活不发 `worker_done`**，只在屏上自述；Orca 只能发 `escalation/msg_stop_*`（"ended its turn without sending worker_done"）。也就是说 inject 派发**没有完成保证**：我不读屏就永远不知道它是干完了还是崩了。h2 后来改用 `ask` 问了两个真问题（是否分两 commit、go 指令取哪值），说明协议本身能用，缺的是「收尾必须回报」这一条纪律 —— 已对两个 worker 明文重申。
下一批（h4/h5/h7/h8/h10）如果继续用 codebuddy/qoderclicn，**要么接受这个缺口，要么换受监督的 agent**（但 codex 本机 bootstrap 崩、opencode 凭据坏，见上表）。


## 新发现（2026-11-06 18:0x）：`go build all` 在默认 go1.27.3 下红，和 h2 的「exit 0」证据冲突

**实测（主仓 `e288553`，有未跟踪 `go.work`）**：

```text
$ go version
go version go1.27.3 darwin/amd64
$ go build all
# google.golang.org/grpc/internal/transport
.../google.golang.org/grpc@v1.71.0/internal/transport/handler_server.go:271:18: undefined: http2.TrailerPrefix
exit=1

$ GOTOOLCHAIN=local /usr/local/go/bin/go version   # 1.26.4
$ GOTOOLCHAIN=local /usr/local/go/bin/go build all
exit=0
```

因果链：`app/go.mod` 声明 `grpc v1.71.0` + `x/net v0.44.0`（indirect），但 workspace MVS 实际选到 **`x/net v0.54.0`**；`x/net@v0.54.0/http2/server.go` 的 `TrailerPrefix` 在 `go1.27` 下被构建标签编译掉 → grpc 1.71 编不过。受影响的只有 `app/internal/delivery/grpc{,/interceptor}`，业务包不受影响。

**影响与归属**：① h2 记录里的「`go build all` exit=0」是**工具链相关**的，不能作为 CI 门禁口径；要么钉 toolchain（1.26.x），要么升 grpc / 钉 x/net，要么加 `-tags http2legacy` —— 这会碰 `go.mod`（h2 锁区）或 CI 工具链（h3），必须二选一后写进 h3 Brief。② h1 的「纯跟踪树 `go build all` exit 0」验收在此之前**本来就不可能绿**（还有 thunder/logs 缺失），现在多了一层工具链原因，验收措辞要分开写。

## Notes

- **派发落点缺陷已修复（重要，防再犯）**：`worker-start --worktree new-child` 默认按 repo 缓存的旧 base 开，首轮两个 worktree 落在 **`a94451c`**（`rag`/`authz`/`txdb` 与 h4–h8 Brief 在那上面根本不存在）。取证：`git worktree list` + `git log`；两处均 clean，零工作丢失。处置 = `git merge --ff-only main`（`a94451c` 是 `e288553` 的严格祖先）把分支抬到 HEAD，永久修 = `orca repo set-base-ref --ref main`（已执行，`worktreeBaseRef: main`），单次修 = 建 worktree 时显式 `--base-branch main`
- **agent 可用性实测**：`worker-start --agent qoderclicn/codebuddy` → `agent_unconfigured`（「A configured single-executable agent alias is required」）；但二者 CLI 在 PATH 上且 `terminal create --command` 能起（CodeBuddy v2.180.0 / Orca 认 `qoder-cn`），故改走 **`terminal create` + `orchestration dispatch --inject`**（有 dispatch 记录，但不是 supervised worker 生命周期）。**opencode 凭据已坏**：屏上 `The API key doesn't exist`（Volcano Engine / `ark-code-latest`，OpenCode 1.19.5），所以 h2 换成 **codex**（oauth 可用，配额 24%/33%）；那个坏凭据的 opencode 终端 `term_8fde6c0f…` 已 `worker-abandon`（`stop_unknown` → abandoned），`worker-release` 以 `identity_unproven` 拒绝回收，**资源 retained 未强杀**，等人处置
- **派发实操坑（已踩过一次）**：`terminal create --command codebuddy|qoderclicn` 会带上 `--dangerously-skip-permissions`，但 **`rm -rf` 类命令仍被「High risk operation detected — requires confirmation every time」拦住** → worker 停在 TUI 审批界面，Orca 发 `escalation/msg_stop_*`。coordinator 的处置 = 按 Escape（等价选项 3「tell it what to do differently」）+ `terminal send --enter` 下发指令：**明确禁止 rm -rf**、把越界的探索性实验拉回 Brief 白名单、并重申完成后必须 `worker_done` 附验证输出。首个案例：h1 想试 `go work init` 绝对路径（不在验收项内），已纠回
- **h2 首跳已失败但工作未丢（2026-11-06 08:0x）**：codex worker `ctx_d75244e6e40f` → `state=failed / liveness=exited`，`worker-read` 报 `worker_identity_changed` 读不到原因；但 `harden-h2` 工作区留下了**正确的半成品**（`app/go.mod: gorm 1.25.12→1.31.2`，另把 `go 1.24.6→1.25.0`，`go.sum` 随之变化），未 commit。`go 1.25.0` 不是人批项，已列为 h2 重跑时必须交代或回退的点。（`--retry-of` 实测被拒：Task 已自动回到 `ready`，`retry-of` 只对 failed/blocked 有效 → 改为在原 worktree 直接 `worker-start --task`，见上表）
- **环境实测（影响所有 worker）**：`proxy.golang.org` 从我的会话 curl 也是 **000（无外网）** → 任何 `go mod` 拉新版的动作都会失败；好消息是 `$(go env GOMODCACHE)/gorm.io` 里 **v1.25.12 与 v1.31.2 都已在缓存**，所以 h2 的「回写声明」无需联网，且保留回退能力。Docker 现已可达：`my_go_study_postgres`（healthy，up 18min）+ `my_go_study_redis`；**ES / RabbitMQ / lan-up 栈仍未起**，所以依赖 ES/RabbitMQ 的人证还是 Partial Deferred。h2 worker 已自建 scratch 库 `h2_a_fresh` / `h2_b_lab`（两种形态冒烟），清理时**只删这两个**
- **coordinator 本轮只产出规划文档 + Task DAG，零 `.go` 改动**（`git diff --name-only | grep '\.go$'` 为空）
- **前提变更（2026-11-05 提交 `3f0963e`）**：原盘点「`rag`/`authz`/`txdb`/`schemaboot`/`payment.go`/`migrations`/`plans`/`docs` 未跟踪、HEAD 编不过」已解除；`go build all` 在 HEAD 实测通过。H1 范围收窄为「import 覆盖机检 + `app/worker`（39MB 二进制）杂物 ignore」；风险 R1 标注已解除
- **四处取证更正**已写入 Brief，防止 worker 按错前提干活：
  0. H4 —— 越权是**两处**（`jpush_controller.go:91-93` + `realtime_controller.go:109-112`），不是一处；H6 —— 金额口径是**三套**（整数分 / decimal-as-string / float64），mall 不是分口径，且 `transactions.amount` 在 DB 侧是**裸 `numeric`（无精度）**、全链路零 `Round(_,2)`
  1. H7 —— `configs/config.prod.yaml` **存在**；真问题是 `APP_ENV=production` 找不到同名文件时**静默回退 `config.yaml`（debug + sslmode disable）**（`common/config/config.go:844-852`）
  2. H8 —— `main.go:246` 已把 `paymentSvc` 注入 `membership.NewService`，注册可在包内完成，**零 main.go 改动** → h8 可与 h4/h5/h7 并行
- **新发现且未排 Slice**：`app/internal/rag/` 5 个文件零测试（Program §背景 第 10 条），已显式登记不静默丢弃
- 待处置杂物：`app/worker` 39MB Mach-O 可执行文件（未 ignore，**未删除**，等人决定）
- 并行建议：h1、h2 可同时起步；h3 完成后 h4/h5/h6/h7/h8 五条可同批派发（文件锁互不重叠）
- 遗留人证（沿上一轮）：lan-up / 飞书 / ES / queue / Agent curl 均为 Partial Deferred；Flutter 客服页未做
- scratch 库待清理（Docker 恢复可达后，且只删本轮自建的）：`p01_*` / `p02_it` / `p02_mig` / `p02_smoke` / `p03_it`

## 历史指针

| 时间 | Active | 备注 |
|------|--------|------|
| 2026-11-04 前 | `cs-agent-infra` C1–C4 | C1 `bd116e1` · C2 `089dfa1` · C3 `748e517` · C4；lan-up/飞书/ES/queue/Agent curl 人证 Partial Deferred |
