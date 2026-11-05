# Slice — parity-p0-2-payment-notify-ledger

## Slice Brief
- SOURCE_MODULE: `app/internal/payment`（只有 prepay）+ `mall`/`membership` 的「本地标记已付」
- TARGET_MODULE: 支付流水 + 渠道回调核销 + 查单兜底 + T+1 对账
- Source entry: `payment/router.go:37`（仅 `POST /api/v1/payments/prepay`）、`payment/service.go:84`（`out_trade_no` 生成即丢）、`payment/service.go:116-189`（微信 v2 unifiedorder 签名）、`:294-338`（支付宝 RSA2）、`mall/service.go:526-543`、`membership/service.go:279`
- Target entry: `wys_payment_records` + `POST /api/v1/payments/notify/{channel}` + Worker 查单/对账任务
- 本轮 ONLY:
  - 新表 `wys_payment_records`（out_trade_no 唯一、user_id、channel、amount_fen、status、notify_raw jsonb、paid_at、ref_type/ref_id）与 `wys_payment_reconcile_diff`；迁移进 `migrations/`（**不走 AutoMigrate**，依赖 P0-1）
  - prepay 落库并关联业务单
  - notify 路由：验签 + 商户号校验，**跳过 SessionAuth**（公网回调），注册在 `payment/router.go` 内
  - 核销幂等（唯一索引 + 状态单向流转，重复投递不重复发货）
  - 金额不符（渠道值 ≠ `amount_fen`）必须拒绝 + 留痕
  - Worker：pending 超时主动查单；T+1 对账写差异表并告警（接 `deploy/observability/alerts.yml`）
  - `mall` 渠道支付改为委托 payment 域（消除「无流水的已付订单」）
  - `membership` 微信/支付宝渠道须有已核销流水才可 `MarkOrderPaid`
- 不做:
  - 退款全链路（只处理 notify/查单/对账所需）
  - 优惠券/组合支付/分期
  - Apple/华为 IAP 验签逻辑改造（现状保留；华为回调 stub 归 P0-5 登记）
  - 不引入新支付渠道
  - 不改 SessionAuth 语义：notify 是公开路由靠**验签**，其它业务接口照旧
- 验收: 见 Epic §P0-2 全部 8 条；硬门槛三条：①notify 重复投递 3 次只核销 1 次（测试断言）②篡改签名/金额不符 → 拒绝且状态不变 ③走微信渠道的商城订单在 `wys_payment_records` 必有流水（SQL 校验）
- 文件白名单:
  - app/internal/payment/
  - app/internal/mall/（仅支付委托点）
  - app/internal/membership/service.go
  - app/internal/delivery/worker/handler.go
  - core/queue/tasks.go
  - model/entity/payment.go
  - model/repository/
  - migrations/
  - configs/
  - common/config/config.go
  - app/internal/delivery/http/router/router.go（仅确认 notify 不进 SessionAuth 链）
  - deploy/observability/alerts.yml
  - docs/
  - .env.example
  - .env.local.example
- 文件黑名单:
  - app/cmd/api/main.go（除必要的构造注册，勿动启动流程）
  - app/internal/transaction/
  - thunder/
- 验证命令:
  - `go build all` && `go -C app build ./cmd/api && go -C app build ./cmd/worker`
  - `go -C app test ./internal/payment/...`（含验签正/负、金额不符）
  - `go -C app test -tags=integration ./internal/payment/... ./internal/mall/...`
  - `make deps-up` → `make run` + `make run-worker`（`queue.enabled=true`）
  - 手工 curl notify：正确签名 → 200 且单据推进；错签名 → 4xx 且状态不变
  - `curl -s http://127.0.0.1:8080/metrics | grep payment`（新增指标可见）
  - `make check-secrets`（密钥只落 `.env.local`）
- 证据: docs/acceptance-records/2026-11-<date>-parity-p0-2-payment-notify-ledger.md

## Context Card — parity-p0-2-payment-notify-ledger

