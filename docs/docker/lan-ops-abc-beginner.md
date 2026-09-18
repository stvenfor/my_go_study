# Docker 联调运维纪要（Flutter 对齐 · 改码生效 · 排障）

> 面向小白：把「Docker 跑起来之后怎么联调」拆成 **A / B / C** 三篇合订。  
> 仓库真相源：本文；飞书知识库 **tool → Docker 相关文档纪要**。  
> 相关：[`lan-backend-host.md`](../lan-backend-host.md) · [`dual-end-lan-startup.md`](../dual-end-lan-startup.md) · [`grpc/analytics-flutter-trial.md`](../grpc/analytics-flutter-trial.md)

<callout emoji="📌" background-color="light-blue">
<text color="green">**绿色加粗**</text> = 正确做法 / 记住就能用  
<text color="red">**红色加粗**</text> = 易错点 / 禁止事项 / 优先排查
</callout>

---

## 0. 先锁死的 unconditional 事实

1. <text color="green">**程序必须跑在某台「电脑」上**</text>：要么是你的 Mac（`make run`），要么是 Docker **容器**（`make lan-up`）。
2. <text color="green">**Compose 一次起多台小电脑**</text>：`postgres` + `redis` + `app` + `worker`。
3. <text color="red">**容器跑的是镜像里烤好的二进制，不是你正在编辑的 `.go` 文件**</text>。改 Go → 必须 **`--build`**（`make lan-up` 已带）。
4. <text color="green">**HTTP `:8080` 与 gRPC `:9090` 是两扇门**</text>；Flutter **host 相同、端口不同**。

```text
真机 / 模拟器
  │  HTTP/WS  → LAN_IP:8080
  │  gRPC     → LAN_IP:9090
  ▼
Mac 宿主机（端口映射）
  ▼
Docker：my_go_study_app（同进程听 8080+9090）
         + worker + postgres + redis
```

| 入口 | 含义 |
|------|------|
| <text color="green">**`make lan-up`**</text> | 基础 compose + LAN overlay；真机联调推荐 |
| `make docker-up` | 仅基础 compose；偏本机 dev |
| <text color="red">**勿与 `make run` 同时开**</text> | 会抢 `:8080` |

---

## A. Flutter 如何对齐 Docker 的 8080 / 9090

### A1. 两边必须填同一个 IP

| 仓库 | 文件 | <text color="green">**关键变量**</text> | 填什么 |
|------|------|------------------------------------------|--------|
| Go `my_go_study` | `.env.lan` | <text color="green">**`REALTIME_PUBLIC_WS_HOST`**</text> | Mac 局域网 IPv4 |
| Flutter `my_ai_project` | `.env.lan` | <text color="green">**`BACKEND_HOST`**</text> | <text color="green">**同一个**</text> IPv4 |

端口一般 **不用手写**：

| 用途 | Go / Docker | Flutter |
|------|-------------|---------|
| HTTP / WS | 映射 **8080** | 默认 `http://…:8080` |
| gRPC | 映射 **9090** | `BackendGrpcConfig` 默认 **9090**（可用 `BACKEND_GRPC_PORT` 覆盖） |

```bash
# Go
cp .env.lan.example .env.lan   # 填 REALTIME_PUBLIC_WS_HOST
make lan-up

# Flutter
cp .env.lan.example .env.lan   # 填 BACKEND_HOST=同一 IP
flutter run --dart-define-from-file=.env.lan
# 或 IDE「LAN 真机」/ ./scripts/run_app.sh --lan
```

<callout emoji="⚠️" background-color="light-red">
<text color="red">**IP 两边不一致**</text>是 Realtime「HTTP 通、WS 挂」的头号原因：ticket 里仍是 `127.0.0.1`，手机连自己。
</callout>

### A2. Flutter 地址怎么算出来的

**HTTP（登录 / 业务）：**

```text
EnvConfig 默认 http://127.0.0.1:8080
        │
        ▼
BackendHttpConfig.remapLocalhostForPlatform()
        ├─ 已是局域网 IP → 原样
        ├─ 127.0.0.1 + 有 BACKEND_HOST → 换成 BACKEND_HOST:8080
        ├─ Android 模拟器 → 10.0.2.2:8080
        └─ 真机未注入 → LanHost.fallback（代码备用 IP）
```

**gRPC（数据分析）：**

```text
BackendGrpcConfig.resolveHost()
  → 优先从 HTTP baseUrl 抠同一 host
  → 端口：BACKEND_GRPC_PORT 或默认 9090
```

白话：<text color="green">**登录和数据分析连同一台门牌（IP）；一个走 8080 门，一个走 9090 门。**</text>

### A3. 场景对照表

| 你怎么跑 App | HTTP | gRPC |
|--------------|------|------|
| iOS 模拟器 + Docker | `127.0.0.1:8080` | `127.0.0.1:9090` |
| Android 模拟器 + Docker | <text color="green">**`10.0.2.2:8080`**</text> | <text color="green">**`10.0.2.2:9090`**</text> |
| 真机 + 已注入 `BACKEND_HOST` | `LAN_IP:8080` | `LAN_IP:9090` |
| 真机未注入、靠 fallback | fallback:8080 | 同 host:9090 |

