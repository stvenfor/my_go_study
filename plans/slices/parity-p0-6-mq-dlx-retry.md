# Slice — parity-p0-6-mq-dlx-retry

## Slice Brief
- SOURCE_MODULE: `core/queue` + `app/internal/delivery/worker`
- TARGET_MODULE: 有界重试 + 死信队列 + 入队失败可见 + Worker 在途等待
- Source entry: `core/queue/consumer.go:50-51,87`（prefetch/手动 ack）、`:100`（`_ = d.Nack(false, true)` 无限重投且丢弃错误）、`core/queue/client.go:32`（`QueueDeclare` args=nil）、`:64`（无 publisher confirm）、`core/queue/hourly_ticker.go`、`app/cmd/worker/main.go:132-135`（只 `cancel()`）
- Target entry: `task.dlq` + `x-dead-letter-exchange` + `queue.max_retries` + DLQ gauge 指标
- 本轮 ONLY:
  - 重试计数（`x-death`/`x-delivery-count` 或自定义头）+ 上限；超限 `Nack(false,false)` 进死信；**检查 Nack/Reject 返回的 error**
  - 声明 DLX 与 `task.dlq` 及绑定；因队列参数不可变，提供改名/重建方案与 `scripts/` 说明（旧队列存在时直接改 args 会 `406`）
  - DLQ 长度 gauge（沿用 `common/metrics/metrics.go:16,23,31` 注册风格）+ `deploy/observability/alerts.yml` 新增「DLQ > 0 持续 5m」
  - 发布侧开 publisher confirm，或明确「入队失败返回 error」并核对所有调用点确实处理了 error
  - 单条消息处理超时（`context.WithTimeout`），防止一条卡死 prefetch 窗口
  - Worker 优雅退出：`cancel()` + `WaitGroup` 等在途 + 再 close（对齐 HTTP 侧 `main.go:574-592`）
  - 幂等复核：4 类任务重复投递无重复副作用（推送去重现靠 `worker/handler.go:136` 的 SetNX，覆盖面需核对）
- 不做:
  - 不换消息中间件、**不引入 Asynq**（AGENTS.md 修改规范 9）
  - 不做业务 outbox（P2-6）
  - 不动 Redis Pub/Sub 的 WS 扇出（那是另一条通路，勿混）
  - 不改任务类型语义与载荷结构
- 验收: 见 Epic §P0-6 全部 8 条；硬门槛：①必然失败的消息 N 次后进 DLQ，**不再无限循环**（`rabbitmqctl list_queues` 数字为证）②旧队列存在时 `make lan-up` 不 `406` ③停 RabbitMQ 后入队失败**不返回成功**④Worker SIGTERM 后发 N 消费 N
- 文件白名单:
  - core/queue/
  - app/internal/delivery/worker/
  - app/cmd/worker/main.go
  - common/metrics/metrics.go
  - configs/
  - common/config/config.go
  - docker/docker-compose.yml
  - deploy/observability/alerts.yml
  - scripts/
  - docs/message-queue.md
  - docs/
- 文件黑名单:
  - app/internal/payment/
  - app/internal/transaction/
  - core/database/redis.go（Pub/Sub 广播逻辑勿动）
  - thunder/
- 验证命令:
  - `go build all`
  - `go -C core test ./queue/...`
  - `go -C app test ./internal/delivery/worker/...`
  - `make lan-up`（含 RabbitMQ）→ `make test-queue-push`
  - 毒丸用例：投一条必然失败任务 → 观察重试次数与最终进 DLQ
  - `rabbitmqctl list_queues messages` / `curl -s http://127.0.0.1:8080/metrics | grep dlq`
  - SIGTERM Worker 观察在途等待日志
  - `amtool check-config deploy/observability/alerts.yml`
  - `make check-secrets`
- 证据: docs/acceptance-records/2026-11-<date>-parity-p0-6-mq-dlx-retry.md

