# 本机局域网后端主机（LAN Backend Host）

> 把开发者 Mac 当作家庭/信任 Wi‑Fi 上的 Go BFF，供 iOS / Android 真机联调。  
> **不是**公网生产部署。用语见仓库根 [`CONTEXT.md`](../CONTEXT.md)。  
> 状态：**已实现**；且 **lan = 本地 Auth + Postgres**（见 [local-auth-postgres.md](./local-auth-postgres.md)），不再默认连 Supabase Cloud。  
> **两端（Go + Flutter）逐步启动配置**：请直接看 **[dual-end-lan-startup.md](./dual-end-lan-startup.md)**。

---

## 0. 已决摘要

| # | 决策 | 选择 |
|---|------|------|
| 1 | 访问范围 | 同 Wi‑Fi 局域网真机 |
| 2 | 存活形态 | 按需启停（用时 `docker compose`，不用就停） |
| 3 | 进程形态 | Compose 全栈：postgres + redis + app + worker |
| 4 | 地址策略 | 路由器 DHCP 预留 / Mac 固定局域网 IPv4 |
| 5 | 客户端 | iOS + Android 真机都要 |
| 6 | 传输 | 先 `http`/`ws` 明文；HTTPS/`wss` 作附录预留 |
| 7 | 端口暴露 | 默认只对局域网提供 **8080**；5432/6379 不常驻映射，查库用 `exec` 或临时映射 |
| 8 | 配置姿态 | 独立 **LAN Profile**（`APP_ENV=lan`），不与纯 `dev`、不与 `prod` 混用 |
| 9 | 改造范围 | Go（本仓）+ Flutter（`my_ai_project`）一起做 |
| 10 | IP / 白名单存放 | 仅本机未入库 env；仓库只留 example |
| 11 | 多端同账号 | 测试账号走 session 白名单；普通账号仍单设备 |
| 12 | macOS 防火墙 | 联调期间关闭（用完建议再开） |
| 13 | 本轮交付 | 本详细方案落盘（本文）；实现另开任务 |

---

## 1. 目标架构

```text
iOS / Android 真机（同 Wi‑Fi）
  │  http://<PINNED_LAN_IP>:8080
  │  ws://<PINNED_LAN_IP>:8080/realtime/v1/connect
  ▼
Mac（LAN Backend Host）
  Docker Compose（或 make lan-run 本机进程）
    ├── app      :8080 → 宿主机 8080（唯一常驻对外端口）
    ├── worker   （队列 / 推送消费，不对局域网暴露端口）
    ├── postgres （127.0.0.1:5432；业务 + local auth）
    └── redis    （127.0.0.1:6379；session / realtime / queue）
  │
  ├── Auth 默认 → 本地 Postgres（AUTH_PROVIDER=local）
  └── 本机未入库 env：LAN IP、白名单邮箱/UID、JWT；切回 Cloud 时再要 SUPABASE_*
```

要点：

- API 进程本身监听 `:{port}`（全网卡），局域网可达；缺的是 **对外宣告的主机名**（尤其 Realtime ticket 里的 `realtime.public_ws_host`）与 **Flutter base URL** 对齐到同一固定 IP。
- 默认 **本地 Auth + 本地 Postgres**（见 [local-auth-postgres.md](./local-auth-postgres.md)）。若 `.env.lan` 设 `AUTH_PROVIDER=supabase`，则 Auth 走 Cloud，容器仍注入 `SUPABASE_*`。

---

## 2. 网络与主机准备（人工步骤）

### 2.1 固定局域网 IP

任选其一：

1. **路由器 DHCP 预留**（推荐）：用 Mac 的 Wi‑Fi MAC 绑定固定 IPv4。  
2. **macOS 静态 IP**：系统设置 → 网络 → 详情 → TCP/IP → 手动。

记下该地址为 `PINNED_LAN_IP`（下文占位符）。不要提交进 Git。

### 2.2 防火墙（已决：联调时关闭）

