# 两端联调启动配置手册（Go BFF + Flutter）

> **目标**：本机 Mac 作为局域网后端（本地 Auth + Postgres + Redis），iOS / Android 真机（或模拟器）通过 Flutter 联调。  
> **仓库**：Go `my_go_study` · Flutter `my_ai_project`（独立 Git）。  
> **相关专题**：[lan-backend-host.md](./lan-backend-host.md) · [local-auth-postgres.md](./local-auth-postgres.md) · Flutter [BACKEND_INTEGRATION.md](../../my_ai_project/docs/BACKEND_INTEGRATION.md)

---

## 0. 一分钟总览

```text
Flutter 真机 / 模拟器
  │  http://<LAN_IP>:8080
  │  ws://<LAN_IP>:8080/...（ticket 返回，依赖 REALTIME_PUBLIC_WS_HOST）
  ▼
my_go_study（APP_ENV=lan, AUTH_PROVIDER=local）
  ├── API :8080
  ├── Worker（Compose 内 / 可选本机）
  ├── Postgres（业务 + auth_users）
  └── Redis（session / realtime / queue）
```

| 项 | 约定 |
|----|------|
| 认证 | **本地** `auth.provider=local`（不再默认连 Supabase Cloud） |
| 端口 | API **8080**；Postgres/Redis 仅本机 `127.0.0.1` |
| 两端 IP | Go `REALTIME_PUBLIC_WS_HOST` **=** Flutter `BACKEND_HOST`（同一局域网 IP） |
| 传输 | 明文 `http` / `ws`（家庭 Wi‑Fi）；Android 已允许 cleartext |
| 导入账号 | 若跑过 `import-supabase`，用临时密码登录（见 §6） |

查本机局域网 IP（示例）：

```bash
ipconfig getifaddr en0
# 例如 192.168.0.102
```

下文用 `<LAN_IP>` 表示该地址。

---

## 1. 首次准备（每台 Mac 做一次）

### 1.1 固定局域网 IP（推荐）

路由器 DHCP 预留，或 macOS「网络 → 手动」固定 IPv4，避免 IP 漂移导致 Flutter / WS 双处失效。

### 1.2 防火墙

联调约定：可暂时关闭 macOS 防火墙；用完建议打开。

### 1.3 Go 仓库

```bash
cd /path/to/my_go_study

# 依赖（二选一）
# A) Homebrew 本机 PG + Redis（后面用 make run）
make deps-up

# B) 仅 Docker（后面用 make lan-up）——需已装 Docker Desktop

cp .env.lan.example .env.lan
# 编辑 .env.lan：见 §2.1
```

可选（从 Cloud 拉过数据时已做过可跳过）：

```bash
# 需 configs/supabase.env + .env.local(service_role) + 本地 Postgres
make import-supabase DEFAULT_PASSWORD='你的临时密码'
```

### 1.4 Flutter 仓库

```bash
cd /path/to/my_ai_project

cp .env.lan.example .env.lan
# 编辑 .env.lan：见 §3.1

flutter pub get
```

---

## 2. Go 后端配置与启动

### 2.1 文件：`.env.lan`（不入库）

模板：`.env.lan.example`

| 变量 | 必填 | 说明 |
|------|------|------|
| `APP_ENV` | 是 | 固定 `lan` → 合并 `configs/config.lan.yaml` |
| `AUTH_PROVIDER` | 是 | 固定 `local` |
| `SERVER_PORT` | 建议 | 默认 `8080` |
| `REALTIME_PUBLIC_WS_HOST` | **是** | `<LAN_IP>`，ticket 里 WS 主机 |
| `JWT_SECRET` | 是 | 本地签发 access token 密钥 |
| `AUTH_SESSION_WHITELIST_EMAILS` | 建议 | 逗号分隔；双端真机同账号不互踢 |
| `AUTH_SESSION_WHITELIST_USER_IDS` | 可选 | UUID 白名单 |
| `AUTH_DEV_TEST_PHONE` / `OTP` / `PASSWORD` | 可选 | 测试手机号 OTP（`config.lan` 为 `debug` 时可用） |

示例：

```bash
APP_ENV=lan
AUTH_PROVIDER=local
SERVER_PORT=8080
REALTIME_PUBLIC_WS_HOST=192.168.0.102
AUTH_SESSION_WHITELIST_EMAILS=you@example.com
JWT_SECRET=change-me-in-lan
```

