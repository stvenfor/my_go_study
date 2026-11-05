# Acceptance Record — parity-p0-6-mq-dlx-retry

- 日期：2026-11-03
- Slice：`plans/slices/parity-p0-6-mq-dlx-retry.md`
- Epic：`plans/epics/xhs-parity-p0-correctness.md` §P0-6
- 主交付物：`core/queue/{topology,retry,consumer,client}.go` + `common/metrics` DLQ gauge + `deploy/observability/alerts.yml` + `scripts/rabbitmq-ensure-dlq.sh` + `docs/message-queue.md`

## status: **Degraded**

note（为什么不是一句 Migrated）：

1. **死信链路本身是 Migrated 的**：DLX/DLQ/绑定声明、有界重试、超限进死信、DLQ gauge、告警规则，
   全部在真实 RabbitMQ 3.13 上跑通并留了取证（见「硬门槛」）。
2. **`make lan-up` 闸门未跑**：本机 `:8080` 被一个无关进程（`super-fac`，PID 40935）占用，compose 端口映射必冲突；
   我没有动那个进程。改用**等价证据**：同一份 406 降级代码路径在 **API 进程**与 **Worker 进程**各自实机验证
   （把队列删成无 args 形状后重启进程 → `Warn(status=degraded)` + `health=200`，不死循环不崩）。
   compose 层的「一次 `make lan-up` 通过」仍是缺口，需要你在 `:8080` 空出来时补跑。
3. **单条超时无实机证据**：现存 4 类任务里没有任何一条会真的跑满 1s，拿不到自然的「慢任务」。
   已用单测把行为钉死（handler 卡住 → 50ms 超时截断返回并按失败重试），
   但「线上真有一条卡死」这件事没实测。
4. **幂等只做了复核，没做改造**：Brief 的验收条「重复投递 4 类任务副作用只发生一次」对
   `realtime:push_notify` **不成立**（at-least-once，重试会多推一帧 WS）。
   服务端按 `task_id` 去重（SetNX）我没加——那属于改投递语义，超出「不改任务类型语义与载荷结构」，登记为待办。
5. **`Client` 无自动重连**：broker 重启后进程需一起重启，否则一直 `channel/connection is not open`
   （这条反而让闸门③的失败非常干净）。归 P1。

## 机器验证（本机实跑）

> Brief 的两条验证命令按原文不可执行，已点名替换：
> `go build all`（`github.com/mszlu521/thunder/logs` 缺目录，先于本片失败）→ 分模块 build；
> `amtool check-config alerts.yml`（amtool 只认 alertmanager 自身配置，不认顶层 `groups:`）→ `promtool check rules`。

| 命令 | 结果 |
|------|------|
| `go -C app build ./...` / `go -C core build ./...` / `go -C common build ./...` | exit 0 |
| `go -C app vet ./...` | exit 0 |
| `go -C core test ./queue/... -count=1` | ok（11 个 handle 用例全 PASS） |
| `go -C app test ./internal/delivery/worker/... -count=1` | ok |
| `promtool check rules deploy/observability/alerts.yml`（v2.54.1 容器） | `SUCCESS: 5 rules found`（含新增 2 条 queue 规则） |
| `make check-secrets` | ✅ 通过 |
| `bash scripts/rabbitmq-ensure-dlq.sh status` | 列出 `my_go_study.tasks`(0) / `.dlq` / `.dlx` |
| `APP_ENV=dev bash scripts/rabbitmq-ensure-dlq.sh recreate --yes` | 删队列入选生效，下次启动自动带 args 重建 |

新增断言（`core/queue/consumer_test.go`）：未到上限**重投** `x-retry-count=int64(2)` 并 Ack 原消息、已有计数 2→3 递增、
重投失败不得 Ack（`Nack(requeue=true)` 保消息）、达上限 0 次重投 + `Nack(requeue=false)`、
legacy `x-delivery-count` / `x-death` 上限、**计数未知且 redelivered → 直接进死信**（防无限循环硬门槛）、
单条 deadline 存在、**慢 handler 被超时截断**、父 ctx 取消不传染在途任务。
`topology_test.go`：默认/覆写派生名、`x-dead-letter-routing-key` 必须等于队列名、`isPreconditionFailed` 对
`amqp.Error` 值/指针/包装/404/nil 五种输入的判定。`retry_test.go`：9 种头组合的计数优先级。

