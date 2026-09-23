# iOS 真机局域网联调调试记录（2026-09-16）

> **结论（一句话）**：真机登录提示「无法连接服务端（`http://127.0.0.1:8080`）」时，**不是 Go 没起来**，而是 Flutter **未注入 `BACKEND_HOST`**，手机把请求打到了自己。  
> **相关手册**：[dual-end-lan-startup.md](./dual-end-lan-startup.md) · [lan-backend-host.md](./lan-backend-host.md) · Flutter [BACKEND_INTEGRATION.md](../../my_ai_project/docs/BACKEND_INTEGRATION.md) · Flutter [device-startup-notes.md](../../my_ai_project/docs/device-startup-notes.md)（**无线 release / USB debug**）

---

## 1. 背景与目标

| 项 | 内容 |
|----|------|
| 日期 | 2026-09-16 |
| 场景 | Mac 作局域网 BFF（`APP_ENV=lan` + `AUTH_PROVIDER=local`），iPhone 真机 Flutter 登录 |
| 本机 IP（当时） | `172.16.0.43`（`ipconfig getifaddr en0`） |
| Go 启动方式 | **无 Docker Desktop** → `make lan-run`（Homebrew Postgres + Redis） |
| Flutter 仓库 | `my_ai_project`（与本仓独立 Git） |

同日还完成了 Compose Phase 1 加固（见 §8）；真机联调主路径仍是 `lan-run`。

---

## 2. 现象时间线

```text
1. make docker-up / lan-up
   → 报错「未安装 Docker Desktop」→ 改用 make lan-run

2. iPhone 安装 App，短信登录（13400000000）
   → Toast：无法连接服务端（http://127.0.0.1:8080）
   → 提示文案已指向「请用 LAN 真机 / .env.lan」

3. 启动日志：
   [App] … baseUrl=http://127.0.0.1:8080 BACKEND_HOST=(未注入) …

4. Mac 侧同时验证：
   curl http://127.0.0.1:8080/health     → {"status":"ok"}
   curl http://172.16.0.43:8080/health   → {"status":"ok"}
   API 监听 *:8080，APP_ENV=lan，AUTH_PROVIDER=local
```

**关键信号**：报错括号里的 URL 是 `127.0.0.1`，且日志 `BACKEND_HOST=(未注入)` → 与「Go 未启动」无关。

---

## 3. 根因分析

### 3.1 主因：`BACKEND_HOST` 未进入本次构建

| 机制 | 说明 |
|------|------|
| 配置源 | Flutter `EnvConfig.test.backendBaseUrl` 写死 `http://127.0.0.1:8080` |
| 真机 remap | `BackendHttpConfig.remapLocalhostForPlatform` 仅在存在 `String.fromEnvironment('BACKEND_HOST')`（或后续加的 debug 回退）时，把 host 换成 Mac 局域网 IP |
| 注入方式 | `--dart-define-from-file=.env.lan`（文件内 `BACKEND_HOST=172.16.0.43`） |
| 失败原因 | 实际用 **Xcode / 裸 flutter run / 默认 launch（只带 `.env`）** 安装；`.env` **没有** `BACKEND_HOST` |
| Hot Restart | **无效**；dart-define 只在 **完整重新编译/安装** 时生效 |

```text
iPhone 真机
  │  错误：http://127.0.0.1:8080  → 手机 loopback（无 Go）
  │  正确：http://172.16.0.43:8080 → Mac 上的 my_go_study
  ▼
Mac :8080（make lan-run）
```

### 3.2 次因 / 易踩坑（同日排查清单）

| # | 坑 | 说明 |
|---|----|------|
| 1 | Cursor 开的是 **Go 仓** | F5 不会自动用 `my_ai_project/.vscode/launch.json` 的 LAN 配置 |
| 2 | 默认 `flutter_module_sample` 曾只绑 `.env` | 真机也跑这条配置 → 必现 127.0.0.1 |
| 3 | iOS 本地网络权限 | 无 `NSLocalNetworkUsageDescription` 时，连局域网 IP 可能失败且难排查 |
| 4 | lan 测试短信 OTP | `config.lan.yaml` 默认清空 `AUTH_DEV_TEST_*`；须在 `.env.lan` 打开，并 **重启** `lan-run` |
| 5 | Docker 不可用 | `make lan-up` 失败属预期；无 Docker 用 `make lan-run` |