1. 系统设置 → 网络 → 防火墙 → **关闭**（仅联调窗口）。  
2. 联调结束建议重新打开。  
3. 若改为「开着防火墙放行 Docker」，见排障 §7。

### 2.3 本机校验

```bash
# 在 Mac 上
ipconfig getifaddr en0   # 或 Wi‑Fi 对应网卡；确认等于 PINNED_LAN_IP

# Compose 起来后（见 §4）
curl -s http://127.0.0.1:8080/health
# 应用另一台同 Wi‑Fi 设备或手机浏览器：
curl -s http://<PINNED_LAN_IP>:8080/health
# 期望：{"status":"ok"}
```

---

## 3. Go 侧：LAN Profile

### 3.1 配置文件

已提供 `configs/config.lan.yaml`。相对 `dev`：

| 项 | 行为 |
|----|------|
| `server.mode` | `debug`（便于测试 OTP；勿当生产） |
| `log.level` | `info` |
| `auth.provider` | `local`（可用 env 切回 `supabase`） |
| `auth.dev_test_*` | 默认清空（可用 `.env.lan` 的 `AUTH_DEV_TEST_*` 临时打开） |
| `realtime.public_ws_host` | **不写死 IP**；由 `REALTIME_PUBLIC_WS_HOST` 注入（`pkg/config` 已 BindEnv） |
| `queue.enabled` | `true` |

### 3.2 本机未入库 env

模板：`.env.lan.example` → 复制为 `.env.lan`（已 gitignore）。

### 3.3 Compose：LAN 覆盖

- Overlay：`docker/docker-compose.lan.yml`
- 启动：`make lan-up`（`./scripts/lan-compose.sh up -d --build`）
- 停止：`make lan-down`
- Postgres/Redis 在基础 compose 中绑定 **`127.0.0.1:5432/6379`**（本机可调、局域网不可达）；API `8080:8080` 对局域网开放。
- 查库：`docker compose … exec postgres psql -U postgres -d my_go_study`

### 3.4 镜像与代码变更节奏

Compose 全栈下，**改 Go 代码需重新 `make lan-up`（会 `--build`）**；仅改 `.env.lan` 也建议 `lan-down` 后再 `lan-up` 以便环境变量生效。频繁改代码时可改用 `make lan-run`（本机热编译）+ Compose 只跑 postgres/redis。

### 3.5 密钥与可切回 Cloud Auth

`lan-compose.sh` 会 source `configs/supabase.env`、`.env`、`.env.local`、`.env.lan`；`docker-compose.lan.yml` 将 `SUPABASE_*` / 白名单等映射进 app/worker。默认 `AUTH_PROVIDER=local` 不依赖 Cloud；切回时在 `.env.lan` 设 `AUTH_PROVIDER=supabase` 并保证 anon/service_role 可用。

---

## 4. Flutter 侧（`my_ai_project`）

### 4.1 Base URL 与 WS 主机一致

已有 `--dart-define=BACKEND_HOST=`（见 `BackendHttpConfig`）。局域网联调：

```bash
cp .env.lan.example .env.lan   # BACKEND_HOST=与 Go 相同的 PINNED_LAN_IP
flutter run --dart-define-from-file=.env.lan
```

Realtime 以 ticket 的 `wsUrl` 为准（Go `REALTIME_PUBLIC_WS_HOST`）；若仍为 `127.0.0.1`，`BackendWsConfig` 会在有 `BACKEND_HOST` 时一并改写。

### 4.2 Android 明文 HTTP

主 Manifest 已 `android:usesCleartextTraffic="true"`，一般无需再改。

### 4.3 iOS 本地网络

首次访问本地网可能弹出本地网络权限；需允许。

### 4.4 会话头

业务 API 仍需 `Authorization` + `X-Session-ID` + `X-Device-ID`。白名单账号可多设备。

---

## 5. 推荐日常操作流程

