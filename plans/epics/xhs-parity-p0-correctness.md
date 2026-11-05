# Epic — xhs-parity-p0-correctness（正确性与资金安全）

> Program: [`plans/2026-11-03-xhs-parity-program.md`](../2026-11-03-xhs-parity-program.md)
> 差距来源: [`docs/xhs-benchmark-gap-analysis.md`](../docs/xhs-benchmark-gap-analysis.md) §5.1、§5.4、§5.5、§2.2-I12、§4.8、§7-P0
> Acceptance 口径: 只认 `AGENTS.md` + 本 Epic/各 Slice Brief + `docs/acceptance-records/`（`.harness/rules/dual-truth.md`）

## 差距表

| 能力 | 现状（证据） | 目标 | Slice |
|------|--------------|------|-------|
| 迁移治理 | 四套建表并存，每实例启动跑 DDL，失败仅 `Warn`（`main.go:107`、`:637-651`、8 处 EnsureSchema、`main.go:185`） | DDL 只在发布流程执行；API 启动仅校验 schema 版本，不符则**拒绝启动** | P0-1 |
| 支付闭环 | `/api/v1/payments` 只有 `prepay`（`payment/router.go:37`）；`out_trade_no` 生成即丢（`payment/service.go:84`）；无流水表 | notify 验签 + 流水落库 + 幂等核销 + 主动查单 + T+1 对账 | P0-2 |
| 资金原子性 | 扣积分 `:520`／扣钱包 `:535`／改单 `:541` 三个独立事务；补偿 `_, _ = Credit(...)` 吞错（`:548-553`、`:579`、`:583`） | 一个事务提交或回滚；补偿失败必留痕可重试 | P0-3 |
| 假审核 | 短视频 30s 超时自动判通过（`short_video/service.go:52-58`、`:339-340`） | 默认拒绝 / 人工放行；无「超时=通过」路径 | P0-4 |
| 假实现治理 | 华为回调 `accepted:true`（`membership/handler.go:152-155`）、SMS stub（`worker/handler.go:169-177`）、OTP 空实现（`phone_otp_usecase.go:31-42`）、mock 图/视频（`short_video/service.go:32-39`）、`ConfirmBuyout` 不验支付（`membership/service.go:279`） | 全部登记五态 status + note；未接的**明确 501**，不返回成功 | P0-5 |
| MQ 可靠性 | 失败 `Nack(false,true)` 无限重投（`consumer.go:100`）；无 DLX（`client.go:32` args=nil）；无 publisher confirm（`:64`） | 重试上限 + 死信队列 + 入队失败可见 | P0-6 |
| 密钥治理 | compose 默认 `change-me-in-production`（`docker-compose.yml:94`）；兜底密钥 `config.go:335`；`check-secrets.sh:13` 只匹配 `sb_secret_*` | 启动校验密钥强度；扫描能抓弱值/默认值 | P0-7 |

## Slice backlog（有序）

| # | ID | ONLY 摘要 | 依赖 | Brief 已落盘 |
|---|-----|-----------|------|--------------|
| 1 | `parity-p0-1-migration-gate` | AutoMigrate/EnsureSchema 加 `APP_ENV` 守卫 + schema 版本校验 + 幂等补建迁移 | 无 | ☑ 已验收（Migrated，见 `docs/acceptance-records/2026-11-04-parity-p0-1-migration-gate.md`） |
| 2 | `parity-p0-7-secret-hardening` | 启动期密钥强度校验 + `check-secrets` 增强 + compose 默认值去除 | 无 | ☑ 已验收（Degraded，见 `docs/acceptance-records/2026-11-03-parity-p0-7-secret-hardening.md`） |
| 3 | `parity-p0-3-money-tx-atomic` | service 层 `WithTx` 最小入口 + 商城支付单事务 + 补偿不吞错 + 真库回滚测试 | 无（为 P0-2 提供事务入口） | ☑ 已验收（Migrated，见 `docs/acceptance-records/2026-11-04-parity-p0-3-money-tx-atomic.md`；HTTP 下单-支付-取消冒烟 Deferred 给 P0-2） |
| 4 | `parity-p0-2-payment-notify-ledger` | `wys_payment_records` + notify 路由（免 SessionAuth）+ 幂等核销 + 查单兜底 + 对账任务 | P0-3 | ☐ 待建 |
| 5 | `parity-p0-4-remove-fake-review` | 短视频审核改为默认拒绝/人工放行；删除超时自动通过谓词 | 无 | ☑ 已验收（Degraded，见 `docs/acceptance-records/2026-11-03-parity-p0-4-remove-fake-review.md`） |
| 6 | `parity-p0-5-fake-impl-inventory` | 假实现清单落盘 + status 五态登记 + 未接路径返回 501 + note | P0-4 | ☑ 已验收（Degraded，见 `docs/acceptance-records/2026-11-03-parity-p0-5-fake-impl-inventory.md`） |
| 7 | `parity-p0-6-mq-dlx-retry` | 重试计数 + 上限 + DLX + Nack 不再无脑 requeue + confirm 或 outbox 兜底 | 无 | ☑ 已验收（Degraded，见 `docs/acceptance-records/2026-11-03-parity-p0-6-mq-dlx-retry.md`） |