### 2.2 文件：`configs/config.lan.yaml`（入库）

当前要点：

- `auth.provider: local`
- `server.mode: debug`（便于测试 OTP）
- `queue.enabled: true`

真实 IP **不要**写进此文件。

### 2.3 启动方式 A — Compose 全栈（推荐真机）

```bash
cd my_go_study
make lan-up      # postgres + redis + app + worker
# 停止
make lan-down
```

等价底层：

```bash
./scripts/lan-compose.sh up -d --build
```

改 Go 代码后需重新 `make lan-up`（重建镜像）。

### 2.4 启动方式 B — 本机进程（依赖已 `make deps-up`）

```bash
cd my_go_study
# 将 .env.lan 中的变量 export，或：
set -a && source .env.lan && source configs/supabase.env 2>/dev/null; set +a
# local 模式可不依赖 supabase.env

APP_ENV=lan AUTH_PROVIDER=local \
  REALTIME_PUBLIC_WS_HOST=<LAN_IP> \
  ./scripts/load-env.sh go run ./cmd/api

# 另开终端（需要异步推送 / 定时任务时）
APP_ENV=lan ./scripts/load-env.sh go run ./cmd/worker
```

注意：`load-env.sh` 默认加载 `.env` / `.env.local` / `supabase.env`；请保证 `APP_ENV=lan` 与 `AUTH_PROVIDER=local` 最终生效（`.env.lan` 需自行 `source`，或写进环境）。

更省事：把关键变量写进 `.env` 仅本机使用，或始终用 **方式 A**。

### 2.5 后端验收

```bash
curl -s http://127.0.0.1:8080/health
curl -s http://<LAN_IP>:8080/health
# 期望：{"status":"ok"}
```

手机浏览器访问 `http://<LAN_IP>:8080/health` 也应通。

| 检查 | 期望 |
|------|------|
| 日志含 `Auth provider=local` | 本地认证已启用 |
| 5432 / 6379 从手机扫 | 默认不可达（仅绑 127.0.0.1） |
| 注册/登录 | `POST /api/v1/user/register` · `/login` |

---

## 3. Flutter 客户端配置与启动

### 3.1 文件：`.env.lan`（不入库）

模板：`.env.lan.example`

| 变量 | 必填 | 说明 |
|------|------|------|
| `USE_MOCK_AUTH` | 是 | 必须 `false`（走真实 BFF） |
| `BACKEND_HOST` | **是** | 与 Go `REALTIME_PUBLIC_WS_HOST` **相同**的 `<LAN_IP>` |

示例：

```bash
USE_MOCK_AUTH=false
BACKEND_HOST=192.168.0.102
```

原理：`EnvConfig` 默认 `http://127.0.0.1:8080`；`BackendHttpConfig` 在存在 `BACKEND_HOST` 时把 localhost 换成局域网 IP（真机必需）。Realtime 以服务端 ticket 的 `wsUrl` 为准。

### 3.2 启动命令

```bash
cd my_ai_project

# 真机（推荐）
flutter run --dart-define-from-file=.env.lan

# 指定设备
flutter devices
flutter run -d <device_id> --dart-define-from-file=.env.lan
```

### 3.3 模拟器对照（可不改 BACKEND_HOST）

| 环境 | API 主机 |
|------|----------|
| iOS 模拟器 | `127.0.0.1`（本机 Go） |
| Android 模拟器 | 自动映射 `10.0.2.2` |
| 真机 | **必须** `BACKEND_HOST=<LAN_IP>` |

模拟器联调本机 `make run` 时可用仓库根 `.env`（仅 `USE_MOCK_AUTH=false`）。

### 3.4 平台注意

| 平台 | 注意 |
|------|------|
| Android | 主 Manifest 已 `usesCleartextTraffic=true` |
| iOS | 首次访问本地网可能弹「本地网络」权限 → 允许 |
| 双端同账号 | Go `.env.lan` 配置白名单邮箱，否则单设备互踢 |

### 3.5 客户端验收

1. 设置页确认环境为 **测试**（`AppEnv.test` → baseUrl 指向 Go）  
2. 邮箱登录（导入用户用临时密码；或新注册）  
3. 业务列表（transactions）有数据或可新建  
4. Realtime 调试：能连上且非 `127.0.0.1`（真机）

