# 11-消息队列与 Worker：RabbitMQ 任务队列 + Redis Pub/Sub 扇出

> 一句话定位：本篇讲异步任务体系——API 只负责把任务塞进 RabbitMQ，独立的 Worker 进程消费任务、写 Redis 事件日志，再经 Redis Pub/Sub 扇出给所有 BFF 实例的 WebSocket Hub。所属模块：`core/queue`（基建）+ `app/internal/delivery/worker`、`app/internal/repository/queue`、`app/internal/usecase`（业务）+ `app/cmd/worker`、`app/cmd/scheduler-trigger`（可执行入口）。阅读前置：先懂 Realtime 推送与 WebSocket Hub（见 [../../message-queue.md](../../message-queue.md) 与 Realtime 相关一篇），了解 `config.Config` 装配。

## 1. 通俗理解：这个体系在干什么

把整条链路想象成「餐厅下单」：

- **API 进程 = 前台**：顾客（客户端）点一份「给某用户推通知」，前台不开火，只把小票（Envelope）塞进传菜传送带（RabbitMQ 队列），立刻返回「已下单（queued=true, taskId）」。
- **Worker 进程 = 后厨**：盯着传送带，按小票上的 `type` 分发到不同工位（推送/短信/极光注册/整点广播），做完后 `Ack`；做坏了 `Nack(requeue=true)` 把小票退回传送带重做。
- **Redis Pub/Sub = 后厨的广播喇叭**：Worker 自己不持有任何 WebSocket 连接（连接在 API 进程里），它把「请给用户 X 投递这条消息」喊到一个频道上；每个 API 实例都竖着耳朵听，听到后送进自己进程内的 Hub，由 Hub 找到该用户的 WS 连接写出去。
- **整点 ticker = 定时闹钟**：Worker 进程内每分钟滴答一次，到点（整点且命中 cron 小时段）就往队列塞一张「广播任务」小票；广播任务再展开成「每个活跃用户一张推送小票」。

为什么要拆成两个中间件？RabbitMQ 负责**可靠的点对点任务**（持久化、Ack、重投），Redis Pub/Sub 负责**不可靠但极快的多实例扇出**（在线才收得到，离线靠 Redis 事件日志 + `/realtime/sync` 补拉）。

## 2. 全景与命名

### 2.1 进程与组件

| 组件 | 所在位置 | 职责 |
|---|---|---|
| RabbitMQ `Client`（生产者） | `core/queue/client.go` | API / scheduler-trigger / Worker 都会用：声明队列、发布 JSON 任务 |
| RabbitMQ `Consumer`（消费者） | `core/queue/consumer.go` | 仅 Worker：QoS prefetch、手动 Ack/Nack 的消费循环 |
| 任务信封与载荷 | `core/queue/tasks.go` | `Envelope` + 4 种任务类型常量与 Payload |
| Pub/Sub 发布器 | `core/queue/pubsub_broadcaster.go` `FanoutPublisher` | Worker 用：把投递消息 `PUBLISH` 到 Redis 频道 |
| Pub/Sub 订阅器 | 同文件 `FanoutSubscriber` | 每个 API 实例用：收到消息调 `hub.BroadcastToUser` |
| 整点 ticker | `core/queue/hourly_ticker.go` | Worker 内 goroutine，到点入队广播任务 |
| 业务分发器 | `app/internal/delivery/worker/handler.go` | 实现 `queue.Dispatcher`，按 `type` switch 到 usecase |
| 入队适配器 | `app/internal/repository/queue/push_enqueuer.go` | 把 `usecase.RealtimePushInput` 转成 `queue.PushNotifyPayload` |

### 2.2 队列 / 交换机 / 频道命名（全部来自代码与配置）

| 名字 | 取值 | 来源 |
|---|---|---|
| RabbitMQ 队列名 | 默认 `my_go_study.tasks`，可配 `queue.rabbitmq.queue` / 环境变量 `RABBITMQ_QUEUE` | `QueueConfig.RabbitMQQueue()`，`common/config/config.go:418` |
| AMQP URL | 默认 `amqp://guest:guest@localhost:5672/`，可配 `queue.rabbitmq.url` / `RABBITMQ_URL` | `RabbitMQURL()`，`config.go:410` |
| 交换机 | **没有声明任何自定义交换机**，发布到默认交换机 `""`，routing key = 队列名 | `client.go:64` `PublishWithContext(ctx, "", c.queue, ...)` |
| 队列属性 | durable=true、autoDelete=false、exclusive=false | 生产端 `client.go:32` 与消费端 `consumer.go:45` 各声明一次（幂等） |
| 消息属性 | `Content-Type: application/json`、`DeliveryMode: Persistent`（持久化）、`MessageId = TaskID` | `client.go:64-69` |
| QoS prefetch | 默认 10，可配 `queue.rabbitmq.prefetch` / `RABBITMQ_PREFETCH` | `Prefetch()`，`config.go:426` |
| Redis Pub/Sub 频道 | 默认 `realtime:fanout`，可配 `queue.pubsub.channel` / `QUEUE_PUBSUB_CHANNEL` | `PubSubChannel()`，`config.go:434` |
| 广播去重 Redis Key | `scheduler:dedup:%s`，参数为 `taskID`，TTL 2 小时 | `handler.go:28` `schedulerDedupKeyFmt` |