### 3.3 非根因（已排除）

- Go `/health` 本机与 LAN IP 均正常  
- ATS：`NSAllowsArbitraryLoads=true` 已允许明文 HTTP  
- `USE_MOCK_AUTH`：`.env` / `.env.lan` 均为 `false`（走真实 BFF）

---

## 4. 修复与工程改动

### 4.1 Flutter（`my_ai_project`）

| 改动 | 目的 |
|------|------|
| `.vscode/launch.json`：默认 `flutter_module_sample` → `.env.lan`；保留「模拟器 / 本机 .env」；顶部「LAN 真机」 | IDE 默认真机可用 |
| `scripts/run_app.sh`：`--lan`；`-d` 物理机自动切 `.env.lan` | CLI 不易漏 define |
| `ios/Runner/Info.plist`：`NSLocalNetworkUsageDescription` + `NSBonjourServices` | 弹出/允许「本地网络」 |
| `commons/network/.../lan_host.dart`：`LanHost.debugFallback` | Xcode / 裸 run 未注入时，debug/profile 仍 remap 到固定 LAN IP |
| `BackendHttpConfig.effectiveBackendHost` | dart-define 优先，否则用 debugFallback |
| `EnvironmentServiceImpl.backendBaseUrl` | 设置页展示与真实请求一致（已 remap） |
| `user_auth_api` 连接失败文案 | 区分「仍是 127.0.0.1」与「已是 LAN IP」 |
| `main.dart` 启动日志打印 `BACKEND_HOST` | 一眼确认是否注入 |

**IP 变更时**：同步改三处——

1. Go `.env.lan` → `REALTIME_PUBLIC_WS_HOST`  
2. Flutter `.env.lan` → `BACKEND_HOST`  
3. Flutter `commons/network/lib/http/lan_host.dart` → `debugFallback`（仅 debug 回退）

### 4.2 Go（`my_go_study`）

| 改动 | 目的 |
|------|------|
| `.vscode/launch.json`：`Flutter LAN 真机`（`cwd` 指向 `my_ai_project` + `.env.lan`） | 在 Go 工作区也能正确起 Flutter |
| `.env.lan` 增加 `AUTH_DEV_TEST_PHONE/OTP/PASSWORD`（不入库） | `13400000000` + `123456` 可用 |
| Compose Phase 1（§8） | Docker 路径补强；本机无 Docker 时不依赖 |

### 4.3 验证通过时的期望日志

```text
[App] 应用初始化完成 env=测试
  baseUrl=http://172.16.0.43:8080
  BACKEND_HOST=172.16.0.43
  loggedIn=false …
```

Toast 不应再出现 `127.0.0.1`。短信登录：`13400000000` / `123456`（需 Go 已加载测试 OTP 配置）。

---

## 5. 标准复现与验收（下次照抄）

### 5.1 Go

```bash
cd my_go_study
cp .env.lan.example .env.lan   # 首次
# REALTIME_PUBLIC_WS_HOST=<LAN_IP>
# 可选短信：AUTH_DEV_TEST_PHONE=13400000000 / OTP=123456 / PASSWORD=…

make deps-up          # 若 PG/Redis 未起
make lan-run          # 无 Docker
# 改过 .env.lan 后必须 Ctrl+C 再 lan-run

curl -s http://127.0.0.1:8080/health
curl -s http://<LAN_IP>:8080/health
# iPhone Safari 同址 /health 也应 ok
```

### 5.2 Flutter

```bash
cd my_ai_project
cp .env.lan.example .env.lan
# BACKEND_HOST=<与 Go 相同的 LAN_IP>
# 同步 lan_host.dart 的 debugFallback（若依赖回退）

# 必须完整 Stop → Run（不要 Hot Restart）
# IDE：flutter_module_sample 或「LAN 真机」
./scripts/run_app.sh --lan -d <iphone_id>
```

### 5.3 验收清单