> 执行顺序另见 Program「执行顺序与并行建议」：1、2 可并行；3→4 串行；5→6 串行；7 独立。

## 文件锁

| Slice | 主写 | 勿抢 |
|-------|------|------|
| P0-1 | `app/cmd/api/main.go`、`common/config/config.go`、`migrations/`、`Makefile`、`docker/docker-compose*.yml`、`scripts/` | 勿改业务 usecase 逻辑；勿删已有迁移 |
| P0-2 | `app/internal/payment/`、`app/internal/mall/`（委托 payment）、`model/entity/payment*.go`、`migrations/`、`cmd/worker` 对账任务 | 勿改 SessionAuth 语义（notify 路由是**公开**回调，必须验签而非靠鉴权） |
| P0-3 | `app/internal/mall/service.go`、`app/internal/mall/repo_postgres.go`、`app/internal/{wallet,points}/`（事务传递）、新增 `app/internal/txdb/` | 勿动 P0-1 的启动流程 |
| P0-4 | `app/internal/short_video/service.go`、`configs/*.yaml`（review 开关）、`short_video/handler.go` | 勿顺手做机审（那是 P2-2） |
| P0-5 | 新增 `docs/placeholder-inventory.md` + 各 stub 所在文件**只改返回语义** | 勿实现被登记为 Deferred 的能力 |
| P0-6 | `core/queue/consumer.go`、`core/queue/client.go`、`core/queue/tasks.go`、`app/internal/delivery/worker/handler.go`、`configs/` | 勿换中间件、勿引入 Asynq（AGENTS.md 规范 9） |
| P0-7 | `scripts/check-secrets.sh`、`common/config/config.go`、`docker/docker-compose.yml`、`.env.example`、`.env.lan.example` | 勿把任何真实密钥写进可提交文件 |

---

## 各 Slice 任务拆解与验收标准

### P0-1 `parity-p0-1-migration-gate`

**任务**
1. 新增配置 `database.auto_migrate`（默认 `false`）与 `database.schema_version_check`（默认 `true`），按 `config.go:647-716` 现有 `yaml + BindEnv + 默认值` 三件套写法。
2. `main.go:107` 的 AutoMigrate 调用与 `main.go:637-651` 白名单、8 处 `EnsureSchema` 全部收口到**一个**入口函数（建议 `app/internal/schemaboot/`），由 `auto_migrate` 单一开关守卫；`APP_ENV=production|lan` 时强制关闭。
3. 建 `schema_migrations` 版本读取与校验：启动时比对 `migrations/` 已应用版本，缺失则**返回 error 退出**，替换现在 `main.go:185` 的 `Warn` 继续。
4. 出一份 schema 对齐审计：把四套机制各自建过的表/列/索引 diff 到同一份，落成 `migrations/` 里**幂等**的补建文件（`CREATE TABLE IF NOT EXISTS` / `ADD COLUMN IF NOT EXISTS` / `CREATE INDEX IF NOT EXISTS`）。
5. 补 `migrations/20260924100000_rongcloud_im` 缺失的 `.down.sql`。
6. 在 compose 增加一次性 `migrate` 服务（`migrate up` 后退出），API `depends_on: condition: service_completed_successfully`；`make lan-up` 保持可用。
7. 文档：`docs/local-auth-postgres.md` 或新增 `docs/migration-workflow.md` 写明「加表怎么走、加索引怎么走、本地逃生门 `make migrate-up`」。