> 注意：这套实现里**没有 DLX（死信交换机）、没有延迟队列、没有 TTL 死信**，就是一条最朴素的持久化直连队列。

### 2.3 任务类型（`core/queue/tasks.go:11`）

| 常量 | 字面值 | Payload 类型 | Worker 处理 |
|---|---|---|---|
| `TypeRealtimePushNotify` | `realtime:push_notify` | `PushNotifyPayload` | 调 `RealtimePushUsecase.DeliverPush` 投递 |
| `TypeScheduledBroadcastNotify` | `scheduled:broadcast_notify` | `BroadcastNotifyPayload` | 展开为每活跃用户一条 push 任务 |
| `TypeSendSMS` | `sms:send` | `SendSMSPayload` | **桩**：只打日志 |
| `TypeJPushRegister` | `jpush:register` | `JPushRegisterPayload` | 调 JPushUsecase 注册设备（未注入则跳过） |

## 3. 配置开关（`common/config/config.go` + `configs/*.yaml`）

| 配置 | 作用 | 事实 |
|---|---|---|
| `queue.enabled` / `QUEUE_ENABLED` | 总开关。false 时 Worker 启动直接报错退出；API 不连 RabbitMQ | `cmd/worker/main.go:44` |
| `queue.push_async` / `QUEUE_PUSH_ASYNC` | `/realtime/push` 是否异步入队。**nil（未配）默认为 true**；显式 false 则同步直投本地 Hub | `UseAsyncPush()`，`config.go:442` |
| `scheduler.enabled` + `scheduler.hourly_notify.enabled` | 两个开关都为 true 时 ticker 才工作 | `hourly_ticker.go:19` |
| `scheduler.timezone` | 调度时区，默认 `Asia/Shanghai` | `SchedulerConfig.Location()` |
| `scheduler.hourly_notify.cron` | 默认 `0 10-19 * * *`（每天 10 点到 19 点的整点） | `CronSpec()` |

各环境现状（来自 yaml 原文）：

- `configs/config.dev.yaml:27-40`：`queue.enabled: true` 但 **`push_async: false`**（开发环境同步直投本地 WS，避免 Pub/Sub 延迟/丢投）；`scheduler.enabled: true`、`hourly_notify.enabled: false`（整点靠手动触发）。
- `configs/config.prod.yaml:12-18`：`queue.enabled: true`（push_async 未配 → 默认异步）、scheduler 与 hourly_notify 均 enabled。

## 4. `core/queue` 基建包逐文件函数清单

### 4.1 `core/queue/tasks.go`——信封与载荷

| 函数/类型签名 | 职责 | 备注 |
|---|---|---|
| `type Envelope struct { Type string; TaskID string; Payload json.RawMessage }` | 队列消息统一信封，JSON 字段 `type`/`task_id`/`payload` | Payload 用 RawMessage 延迟解析，分发时按 type 再 unmarshal |
| `type PushNotifyPayload struct { UserID, Topic, Title, Body, Name string; Extra map[string]any }` | 实时推送载荷，JSON 字段为驼峰 `userId`/`topic`/... | 与 `usecase.RealtimePushInput` 字段同名但**故意是两个类型**（见 5.1） |
| `type BroadcastNotifyPayload struct { ScheduleSlot string }` | 广播任务载荷，仅一个 RFC3339 槽位（可空） | 空时 Worker 用当前整点 |
| `type SendSMSPayload` / `type JPushRegisterPayload` | 短信、极光注册载荷 | SMS 为生产 OTP 预留，目前无消费者真发 |
| `func newEnvelope(taskType string, payload any, taskID string) (Envelope, error)` | 内部构造器：marshal payload；taskID 为空则 `uuid.NewString()` | 包内私有 |
| `func NewPushNotifyTask(p) (Envelope, error)` | 构造 `realtime:push_notify`（随机 taskID） | |
| `func NewBroadcastNotifyTask(p) (Envelope, error)` | 构造 `scheduled:broadcast_notify` | |
| `func NewSendSMSTask` / `NewJPushRegisterTask` | 构造另两类任务 | 注意只有 Push/Broadcast 提供了 Parse 函数 |
| `func ParsePushNotifyPayload([]byte) (PushNotifyPayload, error)` | 消费侧解析，坏 JSON 包错误返回 | |
| `func ParseBroadcastNotifyPayload([]byte) (BroadcastNotifyPayload, error)` | 同上 | |

