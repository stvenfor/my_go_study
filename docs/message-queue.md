# 消息队列（RabbitMQ + Redis Pub/Sub）

本文档说明 `my_go_study` 已接入的异步任务队列与多实例 WebSocket 广播方案。

## 架构

```text
Flutter → HTTP POST /realtime/push
              ↓
         Go BFF (cmd/api) ──Publish──► RabbitMQ
              │                              ↓
              │ Subscribe              cmd/worker
              ▼                              ↓
         Redis Pub/Sub ◄──── Publish ── 写 realtime:events + Publish
              ↓
    各 BFF 实例本地 WS Hub → Flutter WebSocket
```

- **RabbitMQ**：任务队列（入队 / 消费）；定时广播由 worker 内 ticker 整点触发。
- **Redis Pub/Sub**：多实例 BFF 间 WS 广播；离线用户仍靠 `realtime:events` + `/realtime/sync` 补拉。

## 配置

`configs/config.yaml`（基础）：

```yaml
queue:
  enabled: false
  rabbitmq:
    url: amqp://guest:guest@localhost:5672/
    queue: my_go_study.tasks
    prefetch: 10
    dlx: my_go_study.tasks.dlx          # 留空则按 <queue>.dlx 派生
    dlq: my_go_study.tasks.dlq          # 留空则按 <queue>.dlq 派生
    max_retries: 3                      # 单条最多尝试次数（含首次），超限进死信
    handler_timeout_seconds: 30         # 单条处理超时，防一条卡死整个 prefetch 窗口
    publisher_confirm: true             # 发布等 broker 落盘确认（吞吐略降，换入队失败可见）
  pubsub:
    channel: realtime:fanout
```

`configs/config.dev.yaml` 默认 `queue.enabled: true`。

环境变量：

| 变量 | 说明 |
|------|------|
| `QUEUE_ENABLED` | `true` / `false` |
| `RABBITMQ_URL` | AMQP URL（默认 `amqp://guest:guest@localhost:5672/`） |
| `RABBITMQ_QUEUE` | 队列名（默认 `my_go_study.tasks`） |
| `RABBITMQ_PREFETCH` | Worker QoS prefetch |
| `RABBITMQ_DLX` / `RABBITMQ_DLQ` | 死信交换机 / 死信队列名 |
| `RABBITMQ_MAX_RETRIES` | 重试上限（<=0 回落 3） |
| `RABBITMQ_HANDLER_TIMEOUT_SECONDS` | 单条超时（<=0 回落 30s） |
| `RABBITMQ_PUBLISHER_CONFIRM` | 关闭发布确认（默认开） |
| `QUEUE_PUBSUB_CHANNEL` | Pub/Sub 频道名 |

## 可靠性（重试上限 / 死信 / 确认）

### 有界重试

消费失败**不**用 `Nack(requeue=true)` 计数——实测 RabbitMQ 3.13 在 requeue 重投的消息上既不带 `x-delivery-count` 也不带 `x-death`（`x-death` 只在真正死信时写入），靠 broker 计数会无限重投。改成**应用侧重投**：失败后往同一队列再投一条带 `x-retry-count = 上次值+1` 的新消息，然后 Ack 原消息；`x-retry-count` 达到 `max_retries` 时 `Nack(requeue=false)` → 死信。

- 计数读取顺序：`x-retry-count` → `x-delivery-count` → `x-death.count`（兼容旧消息与人工投递）。
- 计数拿不到且 `redelivered=true` → 直接进死信，宁可牺牲一次重试也不无限循环。
- 重投本身失败 → `Nack(requeue=true)` 保住原消息，绝不 Ack（不丢任务）。
- 所有 `Ack`/`Nack` 的 error 都会记日志（原先被丢弃）。

死信消息可在 DLQ 里查到完整取证：`x-retry-count`、`x-death[{reason:"rejected",count,queue}]`、`x-first-death-*`。

### DLX / DLQ 拓扑与 406 兼容

`core/queue` 启动时声明 `dlx`(direct) + `dlq` + 绑定 + 带 `x-dead-letter-*` 的业务队列，且用**独立临时 channel** 声明（406 会关掉 channel）。