**验收标准（全部满足才 Accept）**
- [x] `go build all` 通过；`go -C app build ./cmd/api` 与 `go -C app build ./cmd/worker` 均通过。
- [x] `APP_ENV=production`（或 `auto_migrate=false`）启动时，日志中**无任何** DDL 语句、无 AutoMigrate 记录；用 `pg_stat_activity`/general log 佐证。
- [x] 人为把库内版本改低一位 → 进程**非 0 退出**且错误信息指出缺失的迁移名；此时 `curl http://127.0.0.1:8080/health` 连接被拒（不是 200）。
- [x] 空库一条命令到位：`make migrate-up` 后 `make run` → `GET /health` 200（AGENTS.md 检查项），且 55 实体对应表全部存在（脚本清点 `information_schema.tables`）。
- [x] 起 **两个** API 实例并发启动（不同端口），无 DDL 锁等待、无迁移竞态报错（这是 P2-7 的前置证据）。
- [x] `migrations/` 每个 up 都有对应 down（除历史遗留，须在 acceptance-record 里点名）。
- [x] `make check-secrets` 通过。
- [x] `docs/acceptance-records/2026-11-<date>-parity-p0-1-migration-gate.md` 落盘，含 Context Card；未完成项标 `Deferred` + note。

实机证据：[`docs/acceptance-records/2026-11-04-parity-p0-1-migration-gate.md`](../../docs/acceptance-records/2026-11-04-parity-p0-1-migration-gate.md)
（G1–G9 九道门；G8 是补测出来的新漏洞 —— casbin `gorm-adapter` 会绕过闸门自建 `casbin_rule`，已用
`schemaboot.HasTable` 挡住）。compose 全栈 `make lan-up` 端到端记 **Partial**，其余残留见记录末尾表。

**status 口径**：若生产守卫只做在 `APP_ENV` 判断而未做版本校验 → `Degraded` + note，不得算 Migrated。

### P0-7 `parity-p0-7-secret-hardening`

**任务**
1. `scripts/check-secrets.sh`（现 `:13` 只匹配 `sb_secret_*`）增强：默认值字典（`change-me-in-production`、`postgres`、`secret`、`123456`）+ 长度 < 16 告警 + 高熵串检测。
2. `docker-compose.yml:94`、`:146` 去掉 `${JWT_SECRET:-change-me-in-production}` 兜底，改为**必需变量**（`${JWT_SECRET:?...}`），缺失时 compose 直接失败。
3. `common/config/config.go:335` 的硬编码兜底密钥删除；生产环境缺失即启动失败（与 P0-1 的退出策略一致）。
4. 启动期加 `validateSecrets()`：`APP_ENV != local` 时校验 JWT secret 长度/字符集、DB 密码非默认、`SUPABASE_SERVICE_ROLE_KEY` 只允许来自 `.env.local`。
5. `docs/project-detail.md` H11 结论更新（该风险由本文档记录，落地后应回写为「已缓解」）。
6. `.env` 里现存弱默认值属**已入库**内容（AGENTS.md 记录 `.env` 是有意提交的），需在 `.env.example` 与 `docs/startup-guide.md` 明确「本地可弱、部署必改」，并在 README 检查项加一条。

**验收标准**
- [x] `JWT_SECRET` 未设置时 `make lan-up` 在 compose 层即失败并给出可读错误（不是起一个用默认密钥的服务）。
      实测（同一对 `-f` 文件，`make lan-up` 走的就是 `scripts/lan-compose.sh` 里这两条）：
      `error while interpolating services.app.environment.JWT_SECRET: required variable JWT_SECRET is missing a value: JWT_SECRET 必填…` exit=1；
      4 处兜底（`docker-compose.yml:94,146`、`docker-compose.lan.yml:34,54`）全部改为 `${JWT_SECRET:?…}`。
      ⚠️ 字面 `make lan-up` 演示需临时注释 `.env.lan` 里的 `JWT_SECRET`（该脚本 `source .env.lan` 会把它导出）——没动你的本地文件，改用 compose 插值层等价验证。
- [x] 故意用 `JWT_SECRET=change-me-in-production` 启动 → 进程拒绝启动并打印原因。
      实测 `APP_ENV=prod` exit=1 + `prod 启动校验失败：JWT_SECRET 仍是占位默认值，请换成 ≥32 位随机串`；
      合格密钥则前进到 `DATABASE_PASSWORD` 那道拒绝（说明 JWT 关已过）；空值走 `jwt.secret 不能为空`（`configs/config.yaml` 兜底已删为 `""`）。