### 4.2 `core/queue/client.go`——RabbitMQ 生产者

| 函数/方法签名 | 职责 | 备注（坑） |
|---|---|---|
| `func NewClient(cfg config.Config) (*Client, error)` | Dial AMQP → 开 Channel → `QueueDeclare`（durable 队列） | 任一步失败都关闭已建资源；**生产者启动时也会声明队列**，所以 API 先于 RabbitMQ 起会直接启动失败 |
| `func (c *Client) Close() error` | 先关 channel 再关 conn，返回第一个错误 | nil 接收者安全 |
| `func (c *Client) EnqueuePushNotify(ctx, p PushNotifyPayload) (string, error)` | 入队实时推送（自动生成 TaskID） | 返回 taskID |
| `func (c *Client) EnqueuePushNotifyWithTaskID(ctx, p PushNotifyPayload, taskID string) (string, error)` | 同上但**指定幂等 TaskID** | 整点广播展开时用；注意：RabbitMQ 本身不做 taskID 去重，去重靠 Redis SetNX |
| `func (c *Client) EnqueueBroadcastNotify(ctx, p BroadcastNotifyPayload) (string, error)` | 入队整点广播任务 | scheduler-trigger 与 ticker 都调它 |
| `func (c *Client) EnqueueSendSMS(ctx, phone, message string) (string, error)` | 入队短信任务 | 预留 |
| `func (c *Client) EnqueueJPushRegister(ctx, p JPushRegisterPayload) (string, error)` | 入队极光注册任务 | 预留 |
| `func (c *Client) publish(ctx, env Envelope) (string, error)`（私有） | marshal 信封后发到默认交换机，routing key=队列名，持久化投递，MessageId=env.TaskID | 四个 Enqueue 方法的公共底座 |

文件头注释点明了分层约束：本包是基建层，入参用自己的 `PushNotifyPayload`，**不能反向 import 业务层的 `usecase.RealtimePushInput`**。

### 4.3 `core/queue/consumer.go`——消费循环与失败语义

| 函数/类型签名 | 职责 | 备注 |
|---|---|---|
| `type Dispatcher interface { Dispatch(ctx, body []byte) error }` | 消费侧分发契约 | 用接口隔离业务：基建层不认识 worker.Handler |
| `func NewConsumer(cfg, log) (*Consumer, error)` | Dial、开 Channel、声明队列、`Qos(prefetch, 0, false)`（prefetch count，global=false） | log 为 nil 用 Nop logger |
| `func (c *Consumer) Prefetch() int` / `Queue() string` | 读访问器 | Worker 启动日志用 |
| `func (c *Consumer) Close() error` | 关 channel/conn | |
| `func (c *Consumer) Run(ctx, h Dispatcher) error` | 阻塞消费循环，核心 20 行 | 见下 |

`Run` 的关键逻辑（`consumer.go:85-105`）：

```go
deliveries, _ := c.ch.Consume(c.queue, "", false, false, false, false, nil) // autoAck=false
for {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case d := <-deliveries:
        if err := h.Dispatch(ctx, d.Body); err != nil {
            c.log.Error("task failed", ...)
            _ = d.Nack(false, true) // multiple=false, requeue=true
            continue
        }
        _ = d.Ack(false)
    }
}
```

**失败重试语义（零杜撰，按代码说）**：

- 手动确认模式；业务返回 error → `Nack(false, true)` 立即重新入队，**没有退避、没有最大次数、没有死信队列**。一条永远失败的毒消息会被反复投递（且因为 prefetch 的存在会占住一个投递名额）。
- 未知任务类型 `unknown task type: %s` 同样返回 error → 同样无限重投。
- deliveries channel 被关闭 → `Run` 返回 `rabbitmq deliveries closed`（Worker `main` 对此 `log.Fatal`）。
- 成功才 `Ack(false)`。

### 4.4 `core/queue/pubsub_broadcaster.go`——Redis Pub/Sub 扇出

