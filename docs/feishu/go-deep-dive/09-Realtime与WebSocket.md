# 09 Realtime 与 WebSocket：一次性 ticket、内存 Hub、Redis 事件流与 presence

> 一句话定位：Realtime 是「HTTP 换票 → WS 首帧验票 → Hub 内存广播 → Redis 兜底补发」的实时消息子系统，属于 legacy 分层（delivery/ws + delivery/http + usecase + repository/redis），阅读前置：懂 Gin 中间件与 SessionAuth（见 [04-HTTP路由与中间件](04-HTTP路由与中间件.md)）、懂 Redis 基本数据结构。

## 0. 通俗理解

把整套 Realtime 想象成**演唱会入场**：

1. 你先在售票窗口（HTTP `POST /api/v1/realtime/ws-ticket`，凭登录态）换一张**一次性手环**（ticket，UUID，默认 120 秒过期）。
2. 到场馆门口（WebSocket `/realtime/v1/connect`）门不查身份证，进门后第一件事是出示手环（首帧 `{type:"auth", ticket}`）。手环被**剪角**（Redis `GETDEL`），用过即废，别人捡到也进不来。
3. 进场后有个**场内管理员 Hub**：脑子里记着「每个用户有几条连接、每条连接订了哪些频道（topic）」。有人喊「给张三推通知」，Hub 只往张三那些订阅了对应频道的连接手里塞信。
4. 你离场（断线）时 Hub 划掉你的名字；断线期间漏掉的信，前台 Redis 留着最近 200 封，你重连后拿序号（seq）去 HTTP `/sync` 补领。

为什么不在 WebSocket 握手时直接验 JWT？浏览器 WS API 不能随便加请求头，把短时 ticket 放 URL/首帧更通用；ticket 由已鉴权的 HTTP 接口签发，安全性等价。

## 1. 源码地图

| 层 | 文件（相对路径） | 职责 |
|----|------------------|------|
| 契约 | `model/entity/realtime_envelope.go` | WS 消息包络、topic/事件名常量、关闭码 |
| 契约 | `model/entity/sys_notify_payload.go` | 系统通知 payload 结构与定时通知构造 |
| 端口 | `model/repository/ws_ticket_repository.go` | ticket 仓储接口 + `WSTicket` 结构 |
| 端口 | `model/repository/realtime_event_repository.go` | 事件 seq/追加/补发接口 |
| 端口 | `model/repository/presence_repository.go` | 在线状态存储接口 |
| 端口 | `model/repository/conversation_repository.go` | SSE 停留会话接口（同 redis 包，非 WS） |
| Redis 实现 | `app/internal/repository/redis/ws_ticket_repo.go` | ticket SET / GETDEL |
| Redis 实现 | `app/internal/repository/redis/realtime_event_repo.go` | INCR seq + List 留痕 + LRANGE 补发 |
| Redis 实现 | `app/internal/repository/redis/presence_repo.go` | Set 在线集合 + 每用户 TTL key |
| Redis 实现 | `app/internal/repository/redis/conversation_repo.go` | SSE 会话 JSON 存取（旁路） |
| 用例 | `app/internal/usecase/realtime_ticket_usecase.go` | 签发/消费 ticket |
| 用例 | `app/internal/usecase/realtime_sync_usecase.go` | 重连增量同步 |
| 用例 | `app/internal/usecase/realtime_push_usecase.go` | 服务端推送（同步/异步入队） |
| 用例 | `app/internal/usecase/realtime_presence_usecase.go` | presence 上报与广播 |
| WS | `app/internal/delivery/ws/hub.go` | 连接注册中心 + 两类广播 |
| WS | `app/internal/delivery/ws/client.go` | 单连接状态、读写双 pump |
| WS | `app/internal/delivery/ws/handler.go` | Upgrade、首帧验票、消息分发 |
| HTTP | `app/internal/delivery/http/controller/realtime_controller.go` | ticket/sync/push 三个接口 |
| DTO | `app/internal/delivery/http/dto/request/realtime_request.go` | 三个请求体 |
| DTO | `app/internal/delivery/http/dto/response/realtime.go` | 三个响应体 |
| 路由 | `app/internal/delivery/http/router/realtime_routes.go` | `/api/v1/realtime/*` |
| WS 挂载 | `app/internal/delivery/http/router/router.go` L106-111 | `GET {ws_path}` → `ServeWS`（无中间件） |
| 装配 | `app/cmd/api/main.go` L295-347 | Hub/UC/Repo 构造、Pub/Sub 接线 |
| 跨实例 | `core/queue/pubsub_broadcaster.go` | Redis Pub/Sub 多实例扇出 |
| Worker | `app/internal/delivery/worker/handler.go` | 消费 RabbitMQ 后调 `DeliverPush` |
| 脚本 | `scripts/test_realtime_ws.sh` | 登录→换票→push→sync→go test 全链路 |

