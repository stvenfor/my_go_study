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
├── cmd/worker/main.go                 # RabbitMQ Worker
├── configs/                           # YAML 配置（按 APP_ENV 合并）
│   ├── config.yaml
│   ├── config.dev.yaml
│   ├── config.lan.yaml
│   ├── config.prod.yaml
│   ├── supabase.env                   # URL / anon（入库）
│   └── supabase.env.example
├── migrations/
├── docker/
│   ├── Dockerfile                     # 多阶段构建（TARGETARCH；默认 CMD=api）
│   ├── docker-compose.yml             # postgres + redis + app + worker
│   └── docker-compose.lan.yml         # 局域网 overlay（AUTH_PROVIDER=local）
├── scripts/
│   ├── docker-compose.sh              # make docker-up 入口
│   ├── lan-compose.sh                 # make lan-up 入口
│   └── lan-run.sh                     # 无 Docker 的 lan 本机进程
├── .env                               # 应用运行时配置（入库）
├── .env.example
├── .env.lan.example                   # 局域网模板（复制为 .env.lan）
├── .env.local                         # service_role（不入库）
├── Makefile
└── docs/
    ├── startup-guide.md
    ├── dual-end-lan-startup.md
    └── lan-backend-host.md
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
| `JWT_SECRET` | **必填** | JWT 签名密钥；dev/lan 可用占位值（启动会 Warn），prod 硬校验（见 §13） |
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

一条命令启动 **PostgreSQL + Redis + RabbitMQ + Elasticsearch + App + Worker + 观测栈**：

```bash
cd my_go_study
make docker-up
```

等价于：

```bash
./scripts/docker-compose.sh up -d --build
# 脚本会先加载 configs/supabase.env、.env、.env.local，再注入容器
```

**容器说明**：

| 服务 | 容器名 | 端口 | 备注 |
|------|--------|------|------|
| PostgreSQL | `my_go_study_postgres` | `127.0.0.1:5432` | 仅本机 |
| Redis | `my_go_study_redis` | `127.0.0.1:6379` | 仅本机 |
| RabbitMQ | `my_go_study_rabbitmq` | `127.0.0.1:5672` / `15672` | 管理 UI guest/guest |
| Elasticsearch | `my_go_study_elasticsearch` | `127.0.0.1:9200` | 交易全文 |
| Go API | `my_go_study_app` | `8080` | HTTP healthcheck；`/metrics` |
| Worker | `my_go_study_worker` | 无 | RabbitMQ 消费；已 disable 镜像默认 /health |
| Prometheus | `my_go_study_prometheus` | `127.0.0.1:9091` | 抓取 app `/metrics` |
| Alertmanager | `my_go_study_alertmanager` | `127.0.0.1:9093` | 飞书经 feishu-adapter |
| Grafana | `my_go_study_grafana` | `127.0.0.1:3000` | 默认 admin/admin |

观测细节 → [observability-lan.md](./observability-lan.md)；ES → [elasticsearch-transactions.md](./elasticsearch-transactions.md)。

查看日志：

```bash
docker logs -f my_go_study_app
```

停止并删除容器：

```bash
make docker-down
```

> **改 Go 代码后**：须再次 `make docker-up`（会 `--build`）。局域网真机用 `make lan-up`（见 [dual-end-lan-startup.md](./dual-end-lan-startup.md)）。
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
    "password": "123456",
    "device_id": "dev-test-device",
    "platform": "ios"
  }'
```

> `username` 填**邮箱地址**。返回的 `data.token` 为 **Supabase access token**，`data.refresh_token` 用于静默续期，`data.session_id` 用于业务 API 的 `X-Session-ID` 头。

### 5.3.1 刷新 Token

```bash
curl -X POST http://localhost:8080/api/v1/user/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "<refresh_token>"}'
```

### 5.3.2 退出登录

```bash
curl -X POST http://localhost:8080/api/v1/user/logout \
  -H "Authorization: Bearer <access_token>" \
  -H "X-Session-ID: <session_id>" \
  -H "X-Device-ID: <device_id>"