| 函数/类型签名 | 职责 | 备注 |
|---|---|---|
| `type FanoutMessage struct { UserID, Topic string; Envelope entity.RealtimeEnvelope }` | Pub/Sub 消息体，JSON 字段 `userId`/`topic`/`envelope` | 与 RabbitMQ 信封是两套协议 |
| `func NewFanoutPublisher(redisClient, channel) *FanoutPublisher` | 构造发布器 | |
| `func (p *FanoutPublisher) BroadcastToUser(userID, topic, envelope) int` | marshal 后 `redis.Publish` 到频道 | **无论成败恒返回 0**（marshal 失败/Publish 失败也返回 0）；它靠签名匹配 `usecase.RealtimeBroadcaster` 接口，但 Worker 侧因此拿不到真实 delivered 数 |
| `func NewFanoutSubscriber(redisClient, channel, broadcaster func(...), log) *FanoutSubscriber` | 构造订阅器；`broadcaster` 入参在 API 侧传的是 `hub.BroadcastToUser` 方法值 | |
| `func (s *FanoutSubscriber) Run(ctx) error` | `Subscribe` 频道，循环收消息交给 `handleMessage`；ctx 取消退出，defer Close pubsub | |
| `func (s *FanoutSubscriber) handleMessage(payload string)`（私有） | unmarshal → 校验 `userId != "" && envelope.Type != ""`（否则丢弃并 Warn）→ 调 broadcaster → Info 日志带 `delivered` 数 | 坏 JSON 只 Warn 不 panic |

> 设计取舍：Pub/Sub 是「发后即忘」。消息不持久化——但 `DeliverPush` 在广播前已把事件 `Append` 进 Redis 事件日志，离线/漏投由客户端 `/realtime/sync` 按 seq 补拉。

### 4.5 `core/queue/hourly_ticker.go`——Worker 内嵌的整点闹钟

| 函数签名 | 职责 | 备注 |
|---|---|---|
| `func RunHourlyNotifyTicker(ctx, client *Client, cfg, log)` | 每分钟 tick；分钟数为 0 且命中 cron 小时段且本 slot 未处理过 → `EnqueueBroadcastNotify` | 在 Worker `main` 里以 goroutine 启动；两个 enabled 开关任一为 false 直接 return |
| `func hourMatchesCron(cron string, hour int) bool`（私有） | **极简 cron 解析器**，只看第二个字段（小时段） | 支持 `*`、`10-19` 区间、单个数字；字段数不足时兜底 10–19；解析失败返回 false。分钟位 `0` 实际未解析，靠 `now.Minute()!=0 就 continue` 硬判断 |

去重细节：`lastSlot` 是**进程内变量**，同一整点 ticker 重复触发（理论上不会，每分钟只 tick 一次且要求 minute==0）只入队一次；多实例 Worker 各跑一个 ticker 时，跨实例去重由消费侧 Redis SetNX 兜住（见 5.3）。

### 4.6 包内测试文件（行为佐证，非生产代码）

| 测试文件 | 覆盖的事实 |
|---|---|
| `core/queue/tasks_test.go` | 广播/Push/SMS/JPush 四类信封的 type 正确；Push payload 往返一致且 TaskID 非空；坏 JSON 解析失败；`hourMatchesCron` 对 `10-19` 闭区间、`*` 通配的判定（9/20 不命中，10/19 命中） |
| `core/queue/pubsub_broadcaster_test.go` | `FanoutMessage` JSON 往返（type/seq/userId 保持）；`handleMessage` 用假 broadcaster 验证正确回调 userID/topic/envelope（测试里 redis client 传 nil 也能跑，因为只测消息处理） |

## 5. 业务侧（app 模块）

### 5.1 `app/internal/repository/queue/push_enqueuer.go`——防腐层适配器

| 函数/类型签名 | 职责 | 备注 |
|---|---|---|
| `type PushEnqueuer struct { client *pkgqueue.Client }` | 适配器：实现 usecase 层的 `PushEnqueuer` 端口 | 端口定义在 `usecase/realtime_push_usecase.go:42`：`EnqueueRealtimePush(ctx, RealtimePushInput) (string, error)` |
| `func NewPushEnqueuer(client) *PushEnqueuer` | client 为 nil 时返回 nil | 调用方据此表现为「未启用队列」 |
| `func (e *PushEnqueuer) EnqueueRealtimePush(ctx, input usecase.RealtimePushInput) (string, error)` | 调 `client.EnqueuePushNotify` | |
| `func toPayload(input) pkgqueue.PushNotifyPayload`（私有） | 业务类型 → 队列协议类型的逐字段拷贝 | 文件头注释解释：若让 `core/queue.Client` 直接实现业务端口，会形成 usecase → core/queue → usecase 循环依赖；转换放业务侧保证依赖单向 |

### 5.2 `app/internal/delivery/worker/handler.go`——任务分发表

