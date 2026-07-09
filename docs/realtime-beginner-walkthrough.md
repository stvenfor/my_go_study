# Realtime WebSocket 初学者逐行导读

> 面向零基础读者：解释**每一层在做什么、为什么这样写**。
> 协议细节见 [realtime-websocket.md](./realtime-websocket.md)。

---

## 0. 先建立整体印象（3 分钟）

```
Flutter App
   │  ① HTTP 换票（带登录 token）
   ▼
Go BFF  ──► Redis（存 ticket、事件、在线人数）
   │  ② WebSocket 长连接（auth / sub / ping / event）
   ▼
Flutter App（收 event、弹通知、显示 presence）
```

**为什么不直接在 WebSocket 里传 JWT？**

- WebSocket 升级请求不好统一走 HTTP 中间件
- 用**短期一次性 ticket**更安全：即使 ticket 泄露，120 秒内用完即失效

**为什么分 HTTP 和 WebSocket 两种通道？**

| 通道 | 适合做什么 | 原因 |
|------|-----------|------|
| HTTP | 换票、sync 补历史、push 调试 | 请求-响应模型简单可靠 |
| WebSocket | 实时推送、心跳、客户端上报 | 长连接，服务端可主动推 |

---

## 1. 推荐阅读顺序（按文件）

### Go 后端（my_go_study）

| 顺序 | 文件 | 你将学到 |
|------|------|----------|
| 1 | `internal/domain/entity/realtime_envelope.go` | 消息 JSON 长什么样 |
| 2 | `internal/usecase/realtime_ticket_usecase.go` | 换票业务逻辑 |
| 3 | `internal/repository/redis/ws_ticket_repo.go` | ticket 如何存 Redis |
| 4 | `internal/delivery/ws/hub.go` | 连接如何管理、如何广播 |
| 5 | `internal/delivery/ws/client.go` | 读写分离、心跳 |
| 6 | `internal/delivery/ws/handler.go` | auth/sub/ping/event 分发 |
| 7 | `internal/usecase/realtime_push_usecase.go` | Go → Flutter 推通知 |
| 8 | `internal/usecase/realtime_presence_usecase.go` | Flutter → Go → 其他 Flutter |
| 9 | `internal/delivery/http/controller/realtime_controller.go` | HTTP 入口 |

### Flutter 客户端（my_ai_project）

| 顺序 | 文件 | 你将学到 |
|------|------|----------|
| 1 | `packages/infrastructure/realtime/lib/config/realtime_config.dart` | 常量配置 |
| 2 | `lib/api/ws_ticket_api.dart` | 如何换票 |
| 3 | `lib/connection/heartbeat_scheduler.dart` | 心跳为什么 25s |
| 4 | `lib/client/app_realtime_client_impl.dart` | 连接全流程 |
| 5 | `lib/debug/realtime_debug_page.dart` | 如何手动测试 |

---

## 2. 消息包络 RealtimeEnvelope（Go + Flutter 共用）

```go
type RealtimeEnvelope struct {
    ID      string         `json:"id,omitempty"`      // 消息 ID，ping/pong 配对、ack 引用
    Type    string         `json:"type"`              // auth / ping / event / ...
    Topic   string         `json:"topic,omitempty"`   // pub/sub 主题，如 sys.notify
    TS      int64          `json:"ts,omitempty"`      // 毫秒时间戳
    Seq     int64          `json:"seq,omitempty"`     // 用户级递增序号，用于 sync 去重
    Payload map[string]any `json:"payload,omitempty"` // 各类型自定义字段
}
```

**为什么用 map 做 Payload？**

- 不同 `type` 字段不同（通知有 title/body，presence 有 online）
- Go 与 Flutter 共用 JSON，扩展新事件不用改结构体字段

**为什么需要 Seq？**

- 断线重连后会 HTTP sync 补拉
- 客户端用 `seq <= lastSeq` 跳过重复，避免通知弹两次

---

## 3. 连接流程逐行解读（Flutter connect）