队列参数不可变：若已存在**无 args** 的旧队列，声明会 `PRECONDITION_FAILED (406)`。此时进程**不退出**，改用无 args 沿用旧队列并记
`Warn("task queue args mismatch (406) …", status=degraded)`，代价是死信不生效（`dlq_wired=false`，启动日志可见）。
在维护窗口执行下面脚本重建即恢复：

```bash
bash scripts/rabbitmq-ensure-dlq.sh status            # 看队列/交换机现状
APP_ENV=dev bash scripts/rabbitmq-ensure-dlq.sh recreate --yes   # 删旧队列+DLQ，下次启动自动带 args 重建
bash scripts/rabbitmq-ensure-dlq.sh rename            # 生产零丢失方案：RABBITMQ_QUEUE 切 <queue>.v2
```

`recreate` 只在 `APP_ENV=dev|lan|test` 放行，且必须显式 `--yes`（DLQ 里已有死信会一起没，先 `rabbitmqctl export_definitions`）。

### 单条超时与优雅退出

- 每条消息独立 goroutine（并发度受 prefetch 限制），`Dispatch` 拿到 `context.WithTimeout(context.WithoutCancel(ctx), handler_timeout)`：一条卡死只占自己那份 prefetch，关机 cancel 也不会传染在途任务。
- SIGTERM：`cancel()` → 等消费循环退出 → `consumer.Wait()` 排在途任务收尾 → 关连接。未在途的消息由 broker 重投，不丢。

### 发布确认与失败可见

`publisher_confirm=true` 时 `Channel.Confirm` + 每次发布读一条确认（发布与读确认在同一把锁内串行，保证读到的就是自己那条）。broker nack / confirm channel 关闭 / 等待超时都会返回 error，调用方（`/realtime/push`、广播入队、scheduler-trigger CLI）据此返回 500 或非 0 退出，**不返回「已入队」假象**。

> 已知边界：`Client` 不自动重连，broker 重启后需重启 API/Worker 进程（列入 P1 待办）。

### 死信指标与告警

`common/metrics.NewDepthCollector` 注册 `rabbitmq_dlq_messages{queue="…"}`（由 `cmd/api` 挂载，`/metrics` 暴露）。抓取时 `QueueInspect` 出错则**不输出该指标**——避免「broker 挂了」被误读成「死信为 0」，`absent()` 负责报警：

| 规则 | 表达式 | 级别 |
|------|--------|------|
| `MQDeadLettersPresent` | `rabbitmq_dlq_messages > 0`（持续 5m） | critical |
| `MQDeadLetterMetricMissing` | `absent(rabbitmq_dlq_messages)`（持续 10m） | warning |

校验规则文件用 **`promtool check rules`**（`amtool check-config` 校验的是 alertmanager 自身的配置，不认 `groups:` 顶层）。

## 本地运行

在 **`my_go_study/`** 目录下：

```bash
# 依赖：Postgres + Redis + RabbitMQ（Docker 或本机）
# 终端 1：API（HTTP + WS + Pub/Sub 订阅）
make run

# 终端 2：Worker（消费 RabbitMQ 任务）
make run-worker

# 验证
make test-queue-push
```

Docker Compose / `make lan-up` 已包含 `rabbitmq` 与 `worker` 服务（管理 UI：本机 `127.0.0.1:15672`，guest/guest）。

## 任务类型

| 任务 | 类型常量 | 状态 |
|------|----------|------|
| Realtime Push | `realtime:push_notify` | 已实现 |
| 定时广播 | `scheduled:broadcast_notify` | 已实现（每小时系统通知） |
| 短信 OTP | `sms:send` | 占位（生产接入 SMS 厂商） |
| 极光注册 | `jpush:register` | 已实现（写入 `wys_push_devices`） |

信封 JSON：`{ "type", "task_id", "payload" }`。

## Push API 响应（异步模式）

`queue.enabled=true` 时，`POST /api/v1/realtime/push` 立即返回：

```json
{
  "data": {
    "queued": true,
    "taskId": "…",
    "delivered": -1
  }
}
```