## 2. 消息契约：RealtimeEnvelope

源码锚点：`model/entity/realtime_envelope.go`

**通俗理解**：每条 WS JSON 都是一个「信封」——`type` 决定怎么处理，`topic` 是频道，`payload` 是信纸。信封格式 Go 和 Flutter 必须逐字对齐，所以放在 model 层当唯一契约。

### 2.1 包络字段

| 字段 | 类型 | 含义 | 备注 |
|------|------|------|------|
| `ID` | string | 消息编号 | ping/pong 同 id 配对；ack 用 `payload.refId` 指回 |
| `Type` | string | 消息类型 | `auth/auth_ok/ping/pong/sub/unsub/ack/event/error` |
| `Topic` | string | 订阅主题 | 仅 event 需要，如 `sys.notify`、`presence.bulk` |
| `TS` | int64 | 毫秒时间戳 | 调试排序 |
| `Seq` | int64 | 按用户单调递增序号 | 断线 sync 去重；presence 不带 seq |
| `Payload` | map | 业务字段 | 用 map 而非固定 struct，扩展事件不改契约 |

### 2.2 常量

| 常量 | 值 | 用途 |
|------|----|------|
| `TopicSysNotify` | `sys.notify` | 系统通知频道 |
| `TopicPresenceBulk` | `presence.bulk` | 在线状态批量频道 |
| `EventSysNotifyShow` | `sys.notify.show` | 展示 Banner（payload.name） |
| `EventPresenceReport` | `presence.report` | 客户端 → 服务端上报 |
| `EventPresenceUpdate` | `presence.update` | 服务端 → 其他人广播 |
| `WSCloseAuthFailed` | 4001 | auth 缺 ticket |
| `WSCloseKicked` | 4002 | 预留：被踢下线（当前未使用） |
| `WSCloseTicketExpired` | 4003 | ticket 无效/过期/已用 |

`sys_notify_payload.go` 另定义 `SysNotifyPayload`（title/body/category/action/静默/过期等）、`NewScheduledSysNotify`（定时通知构造，category=scheduled）、`ToExtraMap`、`CampaignIDFromSlot`；push usecase 实际只把 Extra map 合并进 payload，未直接强类型化（见 §5.3 坑）。

## 3. 端口（model/repository 接口）

| 接口 / 结构 | 方法 | 备注 |
|-------------|------|------|
| `WSTicket`（struct） | `UserID / ConnID / Platform` | Redis 里存的 JSON |
| `WSTicketRepository` | `Save(ctx, ticket, data, ttl)` | SET EX |
| | `Consume(ctx, ticket) (*WSTicket, error)` | 原子 GETDEL |
| `RealtimeEventRepository` | `NextSeq(ctx, userID) (int64, error)` | `INCR realtime:seq:{uid}` |
| | `Append(ctx, userID, event, retention)` | RPUSH + LTRIM 定长 |
| | `ListSince(ctx, userID, sinceSeq, topics, retention)` | 补发 |
| `PresenceRepository` | `SetOnline(ctx, userID, device) (count, error)` | 返回在线总人数 |
| | `SetOffline(ctx, userID) (count, error)` | 同上 |
| `ConversationRepository`（SSE 旁路） | `Create/Get/SaveTurns/Touch` | 给 AI 小石头 SSE 用，**不在 WS 链路** |

