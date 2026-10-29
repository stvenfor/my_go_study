# 12-SSE 与客服 Agent：流式生成、Provider 切换与 Eino Tool

> 一句话定位：本篇讲两条 HTTP 流式接口——AI 小石头的 `POST /api/v1/sse/completions`（多轮停留会话 + 可切换 LLM Provider）与客服 Agent 的 `POST /api/v1/cs-agent/chat`（关键词触发 Eino 交易搜索工具）。所属模块：legacy 分层的 `delivery/http/{controller,dto,router}` + `usecase` + `repository/{llm,csagent}`，端口在 `model/provider`、帧契约在 `model/entity`。阅读前置：懂 SessionAuth 中间件链、Redis 仓储装配，建议先读设计文档 [../../sse-streaming.md](../../sse-streaming.md) 与 [../../cs-agent-eino.md](../../cs-agent-eino.md)。

## 1. 通俗理解

- **SSE（Server-Sent Events）** 可以理解成「服务器给客户端打的逐字电报」：一个普通 HTTP POST，响应头声明 `text/event-stream` 后连接不关，服务器每生成一点内容就按固定格式（`event:` + `data:`）发一帧并立刻 Flush；客户端逐帧渲染出打字机效果。它和 WebSocket 的区别是**单向的**：客户端说完一整句（请求体），之后只剩服务器在说。
- **小石头 completions** 像一个带「短期记忆」的向导：第一次对话不带 `conversationId`，服务器在 Redis 建一个停留会话（默认 30 分钟 TTL），后续带着 id 来就能取最近 10 轮上下文；回答由一个可插拔的 `StreamProvider` 产生（默认是关键词假回复 mock，可切 OpenAI 兼容上游）。
- **客服 Agent** 像一个「先查账本再念稿」的客服：它本身没有接大模型，而是按关键词（「交易/收支/查…」）决定要不要调用一个 Eino Tool 去 ES 查交易，把工具返回的 JSON 拼进固定话术，然后同样逐字吐出。它演示的是 **Eino Tool 调用链**，不是完整的 ReAct 智能体。

## 2. 两个路由的差异总览

| 维度 | `/api/v1/sse/completions`（小石头） | `/api/v1/cs-agent/chat`（客服 Agent） |
|---|---|---|
| 方法/鉴权 | POST，SessionAuth + 可选账号门 | POST，同左 |
| 请求 DTO | `CompletionRequest`（prompt/conversationId/clientRequestId/options） | `CsAgentChatRequest`（仅 prompt/clientRequestId） |
| Controller | `SseController.Completions` | `CsAgentController.Chat` |
| Usecase | `CompletionUsecase.Stream` | `CsAgentUsecase.Stream` |
| 多轮上下文 | 有（Redis 停留会话，默认 10 轮/30 分钟） | **无**（每次独立，不存历史） |
| 限流 | 有（进程内滑动窗口，默认 20 次/用户/分钟） | 无 |
| 生成来源 | `model/provider.StreamProvider`：mock 或 OpenAI 兼容 | **不调 LLM**；固定话术 + Eino Tool 结果，按 rune 假流式 |
| 工具调用 | 无 | `transaction_search`（Eino InvokableTool，关键词门控） |
| 首帧 seq | meta 的 seq=**0** | meta 的 seq=**1** |
| meta 特征 | 带 `conversationId`、`model` | 无 conversationId，`text:"cs-agent"` |
| 事件通道缓冲 | 32 | 16 |
| 超时/心跳 | 请求超时默认 120s；`: keepalive` 默认 15s | 同左（各自独立配置） |
| 配置段 | `sse.*`（env 前缀 `SSE_`） | `cs_agent.*`（env 前缀 `CS_AGENT_`） |

帧契约两者完全相同（复用 `writeSSEFrame` 与 `entity.SSEFrame`）：成功 `meta → delta* → done`，流中失败 `meta → ... → error`。

## 3. 路由表（HTTP 适配层）

注册处 `app/internal/delivery/http/router/router.go:98-104`，都有 `businessAuth && controller != nil && sessionAuth != nil && cfg.<段>.Enabled` 守卫——配置关闭时路由根本不注册。