```

从 `data.token` 复制 token，用于下一步。

### 5.4 获取个人信息与切换店铺（需业务 access token）

本地登录还需带 `X-Session-ID` / `X-Device-ID`（与登录时 `device_id` 一致）。`store_id` 为正整数（1=沃德龙鼎，2=腾远）。

```bash
# 读资料（可选 ?store_id=1；省略则用 users.current_store_id）
curl "http://localhost:8080/api/v1/profiles/me?store_id=1" \
  -H "Authorization: Bearer <你的token>" \
  -H "X-Session-ID: <session_id>" \
  -H "X-Device-ID: <device_id>"

# 经销商列表（我作为成员的店；含 is_current）
curl "http://localhost:8080/api/v1/me/stores?user_id=<uid>" \
  -H "Authorization: Bearer <你的token>" \
  -H "X-Session-ID: <session_id>" \
  -H "X-Device-ID: <device_id>"

# 切换店铺，响应为该店 stats
curl -X POST "http://localhost:8080/api/v1/profiles/me/store" \
  -H "Authorization: Bearer <你的token>" \
  -H "X-Session-ID: <session_id>" \
  -H "X-Device-ID: <device_id>" \
  -H "Content-Type: application/json" \
  -d '{"store_id":2}'
```

响应 `data.stats`（snake_case）：`store_id` / `store_name` / `days_joined` / `employee_count` / `store_days` / `total_customers` / `role` / `role_label`。数字来自 `wys_user_store_stats`。`role` 来自 `wys_store_member.position`（0 销售顾问，1 销售经理，2 总经理）；还不是该店成员时 `role` 为 null，`role_label` 为空，不再默认成销售顾问。

`GET /me/stores` 的 `data`：`list[]`（`store_id` / `store_name` / `role` / `role_label` / `is_current`）+ `current_store_id`。

门店与权限（仅 `auth.provider=local`；需登录且 `user_id` 与会话一致）：

```bash
# 有效权限（可选 ?store_id=）
curl "http://localhost:8080/api/v1/me/permissions" \
  -H "Authorization: Bearer <token>" -H "X-Session-ID: <s>" -H "X-Device-ID: <d>" \
  -H "Content-Type: application/json" -G --data-urlencode "user_id=<uid>"

# 建店（需 store.create）
curl -X POST "http://localhost:8080/api/v1/stores?user_id=<uid>" \
  -H "Authorization: Bearer <token>" -H "X-Session-ID: <s>" -H "X-Device-ID: <d>" \
  -H "Content-Type: application/json" \
  -d '{"store_id":3,"name":"新店"}'

# 加成员 / 改职务（需 member.write；平台可管任意店，店内角色只能管当前店）
curl -X POST "http://localhost:8080/api/v1/stores/3/members?user_id=<uid>" \
  -H "Authorization: Bearer <token>" -H "X-Session-ID: <s>" -H "X-Device-ID: <d>" \
  -H "Content-Type: application/json" \
  -d '{"target_user_id":"<target>","position":1}'

# 分配角色（平台角色不带 store_id；店内角色必须带）
curl -X POST "http://localhost:8080/api/v1/roles/assignments?user_id=<uid>" \
  -H "Authorization: Bearer <token>" -H "X-Session-ID: <s>" -H "X-Device-ID: <d>" \
  -H "Content-Type: application/json" \
  -d '{"target_user_id":"<target>","role_code":"store_admin","store_id":3}'
```

演示种子 `scripts/navicat_seed_store_stats.sql` 会给 `demo@example.com` 写门店、成员、统计，并授 `platform_admin`。

核对 SQL：

```sql
SELECT store_id, user_id, store_name, role, days_joined, employee_count, store_days, total_customers
FROM wys_user_store_stats
ORDER BY store_id;

