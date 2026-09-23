# my_go_study 文档总览（架构 + 完整使用）

> 本页是仓库文档的**统一入口**。架构与使用说明**已存在**（见下表），这里再汇总成一份可照着跑的完整手册。  
> 更新日期：2026-08-03

---

## 0. 已有文档一览（是否存在？）

| 类型 | 文档 | 状态 |
|------|------|------|
| **架构学习** | [architecture-learning-guide.md](./architecture-learning-guide.md) | ✅ 已有（分层、优缺点、三条竖切时序） |
| **启动与环境** | [startup-guide.md](./startup-guide.md) | ✅ 已有（Docker / 本地 / 配置 / 验证） |
| **Agent / 工程约束** | [../AGENTS.md](../AGENTS.md) | ✅ 已有 |
| Supabase | [supabase-integration.md](./supabase-integration.md) | ✅ |
| 认证导读 | [auth-beginner-walkthrough.md](./auth-beginner-walkthrough.md) | ✅ |
| Transactions | [transactions-beginner-walkthrough.md](./transactions-beginner-walkthrough.md) | ✅ |
| Realtime | [realtime-beginner-walkthrough.md](./realtime-beginner-walkthrough.md) · [realtime-websocket.md](./realtime-websocket.md) | ✅ |
| **SSE 流式 / AI 小石头** | [sse-streaming.md](./sse-streaming.md) · Flutter [sse-streaming-design.md](../../my_ai_project/docs/sse-streaming-design.md) · SPEC `.scratch/ai-little-stone/SPEC.md` · ADR 0005 | ✅ Accepted |
| **Gin HTTP vs gRPC** | [grpc/gin-http-vs-grpc.md](./grpc/gin-http-vs-grpc.md) | ✅ 飞书 tool / grpc |
| **数据分析 gRPC 联调** | [grpc/analytics-flutter-trial.md](./grpc/analytics-flutter-trial.md) | ✅ 列表/详情试验 |
| **Docker 联调运维（A/B/C）** | [docker/lan-ops-abc-beginner.md](./docker/lan-ops-abc-beginner.md) | ✅ Flutter 对齐 · 改码生效 · 排障；飞书 tool / Docker 相关文档纪要 |
| 消息队列 | [message-queue.md](./message-queue.md) | ✅ |
| 飞书同步 | [FEISHU_SYNC.md](./FEISHU_SYNC.md) | ✅ |
| **本机局域网后端** | [lan-backend-host.md](./lan-backend-host.md) | ✅ `make lan-up`（现为本地 Auth） |
| **两端联调启动手册** | [dual-end-lan-startup.md](./dual-end-lan-startup.md) | ✅ Go + Flutter 配置与启动清单 |
| **iOS 真机 LAN 调试记录** | [ios-lan-device-debug-2026-09-16.md](./ios-lan-device-debug-2026-09-16.md) | ✅ 127.0.0.1 / BACKEND_HOST 未注入根因与验收 |
| **本地 Auth + Postgres** | [jpush-integration.md](./jpush-integration.md) | 极光推送前后端 + 遗留清单 |
| [local-auth-postgres.md](./local-auth-postgres.md) | ✅ `auth.provider=local`；`make import-supabase` |
| **华为账号一键登录** | [huawei-account-login.md](./huawei-account-login.md) | HarmonyOS Account Kit → `POST /user/huawei/login`；AGC scope 待申请 |
| ADR | [adr/](./adr/) | ✅ 含 LAN / local Auth / 钱包退款 ADR-0015 |
| **人民币钱包** | [cash-wallet-api.md](./cash-wallet-api.md) · [acceptance 2026-09-23](./acceptance-records/2026-09-23-cash-wallet.md) · SPEC `.scratch/cash-wallet/` | ✅ 余额/绑卡/充值/商城渠道 6 |

**结论：** 架构文档与使用文档都已存在，但此前分散在多篇里。下文是把「怎么装、怎么跑、怎么调、有哪些 API、架构是什么」收成一份的完整使用手册；细节仍以专题文档为准。

---

## 1. 项目是什么

面向 Flutter 客户端（`my_ai_project`）的 **Gin BFF**，不是全量业务库单体。

| 项 | 说明 |
|----|------|
| 框架 | Gin + Clean Architecture |
| ORM（本地库） | GORM → 本地 PostgreSQL（遗留 `users` 等） |
| 业务数据 | Supabase PostgREST + RLS |
| 缓存 / Session / Realtime | Redis |
| 异步 | Asynq（`cmd/worker`） |
| 默认端口 | `8080` |
| 模块路径 | `github.com/stvenfor/my_go_study` |