## 4. Redis 仓储实现（repository/redis）

### 4.1 `ws_ticket_repo.go`

| 函数/方法签名 | 职责 | 备注 |
|---------------|------|------|
| `NewWSTicketRepository(client *redis.Client) *WSTicketRepository` | 构造 | main.go 注入 |
| `(r) Save(ctx, ticket string, data repository.WSTicket, ttl time.Duration) error` | JSON 序列化后 `SET ws:ticket:{ticket} ... EX ttl` | key 前缀常量 `ws:ticket:` |
| `(r) Consume(ctx, ticket string) (*repository.WSTicket, error)` | `GETDEL` 取值即删 | `redis.Nil` → 「ticket 无效或已过期」；原子性防同一 ticket 并发换两条连接 |

### 4.2 `realtime_event_repo.go`

| 函数/方法签名 | 职责 | 备注 |
|---------------|------|------|
| `NewRealtimeEventRepository(client) *RealtimeEventRepository` | 构造 | |
| `(r) NextSeq(ctx, userID) (int64, error)` | `INCR realtime:seq:{userID}` | seq 按用户独立计数 |
| `(r) Append(ctx, userID, event, retention int) error` | pipeline：`RPUSH realtime:events:{uid}` + `LTRIM -retention -1` | retention ≤0 兜底 200；一次网络往返 |
| `(r) ListSince(ctx, userID, sinceSeq, topics, retention) (...)` | `LRANGE 0 -1` 后内存过滤 | 过滤 `seq > sinceSeq`；topics 非空时按主题白名单；返回 events 与 latestSeq |

坑点：`ListSince` 的 `retention` 入参当前**未使用**（截断只发生在 Append）；补发是把整段 List 拉回应用层过滤，量大时有优化空间。文件尾 `var _ repository.RealtimeEventRepository = ...` 编译期保证实现接口。

### 4.3 `presence_repo.go`

| 函数/方法签名 | 职责 | 备注 |
|---------------|------|------|
| `NewPresenceRepository(client) *PresenceRepository` | 构造 | |
| `(r) SetOnline(ctx, userID, device) (int, error)` | pipeline：`SET presence:user:{uid} {device} EX 300` + `SADD presence:online uid` + `SCARD` | 返回第 3 条命令结果=在线人数 |
| `(r) SetOffline(ctx, userID) (int, error)` | `DEL` 用户 key + `SREM` + `SCARD` | 同上 |

Key 设计：`presence:online`（Set，全局在线用户）、`presence:user:{uid}`（String，设备名，TTL 5 分钟）。注意：TTL 过期只删设备 key，**不会自动从 Set 摘除**——当前没有扫描兜底，在线集合可能残留（⚠️ 待确认是否有定时清理，grep 未见）。

### 4.4 `conversation_repo.go`（SSE 旁路，同包）

key 前缀 `sse:conv:`，结构 `sse:conv:{uid}:{convID}`，默认 TTL 30 分钟。`Create` 存空 `[]ChatTurn`；`Get` miss 返回 `repository.ErrConversationNotFound`；`SaveTurns` 覆写+续期；`Touch` 仅 `EXPIRE`（key 不存在同样返回 ErrConversationNotFound）。

## 5. Usecase 层（usecase/realtime_*）

### 5.1 `realtime_ticket_usecase.go`

| 函数/方法签名 | 职责 | 备注 |
|---------------|------|------|
| `NewRealtimeTicketUsecase(tickets, cfg) *...` | 构造 | |
| `(u) Issue(ctx, RealtimeTicketInput{UserID,Platform,ConnID}) (*RealtimeTicketOutput, error)` | 校验 UserID 非空；ConnID 空则生成 `conn_{毫秒}`；ticket=`uuid.NewString()`；`cfg.Realtime.TicketTTL()` 存 Redis；返回 ticket/wsUrl/expiresInSeconds/connId | WS URL 由 `cfg.Realtime.WSURL(server.port)` 拼 `ws://{public_ws_host|127.0.0.1}:{port}{ws_path}` |
| `(u) Consume(ctx, ticket) (*repository.WSTicket, error)` | 透传 repo.Consume | WS handler 调用 |