- [x] `make check-secrets` 对以下样例**全部报错**：`change-me-in-production`、`postgres`、长度 8 的 secret、`sb_secret_...`（回归旧能力）。
      四类 + 额外两类（高熵真值疑似泄漏、值含空白）全部 `❌` + exit 1；仓库自身默认只给 2 条 ⚠️（`.env` 是有意入库的 lab 文件）且 exit 0，
      `CHECK_SECRETS_STRICT=1` 把提示升级为失败。`*.example` 与设计上公开的 `*_APP_ID`/anon key 不误报。
- [ ] `git grep -n "change-me"` 在可提交文件中仅剩文档说明，无配置生效点。
      ✗ 未完全达成：`configs/config.yaml` 兜底与 compose 兜底已清零，但 `.env:9 JWT_SECRET=change-me-in-production` 仍在，
      且它对 dev 是**生效值**——改它属破坏性动作（Brief 的「不做：不改 `.env` 已入库的历史弱值，需先与人确认」与本条冲突）。
      剩余命中只有 `.env` 与 `scripts/check-secrets.sh` 的检测字典本身。需要轮换请单开一次改动 + 双端同步发版。
- [x] 现有登录链路不回归：`POST /api/v1/user/login` 正常（AGENTS.md 检查项），dev 用 `13400000000/123456` OTP 仍可登录（dev 固定码属 P0-5 登记项，本片不动）。
      实测 dev 实例 `:8083`：`/health` 200、`otp/verify` `code=0` + token 291 字符；启动日志出现 `密钥体检 …` Warn。
      注：`POST /api/v1/user/login`（邮箱密码）在本机会 502，因 `.env` 未配 `AUTH_PROVIDER` 时走 Supabase 云且本网络不可达——与本片无关，用 `AUTH_PROVIDER=local` 的 OTP 链路做的回归。
- [x] 未把任何真实密钥写入可提交文件；acceptance-record 落盘。
      负样例只写在 `/tmp`；→ [`docs/acceptance-records/2026-11-03-parity-p0-7-secret-hardening.md`](../../docs/acceptance-records/2026-11-03-parity-p0-7-secret-hardening.md)（status=**Degraded**：残留 `Makefile` 的 `postgres:postgres`、prod 全链路只验到 config 放行、`.env` 弱值按 Brief 保留）

### P0-3 `parity-p0-3-money-tx-atomic`

**任务**
1. 新增最小事务传递能力（建议 `app/internal/txdb/`）：`WithTx(ctx, fn)`，把 `*gorm.DB`（事务句柄）放进 context，仓储层「有则用、无则用默认」。**只做资金路径所需，不做全域推广**（推广是 P2-9）。
2. `mall/service.go` 的支付编排（`:520` 积分 Debit、`:535` 钱包 Debit、`:541` `PayOrderLocal`）合并进同一 `WithTx`。
3. 失败路径改为**同事务回滚**，删除/重写 `restorePayDebits`（`:548-553`）：不再用 `_, _ =` 丢弃错误；若仍保留补偿分支（跨服务场景），补偿失败必须落库 + 记 error 日志 + 计数指标。
4. 同样处理取消订单退款分支（`:579`、`:583`）。
5. 补**真实 Postgres** 集成测试（testcontainers 或本机 `make deps-up` + build tag）：构造「钱包扣款成功、改单失败」场景，断言余额与积分已回滚、订单未变更。这一条不等 P2-10，本片必须自带（R2 风险）。
6. 幂等复核：同一 `Idempotency-Key` 重复支付请求不得重复扣减（现有键机制 `mall_schema.up.sql:98` + `mall/service.go:271-275`）。

**验收标准**
- [x] `go -C app test ./internal/mall/... ./internal/wallet/... ./internal/points/...` 全绿；其中**至少一条测试连真实 PG**（记录连接方式与清理策略），非纯内存 fake。
      `go -C app test ./internal/{mall,wallet,points,txdb}/... -count=1` 全 `ok`；真库用例在 `app/internal/mall/service_money_tx_integration_test.go`（`//go:build integration`），
      连本机 `my_go_study_postgres` 容器、独立 scratch 库 `P03_PG_DBNAME`（默认 `p03_it`），`t.Cleanup` 逆序 DropTable，连不上则 `t.Skipf`。