| 方法 | 路径 | 中间件链 | Handler | 路由文件 |
|---|---|---|---|---|
| POST | `/api/v1/sse/completions` | `sessionAuthChain(sessionAuth, accountGate)` = SessionAuth（Local 或 Supabase）+ 可选 `RequireActiveAccount` | `SseController.Completions` | `router/sse_routes.go` |
| POST | `/api/v1/cs-agent/chat` | 同上 | `CsAgentController.Chat` | `router/cs_agent_routes.go` |

两个路由文件结构一致：`v1.Group("/sse"|"/cs-agent").Use(chain...)` 后挂一个 POST。链构造函数 `sessionAuthChain` 在 `router/session_auth_chain.go`：accountGate 为 nil 时只有 SessionAuth 一环节。

## 4. SSE 帧格式（契约层 `model/entity/sse_event.go`）

事件名与帧类型常量一一对应：`meta` / `delta` / `done` / `error`；结束原因 `stop` / `length`；稳定错误码 `upstream_timeout`、`upstream_4xx`、`provider_unavailable`、`canceled`、`internal`。

`SSEEventData` 是每帧 `data:` 行的 JSON（字段全部驼峰）：

| 字段 | 出现时机 |
|---|---|
| `requestId` / `seq` / `type` | 每帧都有（小石头 requestId 形如 `sse_<uuid>`） |
| `text` | delta 帧的增量文本；客服 meta 帧固定 `cs-agent` |
| `conversationId` / `model` / `clientRequestId` | meta 为主 |
| `finishReason` | done 帧（指针，omitempty） |
| `error: {code, message}` | error 帧 |

线上的字节格式由 controller 里同一个函数写出（`sse_controller.go:111`）：

```go
func writeSSEFrame(c *gin.Context, frame entity.SSEFrame) error {
	raw, _ := json.Marshal(frame.Data)
	_, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", frame.Event, raw)
	return err
}
```

即标准 SSE：`event: <事件名>` 换行 + `data: <单行 JSON>` 换行 + 空行收尾。另有两类写出：

- **心跳注释帧**：`: keepalive\n\n`（SSE 注释，客户端 EventSource 自动忽略），按 keepalive 间隔（默认 15s）在 select 定时器分支写出。
- **响应头**（两个 controller 完全相同）：`Content-Type: text/event-stream; charset=utf-8`、`Cache-Control: no-cache, no-transform`、`Connection: keep-alive`、`X-Accel-Buffering: no`（禁 Nginx 缓冲），然后 `WriteHeader(200)`，之后每帧 `flusher.Flush()`。

> 关键时序：所有**流建立前**的错误（未登录、参数错、禁用、限流）走普通 JSON 错误响应（`response.BackendError`），不会写 SSE 头；一旦写了 200 与 SSE 头，后续错误只能以 `event: error` 帧表达。

## 5. 小石头链路：`/sse/completions`

请求链路：router → `SseController.Completions` → `CompletionUsecase.Stream`（校验/会话/限流）→ goroutine `runStream` → `provider.StreamProvider.Stream` → 通道逐帧回 controller 写出；历史读写经 `model/repository.ConversationRepository`（生产实现是 Redis，key `sse:conv:<userID>:<conversationID>`）。

### 5.1 请求 DTO（`dto/request/sse_completion.go`）

| 类型/方法 | 职责 | 备注 |
|---|---|---|
| `type CompletionRequest struct { Prompt string; ConversationID *string; ClientRequestID string; Options *CompletionOptions }` | HTTP 请求体，JSON `prompt`/`conversationId`/`clientRequestId`/`options` | conversationId 用指针区分「没传（首句）」与「传了空串」 |
| `type CompletionOptions struct { Model string; Temperature *float64; MaxTokens *int }` | 可选生成参数 | 数值用指针，零值与未传可区分 |
| `func (r CompletionRequest) ConversationIDValue() string` | nil 安全取值，空表示首句 | Controller 调它 |

### 5.2 `controller/sse_controller.go`

