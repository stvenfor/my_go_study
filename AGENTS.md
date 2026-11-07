# my_go_study Agent 开发指南

Go BFF 后端 Agent 指南，并包含与 Flutter **`my_ai_project`** 协作的工作区总览。

### Ponytail

写代码默认走 **Ponytail**（最短可用解）：规则见 [`.cursor/rules/ponytail.mdc`](./.cursor/rules/ponytail.mdc)（`alwaysApply`）。与 Cursor hooks 二选一；本仓库用 rule，不开项目级 hooks（要 `/ponytail lite|full|ultra|off` 再装 hooks 并删掉该 rule）。

### Agent 编排（`.harness`）

多角色 / 多会话任务默认走编排层，**不要**跳过 Brief 与验证直接自称完成：

| 入口 | 路径 |
|------|------|
| 编排总览 | [`.harness/README.md`](./.harness/README.md) |
| 角色合同 | [`.harness/agents/`](./.harness/agents/)（conductor → planner → executor → reviewer） |
| 当前指针 | [`.harness/changes/current.md`](./.harness/changes/current.md) |
| 硬规则 | [`.harness/rules/`](./.harness/rules/) |
| **大型任务三槽流程** | [`.harness/rules/three-slot-orchestration.md`](./.harness/rules/three-slot-orchestration.md)（≤3 agent / 2 实现 + 1 bug / 逐片验收 / 不换 agent / 不绕依赖） |
| 文档索引 | [`.harness/wiki/README.md`](./.harness/wiki/README.md) |
| **双仓编码 Playbook** | [`docs/coding-playbook-dual-repo.md`](./docs/coding-playbook-dual-repo.md)（8 步 / 提示词 / 坑） |

新任务未指定角色时，先读 **conductor**；人批 Brief 与 commit/push。

#### 开新模块一键提示词

规划（先不写码）：

```text
结合双仓（本仓 + ../../my_ai_project）设计 <模块>：
1) CONTEXT 术语草案（含 Avoid） 2) 表/API/权限口径 3) Flutter 对标页与模块边界
4) Program/Epic/Slice backlog；标假实现边界。先方案与 Brief 草稿，等我批再落地。
流程见 docs/coding-playbook-dual-repo.md。
```

执行（Brief 已批）：

```text
执行 plans/slices/<id>.md；只改白名单；Go：定向 go test + go -C app build ./cmd/api（或 go build all）；
Flutter：dart analyze 触及包 + 对标 Brief。写 acceptance-records（Partial 可）。
commit/push 等我指令。
```

---

## 一、工作区总览（Flutter + Go）

### 仓库入口

| 项目 | 本地路径（常见） | Git 远程 | Agent 指南 |
|------|------------------|----------|------------|
| **Go BFF（本仓库）** | `my_go_study/` | `stvenfor/my_go_study` | 本文档 |
| **Flutter 客户端** | `../../my_ai_project/` | `stvenfor/my_ai_project` | [AGENTS.md](../../my_ai_project/AGENTS.md) |

**规则**：改 Go 在本仓库；改 Flutter 进 `my_ai_project` 并读其 AGENTS。

### 系统架构

```text
my_ai_project (Flutter)
    │  HTTP JSON :8080（登录 / 业务）
    │  gRPC :9090（数据分析试验）
    ▼
my_go_study (同进程双端口)    ← 你在这里
    │  Auth / Postgres / Redis / RabbitMQ / ES / PostgREST
    ▼
Supabase Cloud 或 local Auth + 本地 Postgres
```

| 能力 | Flutter | Go（本仓库） | 外部 |
|------|---------|--------------|------|
| 登录注册 | `BackendAuthService` | `UserHandler` → Auth usecase + `DeviceSessionUsecase` | Auth + Redis session |
| 收支/二手车 | `TransactionApi` | `TransactionController` | PostgREST/本地 PG + RLS；全文见 ES |
| 交易全文搜索 | — | `GET /api/v1/transactions/search` | Elasticsearch |
| 实时消息 | `AppRealtimeClient` | `RealtimeController` + WS Hub | Redis |
| 异步 Push | — | `cmd/worker` 消费 | RabbitMQ + Redis Pub/Sub |
| 用户资料 | — | `ProfileController` | PostgREST |
| 门店统计 | Mine 统计栏 | `Profile` + `wys_user_store_stats`（数字列 + role 0/1/2） | 本地 Postgres |
| 数据分析 | `AnalyticsGrpcApi`（gRPC） | `AnalyticsService` → `AnalyticsUsecase` | 本地表 `analytics_records` |
| AI 小石头 | SSE 客户端 | `POST /api/v1/sse/completions` | mock / OpenAI 兼容 |
| 客服 Agent | —（页可后置） | `POST /api/v1/cs-agent/chat`（Eino Tool） | 可选 ES 交易搜索 |
| 可观测 | — | `GET /metrics` | Prometheus / Grafana / Alertmanager（`lan-up`） |