| 函数/类型签名 | 职责 | 备注 |
|---|---|---|
| `type Handler struct { deliver; sessions; enqueue; hourly; jpushUC; cfg; redis; log }` | 持有全部业务依赖 | 文件头注释解释为何放 delivery 层：它依赖 usecase/repository，是业务不是基建 |
| `func NewHandler(deliver *RealtimePushUsecase, sessions SessionRepository, enqueue *Client, cfg, redis, log) *Handler` | 构造；内部 `NewHourlyNotifyUsecase(cfg)` | jpushUC 初始为 nil |
| `func (h *Handler) WithJPush(uc *JPushUsecase) *Handler` | 可选注入极光 usecase，返回自身便于链式 | Worker main 用 `.WithJPush(jpushUC)` |
| `func (h *Handler) Dispatch(ctx, body []byte) error` | unmarshal 信封后按 `env.Type` switch 四类任务；未知类型报错 | 实现 `pkgqueue.Dispatcher`；返回 error 即触发 Nack requeue |
| `func (h *Handler) handlePushNotify(...)`（私有） | Parse payload → 转成 `usecase.RealtimePushInput` → `deliver.DeliverPush` | DeliverPush 写 Redis 事件日志 + Pub/Sub 发布 + 极光离线推送 |
| `func (h *Handler) handleBroadcastNotify(...)`（私有） | 整点广播展开，见 5.3 详解 | |
| `func (h *Handler) handleSendSMS(...)`（私有） | **桩**：只打 Info 日志 `sms task stub — integrate SMS provider in production`，永远返回 nil（会 Ack） | |
| `func (h *Handler) handleJPushRegister(...)`（私有） | jpushUC 为 nil → 打日志跳过并返回 nil；否则 `jpushUC.RegisterDevice` | |
| `func NewDeliveryPushUsecase(events RealtimeEventRepository, cfg, publisher *FanoutPublisher) *RealtimePushUsecase` | Worker 专用装配：把 `FanoutPublisher` 当作 broadcaster 传给推送 usecase（第四个参数 queue 端口传 nil） | 关键：Worker 不直接碰 WS Hub |

### 5.3 广播展开的完整逻辑（`handleBroadcastNotify`，`handler.go:109-167`）

这是全篇最绕的一段，逐步拆解：

1. **定槽位 slot**：优先用 payload 的 `ScheduleSlot`（RFC3339，解析失败则回退当前时间），统一转调度时区并截断到整点。
2. **取活跃用户**：`sessions.ListActiveUserIDs(ctx)` —— 用 Redis `SCAN` 扫 `auth:session:*` 前缀的 key（`session_repo.go:56`，前缀常量 `auth:session:`），每批 100。
3. **逐个用户**循环：
	1. `cfg.Auth.IsSessionExempt(userID, "")` 命中（多端豁免/测试账号）→ skipped++ 跳过。
	2. `taskID = hourly.DedupTaskID(userID, slot)`，去重 key = `scheduler:dedup:<taskID>`。
	3. `redis.SetNX(key, 1, 2*time.Hour)`：已存在 → skipped++；Redis 报错 → 整个任务返回 error（Nack 重投）。
	4. `hourly.BuildPushInput(userID, slot)` 组装推送内容，`EnqueuePushNotifyWithTaskID(..., taskID)` 入队**每用户一条** push 任务。
	5. 入队失败 → **删掉刚写的去重 key** 再返回 error（保证重投时该用户能重试），整个广播任务 Nack。
4. 收尾打 Info：`total_sessions / enqueued / skipped`。

幂等保证：SetNX 先占位 → 即使广播任务被重投，已入队过的用户不会重复收到；而「占位成功但入队失败」时主动回滚 key，避免漏投。

### 5.4 `app/internal/usecase/hourly_notify_usecase.go`——整点通知内容组装

| 函数/方法签名 | 职责 | 备注 |
|---|---|---|
| `func NewHourlyNotifyUsecase(cfg) *HourlyNotifyUsecase` | 构造，仅持有 cfg | |
| `func (u *HourlyNotifyUsecase) BuildPushInput(userID string, slot time.Time) RealtimePushInput` | 组装单个用户的推送：截断整点→生成 campaignID→选模板→算过期时间戳→可选 action→`entity.NewScheduledSysNotify` | Topic 固定 `sys.notify`，Name 固定 `sys.notify.show`，Extra 来自 `payload.ToExtraMap()` |
| `func (u *HourlyNotifyUsecase) DedupTaskID(userID, slot) string` | 返回 `<campaignID>:<userID>` | campaignID 由槽位压缩而来 |
| `func resolveHourlyTemplate(hn, hour) (title, body)`（私有） | 10 点固定「上午好/新的一天开始了，查看今日动态」；19 点固定「今日小结/今日内容已更新，点击查看」；其他整点用配置模板替换 `{{hour}}`、`{{message}}` | 模板默认值：标题「整点提醒」、正文「现在是 {{hour}}:00，{{message}}」、默认消息「别错过重要消息」 |

