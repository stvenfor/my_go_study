# my_go_study Agent 开发指南

Go BFF 后端 Agent 指南，并包含与 Flutter **`my_ai_project`** 协作的工作区总览。

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
    │  HTTP JSON（ResultModel / 直出 JSON）
    ▼
my_go_study (Gin BFF :8080)    ← 你在这里
    │  Auth / PostgREST / Redis
    ▼
Supabase Cloud（Auth + PostgreSQL + RLS）
```

| 能力 | Flutter | Go（本仓库） | 外部 |
|------|---------|--------------|------|
| 登录注册 | `BackendAuthService` | `UserHandler` → `SupabaseAuthUsecase` + `DeviceSessionUsecase` | Supabase Auth + Redis session |
| 收支/二手车 | `TransactionApi` | `TransactionController` | PostgREST + RLS |
| 实时消息 | `AppRealtimeClient` | `RealtimeController` + WS Hub | Redis |
| 用户资料 | — | `ProfileController` | PostgREST |

### 本地联调

> **工作目录**：以下命令均在 **`my_go_study/`** 目录执行（`cd my_go_study`），不要在工作区根目录 `my_code_study/` 运行 `make`。

```bash
# 终端 1：Go API（在 my_go_study/ 目录）
make deps-up    # 首次：PostgreSQL + Redis
make run        # :8080

# 终端 2（dev 默认 queue.enabled=true）：Asynq Worker（同样在 my_go_study/）
make run-worker

# 终端 3：Flutter
cd ../../my_ai_project
flutter run --dart-define-from-file=.env   # USE_MOCK_AUTH=false

# 验证
curl http://127.0.0.1:8080/health
make test-realtime    # 需 make run 已启动
make test-queue-push  # 需 make run + make run-worker
make trigger-hourly-notify  # 手动触发定时广播
make test-scheduled-notify  # 定时通知联调
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
| 密钥 | `make check-secrets` |

### 文档地图

**Go（`docs/`）**

| 文档 | 用途 |
|------|------|
| [startup-guide.md](./docs/startup-guide.md) | 环境、Makefile、Docker |
| [supabase-integration.md](./docs/supabase-integration.md) | Supabase 架构 |
| [realtime-websocket.md](./docs/realtime-websocket.md) | WS 协议 |
| [auth-beginner-walkthrough.md](./docs/auth-beginner-walkthrough.md) | 认证导读 |
| [transactions-beginner-walkthrough.md](./docs/transactions-beginner-walkthrough.md) | 收支 API 导读 |
| [realtime-beginner-walkthrough.md](./docs/realtime-beginner-walkthrough.md) | Realtime 导读 |
| [message-queue.md](./docs/message-queue.md) | Asynq + Pub/Sub 异步队列 |

**Flutter（`my_ai_project/docs/`）**

| 文档 | 用途 |
|------|------|
| [BACKEND_INTEGRATION.md](../../my_ai_project/docs/BACKEND_INTEGRATION.md) | HTTP、认证、Realtime |
| [USAGE_GUIDE.md](../../my_ai_project/docs/USAGE_GUIDE.md) | 运行与环境 |

### 全局约束

1. **所有 `go` / `make` 在本仓库根目录执行**，不要在父目录 `my_code_study` 执行
2. Flutter 业务**不直连** Supabase SDK，经 Go BFF
3. `SUPABASE_SERVICE_ROLE_KEY` 仅 `.env.local`；推送前 `make check-secrets`
4. **两套 Token 勿混用**：Flutter 用 Supabase JWT；遗留 `/api/v1/user/list` 用 Go 自建 JWT
5. 两仓库**独立 git**，分别 push

---

## 二、Go 后端（本仓库）

### 项目定位

- **框架**：Gin + GORM + PostgreSQL + Redis + Clean Architecture
- **模块路径**：`github.com/stvenfor/my_go_study`
- **默认端口**：`8080`
- **配对客户端**：Flutter `my_ai_project`

### 分层结构

```text
cmd/api/main.go                 # 依赖注入、启动 HTTP+WS
cmd/worker/main.go              # Asynq Worker（异步 Push / SMS / JPush 占位）
internal/
  delivery/http/
    handler/                    # 自建用户 HTTP（UserHandler）
    controller/                 # Supabase 业务 HTTP（Profile / Transaction / Realtime）
    middleware/                 # JWT / SupabaseAuth / CORS / Logger
    router/                     # 按模块拆分路由
    dto/request|response/       # 请求/响应 DTO
  delivery/ws/                  # WebSocket Hub / Handler / Client
  usecase/                      # 业务用例
  domain/entity|repository/     # 领域实体与仓储接口
  repository/
    postgres/                   # 本地 PostgreSQL
    redis/                      # Realtime ticket / 事件 / presence
    supabase/                   # Supabase PostgREST
pkg/
  config/                       # Viper 配置
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

| 中间件 | 路由示例 | Token 类型 | 用户 ID |
|--------|----------|------------|---------|
| `SupabaseSessionAuth` | `/api/v1/transactions*`、`/api/v1/realtime/*` | Supabase access token + session | UUID |
| `Auth`（JWT） | `/api/v1/user/list` | Go 自建 JWT | `uint` |

Flutter 登录返回 **Supabase token + refresh_token + session_id**，业务 API 走 `SupabaseSessionAuth`。

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
3. PostgreSQL + autoMigrate（仅 API）
4. Redis
5. UserUsecase（遗留 JWT 路由）+ DeviceSessionUsecase（Redis session）
6. 若 Supabase.Enabled：
     Auth / Profile / Transactions
     Realtime：Hub → Ticket/Sync/Push/Presence → WS + Controller
     若 queue.enabled：Asynq 入队 + Pub/Sub 订阅（API）；Worker 消费并入队广播
7. router.Setup（仅 API）
8. ListenAndServe / Asynq Server.Run
9. SIGTERM → 优雅 Shutdown
```

| 组件 | 未配 Supabase | 已配 Supabase |
|------|---------------|---------------|
| `/api/v1/user/login` | 503 | 正常 |
| `/api/v1/transactions*` | 未注册 | 正常 |
| `/api/v1/realtime/*` | 未注册 | 需 Redis |

**新增组件**：`main.go` 构造 → `router.Options` → `router/` 挂路由。

### 常用命令

```bash
make run
make test
make test-transactions
make test-realtime
make test-single-device-login
make check-secrets
./scripts/check_transactions_rls.sh
```

### Agent 修改规范

1. 在本仓库根目录执行 `go` / `make`
2. 新增 Supabase 表：entity → repository → usecase → controller → router（`SupabaseSessionAuth`）
3. 认证错误：`mapSupabaseAuthError` + `UserHandler.handleUsecaseError`
4. PostgREST 必须 `WithUserToken`，禁止 Admin 绕过 RLS
5. transactions 必须 `.Eq("user_id", userID)` + RLS 迁移
6. Flutter 兼容响应注意 snake_case / `{ items: [] }`
7. service_role 仅 `.env.local`；推送前 `make check-secrets`

### 相关文档

- [启动指南](./docs/startup-guide.md)
- [Supabase 集成说明](./docs/supabase-integration.md)
- [Realtime WebSocket 协议](./docs/realtime-websocket.md)
- [认证初学者导读](./docs/auth-beginner-walkthrough.md)
- [Transactions 初学者导读](./docs/transactions-beginner-walkthrough.md)
- [Realtime 初学者导读](./docs/realtime-beginner-walkthrough.md)
- Flutter [AGENTS.md](../../my_ai_project/AGENTS.md)