## 硬门槛（①②③④）实测

 broker 为 `docker` 里的 `my_go_study_rabbitmq`（3.13），注入走 management API
 `POST /api/exchanges/%2F/amq.default/publish`。

### ① 必然失败的消息 N 次后进 DLQ，不再无限循环 —— **PASS（同拓扑重建后复跑两次）**

```
WARN task failed, republished for retry   {"finished_attempt":1,"next_attempt":2,"max_retries":3,"message_id":"poison-final","error":"unknown task type: bogus:final"}
WARN task failed, republished for retry   {"finished_attempt":2,"next_attempt":3,"max_retries":3,"message_id":"poison-final",...}
WARN task exhausted retries, moved to DLQ {"dlq":"my_go_study.tasks.dlq","dlq_wired":true,"attempt":3,"attempt_known":true,...}

rabbitmqctl list_queues name messages
my_go_study.tasks.dlq   1
my_go_study.tasks       0        ← 数字停在 0，没有无限增长
```

DLQ 死信取证（management API `ackmode=ack_requeue_true` 读回）：
`message_id=poison-final`、`x-retry-count=3`、`x-death=[{reason:"rejected"}]`、payload 原文完整。

> 关键设计事实：**RabbitMQ 对 `Nack(requeue=true)` 的重投消息不写 `x-delivery-count`，也不写 `x-death`**。
> 第一版按 broker 计数实现时，第 2 次投递就 `attempt_known:false` 直接进死信（只重试了 1 次）。
> 所以计数必须由应用自己重投带 `x-retry-count` 的新消息来承担，broker 计数只作兼容回退。

### ② 旧队列已存在时不因 406 启动失败 —— **PASS（进程级，双进程）/ PARTIAL（compose 级未跑）**

删掉队列并以**无 args** 形状重建（模拟历史遗留），再起进程：

```
WARN queue/topology.go:81 task queue args mismatch (406): 沿用旧队列，死信未生效；请在维护窗口执行 scripts/rabbitmq-ensure-dlq.sh 重建
     {"queue":"my_go_study.tasks","dlq":"my_go_study.tasks.dlq","status":"degraded",
      "error":"declare queue my_go_study.tasks: Exception (406) Reason: \"PRECONDITION_FAILED - inequivalent arg 'x-dead-letter-exchange' ...\""}
health=200
```

Worker 侧同样：`RabbitMQ Worker 启动 {"dlq_wired":false}` 正常消费，不退出。
执行 `recreate --yes` 后重启 → `{"prefetch":10,"queue":"my_go_study.tasks","dlq":"my_go_study.tasks.dlq","dlq_wired":true}`。

### ③ 停 RabbitMQ 后入队失败不返回成功 —— **PASS**

`docker stop my_go_study_rabbitmq` 后，用**已建立连接**的 API 进程发异步推送：

```
http=500
{"error":"enqueue push notify: Exception (504) Reason: \"channel/connection is not open\"","code":50000}
```

正向对照（broker 在线，同一进程同一账号）：`{"queued":true,"taskId":"cf23393f-…","delivered":-1}` + HTTP 200。
CLI 侧 `go -C app run ./cmd/scheduler-trigger` → `exit=1`，`连接 RabbitMQ 失败: rabbitmq dial: … connection refused`。
入队调用点全量核对：`Client.Enqueue*` 全部包装返回 error；`PushToUser` 上抛 → controller `BackendError(500)`；
`communityPushAdapter.Notify` 返回 err；`handleBroadcastNotify` 失败先删 dedup key 再返回 err；`hourly_ticker` 记 error 跳过。
（`EnqueueSendSMS` / `EnqueueJPushRegister` 目前**无生产调用点**，只有测试——已在 P0-5 清单登记。）

