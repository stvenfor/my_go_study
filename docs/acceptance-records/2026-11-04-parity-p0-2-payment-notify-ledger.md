# Acceptance Record — parity-p0-2-payment-notify-ledger

- 日期：2026-11-04
- Slice：`plans/slices/parity-p0-2-payment-notify-ledger.md`
- Epic：`plans/epics/xhs-parity-p0-correctness.md` §P0-2
- 主交付物：`wys_payment_records` / `wys_payment_reconcile_diff`（迁移，非 AutoMigrate 建表）
  + 公开回调路由 `POST /api/v1/payments/notify/{wechat,alipay}`（验签 + 商户号 + 金额校验）
  + 核销幂等（唯一索引锚 + 状态单向流转）+ 查单兜底 / T+1 对账（RabbitMQ 任务）
  + `mall` 渠道支付委托 payment 域、`membership` 渠道购买需已核销流水

## status: **Migrated**（流水 / 回调 / 核销 / 查单 / 单向对账）＋ 7 条残留登记在末尾

---

## 一、先说清楚「和验收标准字面不一样」的两处

### 1. 篡改签名不是 4xx，是 **200 + 正文 FAIL**（刻意偏离 epic 原文）

epic 第 148 行写「notify 篡改签名 → 4xx 且不改变单据状态」。实现按**渠道协议**来：

| 渠道 | HTTP 状态 | 正文 | 依据 |
|------|-----------|------|------|
| 微信 v2 | **恒 200** | `<return_code>FAIL</return_code>` + `<return_msg>` | 微信文档约定：非 200/非 SUCCESS 会被**无限重投**，5xx 更是雪崩来源 |
| 支付宝 | **恒 200** | 纯文本 `failure` | 同上，支付宝以 body 字符串判定 |

所以验收改成断言「**拒绝 + 不落状态 + 打点**」，比 4xx 更强：
`TestWechatNotifyTamperedSignatureRejected`（状态仍 pending、`NewlySettled=false`）、
`TestWechatNotifyRouteIsPublicAndRepliesSUCCESS` / `TestWechatNotifyTamperedRepliesFAILAndCountsReject`
（HTTP 层：从 `/metrics` 抓 `payment_notify_reject_total{channel="wechat",reason="sign"}` 的**增量**，
断言计数器真的 +1，而不是只看返回值）。

### 2. Flutter 契约变了，客户端**还没跟**（本片未跨仓改动）

`POST /api/v1/mall/orders/:id/pay` 选微信/支付宝渠道时，`data` 不再是订单详情，而是：

```json
{"status":"pending","out_trade_no":"...","prepay_channel":"wechat","prepay_params":{...}}
```

本地结单（钱包/积分）分支仍返回 `data = MallOrderDetail`，未受影响。

已核对客户端现状（`my_ai_project` 只读，未改）：
`features/mall/lib/mall/api/mall_api.dart:181-197` 把 pay 响应用 `MallOrderDetail.fromJson` 解析；
`mall_order.dart:116-122` 对缺失字段是**软降级**（拿空 map），
`mall_order_detail_controller.dart:107-111` 随后直接 toast「支付成功」。
→ 症状是**误导性的成功提示 + 空订单详情**，不是崩溃。订单本身仍是待支付，钱没扣、权益没发，
**不会造成资损**，但渠道支付在 App 上不可用，必须做 F1（见末尾残留表）。

---

## 二、验收标准逐条（只勾有证据的）

- [x] **`go -C app test ./internal/payment/...` 全绿，含验签正/负与金额不符**
  30 条非 tag 用例 + 3 条 `-tags=integration` 用例。验签正/负、金额不符、商户号不符、
  未知单号、缺密钥必拒、弱 `sign_type` 必拒、支付宝 content 排序且排除 `sign`/`sign_type` 都锁死：
  `TestWechatNotify*`（6 条）、`TestAlipayNotify*`（6 条）、`TestSettle*`（3 条）、`TestPrepay*`（3 条）、
  `TestQueryStalePending*`（5 条）、`TestReconcile*`（2 条）、`TestWechatNotifyThreeDeliveriesSettleOnce`。
  测试密钥全部为用例内固定串（`testMchAPIKey` / 测试 RSA 密钥对），无生产密钥。
- [x] **端到端每步可查库**：prepay 落 `pending` → notify 验签 → `paid` + 订单推进 + 发货，
  在**真 PG16** 上跑通：`app/internal/mall/payment_ledger_integration_test.go`
  （`TestChannelPayWritesPendingLedger` / `TestChannelNotifySettlesMallOrder`：
  断言 `wys_payment_records` 状态、`wys_mall_payment` 1 条、库存 5→4、`NewlySettled`）。
  notify 用**独立实现的 MD5 签名器**（`signedWechatNotify`），顺带把签名算法本身锁住。