## Context Card — parity-p0-6-mq-dlx-retry
- 已完成:
  - `core/queue/topology.go`：DLX/DLQ/绑定声明 + 406 判定与降级（临时 channel，406 会关 channel）
  - `core/queue/retry.go` + `consumer.go`：应用侧 `x-retry-count` 有界重试；超限 `Nack(false,false)` 进死信；
    计数未知且 redelivered → 直接死信；重投失败保消息；Ack/Nack 错误全部记日志
  - `consumer.go`：每条独立 goroutine + `context.WithTimeout(context.WithoutCancel(ctx), handler_timeout)` + `Wait()` 在途等待
  - `client.go`：publisher confirm（发布与读确认同锁串行）、`DLQDepth()`、`NewClient(cfg, log)`
  - `common/metrics.NewDepthCollector` + api main.go 注册 `rabbitmq_dlq_messages`；alerts.yml `queue` 组 2 条
  - `scripts/rabbitmq-ensure-dlq.sh`：status / recreate --yes / rename（零丢失改名迁移）
  - `docs/message-queue.md`：可靠性章节 + 入口表更正（`pkg/queue` → `core/queue`）+ 重复投递语义表
  - 硬门槛①③④ 实机 PASS；② 进程级 PASS（双进程），compose 级未跑
- 未做/Deferred:
  - `realtime:push_notify` 服务端 `task_id` 去重（Brief 禁改投递语义）→ 验收条 7 保持未勾
  - `Client` 自动重连 / 消费循环断线重启（现在 broker 重启需重启进程）→ P1
  - DLQ 死信手工重放工具 → 运维需要时再加
  - `make lan-up` compose 闸门（本机 `:8080` 被无关进程占）→ 待环境空出来补跑
- 关键文件: `core/queue/{topology,retry,consumer,client}.go`、`common/config/config.go`、
  `common/metrics/metrics.go`、`app/cmd/{worker,api,scheduler-trigger}/main.go`、
  `deploy/observability/alerts.yml`、`scripts/rabbitmq-ensure-dlq.sh`、`docs/message-queue.md`
- harness: post ok? — 是（record `docs/acceptance-records/2026-11-03-parity-p0-6-mq-dlx-retry.md`，status=Degraded + note；
  白名单偏差 2 处已在 record 点名：`app/cmd/api/main.go`、`app/cmd/scheduler-trigger/main.go`）
- 下一 Slice 建议: M1 收尾汇报 → 进 P1-1 `parity-p1-1-request-id`（Worker 侧日志需与 request-id 打通）
- 已知坑:
  - RabbitMQ 队列参数**不可变**：已存在的队列改 `x-dead-letter-*` 会 `PRECONDITION_FAILED (406)`，必须走改名或删建流程（测试环境可删，生产需迁移窗口）
  - **`Nack(requeue=true)` 的重投消息不带 `x-delivery-count`，也不带 `x-death`**（3.13 实测）：靠 broker 计数会在第 2 次就误判耗尽 → 必须应用侧重投计数
  - 406 会**关掉整个 channel**，拓扑声明必须在独立临时 channel 上做，否则消费 channel 直接报废
  - `module-boundary` 规则：`core/queue` 是纯基建，**不得**反向依赖 `app/internal/usecase`（该问题历史上已修，见 AGENTS.md 约束 10）；同理 `core` 不应引入 prometheus —— DLQ gauge 用「`common/metrics` 出通用 collector + api 进程注册」避开分层破坏
  - confirm 会降吞吐：默认开启（`publisher_confirm: true`），可 env 关；本机未做吞吐实测（guest localhost 往返内 publish 成功路径未见阻塞）
  - DLQ 只声明不接告警 = `Degraded` + note，不能算 Migrated → 本片已接 `MQDeadLettersPresent` + `absent()` 两条
  - 抓取失败**不输出**指标（不是输出 0），否则「broker 挂了」会被读成「死信为 0」
  - `amtool check-config` 校验不了 Prometheus 规则文件（顶层 `groups:` 报 `field groups not found`）→ 用 `promtool check rules`
  - 整点 ticker 的**重复执行**问题归 P2-7（需分布式锁），本片只管消息侧
