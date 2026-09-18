# my_go_study Agent 开发指南

Go BFF 后端 Agent 指南，并包含与 Flutter **`my_ai_project`** 协作的工作区总览。

### Agent 编排（`.harness`）

多角色 / 多会话任务默认走编排层，**不要**跳过 Brief 与验证直接自称完成：

| 入口 | 路径 |
|------|------|
| 编排总览 | [`.harness/README.md`](./.harness/README.md) |
| 角色合同 | [`.harness/agents/`](./.harness/agents/)（conductor → planner → executor → reviewer） |
| 当前指针 | [`.harness/changes/current.md`](./.harness/changes/current.md) |
| 硬规则 | [`.harness/rules/`](./.harness/rules/) |
| 文档索引 | [`.harness/wiki/README.md`](./.harness/wiki/README.md) |

新任务未指定角色时，先读 **conductor**；人批 Brief 与 commit/push。

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
    │  Auth / Postgres / Redis / PostgREST
    ▼
Supabase Cloud 或 local Auth + 本地 Postgres
```

| 能力 | Flutter | Go（本仓库） | 外部 |
|------|---------|--------------|------|
| 登录注册 | `BackendAuthService` | `UserHandler` → Auth usecase + `DeviceSessionUsecase` | Auth + Redis session |
| 收支/二手车 | `TransactionApi` | `TransactionController` | PostgREST + RLS |
| 实时消息 | `AppRealtimeClient` | `RealtimeController` + WS Hub | Redis |
| 用户资料 | — | `ProfileController` | PostgREST |
| 数据分析 | `AnalyticsGrpcApi`（gRPC） | `AnalyticsService` → `AnalyticsUsecase` | 本地表 `analytics_records` |

### 本地联调

> **工作目录**：以下命令均在 **`my_go_study/`** 目录执行（`cd my_go_study`），不要在工作区根目录 `my_code_study/` 运行 `make`。

```bash
# 方案 A（推荐联调）：Docker 统一管理 HTTP:8080 + gRPC:9090 + PG + Redis + Worker
# cp .env.lan.example .env.lan 后：
make lan-up                 # docs/lan-backend-host.md；改 Go 后需再执行（会 --build）
make lan-down
# 勿与本机 make run 同时开，会抢 :8080

# 方案 B：本机进程
make deps-up                # 首次：PostgreSQL + Redis
make run                    # HTTP :8080 + gRPC :9090（同进程）
make run-worker             # queue.enabled=true 时另开终端

# Flutter（路径相对本仓库：../../my_ai_project）
cd ../../my_ai_project
flutter run --dart-define-from-file=.env.lan   # 真机：BACKEND_HOST=局域网IP；gRPC 默认 9090

# 验证
curl http://127.0.0.1:8080/health
make test-realtime          # 需 API 已启动
make test-queue-push        # 需 API + Worker
make trigger-hourly-notify
make test-scheduled-notify
make test-single-device-login
make test-phone-otp-login
```

| 检查项 | 说明 |
|--------|------|
| 健康 | `GET /health` |
| 登录 | `POST /api/v1/user/login` |
| 刷新 Token | `POST /api/v1/user/refresh` |
| 退出登录 | `POST /api/v1/user/logout` |
| 测试手机号 OTP | `POST /api/v1/user/phone/otp/send` · `verify`（dev：`13400000000` + `123456`） |
| Realtime | Flutter 设置 → Realtime 调试 |
| 异步 Push | `make test-queue-push`（需 Worker） |
| 定时通知 | `make trigger-hourly-notify` + `make test-scheduled-notify` |
| 局域网 / Docker | `make lan-up`：映射 **8080 + 9090**；Flutter 只需 `BACKEND_HOST`（gRPC 端口默认 9090） |
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
| [message-queue.md](./docs/message-queue.md) | Asynq + Pub/Sub 异步队列 |
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

---

## 二、Go 后端（本仓库）

### 项目定位

- **框架**：Gin + gRPC + GORM + PostgreSQL + Redis + Clean Architecture
- **模块路径**：`github.com/stvenfor/my_go_study`
- **默认端口**：HTTP `8080`；gRPC `9090`（`GRPC_ENABLED` / `GRPC_PORT`）
- **配对客户端**：Flutter `my_ai_project`

### 分层结构

```text
cmd/api/main.go                 # 依赖注入、启动 HTTP+WS+gRPC
cmd/worker/main.go              # Asynq Worker（异步 Push / SMS / JPush 占位）
api/
  proto/analytics/v1/           # Protobuf 契约（单一真相源）
  gen/go/analytics/v1/          # make proto 生成的 Go stubs