- [x] **幂等：重复投递 3 次只核销 1 次**（断言，非日志）
  `TestWechatNotifyThreeDeliveriesSettleOnce`（第 1 次 `NewlySettled=true`，后两次 false）
  + 真库并发版 `TestPostgresConcurrentNotifySettlesOnce`（两路并发只有 1 条 paid、发货 1 次）。
- [x] **篡改签名 / 金额不符 → 拒绝且状态不变**
  `TestWechatNotifyAmountMismatchRejected`、`TestAlipayNotifyTamperedAmountRejected`、
  真库 `TestChannelNotifyTamperedAmountKeepsUnpaid`、`TestPostgresConcurrentAmountMismatchKeepsPending`。
- [x] **notify 免会话可达，业务接口未被削弱**
  `TestWechatNotifyRouteIsPublicAndRepliesSUCCESS`（无 `X-Session-ID` 可通）
  + `TestPrepayRouteStillRequiresSession`（同前缀 `/api/v1/payments/prepay` 仍 401）
  + 路由基线 `app/internal/delivery/http/router/domain_routes_test.go`：
  批 3 六域 sessionAuth=nil 时**只剩**两条 notify（`want` 白名单显式列出），多一条注册就红。
- [x] **走微信渠道的商城订单必有流水**
  SQL 校验（列名与关联口径已在真库确认，`ref_type='mall_order'` / `ref_id=order_id::text`）：

  ```sql
  SELECT o.order_id, o.order_no, o.status, o.payment_channel, r.out_trade_no, r.status
  FROM wys_mall_order o
  LEFT JOIN wys_payment_records r
    ON r.ref_type = 'mall_order' AND r.ref_id = o.order_id::text
  WHERE o.status = 1 AND o.payment_channel IN (1,2)
    AND (r.out_trade_no IS NULL OR r.status <> 1);
  -- (0 rows)
  ```

  **诚实口径**：这条跑在 `p02_smoke`（真迁移链建起来的空库），库里没有渠道已付订单，**0 行属空集，
  不能当证据**。真正的证据是同一条不变式在 `TestChannelNotifySettlesMallOrder` 里两侧同时断言
  （订单 paid **且** 流水 paid **且** `wys_mall_payment` 有 `out_trade_no`）。
  旧「本地标记已付」路径已从 `mall/service.go` 删除，微信/支付宝分支只剩委托 `payment.Prepay`。
- [~] **查单兜底：pending 超时单收敛** —— **Migrated（测试）+ Partial（线上形态）**
  收敛三种结局都有测试：`TestQueryStalePendingSettlesLostCallback`（补核销）、
  `TestQueryStalePendingClosesFailedOrders`（`CLOSED/PAYERROR/REVOKED` → failed，避免每轮重扫）、
  `TestQueryStalePendingKeepsUnknownOrders`（渠道未明确 → 保持 pending）。
  真 broker 侧做了冒烟（命令见第四节）：ticker 到点 → `client.Enqueue` → RabbitMQ
  → worker Dispatch → 扫到 1 条 → Warn。**但**冒烟环境没配商户密钥，查单调用失败，
  单据**停留在 pending（fail-safe）**，没有真的收敛成 paid —— 收敛本身只由上面 3 条测试证明。
- [~] **对账产出差异表并可被告警消费** —— **Migrated（进程内）+ Deferred（当日整点实跑）**
  `TestReconcileWritesDiffsAndLateSettles`、`TestReconcileRerunDoesNotDuplicateDiffs`
  + 真库唯一性 `TestPostgresReconcileDiffUnique`。指标 `payment_reconcile_diff_total` 与
  `deploy/observability/alerts.yml` 的 `PaymentReconcileDiffFound` 表达式逐字对齐。
  冒烟配置 `reconcile_enabled: false`（未到 `reconcile_hour=3`），**没跑到真实 T+1 那一轮**。
- [x] `make check-secrets` 通过（2 条 lab 弱值提示为既有项，非本片引入）；本记录落盘。

## 三、回归检查（不止 payment 包）

- `go build all` → 退出 0
- `go -C app test ./internal/payment/... ./internal/mall/... ./internal/membership/... ./internal/wallet/... ./internal/points/... ./internal/txdb/... ./internal/delivery/http/... ./internal/delivery/worker/... -count=1` → 全 `ok`
- `go -C app test ./... -count=1` → 无 FAIL
- `go -C app test -tags=integration ./internal/payment/`（库 `p02_it`）→ `ok 2.077s`
- `go -C app test -tags=integration ./internal/mall/ -run 'TestChannelPay|TestChannelNotify|TestMoneyTx'`（库 `p03_it`）→ 全 PASS，
  含 P0-3 三条资金原子性用例未退化（`TestMoneyTxConcurrentPayDebtsOnce` 等）