| 函数/方法签名 | 职责 | 备注（坑） |
|---|---|---|
| `type SseController struct { uc *CompletionUsecase; cfg SSEConfig }` | 控制器 | |
| `func NewSseController(uc, cfg) *SseController` | 构造 | |
| `func (ctrl *SseController) Completions(c *gin.Context)` | 主 handler：鉴权取 `user.ID` → bind DTO → 映射成 `usecase.CompletionRequest`（options 指针解引用）→ 包请求超时 context → `uc.Stream` | 预流错误经 `mapCompletionPreStreamError` 映射 JSON |
| 同上（流循环部分） | 写 SSE 响应头 → `select` 三路：ctx.Done 退出、ticker 发 keepalive、读 ch 写帧并 Flush；收到 done/error 帧后退出 | 若 `http.Flusher` 断言失败直接静默 return |
| `func writeSSEFrame(c, frame) error` | 帧序列化写出 | 客服 controller **直接复用**（同包） |
| `func mapCompletionPreStreamError(err) (int, string)` | 哨兵错误→HTTP 状态：禁用 503、prompt 空 400、超长 400、限流 429、会话失效 400、provider 不可用 503、其他 500 | 用 `errors.Is` 匹配 |

### 5.3 `usecase/completion_usecase.go`（函数级清单）

预流哨兵错误：`ErrSSEDisabled`、`ErrPromptEmpty`、`ErrPromptTooLarge`、`ErrRateLimited`、`ErrConversationInvalid`、`ErrProviderUnavailable`。

| 函数/类型签名 | 职责 | 备注 |
|---|---|---|
| `type CompletionRequest struct { Prompt; ConversationID; ClientRequestID; Model string; Temperature float64; MaxTokens int }` | usecase 入参，与 HTTP DTO 解耦 | |
| `type RateLimiter interface { Allow(userID string) bool }` | 按用户限流端口 | |
| `func NewCompletionUsecase(p StreamProvider, convs ConversationRepository, limiter RateLimiter, cfg) *CompletionUsecase` | 构造；limiter 传 nil 时自动建内存限流器 | |
| `func (u *CompletionUsecase) Stream(ctx, userID, req) (<-chan SSEFrame, error)` | 同步校验 + 建会话/取历史，然后**开 goroutine** 返回只读帧通道（缓冲 32） | 校验顺序：enabled→userId→trim 后 prompt 非空→字节数 ≤ MaxPromptBytes（默认 8192）→限流→会话 |
| 同上会话分支 | 无 conversationId：`convs.Create(ttl)`，history 为 nil；有：`convs.Get`，`ErrConversationNotFound` 转 `ErrConversationInvalid`，并 `Touch` 滑动续期 | maxTokens 未传/非正用配置默认（2048），超过也 clamp 到默认值；model 空则取 `provider.Name()`；requestID=`sse_<uuid>` |
| `func (u *CompletionUsecase) runStream(ctx, a runStreamArgs)`（私有） | 实际流式编排，`defer close(out)` | 见 5.4 时序 |
| `func (u *CompletionUsecase) persistTurn(ctx, a, assistantText)`（私有） | 把本轮 user + 非空 assistant 追加到历史副本，`trimToMaxTurns` 后 `SaveTurns` | 流中失败/取消也调，避免下一轮丢上下文 |
| `func trimToMaxTurns(turns, maxTurns)`（包内私有） | 超过 `maxTurns*2` 条时只保留尾部 | 一轮 = user+assistant 两条 |
| `func mapProviderError(err) (code, message)`（私有） | provider 错误→稳定错误码：实现 `CodeValue()` 用其码；`DeadlineExceeded` → upstream_timeout；否则 internal | 对应 llm 包的 `ProviderError` |
| `type MemoryRateLimiter` + `NewMemoryRateLimiter(limit)` + `Allow` | 进程内每用户 1 分钟滑动窗口；limit≤0 全放行 | 多实例部署不共享（单机限流） |
| `type MemoryConversationRepository` 及其 `Create/Get/SaveTurns/Touch/ExpireNow` | **测试用**内存会话仓储（同文件），编译期断言实现端口 | 生产用 Redis 实现，不在本篇文件清单内 |