- [x] 回滚用例：注入改单失败后，`wys_cash_wallet`/`wys_points_wallet` 余额 = 支付前值，`wys_mall_order.status` 未变（断言在测试里，不是日志）。
      `TestMoneyTxRollbackOnOrderFailure` 三条断言（余额、积分、订单状态）均在测试内。
- [x] 并发用例：同一订单两个并发支付请求 → 只扣一次款（另一条得到已支付/幂等结果，不是双重扣减）。
      `TestMoneyTxConcurrentPayDebtsOnce`；顺带修掉原实现「重复支付二次扣款」的真实资金 bug（锁行后判 `alreadySettled`）。
- [x] 代码内不再出现 `_, _ = u.pointsSvc.Credit` / `_, _ = u.walletSvc.Credit`（`git grep` 证据）。
      2026-11-04 复验：`git grep -E '_, _ = .*(Credit|Debit|Refund)' -- app` 全仓仅剩 `wallet/service_test.go:249`
      （测试自身丢弃返回值）；membership 两处当时在包外未动，已由 follow-up 清零。
      **限定范围**：`app/internal/{mall,wallet,points}/` 已 0 命中；`app/internal/membership/service.go:228`、`:233` 仍有同款吞错补偿，属 P0-3 白名单外，另立 Task #12 跟进。
- [x] 事务内**无**外部 HTTP/ES/MQ 调用（人工核对 + 在 record 里列出事务边界）。
      边界清单见 [`docs/acceptance-records/2026-11-04-parity-p0-3-money-tx-atomic.md`](../../docs/acceptance-records/2026-11-04-parity-p0-3-money-tx-atomic.md)：锁订单行 → 扣积分 → 扣钱包 → 改单 → 写流水，全部为同库 SQL。
- [ ] `/health` 200，商城下单-支付-取消冒烟通过（`scripts/` 或手工 curl，写入 record）。
      `/health` 200 已在新二进制 + scratch 库上实测；**下单-支付-取消 HTTP 冒烟未跑**（需业务库 seed 商品与登录态，且会把真实 DDL 打到 dev 库）→ 与 P0-2 渠道支付改造一并做，登记为 Deferred。
- [x] acceptance-record 落盘；若取消退款分支未完全改造 → `Degraded` + note。
      → [`docs/acceptance-records/2026-11-04-parity-p0-3-money-tx-atomic.md`](../../docs/acceptance-records/2026-11-04-parity-p0-3-money-tx-atomic.md)（status=**Migrated**：取消退款分支已进同一事务且错误必返回，新增 `ErrRefundUnavailable`；残留只有 membership 与上述 HTTP 冒烟）

### P0-2 `parity-p0-2-payment-notify-ledger`

**任务**
1. 新表 `wys_payment_records`（`out_trade_no`、`user_id`、`channel`、`amount_fen`、`status`、`notify_raw jsonb`、`paid_at`、`ref_type/ref_id`）+ `wys_payment_reconcile_diff`（对账差异）。金额一律**整数分**（现 `entity.ParseYuanToFen` 已有，沿用）。
2. `prepay` 落库：`payment/service.go:84` 生成的 `out_trade_no` 必须持久化并与业务单关联，不再「生成即丢」。
3. 回调路由 `POST /api/v1/payments/notify/{channel}`（微信 v2 / 支付宝 RSA2）：**跳过 SessionAuth**（公网回调），改为**验签 + 商户号校验**；在 `payment/router.go` 内注册，不污染主 `router.go`。
4. 核销幂等：`notify` 重复投递不得重复发货/重复标记（唯一索引 `out_trade_no` + 状态机单向流转）。
5. 主动兜底：查单任务（复用 Worker，`queue.enabled` 下）对「prepay 未终态 > N 分钟」的单据主动查渠道；T+1 对账任务比对账单与本地流水，差异写 `wys_payment_reconcile_diff` 并告警。
6. 把 `mall` 的渠道支付改为**委托 payment 域**（现在 `mall/service.go:526-543` 选微信/支付宝时直接本地标记已付）。
7. 处理 `membership`：`ConfirmBuyout`（`membership/service.go:279`）不再无校验 `MarkOrderPaid`——微信/支付宝渠道必须已有已核销流水；Apple/华为验签路径维持，华为回调归 P0-5 登记。
8. 验签失败、金额不符（渠道金额 ≠ 本地 `amount_fen`）**必须拒绝**并留痕。