配置默认值（`common/config/config.go` L507-536、L745-755）：`ws_path=/realtime/v1/connect`、`ticket_ttl_seconds=120`、`max_connections_per_user=3`、`event_retention=200`。注意 `MaxConnectionsPerUser` 有配置和默认值，但当前 Hub/Handler **没有执行**该上限（零杜撰：grep 未见使用点）。

### 5.2 `realtime_sync_usecase.go`

| 函数/方法签名 | 职责 | 备注 |
|---------------|------|------|
| `NewRealtimeSyncUsecase(events, cfg)` | 构造 | |
| `(u) Sync(ctx, RealtimeSyncInput{UserID,SinceSeq,Topics}) (*RealtimeSyncOutput, error)` | 调 `events.ListSince(..., cfg.Realtime.EventRetention)`；nil 切片归一化为 `[]` | 返回 events + latestSeq |

### 5.3 `realtime_push_usecase.go`

| 函数/方法签名 | 职责 | 备注 |
|---------------|------|------|
| `RealtimeBroadcaster` 接口 | `BroadcastToUser(userID, topic, envelope) int` | Hub 实现；解耦便于 mock |
| `PushEnqueuer` 接口 | `EnqueueRealtimePush(ctx, RealtimePushInput) (taskID, err)` | RabbitMQ 适配器实现（`repository/queue/push_enqueuer.go`） |
| `OfflinePusher` 接口 | `NotifyUser(ctx, userID, title, body, deeplink, extras)` | 极光；占位边界见 §10 |
| `NewRealtimePushUsecase(events, cfg, hub, queue)` | 构造 | queue 可为 nil（同步模式） |
| `(u) SetOfflinePusher(p)` | 构造后注入极光 | main.go 装配后设置 |
| `(u) PushToUser(ctx, input) (RealtimePushOutput, error)` | 总入口：`cfg.Queue.UseAsyncPush() && queue!=nil` → 入队返回 `{Queued:true, TaskID, Delivered:-1}`；否则直接 `DeliverPush` 同步投递 | HTTP push、community 适配器、定时通知都走这里 |
| `(u) DeliverPush(ctx, input) (envelope, delivered, error)` | 同步投递核心（Worker 也调它） | 顺序：校验 → topic 默认 `sys.notify`、name 默认 `sys.notify.show` → `NextSeq` → 组装 payload（name/notifyId/title/body + Extra 覆盖合并）→ 信封 `evt_{毫秒}` → `Append` 留痕 → `hub.BroadcastToUser` → 若有 offline pusher 则 best-effort 极光（deeplink 从 Extra 提取，错误忽略） |

设计要点：**先 Append 再广播**——离线时 delivered=0，事件仍在 Redis，重连可 /sync 补；seq 已持久化，客户端按序号去重。

### 5.4 `realtime_presence_usecase.go`

| 函数/方法签名 | 职责 | 备注 |
|---------------|------|------|
| `RealtimePresenceBroadcaster` 接口 | `BroadcastToTopicExcept(excludeUserID, topic, envelope) int` | Hub 实现 |
| `NewRealtimePresenceUsecase(presence, broadcaster)` | 构造 | |
| `(u) Report(ctx, RealtimePresenceInput{UserID,Online,Device}) (envelope, delivered, error)` | online=true→`SetOnline` 否则 `SetOffline`；组装 `presence.update` 事件（userId/online/onlineCount，device 可选）；`BroadcastToTopicExcept` 发给**除上报者外**所有订阅者 | **不写 seq、不 Append**：瞬时状态无补发价值；Flutter 端 acceptSeq(null) 直接放行 |

## 6. WS 层（delivery/ws）

### 6.1 `hub.go`：连接管理中心

**通俗理解**：Hub 是俱乐部的签到板，两块板子——「全场所有人」「按用户分组」。推消息时先按名字找到组，再逐个问「你订这个频道了吗？」。