```text
Flutter ──HTTP/WS──▶ Gin BFF(:8080)
                       ├─ Supabase Auth / PostgREST
                       ├─ Redis（session / ticket / PubSub）
                       ├─ 本地 Postgres（GORM，遗留）
                       └─ Asynq Worker（可选）
```

更细的架构、优缺点、时序图 → [architecture-learning-guide.md](./architecture-learning-guide.md)

---

## 2. 架构速览

### 2.1 分层

```text
cmd/api、cmd/worker          进程入口（手动 DI）
internal/delivery/           HTTP / WS / 中间件 / DTO
internal/usecase/            业务用例
internal/domain/             实体 + repository 接口
internal/repository/         postgres | redis | supabase
pkg/                         config、jwt、auth、queue、logger…
```

依赖方向：**delivery → usecase → domain 接口 → repository → 外部系统**

新增能力刀路：`entity → repository → usecase → controller/handler → router`

### 2.2 两套认证（勿混用）

| 体系 | 中间件 | 用途 |
|------|--------|------|
| Supabase JWT + Redis session | `SupabaseSessionAuth` | Flutter 业务（transactions / realtime / profile） |
| Go 自建 JWT | `Auth` | 遗留 `/api/v1/user/list` 等 |

业务请求头：`Authorization: Bearer <token>` + `X-Session-ID` + `X-Device-ID`

### 2.3 双响应信封

| 风格 | 形态 | 典型接口 |
|------|------|----------|
| ResultModel | `{ code, message, data, timestamp }` | 登录 / `/transactions/manage` |
| 直出 JSON | `{ items: [...] }` 或 `{ error }` | Flutter `/api/v1/transactions` |

---

## 3. 环境准备与配置

> **所有 `go` / `make` 必须在仓库根目录 `my_go_study/` 执行**，不要在父目录 `my_code_study/` 执行。

### 3.1 依赖

| 工具 | 用途 |
|------|------|
| Go 1.21+ | 编译运行 |
| PostgreSQL + Redis | 本地库与 session / 队列 |
| Docker（可选） | 一键起依赖或整栈 |
| Supabase 项目 | Auth + 业务表 + RLS |

### 3.2 配置文件

```bash
cp .env.example .env
cp .env.local.example .env.local   # 填写 SUPABASE_SERVICE_ROLE_KEY（不入库）
# configs/supabase.env 通常已入库（URL + anon key）
```

| 文件 | 入库 | 内容 |
|------|------|------|
| `configs/config.yaml` + `config.{env}.yaml` | 是 | 端口、DB、Redis、queue… |
| `configs/supabase.env` | 是 | `SUPABASE_URL`、`SUPABASE_ANON_KEY` |
| `.env` | 是 | 运行时覆盖 |
| `.env.local` | **否** | `service_role` 等密钥 |

加载顺序：`config.yaml` → `config.{APP_ENV}.yaml` → 环境变量 → supabase.env / `.env.local`。

推送前：`make check-secrets`。

完整配置说明 → [startup-guide.md](./startup-guide.md) §3

---

## 4. 启动（完整使用）

### 4.1 推荐：本机依赖 + 本地 Go

```bash
cd /path/to/my_go_study
make deps-up      # 首次：PostgreSQL + Redis
make run          # API :8080

# 另开终端（dev 若 queue.enabled=true）
make run-worker
```

### 4.2 Docker 整栈

```bash
make docker-up      # postgres + redis + app
make docker-down
```

### 4.3 与 Flutter 联调

```bash
# 终端 1：API
make run
# 终端 2：Worker（需要异步 Push / 定时通知时）
make run-worker
# 终端 3：Flutter
cd ../../my_ai_project
flutter run --dart-define-from-file=.env   # USE_MOCK_AUTH=false
```

### 4.4 健康检查

```bash
curl http://127.0.0.1:8080/health
# {"status":"ok"}
```

---

## 5. API 一览（当前路由）

前缀默认：`http://127.0.0.1:8080`

### 5.1 公共

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| GET | `/health` | 无 | 健康检查 |
| GET | `/realtime/v1/connect` | 无（首帧 ticket） | WebSocket |

### 5.2 用户 / 认证 `/api/v1/user`

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| POST | `/register` | 无 | 注册（Supabase） |
| POST | `/login` | 无 | 登录；需 `device_id` + `platform`（ios/android） |
| POST | `/refresh` | 无 | refresh_token 续期 |
| POST | `/logout` | SupabaseSession | 删 Redis session + 撤销 refresh |
| POST | `/phone/otp/send` | 无 | 发 OTP |
| POST | `/phone/otp/verify` | 无 | 验 OTP 登录 |
| GET | `/list` | Go JWT | 遗留 |
| GET | `/profile` | Go JWT | 遗留 |