**表命名**：新建业务表一律 `wys_` 前缀（规范见 [docs/local-auth-postgres.md](./docs/local-auth-postgres.md) §3）。

### 本地联调

> **工作目录**：以下命令均在 **`my_go_study/`** 目录执行（`cd my_go_study`），不要在工作区根目录 `my_code_study/` 运行 `make`。

```bash
# 方案 A（推荐联调）：Docker 统一管理 HTTP:8080 + gRPC:9090
# + PG + Redis + RabbitMQ + ES + Worker + Prometheus/Grafana/Alertmanager
# cp .env.lan.example .env.lan 后：
# 不用手改 IP：lan-up 会先按 scripts/lan-ip.clients 的注释开关同步（行首 # = 关闭）
make lan-up                 # docs/lan-backend-host.md；改 Go 后需再执行（会 --build）
# 只想刷 IP 不启动: make sync-lan-ip
# 临时加客户端（不改清单文件）: make lan-up LAN_CLIENTS=my_kmp_project,kmp_kuikly
#                              # LAN_CLIENTS=all 写清单内全部未注释行
make lan-down
# 勿与本机 make run 同时开，会抢 :8080

# 方案 B：本机进程
make deps-up                # 首次：PostgreSQL + Redis（RabbitMQ/ES 建议用 compose）
make run                    # HTTP :8080 + gRPC :9090（同进程）
make run-worker             # queue.enabled=true 时另开终端

# Flutter（路径相对本仓库：../../my_ai_project）
cd ../../my_ai_project
flutter run --dart-define-from-file=.env.lan   # 真机：BACKEND_HOST=局域网IP；gRPC 默认 9090

# 验证
curl http://127.0.0.1:8080/health
curl -s http://127.0.0.1:8080/metrics | head
make test-realtime          # 需 API 已启动
make test-queue-push        # 需 API + Worker + RabbitMQ
make trigger-hourly-notify
make test-scheduled-notify
make test-single-device-login
make test-phone-otp-login
```

| 检查项 | 说明 |
|--------|------|
| 健康 | `GET /health` |
| 指标 | `GET /metrics`（Prometheus；无 SessionAuth） |
| 登录 | `POST /api/v1/user/login` |
| 刷新 Token | `POST /api/v1/user/refresh` |
| 退出登录 | `POST /api/v1/user/logout` |
| 测试手机号 OTP | `POST /api/v1/user/phone/otp/send` · `verify`（dev：`13400000000` + `123456`） |
| 微信登录 | `POST /api/v1/user/wechat/login`（需 `WECHAT_APP_ID` + `WECHAT_APP_SECRET`；仅 local Auth） |
| 华为一键登录 | `POST /api/v1/user/huawei/login`（需 `HUAWEI_CLIENT_ID` + `HUAWEI_CLIENT_SECRET`；仅 local Auth；AGC 须申请 `quickLoginMobilePhone`） |
| 预支付 | `POST /api/v1/payments/prepay`（需登录；微信商户 / 支付宝私钥见 `.env.example`） |
| 交易全文 | `GET /api/v1/transactions/search?q=`（需登录 + ES；见 [elasticsearch-transactions.md](./docs/elasticsearch-transactions.md)） |
| 客服 Agent | `POST /api/v1/cs-agent/chat`（SSE；见 [cs-agent-eino.md](./docs/cs-agent-eino.md)） |
| Realtime | Flutter 设置 → Realtime 调试 |
| 异步 Push | `make test-queue-push`（需 Worker + RabbitMQ） |
| 定时通知 | `make trigger-hourly-notify` + `make test-scheduled-notify` |
| 局域网 / Docker | `make lan-up`：映射 **8080 + 9090**；观测本机 `9091`/`9093`/`3000`；Flutter 只需 `BACKEND_HOST`（gRPC 默认 9090） |
| 可观测 | Prometheus `127.0.0.1:9091` · Grafana `3000` · Alertmanager `9093`；飞书 [observability-lan.md](./docs/observability-lan.md) |
| 数据分析 gRPC | Flutter 首页「数据分析」；冒烟见 [analytics-flutter-trial.md](./docs/grpc/analytics-flutter-trial.md) |
| 密钥 | `make check-secrets` |