| 函数/方法签名 | 职责 | 备注 |
|---------------|------|------|
| `Hub` struct | `mu sync.RWMutex` + `clients map[*Client]struct{}` + `byUser map[string]map[*Client]struct{}` | 指针做 key：注册/注销 O(1)，`struct{}` 不占内存；读多写少用 RWMutex |
| `NewHub() *Hub` | 初始化两张 map | main.go 全局一个 |
| `(h) Register(client)` | 加入 clients；userID 非空才加入 byUser（懒建用户 set） | auth 成功后才调；Upgrade 后未验票的连接只在 clients |
| `(h) Unregister(client)` | 两张表删除；用户 set 空了删 key；`close(client.send)` | close send → writePump 退出发 Close 帧 → TCP 释放 |
| `(h) BroadcastToUser(userID, topic, envelope) int` | RLock 下**复制**一份 client 切片再解锁，逐个 `IsSubscribed` 后 `Send`，返回送达连接数 | 持锁时间最短，避免 Send 阻塞其他广播 |
| `(h) BroadcastToTopicExcept(excludeUserID, topic, envelope) int` | 复制「userID 非空且不等于 exclude」的全部连接，按订阅过滤发送 | presence 用：不给上报者发自己的状态 |

### 6.2 `client.go`：一条连接

**通俗理解**：一条 WebSocket 连接雇了两个快递员——一个只管收（readPump），一个只管发（writePump），中间用 32 格的传送带（buffered channel）衔接，避免两个 goroutine 抢同一支笔写 socket。

常量：`writeWait=10s`、`pongWait=60s`、`pingPeriod=54s`（pongWait×9/10，提前发）、`maxMsgSize=64KB`。

| 函数/方法签名 | 职责 | 备注 |
|---------------|------|------|
| `Client` struct | hub/conn/log + `send chan RealtimeEnvelope`（cap 32）+ userID/sessionID/connID + `topics map` + `topicsMu` | |
| `NewClient(hub, conn, log)` | Upgrade 后创建，send 缓冲 32 | 此时未 auth，userID 空 |
| `(c) IsSubscribed(topic) bool` | RLock 查 topic set | 广播前过滤 |
| `(c) Subscribe(topics) []string` | 加订阅并返回当前全量 topic | 给 ack 用 |
| `(c) Unsubscribe(topics) []string` | 删订阅，返回全量 | |
| `(c) Send(envelope)` | **非阻塞**入队：`select { case send<-: default: 打 warn 丢弃 }` | 队列满直接丢，绝不拖死 Hub 广播循环 |
| `(c) readPump(handle func(*Client, Envelope))` | defer `hub.Unregister`+`conn.Close`；设 64KB 读限；设读截止 60s；PongHandler 刷新截止；循环 `ReadMessage`→JSON 反序列化→`handle` | 坏 JSON 只 warn+continue，不断连 |
| `(c) writePump()` | 54s ticker；select send 通道（关闭则发 Close 帧退出；marshal 后写 TextMessage）或 ticker 到点写协议层 Ping | Ping/Pong 是 WebSocket 控制帧，防 NAT 断连，与应用层 ping 不是一回事 |
| `(c) CloseWithCode(code, reason)` | 写自定义 Close 控制帧并关连接 | 4001/4003 验票失败用 |

### 6.3 `handler.go`：交通警察

`upgrader`：读写缓冲各 1KB，`CheckOrigin` 恒 true（开发放行任意来源；**生产应限制域名**，文件注释自述）。

| 函数/方法签名 | 职责 | 备注 |
|---------------|------|------|
| `Handler` struct | hub + ticketUC + presenceUC + log | 注意没有 sync/push UC：那些是纯 HTTP |
| `NewHandler(hub, ticketUC, presenceUC, log)` | 构造 | main.go L303 |
| `(h) ServeWS(c *gin.Context)` | `upgrader.Upgrade` → NewClient → go writePump + go readPump(handleEnvelope) | 挂在 GET `/realtime/v1/connect`，路由层**无任何鉴权中间件** |
| `(h) handleEnvelope(client, envelope)` | 按 type switch 分发：`auth/ping/sub/unsub/event`，其他 debug 忽略 | 类似 WS 版路由表 |
| `(h) handleAuth(client, envelope)` | 验票全流程，见 §8.2 | |
| `(h) handleEvent(client, envelope)` | defer 必回 ack；未 auth 只 warn；只处理 `topic=presence.bulk` + `payload.name=presence.report`，其余 event **仅 ack**；取 online（默认 true）/device，调 `presenceUC.Report`，记日志含 delivered/eventId | presence 是当前唯一上行 event |
| `(h) sendAck(client, refID, payload)` | 组 `ack_毫秒` 信封，payload 注入 refId | refID 空则不加 |
| `stringSlice(v any) []string` | `[]any` → `[]string`，过滤非字符串/空串 | JSON 解码后 topics 的类型断言 |