| # | 检查 | 通过标准 |
|---|------|----------|
| 1 | 启动日志 `BACKEND_HOST=` | 等于 `<LAN_IP>`，不是 `(未注入)` |
| 2 | `baseUrl=` | `http://<LAN_IP>:8080` |
| 3 | 设置页环境 | **测试**，地址为局域网 IP |
| 4 | iPhone Safari `/health` | `{"status":"ok"}` |
| 5 | 本地网络权限 | 已允许 |
| 6 | 登录 | 邮箱或测试短信成功，不再「无法连接服务端」 |

---

## 6. 排障决策树

```text
登录「无法连接服务端」
  │
  ├─ 括号 URL = 127.0.0.1 / 日志 BACKEND_HOST=(未注入)
  │     → 未注入 define：换 LAN launch / --lan / 依赖 lan_host 回退后【完整重装】
  │
  ├─ 括号 URL = http://<LAN_IP>:8080
  │     ├─ Mac curl <LAN_IP>:8080/health 失败 → 先起 make lan-run / 查端口
  │     ├─ Mac 通、手机 Safari 不通 → 同 Wi‑Fi？防火墙？AP 隔离？
  │     └─ Safari 通、App 不通 → 本地网络权限；清装重跑
  │
  └─ 能连上但短信 OTP 失败
        → .env.lan 的 AUTH_DEV_TEST_* + 重启 lan-run；码是否 123456
```

| 日志 / Toast | 含义 | 动作 |
|--------------|------|------|
| `BACKEND_HOST=(未注入)` 且无 debugFallback | 必失败 | 完整重跑带 `.env.lan` |
| `BACKEND_HOST=172.x.x.x` 仍连不上 | 网络层 | Safari health + 防火墙 + 权限 |
| `认证服务暂时不可用` | Auth/Supabase 配置 | lan 应用 `AUTH_PROVIDER=local` |

---

## 7. 知识点备忘（给以后的自己 / Agent）

1. **真机上的 `127.0.0.1` ≠ Mac**；模拟器才接近「本机 Go」。  
2. **`--dart-define*` 是编译期常量**；改 `.env.lan` 或 launch 后必须重新 `flutter run` / IDE Run。  
3. **`.env` 与 `.env.lan` 职责不同**：前者多为 Mock 开关；后者必须含 `BACKEND_HOST`。  
4. Go / Flutter **两个仓库、两套 `.env.lan`**，IP 必须手写对齐。  
5. `LanHost.debugFallback` 是 **开发兜底**，不是生产方案；IP 漂移时要改代码常量或改回强制 dart-define。  
6. 无 Docker：`make lan-run`；有 Docker：`make lan-up`（改 Go 需 `--build`）。

---

## 8. 同日附带：Docker Compose Phase 1

本机未装 Docker，未用 Compose 完成真机验收；工程侧已落地：

| 项 | 处理 |
|----|------|
| `Dockerfile` | `TARGETARCH`（Apple Silicon） |
| worker | `healthcheck: disable`（避免套用 API `/health`） |
| env 注入 | `scripts/docker-compose.sh` + compose 映射 `SUPABASE_*` 等 |
| lan overlay | 可切回 `AUTH_PROVIDER=supabase` |
| `.dockerignore` | 收紧构建上下文 |
| 文档 | `startup-guide` / `lan-backend-host` / `dual-end-lan-startup` 对齐 |

装好 Docker Desktop 后：`make docker-up`（本机）或 `make lan-up`（真机全栈）。

---

## 9. 文档与入口索引

| 文档 | 用途 |
|------|------|
| **本文** | 本次调试记录 + 根因 + 验收 |
| [dual-end-lan-startup.md](./dual-end-lan-startup.md) | 日常两端启动清单 |
| [lan-backend-host.md](./lan-backend-host.md) | LAN Profile / Compose 决策 |
| [startup-guide.md](./startup-guide.md) | Docker / 本地启动总览 |
| Flutter `BACKEND_INTEGRATION.md` | 客户端 baseUrl / LAN 启动 |

Agent 总览表见仓库根 [AGENTS.md](../AGENTS.md)。