### 文档地图

**Go（`docs/`）**

| 文档 | 用途 |
|------|------|
| [README.md](./docs/README.md) | **文档总览**：架构摘要 + 完整使用手册（入口） |
| [startup-guide.md](./docs/startup-guide.md) | 环境、Makefile、Docker |
| [architecture-learning-guide.md](./docs/architecture-learning-guide.md) | 分层、优缺点、认证/交易/Realtime 时序 |
| [supabase-integration.md](./docs/supabase-integration.md) | Supabase 架构 |
| [realtime-websocket.md](./docs/realtime-websocket.md) | WS 协议 |
| [auth-beginner-walkthrough.md](./docs/auth-beginner-walkthrough.md) | 认证导读 |
| [transactions-beginner-walkthrough.md](./docs/transactions-beginner-walkthrough.md) | 收支 API 导读 |
| [realtime-beginner-walkthrough.md](./docs/realtime-beginner-walkthrough.md) | Realtime 导读 |
| [message-queue.md](./docs/message-queue.md) | RabbitMQ + Redis Pub/Sub 异步队列 |
| [observability-lan.md](./docs/observability-lan.md) | Prometheus / Grafana / Alertmanager / 飞书 webhook（`lan-up`） |
| [elasticsearch-transactions.md](./docs/elasticsearch-transactions.md) | 交易全文索引与搜索 API |
| [cs-agent-eino.md](./docs/cs-agent-eino.md) | 客服 Agent（Eino Tool + SSE） |
| [sse-streaming.md](./docs/sse-streaming.md) | AI 小石头 SSE（与客服路由分离） |
| [lan-backend-host.md](./docs/lan-backend-host.md) | 本机 Mac 当局域网 BFF（现为本地 Auth） |
| [dual-end-lan-startup.md](./docs/dual-end-lan-startup.md) | **Go + Flutter 两端启动配置手册** |
| [ios-lan-device-debug-2026-09-16.md](./docs/ios-lan-device-debug-2026-09-16.md) | iOS 真机「无法连接服务端 / 127.0.0.1」调试记录 |
| [local-auth-postgres.md](./docs/local-auth-postgres.md) | `auth.provider=local` / 可切回 supabase |
| [grpc/gin-http-vs-grpc.md](./docs/grpc/gin-http-vs-grpc.md) | Gin HTTP vs gRPC 概念对比 |
| [grpc/analytics-flutter-trial.md](./docs/grpc/analytics-flutter-trial.md) | **数据分析 gRPC**：proto、`:9090`、Flutter 列表/详情联调 |
| [docker/lan-ops-abc-beginner.md](./docs/docker/lan-ops-abc-beginner.md) | **Docker 联调运维**：Flutter 对齐 / 改码生效 / 排障（飞书 tool） |

**Flutter（`my_ai_project/docs/`）**

| 文档 | 用途 |
|------|------|
| [BACKEND_INTEGRATION.md](../../my_ai_project/docs/BACKEND_INTEGRATION.md) | HTTP、认证、Realtime |
| [USAGE_GUIDE.md](../../my_ai_project/docs/USAGE_GUIDE.md) | 运行与环境 |

### 全局约束