**验收标准**
- [x] `go -C app test ./internal/payment/...` 全绿，含验签正/负用例（用固定测试密钥，勿用生产密钥）与金额不符用例。
- [x] 端到端（可用沙箱或本地伪造签名 + 白名单开关）：prepay → 落库 `pending` → notify 验签通过 → `paid` + 订单状态推进 + 通知发出。**每一步数据库状态可查**。
      注：**「通知发出」子项未实现** —— 本仓支付链路（含改造前的本地结单）从来不调推送；已登记残留 F7，不由本片冒充完成。
- [x] notify **幂等**：同一通知重复投递 3 次，只核销一次、只发货一次（断言，非日志）。
- [x] notify 篡改签名 → 4xx 且不改变单据状态；金额不符 → 拒绝 + 落差异。
      注：**HTTP 状态按渠道协议恒 200**（微信回 `<return_code>FAIL`、支付宝回 `failure`），非 4xx —— 4xx 会让渠道无限重投。
      拒绝的强度改为断言「单据状态不变 + `payment_notify_reject_total{reason="sign"}` 计数 +1」，见 acceptance-record §一.1。
- [x] 未登录（无 `X-Session-ID`）可访问 notify（这是回调路由正确性），但**业务接口未受影响**：`/api/v1/transactions*` 仍需 SessionAuth（回归检查）。
      证据：`TestPrepayRouteStillRequiresSession`（同前缀业务路由仍 401）+ `domain_routes_test.go` 路由基线（transaction 两组路由全在 SessionAuth 组内）。
- [x] `mall` 走微信渠道时 `wys_payment_records` 有对应流水；不再出现「无流水的已付订单」（SQL 校验）。
      注：SQL 在空库 `p02_smoke` 跑通口径但 **0 行属空集**，真证据是 `TestChannelNotifySettlesMallOrder` 两侧同时断言。
- [x] 查单兜底任务：造一条 `pending` 超时单 → 任务跑后状态收敛（人证或测试）。
      收敛三分支由 `TestQueryStalePending{SettlesLostCallback,ClosesFailedOrders,KeepsUnknownOrders}` 证明；
      真 broker 冒烟只跑到「入队 → 消费 → 扫到 1 条 → 查单失败保持 pending」（环境无商户密钥），见 record §四。
- [x] 对账任务产出差异表并可被告警消费（先接现有 Alertmanager `deploy/observability/alerts.yml` 的接收通道）。
      注：`reconcile_enabled` 在冒烟时为 false，**未跑到真实 T+1 整点**；差异表与 `PaymentReconcileDiffFound` 指标名逐字对齐（record F6）。
- [x] `make check-secrets` 通过；acceptance-record 落盘；若只做了 notify 未做对账 → `Degraded` + note。
      对账做了（单向），`local_missing` 为 **Deferred**（F3）；整体 status 记 **Migrated + 7 条残留**。

### P0-4 `parity-p0-4-remove-fake-review`

**任务**
1. 删除「超时即判通过」谓词（`short_video/service.go:339-340`）与其调用点；`reviewDelay` 配置语义改为「最长待审时长」或**彻底移除**（推荐移除，配置项 `configs/*.yaml` 同步）。
2. 审核默认策略改二选一：`review.mode = reject_until_manual`（默认，需人工放行）或 `review.mode = auto_allow_local_only`（仅 `APP_ENV=local` 生效，供演示）。
3. 入库不再自动 `reviewing → approved`（`:160` 保留置 `reviewing`）；放行只走人工接口（最小实现：`POST /api/v1/short-videos/{id}/review` 带角色校验，或先用 `cmd/` 手工命令）。
4. Feed/列表查询侧：未通过审核的内容对**非作者**不可见（现在是否泄漏需先查 `short_video/repo_postgres.go` 的可见性过滤）。
5. 在 `docs/placeholder-inventory.md`（P0-5）登记为「已从 Placeholder 改为显式未实现」。

**验收标准**
- [x] 代码中不存在「按时间自动置 approved」的路径（`git grep reviewDelay` 证据）。
- [x] 新上传短视频 → 状态 `reviewing` → **等待 > 原 30s 后仍为 `reviewing`**（这是本片最关键的反证测试）。
- [x] 未过审内容：作者自己可见、他人列表不可见（两个账号人证或测试断言）。
- [x] 人工放行接口有权限校验（非任意用户可放行；用 `accountGate`/role 现成机制）。
- [x] `APP_ENV=local` 演示模式仍一键可用（`make run` + 客户端能看到内容），不阻塞联调。
- [x] acceptance-record 落盘。