配套的领域工具（`model/entity/sys_notify_payload.go`）：

- `CampaignIDFromSlot("2026-10-23T10:00:00+08:00")`：依次删除 `:`、`-`、`+` 并把 `T` 替换成 `-`，得到 `hourly-20261023-1000000800`（前缀 `hourly-`）。
- `NewScheduledSysNotify` 固定 `category=scheduled`、`messageType=hourly_digest`、`locale=zh-CN`、priority=normal、metadata 默认 `{"source":"scheduler","version":"1"}`。
- `ExpiresMinutesOrDefault` 默认 120 分钟；action 默认 `type=deeplink`、`route=/home`（config defaults）。

### 5.5 `app/internal/usecase/realtime_push_usecase.go`——同步/异步双模式（被两侧复用）

Worker 篇必须讲清这个 usecase，因为 API 和 Worker 都调它、但注入的依赖不同：

| 函数/方法签名 | 职责 | 备注 |
|---|---|---|
| `type PushEnqueuer interface { EnqueueRealtimePush(...) }` | 异步入队端口 | API 侧实现是 5.1 的适配器；Worker 侧构造时传 nil |
| `type RealtimeBroadcaster interface { BroadcastToUser(userID, topic, envelope) int }` | 广播端口 | API 侧实现是 WS `Hub`；Worker 侧实现是 `FanoutPublisher` |
| `type OfflinePusher interface { NotifyUser(...) }` | 离线通道（极光） | 失败不阻断主流程 |
| `func (u *RealtimePushUsecase) PushToUser(ctx, input) (RealtimePushOutput, error)` | API 入口：`UseAsyncPush() && queue!=nil` 时入队返回 `{Queued:true, TaskID, Delivered:-1}`；否则直接 `DeliverPush` 同步投递 | 见 `realtime_push_usecase.go:81` |
| `func (u *RealtimePushUsecase) DeliverPush(ctx, input) (envelope, delivered, error)` | Worker 与同步模式共用：校验 → `events.NextSeq` 取序号 → 组装 payload（含新 uuid notifyId）→ `events.Append` 写事件日志（带保留期）→ `hub.BroadcastToUser` → 若注入了 OfflinePusher 再极光补发（错误忽略） | 先写日志再广播，保证离线可补拉 |
| `func (u *RealtimePushUsecase) SetOfflinePusher(p)` | 构造后注入极光 | Worker/API main 都在 JPush 表初始化成功后调 |

### 5.6 两个可执行入口

#### `app/cmd/worker/main.go`（`make run-worker`）

`run()` 装配顺序（按代码顺序）：

1. `APP_ENV`（默认 dev）→ `config.Load`；`queue.enabled=false` 直接报错退出。
2. 初始化 logger、Redis；建 `RealtimeEventRepository`、`SessionRepository`、`FanoutPublisher`（频道取 cfg）。
3. `worker.NewDeliveryPushUsecase(...)` 得到 Worker 版推送 usecase。
4. `queue.NewClient`（入队广播展开后的 per-user 任务、ticker 也要用）。
5. 连 Postgres；初始化极光设备表（失败仅 Warn），成功则建 JPush client/usecase 并 `pushUC.SetOfflinePusher`。
6. `worker.NewHandler(...).WithJPush(jpushUC)` + `queue.NewConsumer`。
7. 起两个 goroutine：`consumer.Run(ctx, handler)`（异常退出 `log.Fatal`）与 `queue.RunHourlyNotifyTicker(ctx, queueClient, cfg, log)`。
8. 等 SIGINT/SIGTERM → `cancel()` 优雅停止。

| 关键点 | 说明 |
|---|---|
| Worker 同时持有 Consumer 与 Client | 因为广播任务要二次入队 per-user push |
| Worker 连 Postgres 只为 JPush 设备表 | 推送主路径只依赖 Redis + RabbitMQ |
| Worker 不订阅 Pub/Sub、不持有 Hub | 它只发布；订阅方是 API 实例 |

#### `app/cmd/scheduler-trigger/main.go`（`make trigger-hourly-notify`）

约 40 行的一次性小工具：加载配置（queue 未启用则退出）→ `NewClient` → `EnqueueBroadcastNotify(ctx, BroadcastNotifyPayload{})`（空槽位，Worker 用当前整点）→ 打印 taskId 并提示「请确认 make run-worker 正在运行」。

### 5.7 API 侧的发布与订阅接线（`app/cmd/api/main.go`）

为讲全链路，补充 API 进程内的三段接线：