1. **所有 `go` / `make` 在本仓库根目录执行**，不要在父目录 `my_code_study` 执行
2. Flutter 业务**不直连** Supabase SDK，经 Go BFF
3. `SUPABASE_SERVICE_ROLE_KEY` 仅 `.env.local`；推送前 `make check-secrets`
4. **两套 Token 勿混用**：Flutter 用业务 access token + session；遗留 `/api/v1/user/list` 用 Go 自建 JWT
5. 两仓库**独立 git**，分别 push
6. **HTTP `:8080` 与 gRPC `:9090` 勿混端口**；gRPC 鉴权走 metadata（`authorization` / `x-session-id` / `x-device-id`），对齐 HTTP SessionAuth
7. `make lan-up` 后后端在 Docker 内运行（镜像构建时打入代码），**勿再本机 `make run`**；改 Go 需重新 `make lan-up`
8. **多模块**：workspace 根无 `go.mod`，跨模块用 `go -C <模块>` 或 `go build all`；`go build ./...` 已失效
9. **✱ 勿盲跑逐模块 `go mod tidy`**：某模块「缺依赖」时它会去拉**最新版**（实测把 viper 1.19→1.21、zap 1.27→1.28、gorm 1.25.12→1.31.2）。
   要加依赖就在该模块 `go.mod` 里显式写 pinned 版本再 tidy；升级依赖必须单独一次改动 + 全量 `make test`
10. **模块边界不可反向**：`model` / `common` 不得 import `app` 或 `core`。原 `pkg/queue` 曾反向依赖 `internal/usecase`，
    已拆为 `core/queue`（纯基建）+ `app/internal/delivery/worker`（业务分发器）+ `app/internal/repository/queue`（适配器）

---

## 二、Go 后端（本仓库）

### 项目定位

- **框架**：Gin + gRPC + GORM + PostgreSQL + Redis + RabbitMQ + Elasticsearch + Clean Architecture
- **模块结构**：**go.work 多模块工作区** —— `model` / `common` / `core` / `app`，外加本地 `thunder` 三方库
- **默认端口**：HTTP `8080`；gRPC `9090`（`GRPC_ENABLED` / `GRPC_PORT`）；观测本机 `9091`/`9093`/`3000`（见 observability-lan）
- **配对客户端**：Flutter `my_ai_project`

### 分层结构（go.work 多模块）

**依赖方向单向**：`app → {common, core, model}`、`core → {common, model}`；`common` 与 `model` 是叶子，互不依赖。
**workspace 根不再是 module**，所以 `go build ./...` 失效 —— 用 `go build all`（全工作区）或 `go -C app <cmd>`（指定模块）。

```text
go.work                         # use ./app ./common ./core ./model + replace thunder => ./thunder

app/                            # module app —— 唯一的可执行模块
  cmd/api/main.go               # 依赖注入、启动 HTTP+WS+gRPC
  cmd/worker/main.go            # RabbitMQ Worker（异步 Push / SMS / JPush 占位）
  cmd/{import-supabase,seed-analytics,seed-home-todo,scheduler-trigger}/
  api/
    proto/analytics/v1/         # Protobuf 契约（单一真相源）
    gen/go/analytics/v1/        # make proto 生成的 Go stubs
  internal/
    delivery/http/
      handler/                  # 自建用户 HTTP（UserHandler）
      controller/               # 业务 HTTP（Profile / Transaction / Realtime / SSE / CsAgent）
      middleware/               # JWT / SessionAuth / CORS / Logger / PrometheusHTTP
      router/                   # 按模块拆分路由
      dto/request|response/     # 请求/响应 DTO
    delivery/grpc/              # gRPC Server + AnalyticsService + 鉴权 interceptor
    delivery/ws/                # WebSocket Hub / Handler / Client
    delivery/worker/            # RabbitMQ 任务分发器（由 pkg/queue/handler.go 迁入）
    usecase/                    # 业务用例（含 Analytics / Completion / CsAgent）
    repository/
      postgres/                 # 本地 PostgreSQL（含 analytics_records）
      redis/                    # Realtime ticket / 事件 / presence
      es/                       # 交易全文 Elasticsearch
      csagent/                  # 客服 Eino Tool 适配
      supabase/                 # Supabase PostgREST
      llm/                      # SSE StreamProvider（mock / openai_compatible）
      queue/                    # 队列端口适配器（usecase 入参 → 队列载荷）

common/                         # module common —— 通用底座（零项目内依赖）
  config/                       # Viper 配置（含 grpc.* / queue.rabbitmq / elasticsearch / cs_agent）
  logger/                       # zap 初始化
  jwt/                          # 自建 JWT（遗留路由）
  auth/                         # Token 校验（/auth/v1/user）
  metrics/                      # Prometheus Registry + /metrics Handler

core/                           # module core —— 外部系统驱动（→ common, model）
  database/                     # Postgres / Redis 连接
  queue/                        # RabbitMQ 客户端/消费者、Redis Pub/Sub 广播
  supabase/                     # Supabase 客户端封装
  jpush/ rongcloud/             # 三方推送 / IM 客户端

model/                          # module model —— 领域层（零外部依赖）
  entity/                       # 领域实体
  repository/                   # 仓储接口
  provider/                     # StreamProvider 等端口

thunder/                        # 本地三方库（mashenGOAI 的 Gin 框架 SDK），由 replace 引入、不进 workspace
```