### 5.4 `runStream` 的帧时序与取消语义

1. 闭包 `emit`：每帧写入 `requestId` 与当前 `seq`（**meta 拿到 seq=0，之后自增**），channel 满或 ctx 取消时阻塞/放弃。
2. 先发 `meta`（type=meta、conversationId、model、clientRequestId）。
3. 开一个 goroutine 调 `provider.Stream(ctx, CompletionInput{Prompt, Messages:历史, Model, MaxTokens, Temperature}, deltaCh)`，结束时关闭 deltaCh 并带回 `finishReason, err`。
4. 主循环逐 chunk 累积到 `strings.Builder` 同时发 `delta`（type=delta、text=增量）。
5. `wg.Wait()` 后三分支：
	- **取消**（ctx 取消或上游返回 context.Canceled）：用 `context.WithoutCancel(ctx)` 把已生成片段截断入库，**不发 done**，直接返回。
	- **上游出错**：发 `error` 帧（码经 `mapProviderError`），随后仍入库（有片段存片段，没有只存用户句）。
	- **成功**：finishReason 空则补 `stop`，发 `done`，再入库。

### 5.5 Provider 端口（`model/provider/stream_provider.go`）

| 类型/方法 | 职责 |
|---|---|
| `type CompletionInput struct { Prompt string; Messages []entity.ChatTurn; Model string; MaxTokens int; Temperature float64 }` | Provider 入参；注释明确 Messages 不含本轮，本轮在 Prompt |
| `type StreamProvider interface { Name() string; Stream(ctx, in CompletionInput, out chan<- string) (finishReason string, err error) }` | 流式端口：往 out 发**纯文本增量**；返回 stop/length；err 非空时 usecase 发 error 帧 |

### 5.6 Mock Provider（`app/internal/repository/llm/mock_stream_provider.go`）—— 默认假实现

| 函数/方法签名 | 职责 | 备注 |
|---|---|---|
| `const mockModelName = "mock-v1"` | `Name()` 返回值 | meta.model 默认就是它 |
| `func NewMockStreamProvider() *MockStreamProvider` | 默认每段间隔 40ms | |
| `func (p *MockStreamProvider) Name() string` | 端口实现 | |
| `func (p *MockStreamProvider) Stream(ctx, in, out) (string, error)` | `mockAnswer` 选话术；MaxTokens 按 **rune 数截断**并以 `length` 结束，否则 `stop` | 假实现：不发起任何网络请求 |
| `func (p *MockStreamProvider) emit(...)`（私有） | 逐 rune 写入 out，每字间隔 ChunkDelay，处处检查 ctx 取消 | |
| `func mockAnswer(in) string`（私有） | 关键词剧本：二手车/登录/数据分析/短视频/会员支付/自我介绍各有固定回复；历史非空且含「然后/还有/呢/then」给追问续答；兜底回复回显截断 40 rune 的 prompt | 多轮能力是写死的 |
| `containsAny` / `truncate`（私有） | 关键词包含与 rune 截断工具 | |

文件尾 `var _ provider.StreamProvider = (*MockStreamProvider)(nil)` 编译期保证实现端口。

### 5.7 OpenAI 兼容 Provider（`repository/llm/openai_compatible_provider.go`）

| 函数/类型签名 | 职责 | 备注（坑） |
|---|---|---|
| `type OpenAICompatibleConfig struct { BaseURL, APIKey, Model string; Timeout time.Duration }` | 上游配置 | |
| `func NewOpenAICompatibleProvider(cfg) *OpenAICompatibleProvider` | 归一化：timeout 默认 120s、baseURL 默认 `https://api.openai.com/v1`、model 默认 `gpt-4o-mini`；建带超时的 http.Client | 允许无 key 构造（延迟到 Stream 才失败） |
| `func (p *OpenAICompatibleProvider) Name() string` | 返回配置模型名 | |
| `func (p *OpenAICompatibleProvider) Stream(ctx, in, out) (string, error)` | 核心上游调用，见下 | |
| `type ProviderError struct { Code, Message string }` + `Error()` + `CodeValue()` | 带稳定错误码的错误；`CodeValue` 空码兜底 internal | 被 usecase 的 `codedProviderError` 接口识别（避免跨层 import） |
| `type openAIStreamChunk`（私有） | 只解析上游 chunk 的 `choices[].delta.content` 与 `finish_reason` | 其余字段忽略 |