### P0-5 `parity-p0-5-fake-impl-inventory`

**任务**
1. 新建 `docs/placeholder-inventory.md`，按 `.harness/rules/status-enums.md` 五态登记每一条：`Migrated/Degraded/Placeholder/Deferred/Removed` + note。
2. 首批必录（均来自差距文档，逐条给 owner 与解除条件）：华为 IAP 回调 stub（`membership/handler.go:152-155`）、SMS stub（`worker/handler.go:169-177`）、手机 OTP 发送空实现 + dev 固定码（`phone_otp_usecase.go:31-42`）、mock 图/视频 URL（`short_video/service.go:32-39`、`community/service.go:30-37`）、LLM mock provider 与无 key 回退（`repository/llm/mock_stream_provider.go`、`main.go:608-616`）、客服 Agent 不调 LLM（`cs_agent_usecase.go:98,153`）、analytics 演示种子（`analytics_usecase.go:37-49`）、各域 seed（`community/repo_postgres.go:371-434`、`short_video/repo_seed.go`、`mall/repo_seed.go`）、占位邮箱 `wx_/hw_` 域名（`local_auth_usecase.go:252,326`）、JPush 占位分支。
3. 语义修正（代码动作，很小）：**未真实实现却返回成功**的分支，改为返回 `501 Not Implemented` 或业务码「未开通」，并在响应中说明；`Degraded` 类保留成功返回但加 note + 指标计数。
4. dev 固定 OTP/seed 加 `APP_ENV=local|dev` 守卫，生产环境不可用（与 P0-7 启动校验同一入口）。
5. 挂一条 CI/脚本可检项（可选，若成本高分 Deferred）：扫描 `TODO|FIXME|mock|stub` 输出到清单，防新增未登记项。

**验收标准**
- [x] `docs/placeholder-inventory.md` 覆盖上述 10+ 条（实得 17 条 + 4 条指向其它 Slice），每条有：文件:行、status、note、影响面、解除条件、对应 Slice。
- [x] 清单与代码一致：抽查 3 条（华为回调 501、SMS Warn+ACK、OTP dev/prod 行为）与登记 status 相符。
- [x] 正式环境（实测 `APP_ENV=prod`）下：OTP 固定码 `123456` 不可登录（400/10001）；10 处 seed 全部跳过；mock 媒体不被填充而是 400。
      ⚠️ 字面上的 `APP_ENV=production` 目前**起不来**：`config.Load` 要求 `configs/config.production.yaml`，仓库只有 `config.prod.yaml`（已登记为坑，归 P0-7/P0-1 收口）。
- [x] 未实现的回调不再返回成功语义：dev/prod 双实例 curl → `501` + 业务码 `10005`。
- [x] LLM 无 key：启动 `Warn(未配置 LLM …status=degraded)`，SSE 仍可用但 `degraded_path_total{name="llm_mock"}` 计数可见（客服模板同口径）。
- [x] Program checklist 与本清单交叉核对一次（清单末节），未发现新增未登记项。
- [x] acceptance-record 落盘：`docs/acceptance-records/2026-11-03-parity-p0-5-fake-impl-inventory.md`（status=Degraded + note）。

### P0-6 `parity-p0-6-mq-dlx-retry`

**任务**
1. 消费失败不再无条件 requeue：`core/queue/consumer.go:100` 的 `_ = d.Nack(false, true)` 改为读 `x-delivery-count` 或自定义重试头，超过 `queue.max_retries` 走 `Nack(false,false)` 投死信；**且检查 Nack/Reject 自身返回的 error**（现在被丢弃）。
2. 声明 DLX：`client.go:32` 的 `QueueDeclare` args 由 `nil` 改为 `{x-dead-letter-exchange, x-dead-letter-routing-key}`；新增 `task.dlq` 队列与绑定；**注意队列参数不可变**——需同时提供改名迁移或 `scripts/` 重建说明（旧队列存在时声明会 `406`）。
3. 死信可见：DLQ 长度暴露为 Prometheus gauge（复用 `common/metrics/metrics.go` 注册风格）；`deploy/observability/alerts.yml` 加「DLQ > 0 持续 5m」告警。
4. 发布侧：开启 publisher confirm（`client.go:64`），或明确记录「入队失败即返回 error 给调用方」并核对现有调用点是否处理了 error（不要静默）。
5. 毒丸防护：单条消息处理加超时（`context.WithTimeout`），避免一条卡死 prefetch 窗口。
6. Worker 优雅退出补在途等待：`app/cmd/worker/main.go:132-135` 仅 `cancel()`，改为 cancel + `WaitGroup` 等在途任务 + 再 close channel（对齐 HTTP 侧 `main.go:574-592` 的做法）。
7. 重投幂等：确认 4 类任务各自**重复投递不产生重复副作用**（推送去重现靠 `worker/handler.go:136` 的 SetNX，覆盖是否完整需核对）。