### Supabase 集成要点

完整文档：[docs/supabase-integration.md](./docs/supabase-integration.md)

| 能力 | 入口 | Supabase 交互方式 |
|------|------|-------------------|
| 邮箱注册/登录 | `UserHandler` → `SupabaseAuthUsecase` | gotrue-go |
| Token 校验 | `middleware.SupabaseSessionAuth` | `GET /auth/v1/user` + Redis session |
| Profile CRUD | `ProfileController` | PostgREST + `WithUserToken` |
| Transactions CRUD | `TransactionController` | PostgREST + `user_id` + RLS |
| Transactions 全文 | `GET /transactions/search` | Elasticsearch（会话 `user_id`）；CUD best-effort 同步 |
| Realtime WS | `RealtimeController` + `ws.Handler` | Redis ticket → Hub 广播 |
| 异步任务 | API Publish → Worker | RabbitMQ；扇出仍 Redis Pub/Sub |

**启用条件**：`SUPABASE_URL` + `SUPABASE_ANON_KEY` 非空。Realtime 随 Supabase 启用一并注册。

**单设备登录**：登录时 Redis `auth:session:{user_id}` 存 `{session_id, device_id}`；业务 API 需 `X-Session-ID` + `X-Device-ID`；新 mobile 登录覆盖旧 session。

**Session 永不过期**：`auth.session_ttl_hours: 0`（或 `AUTH_SESSION_TTL_HOURS=0`）时 Redis session 无 TTL，仅主动 logout 或其它设备登录时失效。

**Token 续期**：登录/注册返回 `token` + `refresh_token`；access token 过期前客户端调 `POST /api/v1/user/refresh` 静默续期。

**主动退出**：`POST /api/v1/user/logout`（SupabaseSessionAuth）删除 Redis session 并撤销 Supabase refresh token。

**账号白名单豁免**：`auth.session_whitelist_user_ids` / `session_whitelist_emails`（或环境变量 `AUTH_SESSION_WHITELIST_*`）中的账号跳过 Redis session 校验，可多设备同时在线；普通用户不受影响。

### 两套认证（勿混淆）

| 中间件 / 拦截器 | 入口示例 | Token 类型 | 用户 ID |
|----------------|----------|------------|---------|
| `SupabaseSessionAuth` / SessionAuth | `/api/v1/transactions*`、`/api/v1/realtime/*` | access token + session | UUID |
| gRPC unary interceptor | `AnalyticsService/*` | 同上（metadata） | UUID |
| `Auth`（JWT） | `/api/v1/user/list` | Go 自建 JWT | `uint` |

Flutter 登录返回 **token + refresh_token + session_id**；HTTP 业务走 SessionAuth；gRPC 试验走同一套 session（metadata 键名小写亦可）。

### 配置与环境变量

```bash
cp .env.example .env
cp configs/supabase.env.example configs/supabase.env
cp .env.local.example .env.local   # service_role
```

| 文件 | 入库 | 说明 |
|------|------|------|
| `configs/supabase.env` | 是 | `SUPABASE_URL`、`SUPABASE_ANON_KEY` |
| `.env` | 是 | DB、Redis、JWT 等 |
| `.env.local` | 否 | `SUPABASE_SERVICE_ROLE_KEY` |

### 应用启动与依赖注入

**API**：`make run` → `./scripts/load-env.sh go -C app run ./cmd/api`

**Worker**（`queue.enabled=true` 时）：`make run-worker` → `go -C app run ./cmd/worker`