`Stream` 流程与错误映射：

1. APIKey 空 → 直接返回 `ProviderError{provider_unavailable, ...提示 SSE_OPENAI_API_KEY}`。
2. 拼 messages：固定 system 提示（「你是 AI小石头…不要输出 Markdown」）+ 历史中合法 user/assistant 轮次 + 本轮 user；body 固定 `stream:true`，按入参附带 `max_tokens`/`temperature`（temperature 仅 >0 时带）。
3. POST `<BaseURL>/chat/completions`，Header 带 `Authorization: Bearer <key>`、`Accept: text/event-stream`。
4. 网络错误 → `upstream_timeout`；HTTP ≥400：4xx → `upstream_4xx`，5xx → `provider_unavailable`，响应体截断 2048 字节带进 message。
5. 用 `bufio.Scanner`（buffer 上调到 1MB）逐行解析上游 SSE：忽略空行/注释行/非 `data:` 行；`data: [DONE]` 结束；JSON 坏块跳过；delta.content 非空才转发进 out；途中持续检查 ctx。
6. scanner 错误 → `upstream_timeout`；正常返回最后见到的 `finish_reason`（默认 stop）。

### 5.8 Provider 切换配置与装配

- 配置段 `sse`（`common/config/config.go:545`，默认值在 `configs/config.yaml:87-100`）：

| 键 / 环境变量 | 默认 | 作用 |
|---|---|---|
| `sse.enabled` / `SSE_ENABLED` | true | 关闭则不装配、不注册路由 |
| `sse.provider` / `SSE_PROVIDER` | `mock` | 仅 `openai_compatible` 精确命中，其他一律归一成 mock（`ProviderName()`） |
| `sse.openai.base_url` / `SSE_OPENAI_BASE_URL` | `https://api.openai.com/v1` | 任何 OpenAI 兼容根路径 |
| `sse.openai.model` / `SSE_OPENAI_MODEL` | `gpt-4o-mini` | 上游模型 |
| `sse.openai.api_key` / `SSE_OPENAI_API_KEY` | 空 | **只走环境变量**；Load 末尾（config.go:837）从 env 覆盖，yaml 中不放密钥 |
| prompt/token/会话等 | 8192 / 2048 / 30min / 10 轮 / 20 次每分钟 / keepalive 15s / timeout 120s | 见第 2 节与配置表 |

- 装配函数 `buildSSEProvider`（`app/cmd/api/main.go:600`）：provider=openai_compatible 时构造 OpenAI Provider，**但 key 为空会打 Warn 并回退 mock**；API main 在 `cfg.SSE.Enabled` 块内建 Redis 会话仓储、内存限流器、usecase、controller（main.go:349-360）。

切换真实上游的标准操作：`SSE_PROVIDER=openai_compatible` + `export SSE_OPENAI_API_KEY=sk-...`（base_url 可指到任意兼容网关）。

## 6. 客服 Agent 链路：`/cs-agent/chat`

请求链路：router → `CsAgentController.Chat` → `CsAgentUsecase.Stream`（仅校验）→ goroutine `run` → （关键词命中时）`runSearchTool` → Eino `txSearchTool.InvokableRun` → `transaction.Service.Search`（ES）→ `buildCsAnswer` 拼稿 → 逐 rune 发 delta。

### 6.1 DTO 与 Controller

`dto/request/cs_agent_chat.go` 只有一个类型：`CsAgentChatRequest{ Prompt string; ClientRequestID string }`（JSON `prompt`/`clientRequestId`），无会话、无 options。

`controller/cs_agent_controller.go`：