internal/
  delivery/http/
    handler/                    # 自建用户 HTTP（UserHandler）
    controller/                 # 业务 HTTP（Profile / Transaction / Realtime）
    middleware/                 # JWT / SessionAuth / CORS / Logger
    router/                     # 按模块拆分路由
    dto/request|response/       # 请求/响应 DTO
  delivery/grpc/                # gRPC Server + AnalyticsService + 鉴权 interceptor
  delivery/ws/                  # WebSocket Hub / Handler / Client
  usecase/                      # 业务用例（含 AnalyticsUsecase）
  domain/entity|repository/     # 领域实体与仓储接口
  repository/
    postgres/                   # 本地 PostgreSQL（含 analytics_records）
    redis/                      # Realtime ticket / 事件 / presence
    supabase/                   # Supabase PostgREST
pkg/
  config/                       # Viper 配置（含 grpc.*）
  queue/                        # Asynq 客户端/处理器、Redis Pub/Sub 广播
  supabase/                     # Supabase 客户端封装
  auth/                         # Token 校验（/auth/v1/user）
  jwt/                          # 自建 JWT（遗留路由）
```

### Supabase 集成要点

完整文档：[docs/supabase-integration.md](./docs/supabase-integration.md)

| 能力 | 入口 | Supabase 交互方式 |
|------|------|-------------------|
| 邮箱注册/登录 | `UserHandler` → `SupabaseAuthUsecase` | gotrue-go |
| Token 校验 | `middleware.SupabaseSessionAuth` | `GET /auth/v1/user` + Redis session |
| Profile CRUD | `ProfileController` | PostgREST + `WithUserToken` |
| Transactions CRUD | `TransactionController` | PostgREST + `user_id` + RLS |
| Realtime WS | `RealtimeController` + `ws.Handler` | Redis ticket → Hub 广播 |

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

**API**：`make run` → `./scripts/load-env.sh go run ./cmd/api`

**Worker**（`queue.enabled=true` 时）：`make run-worker` → `go run ./cmd/worker`

```text
1. APP_ENV → config.Load
2. logger.Init
3. PostgreSQL + autoMigrate（仅 API；含 AnalyticsRecord）
4. Redis
5. UserUsecase（遗留 JWT 路由）+ DeviceSessionUsecase（Redis session）
6. 若业务启用（Supabase 或 local Auth）：
     Auth / Profile / Transactions
     Realtime：Hub → Ticket/Sync/Push/Presence → WS + Controller
     若 queue.enabled：Asynq 入队 + Pub/Sub 订阅（API）；Worker 消费并入队广播
7. router.Setup（仅 API，HTTP）
8. 若 grpc.enabled：AnalyticsUsecase（空表种子）→ gRPC Listen :9090（需 DeviceSession + Auth）
9. HTTP ListenAndServe / Asynq Server.Run
10. SIGTERM → HTTP Shutdown + gRPC GracefulStop
```

| 组件 | 未配业务后端 | 已配（Supabase 或 local） |
|------|--------------|---------------------------|
| `/api/v1/user/login` | 503 / 按 provider | 正常 |
| `/api/v1/transactions*` | 未注册 | 正常 |
| `/api/v1/realtime/*` | 未注册 | 需 Redis |
| gRPC `AnalyticsService` | 跳过或未启 | `:9090`（需 session 鉴权） |

**新增 HTTP 组件**：`main.go` 构造 → `router.Options` → `router/` 挂路由。  
**新增 gRPC 服务**：改 `api/proto/**` → `make proto` → `internal/delivery/grpc` 实现 → `main.go` 注册；Flutter stubs 在 `my_ai_project/commons/network`（proto 仍以本仓库为源）。

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
6. transactions 必须 `.Eq("user_id", userID)` + RLS 迁移
7. Flutter 兼容响应注意 snake_case / `{ items: [] }`（HTTP）；gRPC 字段以 proto 为准
8. service_role 仅 `.env.local`；推送前 `make check-secrets`

### 相关文档

- [文档总览（架构 + 使用）](./docs/README.md)
- [启动指南](./docs/startup-guide.md)
- [架构学习指南](./docs/architecture-learning-guide.md)
- [Supabase 集成说明](./docs/supabase-integration.md)
- [Realtime WebSocket 协议](./docs/realtime-websocket.md)
- [认证初学者导读](./docs/auth-beginner-walkthrough.md)
- [Transactions 初学者导读](./docs/transactions-beginner-walkthrough.md)
- [Realtime 初学者导读](./docs/realtime-beginner-walkthrough.md)
- [Gin HTTP vs gRPC](./docs/grpc/gin-http-vs-grpc.md)
- [数据分析 gRPC 联调](./docs/grpc/analytics-flutter-trial.md)
- Flutter [AGENTS.md](../../my_ai_project/AGENTS.md)