SELECT email, current_store_id FROM users WHERE email = 'demo@example.com';
```

本地建表 + 种子（Navicat / psql）：

```bash
# 1) 若未 AutoMigrate：执行 scripts/navicat_local_schema.sql
# 2) 演示账号：scripts/navicat_seed_demo_data.sql（demo@example.com / 123456）
# 3) 门店统计：scripts/navicat_seed_store_stats.sql
```

> `/api/v1/user/profile` 使用自建 JWT 中间件，与业务登录 token **不兼容**。Flutter 请走 `/api/v1/profiles/me` 或 `/api/v1/me/profile`。

---

## 6. API 一览

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| GET | `/health` | 否 | 健康检查 |
| POST | `/api/v1/user/register` | 否 | 邮箱注册（Supabase Auth） |
| POST | `/api/v1/user/login` | 否 | 邮箱登录，返回 access token + refresh_token + session_id |
| POST | `/api/v1/user/refresh` | 否 | 用 refresh_token 换取新 access token |
| POST | `/api/v1/user/logout` | Supabase token + session | 主动退出，删除 Redis session |
| GET | `/api/v1/user/list` | Bearer 自建 JWT | 本地用户列表（遗留，`?page=1&size=20`） |
| GET | `/api/v1/user/profile` | Bearer 自建 JWT | 本地用户详情（遗留，与 Supabase 登录不兼容） |

### 6.1 Supabase 接口（需 Supabase JWT）

认证方式：**Supabase access token**（`Authorization: Bearer <token>`）

#### Flutter 兼容接口（snake_case 直出 JSON）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/me/profile` | 读取 profiles（含 `stats`；可选 `?store_id=`） |
| PATCH | `/api/v1/me/profile` | 更新 display_name / avatar_url |
| GET | `/api/v1/transactions` | 列表 `?type=&limit=&offset=` → `{ "items": [...] }` |
| POST | `/api/v1/transactions` | 创建收支 |
| GET | `/api/v1/transactions/:id` | 单条详情 |
| PUT | `/api/v1/transactions/:id` | 更新 |
| DELETE | `/api/v1/transactions/:id` | 删除（204） |

#### 统一管理接口（统一响应 `{ code, message, data }`，camelCase 字段）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/profiles/me` | 获取当前用户资料（含 `stats`；可选 `?store_id=` 正整数） |
| PATCH | `/api/v1/profiles/me` | 更新资料 |
| POST | `/api/v1/profiles/me/store` | 切换当前店铺，body `{"store_id":N}`，返回该店 stats |
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

> 在 **`my_go_study/`** 目录执行（先 `cd my_go_study`）。

| 命令 | 说明 |
|------|------|
| `make run` | `go -C app run ./cmd/api` |
| `make run-worker` | `go -C app run ./cmd/worker`（RabbitMQ 消费，dev 默认需与 API 同启） |
| `make build` | 编译到 `bin/api` 与 `bin/worker` |
| `make test` | 运行全部单元测试 |
| `make tidy` | `go mod tidy` |
| `make air` | Air 热加载开发 |
| `make migrate-up` | 应用 `migrations/` 全部未执行迁移（DSN 取自 `.env` 的 `DATABASE_*`） |
| `make migrate-down` | 回滚最近一次迁移 |
| `make migrate-status` | 库内版本 + 目录最新版本 |
| `make migrate-baseline` | 历史 AutoMigrate 库首次接入迁移链（`force` 到最新，不重放 SQL） |
| `make migrate-force VERSION=<n>` | 迁移中断留下 `dirty` 时重置版本号 |
| `make docker-up` | 构建并启动全部容器（app + worker + postgres + redis；注入 SUPABASE_*） |
| `make docker-down` | 停止并移除上述容器 |
| `make lan-up` / `lan-down` | 局域网全栈（需 `.env.lan`；改代码需重建镜像） |
| `make deps-up` | Homebrew 安装并启动 PostgreSQL + Redis（无需 Docker） |
| `make docker-down` | 停止并移除容器 |
| `make clean` | 删除 `bin/`、`tmp/`、日志 |
| `make test-transactions` | transactions CRUD 联调（`SUPABASE_SERVICE_ROLE_KEY` 放 `.env.local`） |
| `make test-realtime` | WebSocket 联调（ws-ticket / sync / push + WS auth） |
| `make test-queue-push` | 异步 Push 联调（需 `make run` + `make run-worker`） |
| `make trigger-hourly-notify` | 手动入队定时广播任务 |
| `make test-scheduled-notify` | 定时通知联调（登录 + 触发 + sync） |
| `make test-auth-refresh-logout` | refresh + logout 联调 |
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