- 已完成:
  - 迁移 `migrations/20261104160000_payment_ledger.{up,down}.sql`：`wys_payment_records` +
    `wys_payment_reconcile_diff`（`notify_raw` 用 **text 不是 jsonb**；`out_trade_no` 建成
    **具名唯一索引** `uk_wys_payment_records_out_trade_no`，与 entity tag 同名，否则已迁移库上
    `EnsureSchema` 会 SQLSTATE 42704）；`model/entity/payment.go`。
  - prepay 落库（`payment/service.go` `StartChannelPay` 前置写 pending）+ `HandleWechatNotify` /
    `HandleAlipayNotify`（`notify.go` / `verify.go`）：验签 + 商户号 + 金额三重校验，
    核销幂等靠唯一索引 + 状态单向流转，返回 `NotifyResult.NewlySettled`。
  - 公开回调路由注册在 `payment/router.go`（**不进 SessionAuth 链**），同前缀业务路由不受影响。
  - 查单兜底 `query.go` + T+1 对账 `reconcile.go`：`core/queue/payment_ticker.go` 只入队，
    `delivery/worker/handler.go` Dispatch 执行（模块边界不反向）。
  - 指标 `metrics.go` 5 条序列 ↔ `deploy/observability/alerts.yml` `payment` 组 4 条规则逐字对齐。
  - `mall` 微信/支付宝渠道改委托 payment 域（`StartChannelPay` → `SettleChannelPaid` 实现
    `payment.Settler`，与核销同事务）；`membership.ConfirmBuyout` 渠道分支必须已有已核销流水，
    `u.pay == nil` 直接 `ErrPaymentNotConfirmed`（fail-closed）。
  - 测试：payment 包 30 条单测（含 4 条 HTTP/路由层）+ 3 条真库集成；mall 4 条真库渠道支付/核销集成
    + P0-3 三条资金原子性用例回归通过。验收记录 `docs/acceptance-records/2026-11-04-parity-p0-2-payment-notify-ledger.md`。
- 未做/Deferred:
  - **F1 跨仓**：Flutter 未处理 `status=pending`（`mall_api.dart:181` 用 `MallOrderDetail.fromJson`
    解渠道响应 → 软降级成空详情 + toast「支付成功」）。**渠道支付在 App 上不可用**，需另开 my_ai_project 切片。
  - **F2** `membership_order` 未登记业务核销器（`settle.go:103-116` `settled_no_settler`）。
    原并列的「`MarkOrderPaid` 与 `ExtendEntitlement` 不在同一事务」已于同日 follow-up 解除
    （`docs/acceptance-records/2026-11-04-membership-money-tx-followup.md`）。
  - **F3** 对账只做渠道方向，`local_missing` 无写入方（需渠道账单下载）。
  - **F4** 商城 Apple/华为 IAP 渠道仍本地标记已付（epic 明确不做）。
  - **F5** `unifiedorder` 响应未验签（endpoint 只来自运维配置，无 SSRF 面）。
  - **F7** 核销成功后不发通知（既有缺口，改造前也没有）。
- 关键文件:
  - `app/internal/payment/{service,notify,verify,settle,query,reconcile,metrics,repo_postgres,repository,channel,router,handler}.go`
  - `app/internal/mall/{service,handler,repository,repo_postgres}.go`、`app/internal/membership/service.go`
  - `core/queue/{payment_ticker,tasks,client}.go`、`app/cmd/{api,worker}/main.go`
  - `migrations/20261104160000_payment_ledger.up.sql`、`model/entity/payment.go`
  - `app/internal/payment/{ledger_integration,notify_http,jobs,notify}_test.go`、
    `app/internal/mall/payment_ledger_integration_test.go`
- harness: post ok? —— 是：`go build all` 退出 0；`go -C app test ./... -count=1` 无 FAIL；
  定向集成（`p02_it` / `p03_it`）全 PASS；`make check-secrets` 通过；真 RabbitMQ worker 冒烟跑到
  「ticker → Enqueue → 消费 → 扫单 1 条」；worker 进程与 rabbitmq 容器已停回原状。
  **未跑** `make lan-up` 全栈（会重建镜像并改共享容器，非本片必要）。
- 下一 Slice 建议: F1（Flutter 渠道支付契约）优先级最高 —— 后端已就绪，客户端不接就是「看不见的修复」；
  其次 task #12（membership 接 `txdb`，同时解掉 F2 与清单条目 18）。
- 已知坑:
  - 商户私钥/APPID 走 `.env.local`（AGENTS.md：`SUPABASE_SERVICE_ROLE_KEY` 同规则；推送前 `make check-secrets`）
  - 金额**必须整数分**，禁止 float 参与运算
  - notify 免鉴权 ≠ 免校验：漏验签等于任何人都能标记已付，这是本片最高风险
  - 微信 v2 回调需返回特定 XML/文本格式，别只回 JSON 200（会导致渠道持续重推）
  - **新增坑**：渠道回调判定成功要 `result_code == SUCCESS` **且** `trade_state ∈ {"",SUCCESS,REFUND}`；
    缺 `result_code` 的报文是 `Reason="not_paid"` + **nil error**，手写测试报文时极易漏。
  - **新增坑**：改 `migrations/` 里已写过的唯一约束形态（constraint ↔ index）会让**已迁移库**的
    GORM `EnsureSchema` 报错；两份 schema 资产的索引名必须一致。
  - **新增坑**：notify 用**独立签名实现**测试（`signedWechatNotify`），否则签名算法写错也测不出来。
  - 与 Flutter 契约变更需跨仓同步（`../../my_ai_project`，读其 AGENTS.md）
  - 事务边界复用 P0-3 的 `WithTx`，不要新造