登录示例：

```bash
curl -s http://127.0.0.1:8080/api/v1/user/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"你的邮箱","password":"...","device_id":"dev-1","platform":"ios"}'
```

返回 `data.token` / `data.refresh_token` / `data.session_id`。

### 5.3 Transactions（需 SupabaseSession）

| 方法 | 路径 | 响应风格 |
|------|------|----------|
| GET/POST/GET:id/PUT/DELETE | `/api/v1/transactions` | Flutter：`{ items }` 等 |
| 同上 | `/api/v1/transactions/manage` | ResultModel + page/size |

```bash
curl -s 'http://127.0.0.1:8080/api/v1/transactions?limit=10' \
  -H "Authorization: Bearer <token>" \
  -H "X-Session-ID: <session_id>" \
  -H "X-Device-ID: dev-1"
```

### 5.4 Profile（需 SupabaseSession）

| 方法 | 路径 |
|------|------|
| GET/PATCH | `/api/v1/profiles/me` |
| GET/PATCH | `/api/v1/me/profile`（Flutter 兼容） |

### 5.5 Realtime（需 SupabaseSession，除 WS）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/realtime/ws-ticket` | 换一次性 ticket |
| POST | `/api/v1/realtime/sync` | 按 seq 补拉 |
| POST | `/api/v1/realtime/push` | 推送测试（可同步/异步） |
| GET | `/realtime/v1/connect` | WS；首帧 `auth` + ticket |

协议细节 → [realtime-websocket.md](./realtime-websocket.md)

---

## 6. Makefile 常用命令

| 命令 | 作用 |
|------|------|
| `make run` / `make run-worker` | 启 API / Worker |
| `make test` | 单元测试 |
| `make deps-up` | 起本机 Postgres + Redis |
| `make docker-up` / `docker-down` | Docker 整栈 |
| `make test-transactions` | 收支联调 |
| `make test-realtime` | Realtime 联调 |
| `make test-queue-push` | 异步 Push（需 worker） |
| `make test-single-device-login` | 单设备登录 |
| `make test-phone-otp-login` | 手机 OTP（dev：`13400000000` / `123456`） |
| `make test-auth-refresh-logout` | 续期 / 退出 |
| `make trigger-hourly-notify` / `test-scheduled-notify` | 定时通知 |
| `make check-secrets` | 密钥入库检查 |
| `make check-rls` | transactions RLS 脚本 |

---

## 7. 典型使用流程（端到端）

```text
1. make deps-up && make run（+ 可选 make run-worker）
2. POST /api/v1/user/login → 保存 token / refresh_token / session_id
3. 业务 API 带 Bearer + X-Session-ID + X-Device-ID
4. Realtime：POST ws-ticket → WS connect → 首帧 auth
5. access 将过期：POST /refresh（可带 device/session 续写）
6. 退出：POST /logout（需 session 三件套）
```

Flutter 侧契约见：`my_ai_project/docs/BACKEND_INTEGRATION.md`、`USAGE_GUIDE.md`。

---

## 8. 开发约定（改代码前必看）

1. 在本仓库根目录执行 `go` / `make`
2. Supabase 业务：PostgREST 必须 `WithUserToken`，禁止 Admin 绕过 RLS
3. transactions 必须 `.Eq("user_id", userID)` + RLS 迁移
4. `SUPABASE_SERVICE_ROLE_KEY` 仅 `.env.local`
5. 两套 Token 勿混用；Flutter 主路径只用 Supabase token
6. 多角色任务可走 `.harness/`（conductor → planner → executor → reviewer）

完整 Agent 规范 → [../AGENTS.md](../AGENTS.md)

---

## 9. 推荐阅读顺序

1. **本文**（总览 + 用法）
2. [startup-guide.md](./startup-guide.md)（环境踩坑）
3. [architecture-learning-guide.md](./architecture-learning-guide.md)（架构深读）
4. [auth-beginner-walkthrough.md](./auth-beginner-walkthrough.md)
5. [transactions-beginner-walkthrough.md](./transactions-beginner-walkthrough.md)
6. [realtime-beginner-walkthrough.md](./realtime-beginner-walkthrough.md)
7. [message-queue.md](./message-queue.md)
8. [supabase-integration.md](./supabase-integration.md)

---

## 10. 飞书同步（可选）

```bash
# 知识库「项目 docs」
./scripts/sync_docs_to_feishu.sh sync

# 知识库「code」（架构学习等精选）
./scripts/sync_docs_to_feishu.sh sync --config docs/feishu-sync-code.config.yaml
```

说明 → [FEISHU_SYNC.md](./FEISHU_SYNC.md)