| 函数/方法签名 | 职责 | 备注 |
|---|---|---|
| `type CsAgentController struct { uc *CsAgentUsecase; cfg CsAgentConfig }` / `NewCsAgentController` | 控制器与构造 | |
| `func (ctrl *CsAgentController) Chat(c *gin.Context)` | 鉴权→bind→包 120s 超时 ctx→`uc.Stream`→写 SSE 头→与小石头**同构**的 select 三路循环（keepalive/读帧/done-error 退出） | 帧写出复用 `writeSSEFrame` |
| `func mapCsAgentPreStreamError(err) (int,string)` | 禁用→503「cs agent disabled」、prompt 空→400、其他→**400** + err.Error() | 用的是 `err ==` 直接比较而非 `errors.Is`（包装错误匹配不到） |

### 6.2 `usecase/cs_agent_usecase.go`

| 函数/类型签名 | 职责 | 备注（坑/边界） |
|---|---|---|
| `var ErrCsAgentDisabled / ErrCsAgentPrompt` | 仅两个预流哨兵 | |
| `type CsAgentRequest struct { Prompt; ClientRequestID string }` | 入参 | |
| `func NewCsAgentUsecase(searcher csagent.TransactionSearcher, cfg) *CsAgentUsecase` | searcher 为 optional 端口 | API main 传的是 `transactionSvc` |
| `func (u *CsAgentUsecase) Stream(ctx, userID, req) (<-chan SSEFrame, error)` | 校验 enabled/trim prompt/userID，建**缓冲 16** 的通道并 goroutine `run` | requestID 是裸 uuid（无 `sse_` 前缀）；model 取 `cfg.ModelName()` |
| `func (u *CsAgentUsecase) run(...)`（私有） | 帧编排，见下 | seq 从 **1** 开始（先自增再赋值），且每帧强制 `data.Type = event` |
| `func (u *CsAgentUsecase) runSearchTool(ctx, userID, prompt) (string, error)`（私有） | 关键词门控→构造 Eino 工具→固定参数 `{"query":prompt,"limit":5}` 调 `InvokableRun` | 不命中返回 `"", nil` |
| `func shouldSearchTransactions(prompt) bool`（私有） | 小写化后命中任一关键词即 true：`交易/收支/账单/消费/收入/支出/search/查/找` | 规则触发，不是模型决策 |
| `func buildCsAnswer(prompt, toolJSON, toolErr) string`（私有） | 三种固定话术：工具不可用→致歉引导；有结果→「已根据您的问题检索交易：」+ 原始工具 JSON + 补充提示；无搜索→引导例句 + 回显问题 | 开头固定「【客服助手】您好，我是业务客服 Agent（Eino Tool）。」 |

`run` 的帧序：先发 meta（model、`text:"cs-agent"`）→ 执行搜索工具：

- 工具返回非 `shared.ErrSearchUnavailable` 的错误 → 发 `error{internal}` 并结束（这是会中断对话的真错误）；
- `ErrSearchUnavailable`（ES 没配）被当作**可降级**：不发 error，话术里提示「交易搜索当前不可用」；
- 随后 `buildCsAnswer` 生成全文，按 rune 逐字发 delta（每字间隔 `cfg.ChunkDelay()`，默认 20ms），期间 ctx 取消则发 `error{canceled}` 退出；最后发 `done(stop)`。
- 另有 `_ = utf8.RuneCountInString(built.String())`：统计结果被丢弃，属无实际作用的残留代码。

### 6.3 Eino Tool：`repository/csagent/tx_search_tool.go`

依赖 `github.com/cloudwego/eino/components/tool` 与 `eino/schema`。

| 函数/类型签名 | 职责 | 备注 |
|---|---|---|
| `type TransactionSearcher interface { Search(ctx, userID, q string, limit int) ([]entity.Transaction, error) }` | 搜索端口 | 由 `transaction.Service` 实现；userID 靠会话注入，防越权 |
| `type searchHit`（私有） | 出参条目：`id/type/category/amount/date/note(omitempty)` | 对实体的瘦身投影 |
| `func NewTransactionSearchTool(searcher, userID) (tool.InvokableTool, error)` | 构造工具；当前实现永不返回错误（签名预留） | userID 闭包绑定当前会话用户 |
| `func (t *txSearchTool) Info(ctx) (*schema.ToolInfo, error)` | Eino 元信息：名 `transaction_search`、描述「搜索当前用户的收支交易记录（全文：备注/分类/类型）」；参数 `query`（string，必填）、`limit`（integer） | 标准 Eino Tool 描述协议，供未来 ChatModel 决策使用 |
| `func (t *txSearchTool) InvokableRun(ctx, argumentsInJSON, ...tool.Option) (string, error)` | 工具执行体，见下 | 返回字符串（JSON） |

