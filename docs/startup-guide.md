# my_go_study 启动指南

Go 后端服务，采用 Clean Architecture（Gin + GORM + PostgreSQL + Redis + JWT）。

**仓库**：`git@github.com:stvenfor/my_go_study.git`  
**模块路径**：`github.com/stvenfor/my_go_study`  
**默认端口**：`8080`

---

## 1. 环境要求

| 工具 | 版本要求 | 用途 |
|------|----------|------|
| Go | 1.21+ | 编译与运行 |
| Docker | 最新稳定版 | 一键启动 PostgreSQL / Redis / App（推荐） |
| Docker Compose | v2+ | 编排容器 |
| golang-migrate | 可选 | 执行 `migrations/` SQL 迁移 |
| Air | 可选 | 热加载开发（`make air`） |

验证 Go 安装：

```bash
go version
# 期望：go version go1.21+ ...
```

---

## 2. 项目目录（与启动相关）

```text
my_go_study/
├── cmd/api/main.go                    # 应用入口
├── internal/delivery/http/
│   ├── controller/                    # 控制器层（Profile / Transaction 管理）
│   │   ├── profile_controller.go
│   │   └── transaction_controller.go
│   ├── handler/                       # 自建用户 Handler
│   └── router/                        # 路由注册（按模块拆分）
│       ├── router.go
│       ├── user_routes.go
│       ├── profile_routes.go
│       └── transaction_routes.go
├── configs/                 # YAML 配置（按 APP_ENV 合并）
│   ├── config.yaml          # 基础配置
│   ├── config.dev.yaml      # 开发环境覆盖
│   └── config.prod.yaml     # 生产环境覆盖
├── migrations/              # golang-migrate SQL 文件
├── docker/
│   ├── Dockerfile
│   └── docker-compose.yml   # postgres + redis + app
├── configs/
│   ├── config.yaml
│   ├── config.dev.yaml
│   ├── supabase.env          # Supabase URL / anon key（团队常量，入库）
│   └── supabase.env.example
├── .env                     # 应用运行时配置（入库）
├── .env.example
├── .env.local               # service_role 等私密密钥（不入库）
├── Makefile                 # 常用命令
├── .air.toml                # 热加载配置
└── docs/                    # 文档
    ├── startup-guide.md     # 启动与环境
    └── supabase-integration.md  # Supabase 集成详解
```

> **注意**：所有 `go`、`make` 命令必须在 **`my_go_study` 目录内**执行，不要在父目录 `my_code_study` 下执行，否则会报 `go.mod file not found`。

```bash
cd /path/to/my_go_study   # 进入项目根目录
```

---

## 3. 配置说明

### 3.1 加载顺序

1. 读取 `configs/config.yaml`
2. 若设置 `APP_ENV=dev`，合并 `configs/config.dev.yaml`
3. 环境变量覆盖 YAML（见下表）

### 3.2 常用环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `APP_ENV` | `dev` | 配置环境：`dev` / `prod` |
| `SERVER_PORT` | `8080` | HTTP 端口 |
| `DATABASE_HOST` | `localhost` | PostgreSQL 主机 |
| `DATABASE_PORT` | `5432` | PostgreSQL 端口 |
| `DATABASE_USER` | `postgres` | 数据库用户 |
| `DATABASE_PASSWORD` | `postgres` | 数据库密码 |
| `DATABASE_DBNAME` | `my_go_study` | 数据库名 |
| `REDIS_ADDR` | `localhost:6379` | Redis 地址 |
| `JWT_SECRET` | （必填） | JWT 签名密钥，生产环境务必修改 |
| `SUPABASE_URL` | `configs/supabase.env` | Supabase Project URL |
| `SUPABASE_ANON_KEY` | `configs/supabase.env` | anon / publishable key |
| `SUPABASE_SERVICE_ROLE_KEY` | `.env.local` | service_role；勿写入入库文件 |

本地开发：

```bash
cp .env.example .env
cp .env.local.example .env.local   # 填写 service_role
# configs/supabase.env 通常已入库，无需复制
```

### 3.3 启动时自动完成的操作

`cmd/api/main.go` 启动流程：

1. 加载配置（Viper）
2. 初始化 Zap 日志（控制台 + `logs/app.log` 轮转）
3. 连接 PostgreSQL（GORM，含连接池）
4. **GORM AutoMigrate**（自动同步 `users` 表结构）
5. 连接 Redis 并 Ping
6. 依赖注入 → 启动 Gin HTTP 服务
7. 监听 `SIGINT` / `SIGTERM`，优雅关机（10 秒超时）