```dart
// 1. 必须已登录 —— ticket 接口需要 Bearer token
final token = _resolveAccessToken();

// 2. HTTP 换票 —— 不在 WS 里直接传 JWT 的原因见上文
final ticket = await _ticketApi.fetchTicket(accessToken: token);

// 3. 模拟器要把 127.0.0.1 映射成 10.0.2.2
final uri = Uri.parse(BackendWsConfig.resolveWsUrl(ticket.wsUrl));

// 4. 建立 WebSocket TCP 连接
await _transport.connect(uri);

// 5. 监听服务端下发的所有 JSON
_listenInbound();

// 6. 首条业务消息必须是 auth，提交一次性 ticket
await _transport.send(RealtimeEnvelope(
  type: 'auth',
  payload: {'ticket': ticket.ticket, ...},
));
// 7. 等待 auth_ok → 启动心跳 → HTTP sync → 重新 sub
```

---

## 4. Go Handler：收到消息后怎么办

```go
func (h *Handler) handleEnvelope(client *Client, envelope entity.RealtimeEnvelope) {
    switch envelope.Type {
    case "auth":   // 首帧鉴权，消费 Redis ticket
    case "ping":   // 应用层心跳，原样 id 回 pong
    case "sub":    // 记录 client 订阅了哪些 topic
    case "unsub":  // 取消订阅
    case "event":  // 客户端上报（如 presence.report）
    }
}
```

**为什么用 switch 而不是 if-else 链？**

- 消息类型固定且互斥，switch 清晰、易扩展

**为什么 sub 要存到 Client 内存而不是 Redis？**

- 订阅是「这条连接的兴趣列表」，断开即失效
- 放内存查询最快；Redis 适合跨进程持久化（ticket/事件）

---

## 5. Hub：广播的两种模式

### 5.1 推给某个用户的所有设备

```go
hub.BroadcastToUser(userID, "sys.notify", envelope)
```

用于：系统通知只给**目标用户**（push 接口）。

### 5.2 推给除发送者外的所有人

```go
hub.BroadcastToTopicExcept(senderUserID, "presence.bulk", envelope)
```

用于：A 上报在线，B/C/D 应收到，A 自己不需要。

**为什么 Broadcast 前要检查 IsSubscribed？**

- 没订阅 `sys.notify` 的连接不应收到通知（节省流量、避免泄露）
- `delivered: 0` 常见原因：客户端忘了 `sub`

---

## 6. 双向发消息速查

### Go → Flutter

```bash
curl -X POST http://127.0.0.1:8080/api/v1/realtime/push \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title":"Hi","body":"来自 Go"}'
```

或代码：`pushUC.PushToUser(...)`

### Flutter → Go → 其他 Flutter

```dart
await client.sendEvent(
  topic: RealtimeTopics.presenceBulk,
  eventName: 'presence.report',
  payload: {'online': true, 'device': 'ios'},
);
```

Go `handleEvent` → `presenceUC.Report` → `BroadcastToTopicExcept`

### Flutter 接收

```dart
client.watchEvents(eventName: 'presence.update').listen((e) {
  print(e.payload);
});
```

---

## 7. 心跳：两层为什么都需要

| 层 | 谁发 | 间隔 | 作用 |
|----|------|------|------|
| 应用层 ping/pong | Flutter 发，Go 回 | 25s | 业务层检测死连接，Flutter 主动重连 |
| 协议层 Ping 帧 | Go 发 | 54s | 防止中间 NAT/防火墙静默断开 TCP |

**为什么 Flutter 连续 2 次 ping 超时才重连？**

- 一次可能是网络抖动；两次减少误断

---

## 8. 本地动手实验

```bash
# 终端 1
cd my_go_study && make run

# 终端 2
make test-realtime

# Flutter：登录 → 设置 → Realtime 调试 → 连接 → 上报 presence
```

---

## 9. 源码中的注释说明

Realtime 核心 `.go` / `.dart` 文件已补充**逐段中文注释**（说明「做什么 + 为什么」）。
打开上述「推荐阅读顺序」中的文件即可对照阅读。

若需给**全仓库每一行**加注释，建议按模块分批（auth → transactions → realtime），避免注释淹没代码逻辑。

**模块导读文档：**

| 模块 | 文档 |
|------|------|
| Realtime WebSocket | [realtime-beginner-walkthrough.md](./realtime-beginner-walkthrough.md) |
| 认证 login/register | [auth-beginner-walkthrough.md](./auth-beginner-walkthrough.md) |
| Transactions 二手车 | [transactions-beginner-walkthrough.md](./transactions-beginner-walkthrough.md) |

---

## 10. 相关文档

- [Realtime WebSocket 协议与联调指南](./realtime-websocket.md)
- Flutter [BACKEND_INTEGRATION.md](../../../my_ai_project/docs/BACKEND_INTEGRATION.md) §6