**验收标准**
- [x] `go -C core test ./queue/...`、`go -C app test ./internal/delivery/worker/...` 通过；`go build all` 通过。
      ⚠️ 两条口径替换：`go build all` 被**先于本片**存在的 `github.com/mszlu521/thunder/logs` 缺目录卡死 → 改跑
      `go -C {app,core,common} build ./...` + `go -C app vet ./...`，均 exit 0。
- [x] 注入必然失败的消息 → 前 N 次 requeue，第 N+1 次进 DLQ；**不再无限循环**（`rabbitmqctl list_queues` 消息数不无限增长，附截图或输出）。
      实测 `finished_attempt 1→2→3` 后进死信，`my_go_study.tasks` 停在 **0**，DLQ 留 `x-retry-count=3` + `x-death`。
      注：机制不是「requeue N 次」而是**应用侧改投带 `x-retry-count` 的新消息**——RabbitMQ 对 requeue 重投不写任何计数头。
- [ ] 旧队列已存在时 `make lan-up` **不因 406 启动失败**（验证参数变更的兼容路径）。
      ⚠️ 未跑：本机 `:8080` 被无关进程 `super-fac`(PID 40935) 占用，compose 端口必冲突（未动它）。
      等价证据已取：把队列删成无 args 形状后，**API** 与 **Worker** 进程各自 `Warn(status=degraded)` + `health=200` 正常消费，
      `recreate --yes` 后 `dlq_wired:true` 恢复。缺口仅为 compose 一次性通过。
- [x] DLQ gauge 在 `/metrics` 可见（`curl -s http://127.0.0.1:8080/metrics`，AGENTS.md 检查项）且告警规则加载成功（Alertmanager `/api/v2/alerts` 或 `amtool check-config`）。
      实测 `rabbitmq_dlq_messages{queue="my_go_study.tasks.dlq"} 3` 与 broker 数字一致；停 broker 后指标**整条消失**（不是假的 0）。
      规则校验用 `promtool check rules` → `SUCCESS: 5 rules found`（`amtool check-config` 不认顶层 `groups:`，那是 alertmanager 自身配置）。
- [x] 入队失败（停掉 RabbitMQ 后调用发送接口）：API **不返回成功**，且不留「已入队」假象。
      实测 `POST /api/v1/realtime/push` → `500 {"error":"enqueue push notify: Exception (504) Reason: \"channel/connection is not open\""}`；
      正向对照同进程同账号 `queued:true` + 200；`scheduler-trigger` → exit 1。publisher confirm 已默认开启。
- [x] Worker SIGTERM：在途任务完成后退出，日志可见等待过程；期间无消息被重复消费或丢失（发 N 条 → 消费 N 条）。
      实测 prefetch=1 连投 6 条 → 立即 TERM → 6 条全部处理、队列 0、`收到关机信号` 早于 `Worker 已关闭`。
      ⚠️ 口径说明：6 条耗时 <1ms，未真正制造「关机瞬间在途」的重叠；该分支由 `consumer.Wait()` + 单测覆盖。
- [ ] 重复投递 4 类任务各一条，副作用只发生一次。
      ✗ 复核结论：`realtime:push_notify` **不满足**（at-least-once，重试多推一帧 WS）；broadcast 靠 SetNX 满足、sms 无副作用、jpush 是 upsert。
      服务端 `task_id` 去重属改投递语义（Brief「不改任务类型语义与载荷结构」）→ 登记为待办 1。
- [x] acceptance-record 落盘；若 DLQ 仅声明未接告警 → `Degraded` + note。
      → [`docs/acceptance-records/2026-11-03-parity-p0-6-mq-dlx-retry.md`](../../docs/acceptance-records/2026-11-03-parity-p0-6-mq-dlx-retry.md)（status=**Degraded**：compose 闸门未跑、超时缺实机证据、push 去重 Deferred；白名单偏差 2 处已点名）