`ping` 分支：Flutter 每 25s 发应用层 ping，服务端必须回**同 id** 的 pong（与协议层 Ping 双通道心跳）。

## 7. HTTP 层

### 7.1 路由表

源码锚点：`router/realtime_routes.go`、`router/router.go` L86-88、L106-111

| 方法 | 路径 | 中间件链 | Handler |
|------|------|----------|---------|
| POST | `/api/v1/realtime/ws-ticket` | SessionAuth（sbAuth）+ accountGate（可选） | `RealtimeController.WSTicket` |
| POST | `/api/v1/realtime/sync` | 同上 | `RealtimeController.Sync` |
| POST | `/api/v1/realtime/push` | 同上 | `RealtimeController.Push` |
| GET | `/realtime/v1/connect`（默认，走根 router 不在 /api/v1 组） | **无**（WS 首帧 ticket 自证） | `ws.Handler.ServeWS` |

注册条件：`businessAuth && RealtimeController != nil && sessionAuth != nil`。链由 `sessionAuthChain(sbAuth, accountGate)` 组装（`session_auth_chain.go`，accountGate 为 nil 时只有 auth）。

### 7.2 DTO

| DTO | 字段（json） |
|-----|--------------|
| `request.RealtimeTicketRequest` | `platform`、`connId` |
| `request.RealtimeSyncRequest` | `sinceSeq int64`、`topics []string` |
| `request.RealtimePushRequest` | `userId`、`topic`、`title`、`body`、`name`、`extra map` |
| `response.RealtimeTicketData` | `ticket`、`wsUrl`、`expiresInSeconds`、`connId`（对齐 Flutter WsTicketResult） |
| `response.RealtimeSyncData` | `events []Envelope`、`latestSeq` |
| `response.RealtimePushData` | `envelope,omitempty`、`delivered`、`queued,omitempty`、`taskId,omitempty` |

### 7.3 `realtime_controller.go` 函数表

| 方法签名 | 职责 | 备注 |
|----------|------|------|
| `RealtimeController` struct / `NewRealtimeController(ticketUC, syncUC, pushUC)` | 持有三个 UC | |
| `(ctrl) WSTicket(c)` | `supabaseAuthContext(c)` 取会话用户（失败 401）；Bind body（忽略错误，platform/connId 可空）；调 `ticketUC.Issue`；500/200 包 `RealtimeTicketData` | |
| `(ctrl) Sync(c)` | 取用户；**严格** Bind（失败 400）；调 `syncUC.Sync(user.ID, sinceSeq, topics)`；返回 events/latestSeq | |
| `(ctrl) Push(c)` | 取用户；严格 Bind；`req.UserID` 空则**给自己推**（调试接口允许指定他人，⚠️ 无服务端授权校验目标 userId）；调 `pushUC.PushToUser`；返回 envelope/delivered/queued/taskId | 定位为开发环境推送测试 |

## 8. 全链路时序

### 8.1 连接建立与 ticket 校验

```text
Flutter                    Go API                              Redis
   │  POST /realtime/ws-ticket (Bearer+session)                 │
   │ ───────────────────────────► Controller.WSTicket           │
   │                              ticketUC.Issue                 │
   │                              Save(ws:ticket:{uuid}, {uid,connId}, EX 120)
   │                              ──────────────────────────────►│
   │  ◄──────── {ticket, wsUrl, connId, expiresInSeconds}        │
   │                                                             │
   │  GET ws://host:8080/realtime/v1/connect  (Upgrade，无鉴权头) │
   │ ───────────────────────────► ServeWS：101 Switching          │
   │                              NewClient + 两个 pump goroutine │
   │  首帧 {type:"auth", payload:{ticket, connId?}}               │
   │ ───────────────────────────► handleAuth                     │
   │                              ticketUC.Consume = GETDEL       │
   │                              ──────────────────────────────►│ 命中即删
   │  ◄──────── {type:"auth_ok", payload:{userId,sessionId,serverTime}}
   │                              hub.Register(client)（进 byUser）│
```