Worker 投递完成后，客户端通过 WebSocket 实时收到，或断线后 `POST /api/v1/realtime/sync` 补拉。

## 代码入口

| 路径 | 职责 |
|------|------|
| `core/queue/consumer.go` | 消费循环、有界重试、单条超时、在途等待 |
| `core/queue/topology.go` | DLX/DLQ 声明与 406 降级 |
| `core/queue/retry.go` | 尝试次数读取（`x-retry-count`/`x-delivery-count`/`x-death`） |
| `core/queue/client.go` | 入队、publisher confirm、`DLQDepth()` |
| `core/queue/pubsub_broadcaster.go` | Redis Pub/Sub WS 扇出 |
| `app/internal/delivery/worker/handler.go` | 按 `type` 分发业务（基建与业务分层，见 AGENTS.md 约束 10） |
| `app/cmd/worker/main.go` | Worker 进程入口（含整点 ticker、优雅退出） |
| `scripts/rabbitmq-ensure-dlq.sh` | 死信拓扑体检 / 重建 / 改名迁移 |
| `internal/usecase/realtime_push_usecase.go` | 同步/异步入队与投递 |
| `internal/usecase/hourly_notify_usecase.go` | 定时通知消息体组装 |
| `cmd/scheduler-trigger/main.go` | 开发环境手动触发广播 |

### 重复投递语义（复核结论）

队列是 at-least-once，四类任务的重复投递后果：

| 任务 | 重复投递 | 去重手段 |
|------|----------|----------|
| `realtime:push_notify` | 可能多推一帧 WS（客户端按 `notifyId`/seq 幂等展示） | **无服务端去重**（待办：按 `task_id` SetNX） |
| `scheduled:broadcast_notify` | 不重复 | `worker/handler.go` Redis SetNX + 确定性 taskID |
| `sms:send` | 无副作用（占位，Warn 后丢弃） | — |
| `jpush:register` | 幂等 upsert | 主键覆盖 |


## 定时每小时系统通知

每天 **10:00–19:00**（Asia/Shanghai）每小时向**有登录 Session 的用户**（`auth:session:*`）发送 `sys.notify`：

- **在线**：WS 即时 Banner
- **离线登录**：写入 `realtime:events`，重连后 `sync` 补拉

配置（`configs/config.yaml` → `scheduler`）：

```yaml
scheduler:
  enabled: true
  timezone: Asia/Shanghai
  hourly_notify:
    enabled: true
    cron: "0 10-19 * * *"
    title_template: "整点提醒"
    body_template: "现在是 {{hour}}:00，{{message}}"
    default_message: "别错过重要消息"
    expires_minutes: 120
    action:
      type: deeplink
      route: /home
```

Worker 用分钟级 ticker 对齐 cron 小时范围（替代原 Asynq Scheduler）。

**开发环境**（`config.dev.yaml`）默认 `hourly_notify.enabled: false`，用手动触发：

```bash
make trigger-hourly-notify   # 入队 broadcast 任务
make test-scheduled-notify   # 登录 + 触发 + sync 验证
```

**消息体示例**（单用户完整 Envelope）：

```json
{
  "type": "event",
  "topic": "sys.notify",
  "seq": 12,
  "payload": {
    "name": "sys.notify.show",
    "notifyId": "660e8400-e29b-41d4-a716-446655440001",
    "title": "上午好",
    "body": "新的一天开始了，查看今日动态",
    "category": "scheduled",
    "campaignId": "hourly-20260710-10",
    "scheduleSlot": "2026-07-10T10:00:00+08:00",
    "messageType": "hourly_digest",
    "expiresAt": 1739007204000,
    "action": { "type": "deeplink", "route": "/home" }
  }
}
```

## 阶段 2 评估（未实施）

当满足以下**任一**条件时，再评估 NATS JetStream 或 Watermill 抽象层：

- BFF 实例 ≥ 3，Redis Pub/Sub 出现消息风暴或运维痛点
- 拆分为独立 Notification / IM 微服务
- 需要消息回溯、严格顺序的多消费者组

触发前保持当前 **RabbitMQ + Redis Pub/Sub** 方案即可。
