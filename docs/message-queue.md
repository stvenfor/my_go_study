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
| `QUEUE_PUBSUB_CHANNEL` | Pub/Sub 频道名 |

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
| `pkg/queue/` | RabbitMQ 客户端/消费者、Pub/Sub 广播 |
| `cmd/worker/main.go` | Worker 进程入口（含整点 ticker） |
| `internal/usecase/realtime_push_usecase.go` | 同步/异步入队与投递 |
| `internal/usecase/hourly_notify_usecase.go` | 定时通知消息体组装 |
| `cmd/scheduler-trigger/main.go` | 开发环境手动触发广播 |

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
