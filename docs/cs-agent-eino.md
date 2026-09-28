# 客服 Agent（Eino + SSE）

独立于「AI 小石头」`/api/v1/sse/completions`。

| 项 | 值 |
|----|-----|
| 路由 | `POST /api/v1/cs-agent/chat` |
| 鉴权 | SessionAuth（与业务 API 相同） |
| 帧契约 | 同小石头：`meta` → `delta*` → `done` / `error` |
| 编排 | CloudWeGo Eino `InferTool`：`transaction_search` |
| 默认 | `cs_agent.provider=mock`（无真实 LLM ReAct；工具结果拼进回复） |

## 请求

```json
{ "prompt": "查一下餐饮交易", "clientRequestId": "optional" }
```

含「交易/收支/查/找…」等关键词时调用 ES 交易搜索（需 C3 `ELASTICSEARCH_URL`）；否则返回引导话术。

## 配置

```yaml
cs_agent:
  enabled: true
  provider: mock
  chunk_delay_ms: 20
```

环境变量：`CS_AGENT_ENABLED`、`CS_AGENT_PROVIDER`、`CS_AGENT_CHUNK_DELAY_MS`。

## Deferred

- 完整 Eino ChatModelAgent / ReAct + 真实 OpenAI 循环
- Flutter 客服页