- P0-1 闸门未被破坏：worker 冒烟日志 `schema 版本校验通过 {version: 20261104160000, migrations: 26}`，
  启动期 DDL 关闭（`auto_migrate=false`），`payment.EnsureSchema` 只挂在 `schemaboot` 的逃生门里。

## 四、真库 / 冒烟命令与输出（复现口径）

```bash
# 集成用例（独立 scratch 库，绝不连业务库 my_go_study）
docker exec my_go_study_postgres psql -U postgres -d postgres -c "CREATE DATABASE p02_it"
DATABASE_PASSWORD=postgres P02_PG_DBNAME=p02_it \
  go -C app test -tags=integration ./internal/payment/ -count=1 -v
DATABASE_PASSWORD=postgres P03_PG_DBNAME=p03_it \
  go -C app test -tags=integration ./internal/mall/ -run 'TestChannel|TestMoneyTx' -count=1 -v

# 查单兜底实跑：真迁移链建库 → 造 1 条超时 pending → 起真 RabbitMQ worker
docker exec my_go_study_postgres psql -U postgres -d postgres -c "CREATE DATABASE p02_smoke"
MIGRATE_DB=p02_smoke ./scripts/migrate.sh up            # 26 个迁移，版本 20261104160000
docker exec my_go_study_postgres psql -U postgres -d p02_smoke -c \
  "INSERT INTO wys_payment_records (out_trade_no,user_id,channel,amount_fen,status,ref_type,ref_id,created_at)
   VALUES ('SMOKE-PENDING-1','u-smoke','wechat',1500,0,'mall_order','9001', now() - interval '30 min')"
/tmp/run_p02_worker.sh                                   # go -C app run ./cmd/worker（queue.enabled=true）
```

```
12:49:20 INFO  queue/payment_ticker.go:29  Payment jobs ticker 启动 {"query_interval":300,"reconcile_enabled":false,"reconcile_hour":3,"timezone":"Asia/Shanghai"}
12:54:20 SELECT * FROM "wys_payment_records" WHERE status = 0 AND created_at <= '12:44:20' ORDER BY created_at LIMIT 200   [rows:1]
12:54:20 WARN  payment/query.go:41         支付查单失败 {"out_trade_no":"SMOKE-PENDING-1","channel":"wechat","error":"支付未配置：请填写 WECHAT_MCH_* …"}
12:54:20 INFO  worker/handler.go:121       payment query job done {"scanned":1}
# broker 侧：queue my_go_study.tasks 存在且消息已排空（consumers=0 为 worker 停止后）
```

收尾：worker 进程与 `my_go_study_rabbitmq` 容器均已停止，机器恢复原状（只剩 postgres + redis）。

## 五、测试过程中发现并修掉的两个真 bug

| # | 现象 | 根因 | 修法 |
|---|------|------|------|
| 1 | 真库核销**永远失败**：`invalid input syntax for type json (SQLSTATE 22P02)` | 迁移把 `notify_raw` 定成 `jsonb`，而微信回调是 **XML**、支付宝是 form 原文，塞不进 jsonb | 列与 entity 改 `text`（`migrations/20261104160000_payment_ledger.up.sql`、`model/entity/payment.go`），注释写明「为什么不是 jsonb」 |
| 2 | 已迁移库上启动 `EnsureSchema: constraint "uni_wys_payment_records_out_trade_no" does not exist (SQLSTATE 42704)` | 迁移写的是**表级 UNIQUE 约束**，GORM 按 `uniqueIndex` 默认名 `uni_*` 去找要删的旧索引 —— 两份 schema 资产对同一个约束**叫不同名字** | 迁移改为 `CREATE UNIQUE INDEX uk_wys_payment_records_out_trade_no`，entity tag 同名；`p02_mig`（全新迁移链）3 条集成用例复验通过 |

残留风险（写明白）：**验收期间改过迁移文件形态**，已经用旧形态（constraint 版）迁过的库需要手工
`ALTER TABLE ... DROP CONSTRAINT` + `CREATE UNIQUE INDEX`。除我的 scratch 库 `p02_it`（已就地修好）外，
本仓没有任何库是旧形态 —— 迁移未提交、业务库 `my_go_study` 未动。

## 六、白名单越界披露（epic 白名单 vs 实际改动）