---

## 4. 推荐日常流程（清单）

```text
□ 1. 确认 <LAN_IP>（ipconfig getifaddr en0）
□ 2. 关闭防火墙（联调窗口）
□ 3. Go：.env.lan 中 REALTIME_PUBLIC_WS_HOST=<LAN_IP>
□ 4. Flutter：.env.lan 中 BACKEND_HOST=<LAN_IP>
□ 5. make lan-up（或本机 make deps-up + API/Worker）
□ 6. curl http://<LAN_IP>:8080/health
□ 7. flutter run --dart-define-from-file=.env.lan
□ 8. 登录 → 业务 → Realtime
□ 9. 结束：make lan-down；酌情开防火墙
```

---

## 5. 两套模式对照（勿混）

| 场景 | Go | Flutter |
|------|-----|---------|
| **局域网真机（本文）** | `APP_ENV=lan` + `AUTH_PROVIDER=local` + `make lan-up` | `.env.lan` + `BACKEND_HOST` |
| **本机 / 模拟器 + Cloud Auth** | `APP_ENV=dev` + supabase.env | `.env` 仅 `USE_MOCK_AUTH=false`，无 BACKEND_HOST |
| **Mock 登录** | 可不启 Go | `USE_MOCK_AUTH=true` |

切回 Cloud：`AUTH_PROVIDER=supabase` 并配置 `SUPABASE_*`（见 [local-auth-postgres.md](./local-auth-postgres.md)）。

---

## 6. 导入账号登录

若执行过 Cloud 导入：

```bash
make import-supabase DEFAULT_PASSWORD='…'
```

- 邮箱：Cloud 原邮箱  
- 密码：**导入时的 DEFAULT_PASSWORD**（不是 Cloud 原密码）  
- 白名单：把常用邮箱写入 Go `.env.lan` 的 `AUTH_SESSION_WHITELIST_EMAILS`

---

## 7. 排障速查

| 现象 | 处理 |
|------|------|
| 真机 health 不通 | 同 Wi‑Fi？IP 变了？防火墙？访客网络隔离？ |
| HTTP 通、登录后 WS 失败 | Go `REALTIME_PUBLIC_WS_HOST` 是否仍为 127.0.0.1 |
| Android 立刻网络错误 | cleartext；确认用了 `.env.lan` 且 Hot Restart / 重跑 |
| 第二台设备 401 | 单设备互踢 → 加白名单 |
| 登录密码错误（导入用户） | 用导入临时密码，不是 Cloud 密码 |
| `lan-up` 失败缺 IP | `.env.lan` 未设或仍为 `YOUR_LAN_IP` |
| 改 Go 不生效 | Compose 需 `--build` / 再执行 `make lan-up` |
| AutoMigrate 报 bigint→uuid | 已有自动重命名 `transactions_legacy_uint`；或清 Docker volume |

---

## 8. 配置文件清单

### Go（`my_go_study`）

| 文件 | 入库 | 用途 |
|------|------|------|
| `.env.lan.example` | 是 | 模板 |
| `.env.lan` | **否** | 真实 IP / 白名单 / JWT |
| `configs/config.lan.yaml` | 是 | lan 覆盖项 |
| `docker/docker-compose.yml` | 是 | 基础栈 |
| `docker/docker-compose.lan.yml` | 是 | lan overlay（`AUTH_PROVIDER=local`） |
| `.env.local` | 否 | 仅 Cloud 导入需要 service_role |
| `configs/supabase.env` | 是 | 仅 Cloud / 导入时需要 |

### Flutter（`my_ai_project`）

| 文件 | 入库 | 用途 |
|------|------|------|
| `.env.lan.example` | 是 | 模板 |
| `.env.lan` | **否** | `BACKEND_HOST` + `USE_MOCK_AUTH=false` |
| `.env` | 是 | 团队默认（通常只有 Mock 开关） |

---

## 9. 相关命令速查

```bash
# Go
make deps-up
make lan-up / make lan-down
make import-supabase-dry
make import-supabase DEFAULT_PASSWORD='…'
curl http://127.0.0.1:8080/health

# Flutter
flutter run --dart-define-from-file=.env.lan
```