---

## 4. 启动方式

### 方式 A：Docker Compose（推荐，零基础最快）

一条命令启动 **PostgreSQL + Redis + App**：

```bash
cd my_go_study
make docker-up
```

等价于：

```bash
docker compose -f docker/docker-compose.yml up -d --build
```

**容器说明**：

| 服务 | 容器名 | 端口 |
|------|--------|------|
| PostgreSQL | `my_go_study_postgres` | `5432` |
| Redis | `my_go_study_redis` | `6379` |
| Go API | `my_go_study_app` | `8080` |

查看日志：

```bash
docker logs -f my_go_study_app
```

停止并删除容器：

```bash
make docker-down
```

---

### 方式 B：本地开发（无需 Docker，推荐当前环境）

本机未安装 Docker 时，用 Homebrew 启动数据库：

```bash
cd my_go_study
make deps-up    # 安装/启动 PostgreSQL@16 + Redis，创建数据库
go mod tidy
make run
```

若已手动安装过 PostgreSQL / Redis，也可：

```bash
brew services start postgresql@16
brew services start redis
make run
```

---

### 方式 C：本地开发（Docker 仅跑数据库，本机跑 Go）

> 需已安装 Docker Desktop。

```bash
cd my_go_study
docker compose -f docker/docker-compose.yml up -d postgres redis
go mod tidy
make run
```

1. **File → Open** → 选择 `my_go_study` 文件夹（不是父目录）
2. 打开 `cmd/api/main.go`
3. 配置 Run Configuration：
   - **Run kind**：Package
   - **Package path**：`github.com/stvenfor/my_go_study/cmd/api`
   - **Working directory**：`$ProjectFileDir$`（项目根目录）
4. 在 Environment 中添加（或依赖 `.env` + 插件）：

   ```
   APP_ENV=dev
   DATABASE_HOST=localhost
   REDIS_ADDR=localhost:6379
   JWT_SECRET=change-me-in-production
   ```

5. 先确保 PostgreSQL、Redis 已启动，再点击 Run

---

## 5. 验证服务

### 5.1 健康检查

```bash
curl http://localhost:8080/health
```

期望响应：

```json
{"status":"ok"}
```

### 5.2 注册用户

```bash
curl -X POST http://localhost:8080/api/v1/user/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "test",
    "password": "123456",
    "email": "test@example.com"
  }'
```

成功响应示例（`id` 为 Supabase UUID 字符串）：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "3704f664-5f2c-4ea9-acdf-f244256dc935",
    "username": "test",
    "email": "test@example.com"
  }
}
```

### 5.3 登录获取 Token

```bash
curl -X POST http://localhost:8080/api/v1/user/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "test@example.com",
    "password": "123456"
  }'
```

> `username` 填**邮箱地址**。返回的 `data.token` 为 **Supabase access token**（非自建 JWT）。

从 `data.token` 复制 token，用于下一步。

### 5.4 获取个人信息（需 Supabase JWT）

```bash
curl http://localhost:8080/api/v1/me/profile \
  -H "Authorization: Bearer <你的token>"
```

> `/api/v1/user/profile` 使用自建 JWT 中间件，与 Supabase 登录 token **不兼容**。Flutter 请走 `/api/v1/me/profile`。

---

## 6. API 一览

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| GET | `/health` | 否 | 健康检查 |
| POST | `/api/v1/user/register` | 否 | 邮箱注册（Supabase Auth） |
| POST | `/api/v1/user/login` | 否 | 邮箱登录，返回 Supabase access token |
| GET | `/api/v1/user/list` | Bearer 自建 JWT | 本地用户列表（遗留，`?page=1&size=20`） |
| GET | `/api/v1/user/profile` | Bearer 自建 JWT | 本地用户详情（遗留，与 Supabase 登录不兼容） |

### 6.1 Supabase 接口（需 Supabase JWT）

认证方式：**Supabase access token**（`Authorization: Bearer <token>`）

#### Flutter 兼容接口（snake_case 直出 JSON）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/me/profile` | 读取 profiles |
| PATCH | `/api/v1/me/profile` | 更新 display_name / avatar_url |
| GET | `/api/v1/transactions` | 列表 `?type=&limit=&offset=` → `{ "items": [...] }` |
| POST | `/api/v1/transactions` | 创建收支 |
| GET | `/api/v1/transactions/:id` | 单条详情 |
| PUT | `/api/v1/transactions/:id` | 更新 |
| DELETE | `/api/v1/transactions/:id` | 删除（204） |