| 文件 | 是否在白名单 | 为什么必须动 |
|------|--------------|--------------|
| `common/config/third_party.go` | 否（只列了 `config.go`） | 新增 `ALIPAY_PUBLIC_KEY`（验回调签）/ `WECHAT_PAY_ENDPOINT` / `ALIPAY_GATEWAY` |
| `core/queue/payment_ticker.go`（新） | 否（只列了 `tasks.go`） | 到点入队；基建层**只入队**，不 import payment（模块边界不可反向） |
| `core/queue/client.go` | 否 | 本片用到新 task 类型的入队；其余改动属 P0-6 死信 |
| `app/cmd/worker/main.go` | 否 | 启动 ticker + Dispatch 注册两个新任务 |
| `app/cmd/api/main.go` | **黑名单（允许「必要构造注册」）** | `payment.NewService(..., NewPostgresRepository(db))`、`mallSvc.WithPayment`、`membership.NewService(..., paymentSvc, ...)`、schemaboot 步骤；未动启动流程 |
| `app/internal/mall/{handler,repository,repo_postgres}.go` + 2 个测试 | 超出「仅支付委托点」 | 委托后要回 pending 结构体、`Settler` 端口要 `LockOrderForSettle` / `WithTx` |
| `delivery/http/router/domain_routes_test.go` | 否（只列 `router.go`） | notify 是两条新增公开路由，基线表必须更新，否则路由数断言直接红 |
| `.env.lan.example` | 否（只列 `.env.example`/`.env.local.example`） | LAN 联调同样需要渠道环境变量口径 |
| `membership/{handler,router}.go` | 否 | **不属于本片**，是 P0-5「华为回调 501」的改动，同处未提交工作树 |

## 七、已知残留

| ID | 残留 | 状态 | 说明 / 解除条件 |
|----|------|------|-----------------|
| F1 | Flutter 侧未处理 `status=pending`，渠道支付在 App 上表现为「toast 成功 + 空详情」 | **Deferred（跨仓）** | 需改 `mall_api.dart:181`（按 `status` 分支解析）+ `mall_order_detail_controller.dart:107`（pending 时拉起 SDK / 轮询订单）。规则要求跨仓只读，本会话未动 `my_ai_project` |
| F2 | ~~`membership_order` 无业务核销器：回调只把流水置 paid，权益要客户端回来 `ConfirmBuyout` 才发~~ | **已解除（Migrated，2026-11-05 h8）** | `harden-h8-membership-settler`：`membership.Service` 实现 `payment.Settler`，`NewService` 内自注册 `PaymentRefMembershipOrder`，回调核销即发权益，`settled_no_settler` 对该 ref_type 不再出现；写落在 payment 传入的 ctx 事务里、体内零外部 HTTP/ES/MQ；幂等 + 双链并发只加一个权益周期经真库用例证明。证据见 [`2026-11-05-harden-h8-membership-settler.md`](./2026-11-05-harden-h8-membership-settler.md)。~~`MarkOrderPaid` 与 `ExtendEntitlement` 不在同一事务~~ 同日已解除（[`2026-11-04-membership-money-tx-followup.md`](./2026-11-04-membership-money-tx-followup.md)）；清单条目 18 同步改 `Migrated` |
| F3 | 对账只做「渠道有、本地无/未付」一个方向，`local_missing`（本地 paid 渠道无此单）未实现 | **Deferred** | `payment/reconcile.go:47-92`（`reconcileOne`）；`git grep PaymentDiffLocalMissing -- app model` 无命中，常量确实**没有写入方**。需要渠道账单下载（`download_url` → CSV），本片未接 |
| F4 | 商城 Apple / 华为 IAP 渠道仍走本地标记已付，无渠道流水 | **Placeholder** | `mall/service.go:505-517`；IAP 验签专项，epic 明确不做 |
| F5 | `unifiedorder` **响应**未验签（只验回调） | **Degraded** | `payment/service.go:242-259`。缓解：渠道 endpoint 只来自运维配置（`.env` / `configs/`），客户端不可控 → 无 SSRF 面；清单条目 21 |
| F6 | 未跑真实 T+1 整点那一轮（冒烟时 `reconcile_enabled=false`） | **Partial** | 对账语义由 3 条测试覆盖；整点触发由 ticker 代码路径 + 单测覆盖，缺实弹 |
| F7 | 核销成功后**不发通知**（epic 第 149 条的「通知发出」子项） | **Deferred** | 改造前的本地结单路径同样不调推送，属既有缺口而非本片回归。仓内已有能力：`usecase/jpush_usecase.go` + worker 任务；接法是 `SettleChannelPaid` **事务提交后**入队（事务内零外部调用是 P0-3 立的规矩，别破坏） |