同一场景下 `/metrics` 里 `rabbitmq_dlq_messages` **整条消失**（抓取 0 条样本，而不是假的 `0`），
`MQDeadLetterMetricMissing` = `absent(...)` 因此能区分「死信为 0」与「采集不到」。

### ④ Worker SIGTERM 发 N 消费 N —— **PASS（含一处口径说明）**

`RABBITMQ_PREFETCH=1`，连投 6 条 `sms:send` 后立即 `kill -TERM`：

```
收到关机信号，停止消费...   （15:13:48.332）
Worker 已关闭              （15:13:48.332，进程 ~0.5s 内退出）
日志内占位 Warn 行数 = 6
rabbitmqctl list_queues → my_go_study.tasks  0
```

口径说明：6 条处理耗时 <1ms，TERM 到达时大概率已全部消费完，**没有真正制造出「关机瞬间在途」的重叠**。
在途等待这段由 `consumer.Wait()` + `TestHandleIgnoresParentCancellation`（父 ctx 取消不打断 handle）
与超时截断用例覆盖；未丢消息（0 条留在队列）是实测的。

## 改动清单

| 文件 | 改动 | 白名单 |
|------|------|--------|
| `core/queue/topology.go` | 新增：DLX/DLQ 声明 + 406 判定与降级 | ✅ |
| `core/queue/retry.go` | 新增：尝试计数读取 | ✅ |
| `core/queue/consumer.go` | 重写 handle：有界重试 / 单条超时 / 在途 WaitGroup / Ack-Nack 错误可见 | ✅ |
| `core/queue/client.go` | `NewClient(cfg, log)`、拓扑复用、`DLQDepth()`、publisher confirm | ✅ |
| `common/config/config.go` | `dlx/dlq/max_retries/handler_timeout_seconds/publisher_confirm` + 5 个 BindEnv | ✅ |
| `configs/config.yaml` | 上述默认值与不可变参数注释 | ✅ |
| `common/metrics/metrics.go` | `NewDepthCollector`（抓取失败**不输出**指标） | ✅ |
| `app/cmd/worker/main.go` | 启动日志带 dlq 状态、`log.Fatal`→`Error`、`cancel()` → `<-runDone` → `consumer.Wait()` | ✅ |
| `deploy/observability/alerts.yml` | `queue` 组 2 条规则 | ✅ |
| `scripts/rabbitmq-ensure-dlq.sh` | 新增：`status` / `recreate --yes` / `rename` | ✅ |
| `docs/message-queue.md` | 可靠性章节、配置项、入口表、重复投递语义 | ✅ |
| `core/queue/{consumer,topology,retry}_test.go` | 新增/扩充 | ✅ |
| **`app/cmd/api/main.go`** | **偏差**：注册 `rabbitmq_dlq_messages` gauge（不在白名单） | ⚠️ 偏差 |
| **`app/cmd/scheduler-trigger/main.go`** | **偏差**：`NewClient` 签名加 log，传 `nil`（1 行） | ⚠️ 偏差 |

api main.go 的偏差理由：`/metrics` 只在 API 进程暴露，gauge 若不在这里注册就只能塞进 worker（无 HTTP 端点）
或让 `core` 依赖 prometheus（破坏分层）。黑名单目录（payment / transaction / core/database/redis.go / thunder）未触碰。

## 待办（本片不做，需后续 Slice）

1. `realtime:push_notify` 服务端 `task_id` 去重（SetNX，参照 broadcast 的做法）→ 让「重复投递只一次副作用」真正成立。
2. `core/queue.Client` 自动重连 + 消费循环断线重启（现需重启进程）。
3. `:8080` 空出来后补一次 `make lan-up` + `make test-queue-push`，把 ② 的 compose 级证据补齐。
4. 死信「重放」工具（从 DLQ 手工重投回业务队列）——运维需要时再加，本片只保证可查。