- `main.go:306-326`：`queue.enabled` 时建 `queueClient` → `repoqueue.NewPushEnqueuer(queueClient)` 作为 usecase 的入队端口；同时建 `FanoutSubscriber(redis, channel, hub.BroadcastToUser, log)` 并 goroutine `Run`（日志「Redis Pub/Sub 订阅已启动」）。
- `main.go:328`：`NewRealtimePushUsecase(eventRepo, cfg, hub, pushEnqueuer)` —— 同一个 usecase，broadcaster 是本地真 Hub，queue 是适配器。
- HTTP 入口：`POST /api/v1/realtime/push`（路由 `router/realtime_routes.go`，中间件链 = `sessionAuthChain(sbAuth, accountGate)`，即 SessionAuth + 可选账号门）→ `RealtimeController.Push`（`realtime_controller.go:96`）→ `pushUC.PushToUser`；请求体未带 `userId` 时推给登录者本人。

## 6. 全链路时序

### 6.1 链路 A：API 发布 → Worker 消费 → Pub/Sub 扇出 → WS（异步模式）

1. 客户端 `POST /api/v1/realtime/push`（Bearer token + X-Session-ID + X-Device-ID）。
2. `RealtimeController.Push` → `RealtimePushUsecase.PushToUser`；`UseAsyncPush()` 为真 → `PushEnqueuer.EnqueueRealtimePush`（适配器转 payload）→ `queue.Client.EnqueuePushNotify`：消息持久化写入 RabbitMQ `my_go_study.tasks`，HTTP 立刻返回 `{queued:true, taskId, delivered:-1}`。
3. Worker `Consumer.Run` 拉到 delivery（prefetch 内）→ `Handler.Dispatch` 命中 `realtime:push_notify` → `handlePushNotify` → `RealtimePushUsecase.DeliverPush`：
	1. Redis `NextSeq` 取该用户递增序号；
	2. `Append` 事件到 Redis 事件日志（带保留期）；
	3. `FanoutPublisher.BroadcastToUser` → Redis `PUBLISH realtime:fanout <FanoutMessage JSON>`；
	4. 若配了极光，`OfflinePusher.NotifyUser` 补发（失败仅忽略）。
4. 每个 API 实例的 `FanoutSubscriber` 收到消息 → 校验 → 本实例 `hub.BroadcastToUser(userID, topic, envelope)`：遍历该用户在本进程的连接，凡订阅了 topic 的 `client.Send(envelope)`，返回送达连接数并打日志。
5. 业务成功 → Worker `Ack`；任何一步 error → `Nack(false,true)` 重投。
6. 用户不在线则 delivered=0，但事件已落 Redis；客户端重连后 `POST /api/v1/realtime/sync`（`sinceSeq` + topics）补拉。

> 开发环境 `push_async: false` 时第 2 步短路：API 在同一请求内直接 `DeliverPush`，此时 broadcaster 是本地 Hub（同步直投本机 WS），不经过 RabbitMQ/Pub/Sub；响应里 `queued` 为假、`delivered` 为真实连接数。

### 6.2 链路 B：整点广播（两层队列展开）

1. 触发源二选一：Worker 内 ticker（prod：整点分钟 & cron 命中，自动入队）或 `make trigger-hourly-notify`（dev：手动入队）。产生 1 条 `scheduled:broadcast_notify`。
2. Worker `handleBroadcastNotify`：SCAN 活跃 session → 每用户 SetNX 去重（2h）→ `BuildPushInput`（campaignID/模板/action）→ 每用户入队 1 条 `realtime:push_notify`（携带确定性 taskID）。
3. 后续与链路 A 的第 3 步起完全相同。

## 7. 三个联调脚本

| 脚本 / make 目标 | 干什么 | 关键断言与前置 |
|---|---|---|
| `scripts/test_queue_push.sh`（`make test-queue-push`） | 登录拿 token/session → `POST /realtime/push` → sleep 2 → `POST /realtime/sync` 验证事件落库可补拉 | 断言响应 `data.queued is true` 且有 `taskId`；sync 的 `sys.notify` 事件数 ≥1。前置 `make run` + `make run-worker`；登录失败时若配了 `SUPABASE_SERVICE_ROLE_KEY` 会临时建测试用户。⚠️ 注意：`config.dev.yaml` 当前是 `push_async: false`，直接跑会得到同步响应（queued 非 true），要满足脚本断言需用 `QUEUE_PUSH_ASYNC=true` 覆盖或改配置 |
| `scripts/test_scheduled_notify.sh`（`make test-scheduled-notify`） | 登录建 session → `go -C app run ./cmd/scheduler-trigger` 手动入队广播 → sleep 4 → sync 校验 | 断言最后一个事件 `payload.category == "scheduled"` 且 `campaignId` 非空。前置：API + Worker 都在跑且用户有 `auth:session` |
| `scripts/push_notify_user.sh`（`make push-notify-user USER_ID=... EMAIL=... TITLE=... BODY=...`） | 向指定 userId 推一条 sys.notify（默认 userId/邮箱写死在脚本参数里），打印 queued/delivered/taskId/envelope | 文件头注释明确：开发环境 `push_async=false` 时**同步直投 WS**；无连接时提示 delivered=0 可 sync 补拉；可用 `PUSH_OPERATOR_TOKEN`/`PUSH_OPERATOR_SESSION` 跳过临时建号 |