验票失败分支：ticket 空 → 回 error(4001) + `CloseWithCode(4001)`；Consume 出错 → error(4003) + `CloseWithCode(4003)`。成功后：client.userID/connID 落位（ticket 无 connId 时取首帧 payload.connId），sessionID 服务端新生成 UUID，然后才 Register。

### 8.2 订阅与心跳

```text
{sub, payload:{topics:[...]}} → Subscribe → ack(payload.topics=当前全量)
{unsub,...}                   → Unsubscribe → ack
{ping, id}                    → {pong, id 相同}   （应用层，25s）
协议层 Ping/Pong（54s/60s 超时）由两个 pump 自动完成
```

### 8.3 消息广播：sys.notify（同步与异步两条路）

同步（`queue.enabled=false`）：

```text
HTTP /realtime/push → PushToUser → DeliverPush
  ① NextSeq（INCR）  ② Append（RPUSH+LTRIM 留痕）
  ③ hub.BroadcastToUser：复制该用户连接 → IsSubscribed("sys.notify") → Send 入队
  ④ writePump 出队写 TextMessage；可选 OfflinePusher 极光 best-effort
```

异步（`queue.enabled=true`，main.go L305-326）：

```text
API 进程：PushToUser → PushEnqueuer.EnqueueRealtimePush → RabbitMQ（立刻返回 queued/taskId, delivered=-1）
Worker 进程：Consumer → worker.Handler.handlePushNotify
  → NewDeliveryPushUsecase 构造的 pushUC（broadcaster 是 FanoutPublisher，不是本地 Hub）
  → DeliverPush（同样先 NextSeq+Append）
  → FanoutPublisher.BroadcastToUser → PUBLISH Redis 频道（返回 0，不算本地送达）
每个 API 实例：FanoutSubscriber.Run 订阅频道 → handleMessage
  → 本地 hub.BroadcastToUser(userID, topic, envelope) → 对应连接收信
```

**通俗理解**：API 实例可能部署多个，WebSocket 连接张三落在 A 机器、推送请求打到 B 机器，B 不认识张三——于是 Worker 往 Redis 大喇叭（Pub/Sub）喊一嗓子，所有机器听到后各自查本地 Hub。事件先写 Redis List 保证任一台机器都能给离线者补发。

### 8.4 presence 全链路

```text
Flutter A 订阅 presence.bulk 后发：
{type:"event", topic:"presence.bulk", payload:{name:"presence.report", online:true, device:"iphone"}}
  → handleEvent：defer ack（accepted:true）
  → presenceUC.Report
      → Redis pipeline：SET presence:user:{A} EX 300 / SADD presence:online A / SCARD
      → 组 presence.update 信封（无 seq）
      → hub.BroadcastToTopicExcept(A, "presence.bulk", envelope)
  → 同实例上除 A 外所有订阅了 presence.bulk 的连接收到 {userId, online, onlineCount, device?}
```

注意边界：presence 广播**只在单个进程内**扇出，没有走 Redis Pub/Sub（与 push 不同）；多实例部署时跨实例 presence 互不可见（⚠️ 当前实现边界）。

### 8.5 断线补发 sync

```text
重连后 POST /realtime/sync {sinceSeq:N, topics:["sys.notify"]}
  → syncUC.Sync → LRANGE realtime:events:{uid} → 过滤 seq>N + topic 白名单
  → {events:[...], latestSeq:M}
```

## 9. 装配（main.go L295-347 摘要）

