# SSE 流式输出（Go BFF）

> **权威设计文档**（跨仓库，含 Flutter UI / 分期 / 验收）：  
> [`my_ai_project/docs/sse-streaming-design.md`](../../my_ai_project/docs/sse-streaming-design.md)  
> **产品 Spec**：`my_ai_project/.scratch/ai-little-stone/SPEC.md` · **ADR 0005**（SSE + 停留会话，非 Realtime/IM）

请求作用域的生成流用 **`POST /api/v1/sse/completions`** + `text/event-stream`；长连接推送仍用 [Realtime WebSocket](./realtime-websocket.md)。

## 端点

| 方法 | 路径 | 鉴权 |
|------|------|------|
| POST | `/api/v1/sse/completions` | Session Auth（与 realtime HTTP / transactions 相同） |

- 成功：SSE 帧 `meta` → `delta`* → `done`
- 流前失败：JSON `{"error":"..."}`（非 ResultModel）
- 流中失败：`event: error`
- 响应头含 `Cache-Control: no-cache`、`X-Accel-Buffering: no`，每帧 `Flush`
- 可选注释心跳：`: keepalive`（默认 15s）

## 停留会话

- 首句省略 `conversationId` → 服务端 Redis 创建会话，`meta` 下发 id
- 后续请求带 id → 使用最近 **10 轮**上下文（可配）
- 滑动 TTL 默认 **1800s**；取消生成会截断本轮助手内容写入上下文
- 离开助手页由客户端丢弃 id（不跨次续聊）

## 配置（`configs/config.yaml` → `sse.*`）

| 键 | 默认 | 说明 |
|----|------|------|
| `sse.enabled` | `true` | 关闭后不注册路由 |
| `sse.provider` | `mock` | `mock` \| `openai_compatible` |
| `sse.max_prompt_bytes` | `8192` | prompt 上限 |
| `sse.max_tokens` | `2048` | 生成上限（再 clamp） |
| `sse.keepalive_seconds` | `15` | 注释心跳间隔 |
| `sse.request_timeout_seconds` | `120` | 单次生成超时 |
| `sse.rate_limit_per_user_per_minute` | `20` | 按 user 限流 |
| `sse.conversation_ttl_seconds` | `1800` | 停留会话滑动 TTL |
| `sse.max_turns` | `10` | 上下文轮数 |
| `sse.openai.base_url` | OpenAI | 兼容 API 根路径 |
| `sse.openai.model` | `gpt-4o-mini` | 上游模型 |
| `SSE_OPENAI_API_KEY` | — | **仅环境变量**；无 key 时 `openai_compatible` 回退 mock |

切换真实上游示例：

```yaml
sse:
  provider: openai_compatible
  openai:
    base_url: "https://api.openai.com/v1"
    model: "gpt-4o-mini"
```

```bash
export SSE_OPENAI_API_KEY=sk-...
```

## 落点文件

```text
internal/delivery/http/router/sse_routes.go
internal/delivery/http/controller/sse_controller.go
internal/delivery/http/dto/request/sse_completion.go
internal/usecase/completion_usecase.go
internal/domain/entity/sse_event.go
internal/domain/provider/stream_provider.go
internal/domain/repository/conversation_repository.go
internal/repository/redis/conversation_repo.go
internal/repository/llm/mock_stream_provider.go
internal/repository/llm/openai_compatible_provider.go
pkg/config/config.go          # SSEConfig
configs/config.yaml           # sse: 段
cmd/api/main.go               # DI
```

## 测试

```bash
go test ./internal/usecase/ -run Completion -count=1
go test ./internal/repository/llm/ -count=1
```

## 联调

```bash
TOKEN=...   # 登录后 access_token
SESSION=... # X-Session-ID
DEVICE=...  # X-Device-ID

curl -N -X POST http://127.0.0.1:8080/api/v1/sse/completions \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Session-ID: $SESSION" \
  -H "X-Device-ID: $DEVICE" \
  -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -d '{"prompt":"你好，做个自我介绍","clientRequestId":"curl_1"}'
```