#### 统一管理接口（统一响应 `{ code, message, data }`，camelCase 字段）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/profiles/me` | 获取当前用户资料 |
| PATCH | `/api/v1/profiles/me` | 更新资料 |
| GET | `/api/v1/transactions/manage` | 分页列表 `?page=1&size=20&type=` |
| POST | `/api/v1/transactions/manage` | 创建收支 |
| GET | `/api/v1/transactions/manage/:id` | 单条详情 |
| PUT | `/api/v1/transactions/manage/:id` | 更新 |
| DELETE | `/api/v1/transactions/manage/:id` | 删除 |

控制器与路由文件：

| 模块 | 控制器 | 路由文件 |
|------|--------|----------|
| Profile | `controller/profile_controller.go` | `router/profile_routes.go` |
| Transaction | `controller/transaction_controller.go` | `router/transaction_routes.go` |
| 自建用户 | `handler/user_handler.go` | `router/user_routes.go` |

数据通过 PostgREST 访问 Supabase `profiles` / `transactions` 表。

**transactions 用户隔离（双层防护）**：

1. **Go 后端**：所有查询强制带 `user_id = 当前登录用户`（已实现）
2. **Supabase RLS**：需在 Dashboard 执行 `supabase/migrations/003_transactions_user_id_rls.sql`（见下文 §10）

transactions 联调：

```bash
make test-transactions          # 完整 CRUD 测试
./scripts/check_transactions_rls.sh  # 检查 RLS 是否已在 Supabase 启用
```

#### Realtime WebSocket（需 Supabase JWT）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/realtime/ws-ticket` | 签发 WS 一次性 ticket |
| POST | `/api/v1/realtime/sync` | 重连后按 `sinceSeq` 增量同步 |
| POST | `/api/v1/realtime/push` | 开发环境推送 `sys.notify` |
| GET | `/realtime/v1/connect` | WebSocket 升级（无 HTTP 鉴权，首帧 `auth` 换票） |

响应为 **直出 JSON**（非 `ResultModel` 信封）。协议、心跳、消息示例见 [Realtime WebSocket 协议与联调指南](./realtime-websocket.md)。

```bash
make test-realtime   # 一键联调（需 Redis + make run）
```

控制器与路由：`controller/realtime_controller.go`、`router/realtime_routes.go`、`delivery/ws/`。

### 6.2 自建用户接口响应格式

非列表：

```json
{
  "code": 0,
  "message": "success",
  "data": { "id": 1, "username": "test", "createdAt": "..." },
  "timestamp": 1749456789
}
```