`InvokableRun` 逻辑：解析参数 JSON → query 空返回 `{"items":[],"message":"empty query"}` → limit≤0 取 5 → **searcher 为 nil 返回 `{"items":[],"message":"search unavailable"}` 且无 error** → 调 `Search`，错误用 `%w` 包装（因此 usecase 能用 `errors.Is(_, shared.ErrSearchUnavailable)` 识别 ES 未配置）→ 实体投影为 searchHit（Note 是指针，非 nil 才带）→ 返回 `{"items":[...],"count":n}`。

下游事实：`transaction.Service.Search`（`app/internal/transaction/service.go:86`）在 ES store 为 nil 时返回 `shared.ErrSearchUnavailable`（域内别名指向 shared 哨兵），否则走 ES 全文检索。所以「客服 Agent 能查到什么」取决于 `ELASTICSEARCH_URL` 是否配置、交易索引是否就绪。

### 6.4 Eino 调用链全景（当前形态与边界）

```text
用户 prompt
  └─ shouldSearchTransactions（关键词规则，非 LLM）
       └─ NewTransactionSearchTool(transactionSvc, userID)
            └─ tool.InvokableRun(`{"query":...,"limit":5}`)
                 └─ TransactionSearcher.Search → ES
                      └─ []entity.Transaction → searchHit 投影 → JSON 字符串
       └─ buildCsAnswer：固定话术 + 工具 JSON（没有 LLM ReAct 推理）
            └─ 逐 rune delta 帧（20ms 假流式）
```

明确边界（与 `docs/cs-agent-eino.md` 的 Deferred 一致）：当前只用到了 Eino 的 **Tool 抽象**（Info + InvokableRun），没有 ChatModel、没有 ReAct 循环、没有真实大模型；`cs_agent.provider/model` 配置只影响 meta 里的模型名（`ModelName()`：显式 model 优先，否则 provider=openai_compatible 给 `cs-openai`，再否则 `cs-mock`），**不驱动任何生成调用**。`ChunkDelay()` 仅在显式配 0/负值时为 0；provider 为空且未配延迟时默认 20ms。

## 7. 两条链路相同的工程细节

- **鉴权取人**：都用 `supabaseAuthContext(c)` 拿登录用户，未认证 401；与 realtime/transactions 同一套 SessionAuth。
- **超时与心跳**：各自从 cfg 派生 `context.WithTimeout`（默认 120s）与 keepalive ticker（默认 15s）；客户端断连→请求 ctx 取消→后台 goroutine 经 `ctx.Done()` 收敛。
- **通道 + goroutine 模式**：usecase 同步返回「预流错误或只读 channel」，controller 不阻塞在生成逻辑上；channel 有缓冲（32/16）。
- **流后持久化**：只有小石头在流末写 Redis 会话（且用 `context.WithoutCancel` 防止客户端断连导致写不进去）；客服无持久化。
- **SSE 头与 Flush 时机**：两个 controller 逐字不差，维护时改一个别忘了另一个（`writeSSEFrame` 已复用，但头/循环是两份）。

## 8. 占位与假实现清单

1. `MockStreamProvider`：关键词写死回复 + 40ms 逐字，是默认 Provider，不产生真实 AI。
2. 客服 Agent：无 LLM 推理，工具靠关键词触发，回复是固定模板拼接；完整 Eino ChatModelAgent/ReAct 在设计文档里标注为 Deferred，Flutter 客服页同样 Deferred。
3. `NewOpenAICompatibleProvider` 允许空 key 构造；装配层对空 key 的 openai_compatible 静默回退 mock（只打 Warn）。
4. `MemoryRateLimiter` 是单机限流，多实例不共享计数。
5. `MemoryConversationRepository` 仅测试用；生产是 Redis 实现。
6. 客服 `searcher == nil` 与「ES 未配置」是两条不同的降级路径（一个返回 JSON 无错误，一个返回包装哨兵错误），话术也不同。