`configs/config.yaml` 里 `jwt.secret` 刻意留空（不再有兜底默认密钥），值只能由 `JWT_SECRET` 注入。
本机用 `./scripts/load-env.sh <cmd>` 或 `make run`（会自动 source `.env`）；IDE 直跑请把 §9「IDE 环境变量」配全。

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

# 3. 首次/落后时把 schema 推平（本机 APP_ENV=dev 有 AutoMigrate 逃生门，可跳过）
make migrate-up

# 4. 启动服务
make run
```

> 平时**不要**跑 `make tidy` / 逐模块 `go mod tidy`：缺依赖时它会把 viper / zap / gorm 拉到最新版
> （AGENTS.md 约束 9）。要加依赖就在对应模块 `go.mod` 显式写 pinned 版本。

---

## 12. 生产部署提示

1. 设置 `APP_ENV=prod`，使用 `configs/config.prod.yaml`
2. **务必修改** `JWT_SECRET` 和数据库密码（见 §13，prod 由 `config.Load` 硬校验）
3. 数据库 `sslmode` 建议设为 `require`
4. **发布流程先跑迁移，再滚更服务**：`make migrate-up`（K8s 用 Job / compose 用一次性 `migrate` 服务）。
   进程启动期不再建表（`APP_ENV=prod|lan` 代码强制 `database.auto_migrate=false`），
   库版本落后 / `dirty` / 空库一律**拒绝启动**——`/health` 连不上即是信号，见 §14
5. 使用 `make build` 编译二进制，或通过 `docker/Dockerfile` 多阶段构建镜像部署
   （镜像内含 `/app/migrations` + `/app/migrate`，API 启动要做版本比对）
6. `.env` 须入库；`SUPABASE_SERVICE_ROLE_KEY` 仅放 `.env.local`；推送前运行 `make check-secrets`

---

## 13. 密钥

**本地可弱、部署必改**：`.env` 与 `.env.lan` 是**有意入库**的文件（历史 commit `afbc29f`），里面
`JWT_SECRET=change-me-in-production`、`DATABASE_PASSWORD=postgres` 这类占位值只允许出现在本机联调。
任何共享 / 生产环境复用同一组值都等于没有密钥——这是本项目此前最大的未闭环风险（`docs/project-detail.md` H11）。

三道闸门（已落地）：

| 层 | 位置 | 行为 |
|----|------|------|
| compose | `docker/docker-compose.yml`、`docker/docker-compose.lan.yml` | `JWT_SECRET: ${JWT_SECRET:?...}`——**没配就直接拒绝**，不再兜底默认值 |
| 配置加载 | `common/config/secrets.go`（`config.Load` 内） | `APP_ENV=prod/production` 硬校验：JWT 非占位值 / 长度 ≥32 / 无空白 / 非全同字符；`DATABASE_PASSWORD` 非默认且 ≥8；`SUPABASE_SERVICE_ROLE_KEY` 不得来自入库 `.env` |
| 启动日志 | `app/cmd/api/main.go` | lab 环境放行但打印 `密钥体检：存在联调可用、生产必改的配置 {findings:[…]}` |
| 推送前扫描 | `scripts/check-secrets.sh` | 三级判定，见下 |

`configs/config.yaml` 的 `jwt.secret` 现为**空串**：密钥只能从 `JWT_SECRET` 注入，缺失即 `jwt.secret 不能为空` 启动失败。

### `make check-secrets` 判定口径

| 判定 | 触发条件 | 结果 |
|------|----------|------|
| ❌ 硬失败 | `sb_secret_*` / `SUPABASE_SERVICE_ROLE_KEY=sb_`；非 lab 文件里的占位默认值；secret 类变量含空白、疑似高熵真值（≥20 位且 ≥3 类字符）、或长度 <16 | 退出 1，拦推送 |
| ⚠️ 提示 | 有意入库的 `.env` 里的占位弱值 | 仅提示，退出 0（否则每次推送都要改 `.env`） |
| `CHECK_SECRETS_STRICT=1` | 把上面「提示」升级为失败 | 部署前 / CI 用 |

`*.example` 模板与设计上公开的标识位（`SUPABASE_ANON_KEY`、各平台 `*_APP_ID` / `*_APP_KEY`）不参与弱值与高熵判定。
也可显式传文件自检：`./scripts/check-secrets.sh /tmp/我新写的.env`。

### 轮换

改 `JWT_SECRET` 会使**已签发的 access token 全部失效**（Flutter 端会走一次 `/api/v1/user/refresh` 或重新登录）；
`DATABASE_PASSWORD` 需与 Postgres 侧同步改。两者都要双端确认后一起发版，不在联调中途随手换。

---

## 14. 数据库 schema 闸门

一句话：**进程启动不再建表，`migrations/` 是 schema 的单一真相，版本落后就拒绝启动。**

| 配置（`configs/config.yaml` → `database.*`） | env | 默认 | prod/lan |
|-------------------------------------------|-----|------|----------|
| `auto_migrate` | `DATABASE_AUTO_MIGRATE` | `false` | 代码强制 `false`（`common/config.NoDDLAppEnv`） |
| `schema_version_check` | `DATABASE_SCHEMA_VERSION_CHECK` | `true` | 不可关（关掉的后果是老 schema 上跑新代码） |
| `migrations_dir` | `DATABASE_MIGRATIONS_DIR` | `migrations` | 镜像里是 `/app/migrations` |

唯一入口 `app/internal/schemaboot.Boot()`（API 与 Worker 共用）。启动即失败的五种情况与处置：

| 日志 | 含义 | 处置 |
|------|------|------|
| `schema 版本落后：库内 X，迁移目录最高 Y；缺失 N 个迁移：…` | 迁移没跑完 | `make migrate-up` |
| `上一次迁移中断（schema_migrations.dirty=true…）` | 某条迁移跑到一半失败 | 修好 SQL → `make migrate-force VERSION=<n>` |
| `数据库未初始化：缺少 schema_migrations 且无业务表` | 空库直接上正式环境 | 发布流程先 `make migrate-up`（compose 由 `migrate` 服务代劳） |
| `migrations 存在重复版本号` | 两个文件同号，后一个会被永久跳过 | 把后一个改成更大的唯一版本号 |
| `库由历史 AutoMigrate 建立，尚无 schema_migrations 版本记录`（Warn，放行） | 旧库首次接入 | 确认 schema 已最新后 `make migrate-baseline` |

本机想跳过迁移：`APP_ENV=dev`（`configs/config.dev.yaml` 里 `auto_migrate: true`）——空库也能 `make run`，
但**一旦库里出现过 `schema_migrations`，本机同样受版本校验约束**。

compose（`make lan-up` / `make docker-up`）里多了一个一次性 `migrate` 服务，`app`/`worker` 以
`depends_on: service_completed_successfully` 等它；判库与自动基线见
[`scripts/migrate-entrypoint.sh`](../scripts/migrate-entrypoint.sh)（想禁掉自动基线：`MIGRATE_BASELINE=off`）。

加表 / 加索引 / 回滚的完整步骤：[`docs/migration-workflow.md`](./migration-workflow.md)。