### A4. 30 秒对齐清单

1. Mac：`ipconfig getifaddr en0`（或 Wi‑Fi 网卡）
2. Go `.env.lan` 的 <text color="green">**`REALTIME_PUBLIC_WS_HOST`**</text> = 该 IP
3. Flutter `.env.lan` 的 <text color="green">**`BACKEND_HOST`**</text> = 该 IP
4. `docker ps` 看到 <text color="green">**`0.0.0.0:8080→8080`**</text> 与 <text color="green">**`0.0.0.0:9090→9090`**</text>
5. Flutter 启动日志 <text color="red">**`BACKEND_HOST=` 不能是 `(未注入)`**</text>（真机时）

---

## B. 改一行 Go → Docker 里何时生效

### B1. 时间线

```text
① 改 cmd/api 或 internal/... 某行 Go
② make lan-up  →  up -d --build
③ BuildKit：依赖层缓存 → COPY 源码 → go build api+worker → Alpine 运行镜像
④ 停旧容器 → 用新镜像起 app / worker
⑤ 端口映射回来：8080 / 9090
⑥ Flutter 下次请求才打到新逻辑（必要时重进页面 / 重连）
```

| 你改了什么 | <text color="green">**要不要 lan-up**</text> | 还要做什么 |
|------------|----------------------------------------------|------------|
| Go 业务 / 进镜像的代码 | <text color="green">**要**</text> | Flutter 若改 proto stubs，另更新 `my_ai_project` |
| 仅 compose / Dockerfile | <text color="green">**要**</text> | — |
| 仅 Go `.env.lan` | <text color="green">**建议 down + up**</text> | 确认 `docker exec … printenv` |
| 仅 Flutter / Flutter `.env.lan` | <text color="red">**不要**</text> 动 Docker | 重跑 / 热重载 App |
| 仅 `docs/` | 不要 | — |

### B2. 「改了代码容器还是旧的」

| 原因 | 白话 |
|------|------|
| <text color="red">**没带 `--build`**</text> | 只重启旧镜像 |
| 本机还有 **`make run`** | 你连错进程 / 端口冲突 |
| Flutter 连错 IP/端口 | 后端已新，客户端打到别处 |
| 改的是 worker 逻辑 | 确认 **worker** 也重建（同一次 `lan-up` 会一起 build） |

### B3. 构建缓存（为何有时快有时慢）

Dockerfile 顺序：

1. 先 `go.mod` / `go.sum` → `go mod download`（**依赖层**）
2. 再 `COPY . .` → `go build`（**源码层**）

- 只改业务代码 → 依赖层常命中，主要重编译  
- 改了 `go.mod` → 依赖层也重做，更慢  
- `.dockerignore` 排除 `docs`、`.env.lan` 等 → 改文档不弄脏构建；改 `internal/` 会  

### B4. 验证新镜像已上岗

```bash
docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'
docker exec my_go_study_app printenv | grep -E 'APP_ENV|GRPC|AUTH|REALTIME'
curl -s http://127.0.0.1:8080/health
docker inspect my_go_study_app --format '{{.Created}}'
```

刚启动的日志里应有 <text color="green">**`gRPC 服务启动 port=9090`**</text>。

---

## C. 排障决策树

<callout emoji="🧭" background-color="light-yellow">
口诀：<text color="green">**先分清链路哪一段挂了**</text>，不要一上来乱 `lan-down` / `lan-up`。
</callout>

### C0. 30 秒分诊

```bash
docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'
curl -sS -m 3 http://127.0.0.1:8080/health
curl -sS -m 3 http://$(ipconfig getifaddr en0):8080/health
nc -z -v 127.0.0.1 9090
```

| 本机 8080 | 局域网 IP:8080 | 9090 | 去哪一节 |
|-----------|----------------|------|----------|
| ✗ | — | — | **C1** |
| ✓ | ✗ | — | **C2** |
| ✓ | ✓ | ✗ | **C3** |
| ✓ | ✓ | ✓ | **C4** |

```text
health(127.0.0.1) 失败？
  ├─ 是 → C1 容器/进程
  └─ 否 → health(LAN_IP) 失败？
        ├─ 是 → C2 网络/防火墙
        └─ 否 → 哪个功能挂？
              ├─ 数据分析 gRPC → C3
              ├─ 登录 / 401     → C4a/b
              ├─ Realtime       → C4c
              ├─ 推送队列       → C4d
              └─ 改码不生效     → C4e / 篇 B
```

### C1. 本机 health 都不通

1. 有没有 `my_go_study_app`？没有 → `make lan-up`（缺 `.env.lan` / IP 仍是 `YOUR_LAN_IP` 最常见）
2. `unhealthy` / 重启？→ `docker logs my_go_study_app|postgres --tail 80`
3. 端口占用？→ `lsof -nP -iTCP:8080 -sTCP:LISTEN`；<text color="red">**停掉冲突的本机 `make run`**</text>
4. Ports 是否含 <text color="green">**`0.0.0.0:8080->8080`**</text>