```text
1. 确认 PINNED_LAN_IP；关闭 macOS 防火墙（联调窗口）
2. 本仓：配置 .env.lan（REALTIME_PUBLIC_WS_HOST + 白名单）；local auth 不必强依赖 supabase
3. make lan-up（有 Docker）或 make lan-run（无 Docker）
4. curl http://<PINNED_LAN_IP>:8080/health
5. Flutter：.env.lan 指向同一 IP，真机 flutter run
6. 用白名单测试账号在 iOS + Android 同时登录，验证不互踢
7. 验证 Realtime：换票后 WS 主机为 PINNED_LAN_IP，不是 127.0.0.1
8. 结束：make lan-down 或 Ctrl+C；酌情重新打开防火墙
```

---

## 6. 验收清单

| # | 检查项 | 期望 |
|---|--------|------|
| 1 | 手机浏览器或 curl 访问 `http://<IP>:8080/health` | `{"status":"ok"}` |
| 2 | 同网扫描/连接 `5432`/`6379` | 默认不可达（未映射） |
| 3 | Flutter 真机登录 | 成功，拿到 token + session_id |
| 4 | iOS + Android 同白名单账号 | 均可调业务 API，不因互踢 401 |
| 5 | Realtime 连接 | ticket / 实际连接主机 = `PINNED_LAN_IP` |
| 6 | 非白名单第二台 mobile 登录 | 仍符合单设备覆盖行为 |
| 7 | `make check-secrets` | 无 service_role 入库 |

---

## 7. 排障

| 现象 | 优先查 |
|------|--------|
| 真机完全连不上 Mac | 是否同一 Wi‑Fi；IP 是否变了；防火墙是否仍开；AP 隔离（访客网络） |
| Mac 上 health 通、手机不通 | 防火墙；Docker 端口是否发布到 `0.0.0.0:8080` |
| HTTP 通、登录后 WS 失败 | `REALTIME_PUBLIC_WS_HOST` 是否仍为 `127.0.0.1`；Flutter 是否没用 ticket 里的 host |
| Android 立刻网络错误 | cleartext / network security config |
| 第二台设备莫名 401 | 账号是否在白名单；是否单设备被踢 |
| 改代码不生效 | Compose 是否 `--build` 重建了 app/worker |

---

## 8. 附录：以后要 HTTPS / wss（预留，当前不做）

目标：手机仍连 Mac，但使用 `https`/`wss`。

建议形态：

```text
真机 → https://<域名或 IP>:443 → Caddy/Nginx（TLS 终止）→ 127.0.0.1:8080
```

注意：

- 自签证书需在 iOS/Android 信任，摩擦大；更好是内网域名 + 受信 CA，或临时公网隧道（超出本方案范围）。  
- 启用后：`REALTIME_PUBLIC_WS_HOST` 与 Flutter base URL 改为该主机名；ticket 侧协议需与 `wss` 一致（以当时 Realtime 实现为准）。  
- 不改变「默认只暴露反代端口、DB 不映射」的暴露面策略。

---

## 9. 实现任务拆分

| # | 任务 | 状态 |
|---|------|------|
| 1 | Go `config.lan.yaml` + `REALTIME_PUBLIC_WS_HOST` BindEnv + `.env.lan.example` | ✅ |
| 2 | Compose overlay + `make lan-up/down`；DB/Redis 绑 127.0.0.1 | ✅ |
| 3 | 文档索引 / AGENTS / 本文 | ✅ |
| 4 | Flutter `.env.lan.example` + BACKEND_INTEGRATION | ✅ |
| 9 | Compose Phase 1：TARGETARCH、worker healthcheck、env 注入、文档对齐 | ✅ |  

---

## 10. 相关文档

- [两端联调启动手册](./dual-end-lan-startup.md)  
- [iOS 真机 LAN 调试记录（2026-09-16）](./ios-lan-device-debug-2026-09-16.md)  
- [启动指南](./startup-guide.md)  
- [Realtime WebSocket](./realtime-websocket.md)  
- [认证导读（含白名单）](./auth-beginner-walkthrough.md)  
- [Supabase 集成](./supabase-integration.md)  
- 仓库根 [CONTEXT.md](../CONTEXT.md) · [AGENTS.md](../AGENTS.md)