```text
1. APP_ENV → config.Load
2. logger.Init
3. PostgreSQL + autoMigrate（仅 API；含 AnalyticsRecord）
4. Redis
5. UserUsecase（遗留 JWT 路由）+ DeviceSessionUsecase（Redis session）
6. 若业务启用（Supabase 或 local Auth）：
     Auth / Profile / Transactions（可选挂 ES SearchStore）
     Realtime：Hub → Ticket/Sync/Push/Presence → WS + Controller
     若 queue.enabled：RabbitMQ 入队 + Pub/Sub 订阅（API）；Worker 消费并入队广播
     若 sse.enabled：CompletionUsecase → `/api/v1/sse/completions`
     若 cs_agent.enabled：CsAgentUsecase → `/api/v1/cs-agent/chat`
7. router.Setup（仅 API，HTTP；含 `/metrics`）
8. 若 grpc.enabled：AnalyticsUsecase（空表种子）→ gRPC Listen :9090（需 DeviceSession + Auth）
9. HTTP ListenAndServe / Worker：RabbitMQ Consumer + 整点 ticker
10. SIGTERM → HTTP Shutdown + gRPC GracefulStop
```

| 组件 | 未配业务后端 | 已配（Supabase 或 local） |
|------|--------------|---------------------------|
| `/api/v1/user/login` | 503 / 按 provider | 正常 |
| `/api/v1/transactions*` | 未注册 | 正常；search 需 `ELASTICSEARCH_URL` |
| `/api/v1/realtime/*` | 未注册 | 需 Redis |
| `/api/v1/sse/completions` | 未注册 | `sse.enabled` |
| `/api/v1/cs-agent/chat` | 未注册 | `cs_agent.enabled` |
| `/metrics` | 始终注册 | Prometheus 抓取 |
| gRPC `AnalyticsService` | 跳过或未启 | `:9090`（需 session 鉴权） |

**新增 HTTP 组件**：`main.go` 构造 → `router.Options` → `router/` 挂路由。  
**新增 gRPC 服务**：改 `app/api/proto/**` → `make proto` → `app/internal/delivery/grpc` 实现 → `app/cmd/api/main.go` 注册；Flutter stubs 在 `my_ai_project/commons/network`（proto 仍以本仓库为源）。

### 常用命令

```bash
make run
make proto                  # 从 api/proto 生成 Go stubs
make lan-up                 # Docker：8080+9090
make test
make test-transactions
make test-realtime
make test-single-device-login
make check-secrets
./scripts/check_transactions_rls.sh
```

### Agent 修改规范

1. 在本仓库根目录执行 `go` / `make`
2. 新增 Supabase/业务 HTTP 表：entity → repository → usecase → controller → router（SessionAuth）
3. 新增 gRPC：proto（本仓库）→ `make proto` → delivery/grpc + usecase/repo；同步更新 Flutter `commons/network` codegen 与页面
4. 认证错误：`mapSupabaseAuthError` + `UserHandler.handleUsecaseError`（HTTP）；gRPC 返回合适的 `status` 码
5. PostgREST 必须 `WithUserToken`，禁止 Admin 绕过 RLS
6. transactions 必须用会话中的 `userID` 做 `.Eq("user_id", userID)` + RLS 迁移（勿信任客户端传的 user_id）；ES 搜索同样强制会话 `user_id`
7. Flutter 兼容响应注意 snake_case / `{ items: [] }`（HTTP）；gRPC 字段以 proto 为准
8. service_role 仅 `.env.local`；推送前 `make check-secrets`
9. 客服 Agent 与小石头 SSE **分路由**；异步任务走 RabbitMQ，勿再引入 Asynq

### 相关文档

- [文档总览（架构 + 使用）](./docs/README.md)
- [启动指南](./docs/startup-guide.md)
- [架构学习指南](./docs/architecture-learning-guide.md)
- [消息队列（RabbitMQ）](./docs/message-queue.md)
- [可观测 lan-up](./docs/observability-lan.md)
- [交易全文 ES](./docs/elasticsearch-transactions.md)
- [客服 Agent Eino](./docs/cs-agent-eino.md)
- [Supabase 集成说明](./docs/supabase-integration.md)
- [Realtime WebSocket 协议](./docs/realtime-websocket.md)
- [认证初学者导读](./docs/auth-beginner-walkthrough.md)
- [Transactions 初学者导读](./docs/transactions-beginner-walkthrough.md)
- [Realtime 初学者导读](./docs/realtime-beginner-walkthrough.md)
- [Gin HTTP vs gRPC](./docs/grpc/gin-http-vs-grpc.md)
- [数据分析 gRPC 联调](./docs/grpc/analytics-flutter-trial.md)
- Flutter [AGENTS.md](../../my_ai_project/AGENTS.md)