### C2. Mac 通、真机不通

| 检查 | 期望 |
|------|------|
| 同一 Wi‑Fi（非访客网） | 同网段 |
| IP 是否漂移 | 等于两边 `.env.lan` |
| macOS 防火墙 | 联调窗口建议关 |
| 手机浏览器打开 `http://LAN_IP:8080/health` | `{"status":"ok"}` |
| iOS 本地网络权限 | 允许 |
| 路由器 AP 隔离 | 关 / 换主网络 |

### C3. HTTP 通，gRPC「数据分析」挂

**先看门，再看鉴权。**

```bash
docker exec my_go_study_app printenv | grep GRPC
# 期望 GRPC_ENABLED=true  GRPC_PORT=9090
```

gRPC metadata（与 HTTP Session 对齐）：

- <text color="green">**`authorization: Bearer <token>`**</text>
- <text color="green">**`x-session-id`**</text>
- <text color="green">**`x-device-id`**</text>

Flutter 由 `AnalyticsGrpcApi` 自动注入；手动 `grpcurl` 忘了头 → Unauthenticated。

Android 模拟器 host 用 <text color="green">**`10.0.2.2`**</text>，不是手机局域网 IP。

### C4. 门都通，业务仍失败

#### C4a / C4b 登录与 401

| 现象 | 优先查 |
|------|--------|
| OTP 参数错误 | 缺 `platform` 等 |
| 测试 OTP 无效 | `AUTH_DEV_TEST_*` 是否进容器；debug 模式 |
| 业务 401 | session 头；<text color="red">**单设备被踢**</text>；测试号加白名单 |
| Flutter 仍 mock | `USE_MOCK_AUTH=false` + `.env.lan` |

#### C4c Realtime WS

```text
HTTP 用 BACKEND_HOST=192.168.x.x  ✓
WS ticket 却是 127.0.0.1         ✗
```

→ 对齐 <text color="green">**`REALTIME_PUBLIC_WS_HOST`**</text> 后 <text color="green">**`make lan-down && make lan-up`**</text>。

#### C4d 队列 / 推送

```bash
docker ps | grep worker
docker logs my_go_study_worker --tail 50
```

确认 `QUEUE_ENABLED=true`；worker **没有** HTTP `/health`（healthcheck 已 disable）。

#### C4e 改码不生效

→ 篇 B：必须 rebuild；确认没有本机进程抢口。

### C5. 先别重建 vs 该重建

| 情况 | 建议 |
|------|------|
| 防火墙、IP、Flutter 没带 `.env.lan` | <text color="red">**先别**</text> 乱 rebuild |
| 容器 unhealthy、改了 Go | <text color="green">**要 `make lan-up`**</text> |
| 只改 `.env.lan` | <text color="green">**down + up**</text> |
| 怀疑缓存脏 | `./scripts/lan-compose.sh build --no-cache` 再 up |

---

## 附录：Compose / 端口速查

| 端口 | 映射 | 谁能访问 | 用途 |
|------|------|----------|------|
| <text color="green">**8080**</text> | `8080:8080` | 本机 + 局域网 | HTTP / WS |
| <text color="green">**9090**</text> | `9090:9090` | 本机 + 局域网 | gRPC |
| 5432 | `127.0.0.1:5432` | <text color="red">**仅 Mac**</text> | Postgres |
| 6379 | `127.0.0.1:6379` | <text color="red">**仅 Mac**</text> | Redis |

容器内连库：<text color="green">**`DATABASE_HOST=postgres`**</text>（服务名 DNS），<text color="red">**不要写 `127.0.0.1`**</text>。

关键文件：

| 路径 | 作用 |
|------|------|
| `docker/Dockerfile` | 多阶段构建 api + worker |
| `docker/docker-compose.yml` | 四服务基础编排 |
| `docker/docker-compose.lan.yml` | LAN overlay（`APP_ENV=lan`、local Auth、必填 WS host） |
| `scripts/lan-compose.sh` | `make lan-up` 入口 + 校验 `.env.lan` |

飞书同步（本目录）：

```bash
./scripts/sync_docs_to_feishu.sh bootstrap --config docs/feishu-sync-tool-docker.config.yaml
./scripts/sync_docs_to_feishu.sh sync --config docs/feishu-sync-tool-docker.config.yaml
```

---

## 口诀（收工带一句）

1. <text color="green">**IP 两边对齐**</text>（`BACKEND_HOST` = `REALTIME_PUBLIC_WS_HOST`）  
2. <text color="green">**8080 HTTP，9090 gRPC**</text>  
3. <text color="green">**改 Go 必 rebuild**</text>；改 Flutter 只动客户端  
4. <text color="red">**先分诊再重建**</text>；Docker 与本机 `make run` 不同时开  