三个脚本都先 `source scripts/source-env.sh`，通过 `/health` 探活，失败时给出「终端 1 make run / 终端 2 make run-worker」提示。

## 8. 占位与边界（按代码标注）

- `sms:send` 消费者是**纯桩**，只打日志，不接任何短信供应商。
- `EnqueueSendSMS` 有生产者方法但仓库内没有业务调用方（为生产 OTP 预留）。
- `FanoutPublisher.BroadcastToUser` 恒返回 0，Worker 日志里的投递数不代表真实在线连接数；真实 delivered 只在各 API 实例订阅器日志里。
- RabbitMQ 无 DLX/重试上限/退避；毒消息与未知类型消息会无限重投（消费侧现状，改进需引入 DLX 或 retry 计数）。
- Pub/Sub 不持久化，订阅器宕机期间的扇出消息直接丢失，靠事件日志 sync 兜底。
- ticker 的 cron 解析只支持小时字段的 `*`/`a-b`/单值，不是完整 cron。
- Worker 进程内 ticker 多实例运行时靠下游 SetNX 幂等，ticker 自身无分布式锁（会重复入队广播任务，但展开阶段被去重）。

## 9. 学习要点回顾

1. **两种中间件各司其职**：RabbitMQ 做可靠点对点任务（持久化 + 手动 Ack），Redis Pub/Sub 做跨实例在线扇出；事件先落 Redis 日志再广播，把「可靠」与「实时」解耦。
2. **依赖方向靠端口+适配器维持**：usecase 定义 `PushEnqueuer`/`RealtimeBroadcaster` 接口，`core/queue` 与 `app/internal/repository/queue` 各自实现，避免基建层反向依赖业务层成环。
3. **同一 usecase、两套装配**：API 注入「真 Hub + 入队适配器」，Worker 注入「FanoutPublisher + nil 队列」，`PushToUser`（异步分支）与 `DeliverPush`（同步投递）因此可以复用。
4. **消息协议分两层**：RabbitMQ 的 `Envelope{type,task_id,payload}` 是任务协议；Redis 频道的 `FanoutMessage{userId,topic,envelope}` 是投递协议，不要混淆。
5. **手动 Ack 模型**：autoAck=false，成功 Ack、失败 `Nack(requeue=true)`；能保证 at-least-once，所以业务侧必须幂等——本项目用 Redis SetNX + 确定性 taskID 实现。
6. **定时任务也是消息**：没有引入独立调度器，ticker 只负责「到点往队列塞任务」，重试点、幂等点全部统一在 Worker 消费侧。
7. **配置驱动同步/异步切换**：`queue.enabled` + `push_async` 两个布尔决定形态；nil 与显式 false 语义不同（未配默认异步）。
8. **优雅停机**：SIGINT/SIGTERM → cancel context，消费循环与 ticker、Pub/Sub 订阅都通过 `ctx.Done()` 退出。

## 10. 可动手验证

读代码：

- 对照 `consumer.go:85-105` 的 Ack/Nack 分支，在纸上推演一条「解析失败的坏消息」会发生什么（提示：无限 requeue）。
- 追踪 `handleBroadcastNotify` 中 SetNX key 的生命周期：入队失败为何要 `Del(key)`？不删会怎样？
- 对比 API main 与 Worker main 中 `NewRealtimePushUsecase` 的入参差异，说出谁在什么情况下调用 `hub.BroadcastToUser`。

跑命令（本机需起好 RabbitMQ/Redis，见 docker 目录）：

```bash
make run            # 终端 1：API
make run-worker     # 终端 2：Worker
make test-queue-push                 # 异步链路（注意 push_async 需为 true）
make trigger-hourly-notify           # 手动入队一条广播
make test-scheduled-notify           # 广播端到端 + sync 校验
QUEUE_PUSH_ASYNC=false make push-notify-user  # 观察同步直投响应字段差异
```

观察点：Worker 日志的 `prefetch/queue/pubsub_channel` 启动行、`hourly broadcast enqueued` 统计表、`pubsub fanout delivered` 行；RabbitMQ 管理台（默认 guest/guest，15672）可看到持久化队列 `my_go_study.tasks`；Redis 侧可 `SUBSCRIBE realtime:fanout` 直接看扇出消息体。