## 9. 学习要点回顾

1. SSE 本质是「一个不能关的 HTTP 响应 + 固定文本帧协议 + 每帧 Flush」；`X-Accel-Buffering: no` 这类头是为了穿透代理缓冲，逐字交付才有意义。
2. 流前错误与流中错误的通道不同：前者 JSON + 状态码，后者只能 `event:error`——所以校验必须在写 200 头之前全部做完。
3. usecase 用「返回 channel + 后台 goroutine」把生成与传输解耦；ctx 取消要同时通知上游 Provider 与帧循环，并在取消后仍持久化片段（`WithoutCancel`）。
4. 依赖倒置让 Provider 可插拔：`model/provider.StreamProvider` 是端口，mock 与 OpenAI 兼容实现平级；配置 + 装配函数完成切换与降级，业务代码零改动。
5. 调外部 SSE 上游时，自己也是一个 SSE 客户端：Scanner 解析 `data:`/`[DONE]`、透传增量、映射 4xx/5xx/超时为稳定错误码。
6. Eino Tool 的最小形态就是 `Info()`（声明式描述）+ `InvokableRun()`（执行）；工具结果是字符串，能不能被「智能」使用取决于上层有没有 Agent 循环——本项目客服只到工具直调这一步。
7. 越权防护在端口层：交易搜索的 userID 不由工具参数决定，而由构造工具时的闭包绑定会话用户。
8. 两个相似接口要对照着学：帧契约复用、controller 循环复制、但 seq 起点、限流、会话、错误比较方式（errors.Is vs ==）都有细微差别，读代码时这些差异就是考点。

## 10. 可动手验证

读代码任务：

- 在 `completion_usecase.go` 里画出 meta/delta/done/error 四种路径下 `persistTurn` 分别被调用几次、存了什么。
- 追踪 `SSE_OPENAI_API_KEY` 从环境变量到 HTTP Header 的完整路径（config.go Load 末尾 → buildSSEProvider → Provider.Stream）。
- 说出 prompt 命中「查一下餐饮支出」但 `ELASTICSEARCH_URL` 未配置时，客服链路每一帧的 `event` 与最终话术；再推演 searcher 为 nil 的情形。

curl 联调（需先登录拿到 TOKEN/SESSION/DEVICE，参考 [../../sse-streaming.md](../../sse-streaming.md)）：

```bash
# 小石头首句（观察 meta 里的 conversationId，下一句带上它验证多轮）
curl -N -X POST http://127.0.0.1:8080/api/v1/sse/completions \
  -H "Authorization: Bearer $TOKEN" -H "X-Session-ID: $SESSION" \
  -H "X-Device-ID: $DEVICE" -H "Accept: text/event-stream" \
  -H "Content-Type: application/json" \
  -d '{"prompt":"二手车入口在哪","clientRequestId":"curl_1"}'

# 客服 Agent（命中关键词触发 transaction_search）
curl -N -X POST http://127.0.0.1:8080/api/v1/cs-agent/chat \
  -H "Authorization: Bearer $TOKEN" -H "X-Session-ID: $SESSION" \
  -H "X-Device-ID: $DEVICE" -H "Content-Type: application/json" \
  -d '{"prompt":"查一下餐饮支出"}'
```

切换真实上游：

```bash
SSE_PROVIDER=openai_compatible SSE_OPENAI_API_KEY=sk-xxx \
SSE_OPENAI_BASE_URL=https://你的兼容网关/v1 make run
```

测试命令（仓库自带）：

```bash
go -C app test ./internal/usecase/ -run Completion -count=1
go -C app test ./internal/repository/llm/ -count=1
go -C app test ./internal/repository/csagent/ -count=1
```