列表（含 `pagination`）：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [{ "id": 1, "username": "user1", "email": "user1@test.com", "createdAt": "..." }],
    "pagination": { "page": 1, "size": 20, "total": 100, "totalPages": 5 }
  },
  "timestamp": 1749456789
}
```

**错误码**：`0` 成功 · `10001` 参数错误 · `10002` 未授权 · `10004` 不存在 · `50000` 服务器错误

---

## 7. Makefile 命令速查

| 命令 | 说明 |
|------|------|
| `make run` | `go run ./cmd/api` |
| `make build` | 编译到 `bin/api` |
| `make test` | 运行全部单元测试 |
| `make tidy` | `go mod tidy` |
| `make air` | Air 热加载开发 |
| `make migrate-up` | 执行数据库迁移（升级） |
| `make migrate-down` | 回滚最近一次迁移 |
| `make docker-up` | 构建并启动全部容器（需 Docker Desktop） |
| `make deps-up` | Homebrew 安装并启动 PostgreSQL + Redis（无需 Docker） |
| `make docker-down` | 停止并移除容器 |
| `make clean` | 删除 `bin/`、`tmp/`、日志 |
| `make test-transactions` | transactions CRUD 联调（`SUPABASE_SERVICE_ROLE_KEY` 放 `.env.local`） |
| `make test-realtime` | WebSocket 联调（ws-ticket / sync / push + WS auth） |
| `make check-secrets` | 推送前检查入库文件是否含 service_role 密钥 |

---

## 8. 与 Flutter 客户端联调

Flutter 项目 `my_ai_project` 中 `BackendApiClient` 默认连接：

```text
http://127.0.0.1:8080   # iOS 模拟器 / 本机
http://10.0.2.2:8080    # Android 模拟器（需在 EnvConfig 中修改）
```

联调前确认：

1. Go 后端已启动且 `/health` 返回 ok
2. Flutter `.env` 设置 `USE_MOCK_AUTH=false`
3. 登录走 `POST /api/v1/user/login`，Go 后端代理 Supabase Auth，返回 **Supabase access token**
4. `AuthHeaderProvider` 自动在业务请求头附带 `Authorization: Bearer <token>`
5. **业务数据**（profile / transactions）走 `/api/v1/me/*`、`/api/v1/transactions`
6. 详细架构见 [Supabase 集成说明](./supabase-integration.md)

### 8.1 Realtime WebSocket

登录后 Flutter `module_realtime` 自动连接 Go WS 网关：

```text
POST /api/v1/realtime/ws-ticket  → 换票
GET  /realtime/v1/connect        → WebSocket 升级 + auth
POST /api/v1/realtime/sync       → 重连增量同步
POST /api/v1/realtime/push       → 开发推送 sys.notify
```

联调：

```bash
make test-realtime   # 需 make run + Redis；.env.local 配 service_role 可自动建测试用户
```

协议、心跳、收发消息示例见 [Realtime WebSocket 协议与联调指南](./realtime-websocket.md)。

Flutter 调试：设置 → **Realtime / WebSocket 调试**（Debug 模式）。

---

## 9. 常见问题

### `go.mod file not found`

当前目录不是项目根。请执行：

```bash
cd my_go_study
```

### `连接 PostgreSQL 失败` / `PostgreSQL ping 失败`

- 确认 Postgres 已启动：`docker ps | grep my_go_study_postgres`
- 确认 `DATABASE_HOST`：本机开发用 `localhost`，Docker 内 App 用 `postgres`

### `连接 Redis 失败`

- 确认 Redis 已启动：`docker ps | grep my_go_study_redis`
- 确认 `REDIS_ADDR`：本机 `localhost:6379`，Docker 内 `redis:6379`

### 端口 8080 被占用

修改 `SERVER_PORT` 或 `configs/config.yaml` 中的 `server.port`，并重启服务。

### `jwt.secret 不能为空`

设置环境变量 `JWT_SECRET`，或检查 `configs/config.yaml` 中 `jwt.secret` 是否有值。

### `make docker-up` 报 `docker: No such file or directory`

未安装 Docker Desktop。二选一：

**方案 1（推荐，无需 Docker）**：

```bash
make deps-up   # Homebrew 安装并启动 PostgreSQL + Redis
make run
```

**方案 2**：安装 [Docker Desktop for Mac](https://www.docker.com/products/docker-desktop/)，启动后再执行 `make docker-up`。

---

## 10. Supabase transactions RLS（必做）

若未在 Supabase 执行 RLS 迁移，用户通过 **REST API 直连** 仍可能读写他人数据。Go 后端已加 `user_id` 过滤，但 Flutter 若直连 Supabase 或 RLS 未开仍有风险。

### 10.1 一键执行

1. 打开 [Supabase SQL Editor](https://supabase.com/dashboard/project/uqznnzkugvhsrlcudrbj/sql/new)
2. 粘贴 `supabase/migrations/003_transactions_user_id_rls.sql` 全部内容
3. 点击 **Run**
4. 结果应显示 `rls_enabled = true`，`policy_count >= 4`

### 10.2 验证

```bash
./scripts/check_transactions_rls.sh
```

期望输出：

- `✅ Supabase RLS 已生效` — 数据库层隔离正常
- `✅ Go 后端 user_id 过滤已生效` — 应用层隔离正常

### 10.3 你的测试数据

用户 `3704f664-5f2c-4ea9-acdf-f244256dc935` 的 55 条 transactions 在启用 RLS 后，仅该用户登录可见；其他用户列表为空。

---

## 11. 推荐日常开发流程

```bash
# 1. 进入项目
cd my_go_study

# 2. 启动数据库（无 Docker 用 deps-up，有 Docker 用 compose）
make deps-up
# 或: docker compose -f docker/docker-compose.yml up -d postgres redis

# 3. 拉依赖
go mod tidy

# 4. 启动服务
make run
```

---

## 12. 生产部署提示

1. 设置 `APP_ENV=prod`，使用 `configs/config.prod.yaml`
2. **务必修改** `JWT_SECRET` 和数据库密码
3. 数据库 `sslmode` 建议设为 `require`
4. 使用 `make build` 编译二进制，或通过 `docker/Dockerfile` 多阶段构建镜像部署
5. `.env` 须入库；`SUPABASE_SERVICE_ROLE_KEY` 仅放 `.env.local`；推送前运行 `make check-secrets`