```text
businessEnabled 时：
  hub := NewHub()
  ticketRepo / eventRepo / presenceRepo := redis.New*(redisClient)
  ticketUC / syncUC / presenceUC（hub 作为 PresenceBroadcaster）
  wsGateway = ws.NewHandler(hub, ticketUC, presenceUC, log)
  if cfg.Queue.Enabled:
      queueClient = queue.NewClient
      pushEnqueuer = repoqueue.NewPushEnqueuer(queueClient)
      fanoutSub = queue.NewFanoutSubscriber(redisClient, channel, hub.BroadcastToUser, log)
      go fanoutSub.Run(ctx)
  pushUC = NewRealtimePushUsecase(eventRepo, cfg, hub, pushEnqueuer)
  realtimeController = NewRealtimeController(ticketUC, syncUC, pushUC)
  // community 域通过 communityPushAdapter{uc: pushUC} 反向调用推送
```

`router.Options.WSHandler = wsGateway`，router.go 里 `r.GET(wsPath, opts.WSHandler.ServeWS)`。

## 10. 假实现 / 占位 / 边界清单

- `WSCloseKicked=4002`「被踢下线」：常量预留，无任何代码发送。
- `RealtimeConfig.MaxConnectionsPerUser=3`：配置存在，未在 Hub/Handler 强制执行。
- `RealtimePushUsecase.offline`（极光）：有 `SetOfflinePusher` 接口位；是否实际注入取决于 main 装配（JPush 整体属占位性质，见推送专题文档）；失败一律忽略。
- presence 的 Redis Set 无 TTL 自动摘除、无跨实例 Pub/Sub。
- HTTP `/realtime/push` 是调试口：可指定任意 `userId` 为他人推送，无目标授权校验。
- WS `CheckOrigin` 恒 true。
- 上行 event 仅 `presence.report` 一种；其他 event 只回 ack。
- `conversation_repo.go` 虽在 redis 包里且 key 前缀是 `sse:conv:`，但服务于 SSE AI 会话，不属于 WS。

## 11. 学习要点回顾

1. **WS 鉴权两阶段**：HTTP 签发短时一次性 ticket（GETDEL 原子消费），握手不验权、首帧验票——绕开浏览器 WS 不能加头的限制，同时防重放。
2. **Hub 双索引 + 复制后解锁**：广播热点路径避免长持锁；`map[*Client]struct{}` 做 O(1) 集合。
3. **单连接读写分泵 + channel 串行写**：gorilla/websocket 官方模式；Send 非阻塞、缓冲满即丢，保护广播方。
4. **双层心跳**：协议层 Ping/Pong（54/60s）保 TCP，应用层 ping/pong（25s，同 id）业务可观测。
5. **先持久化后广播 + 按用户 seq**：离线不丢、重连可按 seq 去重补发；presence 反例（瞬态不补发、不带 seq）。
6. **同步/异步同一投递函数**：`DeliverPush` 被 HTTP 与 Worker 共用；多实例用 Redis Pub/Sub 把 Worker 的广播带回各实例本地 Hub。
7. **接口倒置**：usecase 定义 `RealtimeBroadcaster/PushEnqueuer/OfflinePusher`，Hub 与队列适配器反向实现，核心逻辑可单测。
8. Redis 各 key 职责分明：`ws:ticket:` 票据、`realtime:seq:`+`realtime:events:` 事件流、`presence:online`+`presence:user:` 在线态。

## 12. 可动手验证

```bash
# 1) 一键全链路脚本（登录→换票→push→sync→go test TestWSAuthFlow）
make test-realtime
# 等价：bash scripts/test_realtime_ws.sh

# 2) 手工看 Redis key（脚本跑完后）
redis-cli KEYS 'ws:ticket:*'
redis-cli LRANGE realtime:events:<你的userId> 0 -1
redis-cli SMEMBERS presence:online

# 3) 读代码任务
#  - 找到 Send 缓冲满时的丢弃日志：delivery/ws/client.go Send
#  - 验证 GETDEL 原子性：repository/redis/ws_ticket_repo.go Consume
#  - 对比两条广播路径：hub.BroadcastToUser vs BroadcastToTopicExcept
#  - 跑单测：go -C app test ./internal/delivery/ws/ -v
```
