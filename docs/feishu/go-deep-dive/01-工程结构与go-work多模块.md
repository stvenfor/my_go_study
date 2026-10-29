# 01 · 工程结构与 go.work 多模块

> 一句话定位：本篇讲清「仓库分几个 Go 模块、谁能依赖谁、怎么跑起来」；所属模块为工作区全局（go.work + app/common/core/model/thunder）；阅读前置：无，会用终端执行 `make`、知道 `go build` 即可。

## 通俗理解：先把这个仓库想象成什么

把项目想成一家餐厅：

- **app** 是门店大堂和厨房（真正出餐的地方）：HTTP/gRPC 服务、Worker、各种一次性命令都在这。
- **common** 是餐具和规章制度（叶子底座）：配置怎么读、日志怎么打、JWT 怎么签，谁都能用，它不依赖任何业务模块。
- **core** 是外接供应商的对接部：Postgres、Redis、RabbitMQ、极光、融云、Supabase 这些「外部驱动」都包在这里。
- **model** 是菜单和单据模板：领域实体（`entity`）与跨层共享的接口/端口（`repository`、`provider`），不含业务逻辑。
- **thunder** 是店里参考用的一套别人的厨具样品：一个独立的第三方 Gin 脚手架（module 名 `github.com/mszlu521/thunder`），挂在同一个 go.work 里方便随时翻看/试验，但 app 等业务模块并不 import 它。

go.work 就是「商场的统一导览图」：在仓库根目录执行 go 命令时，工具链按这张图把五个目录当成同一个工作区，本地模块之间直接用源码互相引用，不需要发版、不需要 `replace`。

源码锚点：[go.work](file:///Users/qpy/Desktop/github/my_go_study/go.work)、[Makefile](file:///Users/qpy/Desktop/github/my_go_study/Makefile)、[docker/docker-compose.yml](file:///Users/qpy/Desktop/github/my_go_study/docker/docker-compose.yml)、[docker/docker-compose.lan.yml](file:///Users/qpy/Desktop/github/my_go_study/docker/docker-compose.lan.yml)。

## 顶层目录速览

| 目录/文件 | 职责 |
|---|---|
| `app/` | 可执行主模块（module `app`）：6 个命令入口 + 全部业务代码 |
| `common/` | 叶子底座模块（module `common`）：config / logger / jwt / auth / metrics |
| `core/` | 外部驱动模块（module `core`）：database / queue / jpush / rongcloud / supabase |
| `model/` | 领域模块（module `model`）：entity 实体、repository 端口接口、provider 端口接口 |
| `thunder/` | 独立第三方框架模块（module `github.com/mszlu521/thunder`），业务代码未引用 |
| `configs/` | YAML 配置（config.yaml + config.dev/lan/prod.yaml）与 `supabase.env` |
| `docker/` | Dockerfile、基础 compose、LAN overlay compose |
| `scripts/` | make 背后的 shell 脚本（环境加载、compose 包装、LAN IP 同步、联调脚本） |
| `migrations/` | 45 个 SQL 的 golang-migrate 迁移文件（up/down 成对） |
| `deploy/observability/` | Prometheus / Alertmanager / Grafana / 飞书告警适配器配置 |
| `docs/` | 项目文档（ADR、API 文档、架构图等） |
| `go.work` / `go.work.sum` | 工作区编排文件 |
| `Makefile` | 全部日常命令入口 |

## go.work 逐行解读

源码锚点：[go.work](file:///Users/qpy/Desktop/github/my_go_study/go.work#L1-L9)，全文 9 行：

```go
go 1.25.0

use (
	./common
	./core
	./model
	./app
	./thunder
)
```

| 行 | 内容 | 说明 |
|---|---|---|
| 1 | `go 1.25.0` | 工作区要求的 Go 版本；Dockerfile 据此固定 `golang:1.25-alpine` 且 `GOTOOLCHAIN=local` |
| 3-9 | `use (...)` | 把 5 个目录纳入工作区；目录里必须各有一个 `go.mod` |

几个容易踩的事实（均来自实际文件，不是推论）：

1. `use` 里有 **5 个**模块：业务四件套 `app/common/core/model` 加 `thunder`。规范里说的「四模块」指业务四模块；thunder 是挂进来的独立框架。
2. 各模块自己的 go 版本并不一致：`app` 声明 `go 1.24.6`，`common/core/model` 声明 `go 1.22.0`，`thunder` 声明 `go 1.25.0`；工作区统一按 go.work 的 1.25.0 工具链编译。
3. [app/go.mod](file:///Users/qpy/Desktop/github/my_go_study/app/go.mod) 里**没有** `require common/core/model`，也没有 `replace`。业务模块对本地模块的解析完全靠 go.work；脱离工作区单拿 app 目录无法构建。Makefile 注释明确写了老写法 `go build ./cmd/api` 与 `go build ./...` 已失效（[Makefile L4-L6](file:///Users/qpy/Desktop/github/my_go_study/Makefile#L4-L6)）。
4. `Makefile` 的 `MODULES := model common core app`（[L8](file:///Users/qpy/Desktop/github/my_go_study/Makefile#L8)）不含 thunder：逐模块 `go test`/`go mod tidy` 只扫业务四模块。

## 四模块职责与依赖方向

### 模块总表

| 模块 | module path | go 版本 | 目录 | 职责 | 主要直接依赖（摘自各 go.mod） |
|---|---|---|---|---|---|
| model | `model` | 1.22.0 | [model/](file:///Users/qpy/Desktop/github/my_go_study/model) | 领域实体 + 端口接口，零业务编排 | google/uuid |
| common | `common` | 1.22.0 | [common/](file:///Users/qpy/Desktop/github/my_go_study/common) | 配置/日志/JWT/Supabase token 校验/Prometheus 指标 | viper、zap、lumberjack、golang-jwt、prometheus/client_golang |
| core | `core` | 1.22.0 | [core/](file:///Users/qpy/Desktop/github/my_go_study/core) | 外部驱动适配：DB/Redis/MQ/推送/IM/Supabase | gorm + postgres 驱动、go-redis、amqp091-go、supabase-go、zap |
| app | `app` | 1.24.6 | [app/](file:///Users/qpy/Desktop/github/my_go_study/app) | 可执行程序与全部业务代码 | gin、grpc、gorilla/websocket、gorm、eino、zap 等 |
| thunder | `github.com/mszlu521/thunder` | 1.25.0 | [thunder/](file:///Users/qpy/Desktop/github/my_go_study/thunder) | 第三方 Gin 脚手架，业务代码未引用 | gorm、viper、eino 全家桶、gopay、OSS/七牛 SDK 等 |

### 依赖方向（有代码证据）

```text
app   -> common / core / model
core  -> common / model
common: 不依赖任何本地模块（叶子）
model : 不依赖任何本地模块（叶子）
thunder: 独立，不被业务四模块引用
```

证据：

- app 的 main.go 同时 import 三个本地模块，例如 `common/config`、`core/database`、`model/entity`（[app/cmd/api/main.go L54-L65](file:///Users/qpy/Desktop/github/my_go_study/app/cmd/api/main.go#L54-L65)）。
- core 依赖 common：[core/database/postgres.go L8](file:///Users/qpy/Desktop/github/my_go_study/core/database/postgres.go#L8) 与 redis.go、queue、supabase 均 import `common/config`。
- core 依赖 model：[core/queue/pubsub_broadcaster.go L10](file:///Users/qpy/Desktop/github/my_go_study/core/queue/pubsub_broadcaster.go#L10) import `model/entity`。
- 在 app/ 下全仓搜索 `mszlu521/thunder` 无任何匹配：业务代码不依赖 thunder。

### model 内部包职责

源码锚点：[model/](file:///Users/qpy/Desktop/github/my_go_study/model)

| 包 | 路径 | 职责 |
|---|---|---|
| entity | [model/entity/](file:///Users/qpy/Desktop/github/my_go_study/model/entity) | 全部 GORM 实体与 DTO（29 个 .go）：user/profile/transaction、`wys_*` 业务表（home_todo、mall、membership、points、community、short_video、used_car_order 等）、IM、实时信封、SSE 事件、系统通知 payload |
| repository | [model/repository/](file:///Users/qpy/Desktop/github/my_go_study/model/repository) | 跨层共享的仓储端口接口（11 个文件），如 `SessionRepository`（Redis 设备会话）、`ProfileRepository`、`PresenceRepository`、`WSTicketRepository`、`StreamProvider` 无关的各类 repo 接口 |
| provider | [model/provider/stream_provider.go](file:///Users/qpy/Desktop/github/my_go_study/model/provider/stream_provider.go) | LLM 流式生成端口：`StreamProvider` 接口 + `CompletionInput`，由 app 内 mock/openai_compatible 两个实现满足 |

端口接口示例（[session_repository.go L17-L23](file:///Users/qpy/Desktop/github/my_go_study/model/repository/session_repository.go#L17-L23)）：接口只定义 Get/Save/Delete/ListActiveUserIDs，实现可以是 Redis 或内存，usecase 不感知。

### common 内部包职责

源码锚点：[common/](file:///Users/qpy/Desktop/github/my_go_study/common)

| 包 | 路径 | 职责 |
|---|---|---|
| config | [common/config/](file:///Users/qpy/Desktop/github/my_go_study/common/config) | Viper 加载链、全部配置结构体、默认值与环境变量绑定（详见 [02-配置体系](02-配置体系.md)） |
| logger | [common/logger/logger.go](file:///Users/qpy/Desktop/github/my_go_study/common/logger/logger.go) | Zap 全局日志：控制台 core + 可选 lumberjack 文件轮转 core |
| jwt | [common/jwt/jwt.go](file:///Users/qpy/Desktop/github/my_go_study/common/jwt/jwt.go) | 遗留 JWT 体系：`Claims{UserID uint, Username}`、`Manager.Generate/Parse`，服务于遗留 `/user/*` 路由 |
| auth | [common/auth/supabase.go](file:///Users/qpy/Desktop/github/my_go_study/common/auth/supabase.go) | `ValidateAccessToken`：拿用户 token 调 Supabase `/auth/v1/user` 远程校验（文件注释明确「不本地解析 JWT」） |
| metrics | [common/metrics/metrics.go](file:///Users/qpy/Desktop/github/my_go_study/common/metrics/metrics.go) | 进程级 Prometheus Registry 与 `http_requests_total`、`http_request_duration_seconds` 指标 |

注意两套认证不许混淆：common/jwt 是遗留 JWT（数字 user_id），业务主路径是 SessionAuth（业务 token + Redis session），后篇路由文档会展开。

### core 内部包职责

源码锚点：[core/](file:///Users/qpy/Desktop/github/my_go_study/core)

| 包 | 路径 | 职责 |
|---|---|---|
| database | [core/database/](file:///Users/qpy/Desktop/github/my_go_study/core/database) | `NewPostgres`（GORM + 连接池 + 5 次重试 ping）、`NewRedis`（5 秒 ping）、ping.go 重试逻辑 |
| queue | [core/queue/](file:///Users/qpy/Desktop/github/my_go_study/core/queue) | RabbitMQ 客户端/消费者、任务类型 tasks.go、整点 ticker（hourly_ticker.go）、Redis Pub/Sub 广播（publisher + FanoutSubscriber） |
| jpush | [core/jpush/client.go](file:///Users/qpy/Desktop/github/my_go_study/core/jpush/client.go) | 极光推送 REST 客户端封装 |
| rongcloud | [core/rongcloud/client.go](file:///Users/qpy/Desktop/github/my_go_study/core/rongcloud/client.go) | 融云 IM 服务端 API 客户端（Token 签发等） |
| supabase | [core/supabase/](file:///Users/qpy/Desktop/github/my_go_study/core/supabase) | Supabase 客户端：client.go、auth_admin.go（Admin 用户 API）、session.go |

### app 内部结构（鸟瞰）

源码锚点：[app/](file:///Users/qpy/Desktop/github/my_go_study/app)

| 路径 | 职责 |
|---|---|
| `cmd/api` | HTTP + WS + gRPC 同进程主服务 |
| `cmd/worker` | RabbitMQ 消费者进程 |
| `cmd/import-supabase` | 一次性：Supabase Cloud → 本地 PG 导入 |
| `cmd/seed-analytics` | 一次性：刷新 analytics 图表种子 |
| `cmd/seed-home-todo` | 一次性：写首页待办装箱演示种子 |
| `cmd/scheduler-trigger` | 一次性：手动入队整点广播任务 |
| `internal/<域>/` | vertical slice 业务域五件套（router/handler/service/repository/repo_postgres…），共 15 个域目录 |
| `internal/delivery/http/` | legacy 分层：router/controller/handler/middleware/dto/authctx |
| `internal/delivery/grpc/` | gRPC server 与鉴权 interceptor |
| `internal/delivery/ws/` | WebSocket hub/client/handler |
| `internal/delivery/worker/` | Worker 端推送 handler |
| `internal/usecase/` | legacy 用例层（认证、Realtime、IM、JPush、SSE、客服等） |
| `internal/repository/{postgres,redis,supabase,llm,queue,csagent}` | legacy 仓储实现 |
| `internal/shared/` | 跨域共享：错误模型、HTTP 工具、`Registrable` 等 |
| `api/proto` + `api/gen` | analytics gRPC proto 与生成代码 |

## thunder：workspace 里的第五个模块

通俗理解：thunder 是作者 fork/参考的开源 Gin 脚手架（见 [thunder/README.md](file:///Users/qpy/Desktop/github/my_go_study/thunder/README.md)：「a fast and lightweight web framework built on top of Gin」）。它被纳入 go.work 只是为了在同一个工作区里能跳转和试改，业务四模块没有任何代码 import 它，`make test`/`make tidy` 也不扫它。

包职责速查（只看包职责，依据目录名、包注释与 README）：

| 包 | 职责 |
|---|---|
| `thunder/config` | Viper 配置初始化 |
| `thunder/server` | Gin server 与 router 封装 |
| `thunder/midd` | Gin 中间件：auth / cache / cros |
| `thunder/database`、`thunder/db` | GORM 接 MySQL/Postgres/Redis（两套并存的库包） |
| `thunder/cache` | Redis 缓存封装 |
| `thunder/req`、`thunder/res` | 请求参数/分页/context 与统一响应码、响应体 |
| `thunder/errs`、`thunder/gorms`、`thunder/types`、`thunder/helper` | 错误模型、GORM 错误、通用类型、辅助函数 |
| `thunder/event` | 事件与事件管理器 |
| `thunder/pay/wxPay`、`thunder/wx` | 微信支付（gopay）、微信公众号 token/校验/Web |
| `thunder/upload` | 阿里云 OSS、七牛上传 |
| `thunder/ai/a2a`、`thunder/ai/einos` | A2A client、Eino 的 chat/embedding/mcp/tools 封装 |
| `thunder/einos/components/document/parser/epub` | EPUB 文档解析器 |
| `thunder/tools/{convert,crypro,gptr,httputils,images,jwt,randoms,timeouts}` | 转换、加密、泛型指针、HTTP、图片、JWT、随机数、超时工具 |

## 命令体系

### 六个可执行入口

| 命令路径 | Make 入口 | 角色 | 是否需要 queue.enabled |
|---|---|---|---|
| [app/cmd/api](file:///Users/qpy/Desktop/github/my_go_study/app/cmd/api/main.go) | `make run` | 主服务（HTTP 8080 + WS + gRPC 9090） | 否 |
| [app/cmd/worker](file:///Users/qpy/Desktop/github/my_go_study/app/cmd/worker/main.go) | `make run-worker` | RabbitMQ 消费者 + 整点 ticker | 是，false 直接报错退出 |
| [app/cmd/import-supabase](file:///Users/qpy/Desktop/github/my_go_study/app/cmd/import-supabase/main.go) | `make import-supabase` | Cloud 数据导入本地（带 `--default-password` 等 flag） | 否 |
| [app/cmd/seed-analytics](file:///Users/qpy/Desktop/github/my_go_study/app/cmd/seed-analytics/main.go) | `make seed-analytics` | 刷新 analytics_records 种子 | 否 |
| [app/cmd/seed-home-todo](file:///Users/qpy/Desktop/github/my_go_study/app/cmd/seed-home-todo/main.go) | 无专门 target | 写装箱演示种子，需手动 `./scripts/load-env.sh go -C app run ./cmd/seed-home-todo` | 否 |
| [app/cmd/scheduler-trigger](file:///Users/qpy/Desktop/github/my_go_study/app/cmd/scheduler-trigger/main.go) | `make trigger-hourly-notify` | 向 RabbitMQ 入队一次广播任务 | 是，false 直接报错退出 |

### Makefile target 全表

源码锚点：[Makefile](file:///Users/qpy/Desktop/github/my_go_study/Makefile)。所有 go 命令都用 `go -C app`（或工作区根的 `go build all`），因为工作区根不是 module。

| Target | 实际命令 | 用途/备注 |
|---|---|---|
| `run` | `./scripts/load-env.sh go -C app run ./cmd/api` | 启动 API；先 source 环境文件 |
| `run-worker` | `./scripts/load-env.sh go -C app run ./cmd/worker` | 启动 Worker |
| `build` | `go -C app build -o ../bin/api ./cmd/api` 且 build worker | 产物在仓库根 `bin/` |
| `build-worker` | 只 build `bin/worker` | |
| `build-all` | 工作区根 `go build all` | 跨 5 个模块整体编译，替代失效的 `go build ./...` |
| `test` | 依次 `cd model/common/core/app && go test ./... -count=1` | 不含 thunder |
| `test-all` | `go build all` 后逐模块测试 | 比 test 多一步全工作区编译 |
| `tidy` | 逐模块 `go mod tidy` | 注释警告：缺依赖时会拉最新版，注意 pin 回 |
| `air` | `air -c .air.toml` | 热重启 |
| `proto` | `protoc` 生成 analytics pb 代码 | 先检查 protoc / 两个插件是否存在 |
| `migrate-up` | `migrate -path migrations ... up` | DSN 写死 localhost postgres/postgres |
| `migrate-down` | 同上 `down 1` | 回滚一版 |
| `docker-build` | `scripts/docker-compose.sh build` | |
| `docker-up` | `scripts/docker-compose.sh up -d --build` | 无 Docker 时提示改用 `make deps-up` |
| `docker-down` | `scripts/docker-compose.sh down` | |
| `deps-up` | brew 装/启 postgresql@16 + redis | 自动建 postgres 角色与 my_go_study 库 |
| `lan-up` | `scripts/lan-compose.sh up -d --build` | 基础 compose + LAN overlay；先同步网卡 IP |
| `lan-down` | `scripts/lan-compose.sh down` | |
| `lan-run` | `scripts/lan-run.sh api` | 无 Docker 的 LAN 本机进程模式 |
| `lan-run-worker` | `scripts/lan-run.sh worker` | |
| `sync-lan-ip` | `scripts/sync-lan-ip.sh` | 支持 `LAN_CLIENTS=` 覆盖客户端清单 |
| `import-supabase` | load-env 后 run import-supabase | 必填 `DEFAULT_PASSWORD`，否则 exit 2 |
| `import-supabase-dry` | 同上加 `--dry-run --skip-users` | 只拉取统计 |
| `seed-analytics` | run seed-analytics | |
| `trigger-hourly-notify` | run scheduler-trigger | dev 关闭 cron 时手动触发 |
| `test-transactions` | `scripts/test_transactions_crud.sh` | 需 Supabase token/service_role |
| `check-rls` | `scripts/check_transactions_rls.sh` | 验证 RLS |
| `test-realtime` | `scripts/test_realtime_ws.sh` | WS 联调 |
| `test-single-device-login` | `scripts/test_single_device_login.sh` | 单设备登录 |
| `test-auth-refresh-logout` | `scripts/test_auth_refresh_logout.sh` | |
| `test-phone-otp-login` | `scripts/test_phone_otp_login.sh` | |
| `test-queue-push` | `scripts/test_queue_push.sh` | 需 run + run-worker 且 queue.enabled |
| `test-scheduled-notify` | `scripts/test_scheduled_notify.sh` | |
| `push-notify-user` | `scripts/push_notify_user.sh $(USER_ID) $(EMAIL) "$(TITLE)" "$(BODY)"` | 向指定用户推 sys.notify |
| `check-secrets` | `scripts/check-secrets.sh` | 推送前防 service_role 入库 |
| `clean` | `rm -rf bin tmp logs/*.log` | |

### 环境加载脚本（函数级）

[scripts/load-env.sh](file:///Users/qpy/Desktop/github/my_go_study/scripts/load-env.sh) 全文 8 行，逐行职责：

| 行 | 代码 | 职责 |
|---|---|---|
| 3 | `set -euo pipefail` | 出错即退、未定义变量报错、管道错误透传 |
| 4 | `ROOT="$(cd "$(dirname "$0")/.." && pwd)"` | 无论从哪个目录调用，都定位到仓库根 |
| 5 | `cd "$ROOT"` | 切到仓库根（保证 `go -C app`、相对路径一致） |
| 7 | `source "$ROOT/scripts/source-env.sh"` | 加载环境变量文件 |
| 8 | `exec "$@"` | 用当前 shell 替换为目标命令（如 `go -C app run ./cmd/api`） |

[scripts/source-env.sh](file:///Users/qpy/Desktop/github/my_go_study/scripts/source-env.sh) 的加载顺序（`set -a` 自动导出）：

1. `configs/supabase.env`（团队常量，入库：SUPABASE_URL / SUPABASE_ANON_KEY）
2. `.env`（本地常规变量，模板 [.env.example](file:///Users/qpy/Desktop/github/my_go_study/.env.example)）
3. `.env.local`（私密覆盖，模板 [.env.local.example](file:///Users/qpy/Desktop/github/my_go_study/.env.local.example)，主要放 SUPABASE_SERVICE_ROLE_KEY）

后者覆盖前者（shell source 的自然行为）。LAN 模式额外 source `.env.lan`（模板 [.env.lan.example](file:///Users/qpy/Desktop/github/my_go_study/.env.lan.example)）。

其他编排脚本：

| 脚本 | 职责 |
|---|---|
| [scripts/docker-compose.sh](file:///Users/qpy/Desktop/github/my_go_study/scripts/docker-compose.sh) | source 环境后执行 `docker compose -f docker/docker-compose.yml "$@"` |
| [scripts/lan-compose.sh](file:///Users/qpy/Desktop/github/my_go_study/scripts/lan-compose.sh) | 校验 `.env.lan` 存在 → 非 down 类命令先 `sync-lan-ip` → source `.env.lan` → 校验 `REALTIME_PUBLIC_WS_HOST` 不是占位值 → 默认 `APP_ENV=lan` → compose 同时挂基础文件 + lan overlay |
| [scripts/lan-run.sh](file:///Users/qpy/Desktop/github/my_go_study/scripts/lan-run.sh) | 无 Docker 版：sync IP + source 全部 env + 默认 `AUTH_PROVIDER=local` + `go -C app run ./cmd/api|worker` |
| scripts/sync-lan-ip.sh | 按当前网卡 IPv4 改写 `.env.lan` 与各客户端配置（开关在 `scripts/lan-ip.clients`） |
| scripts/test_*.sh、push_notify_user.sh、check_transactions_rls.sh、reindex_transactions_es.sh、check-secrets.sh | 联调与防护脚本，均由对应 make target 包装 |

## Docker：本地开发栈与 LAN overlay

### Dockerfile 多阶段

源码锚点：[docker/Dockerfile](file:///Users/qpy/Desktop/github/my_go_study/docker/Dockerfile)。

| 阶段 | 内容 | 关键点 |
|---|---|---|
| builder | `golang:1.25-alpine` | 注入 `TARGETARCH`（arm64/amd64）；`GOPROXY` 默认 goproxy.cn；`GOTOOLCHAIN=local`；对 goproxy.cn 与 alpine CDN 绕过代理；先拷 go.work + 五个模块的 go.mod/go.sum 再 `go mod download`（依赖层缓存） |
| 编译 | `cd app && go build ... ./cmd/api ./cmd/worker` | `CGO_ENABLED=0 -ldflags="-w -s"`，产出 `/app/bin/api` 与 `/app/bin/worker` |
| 运行时 | `alpine:3.20` | 非 root 用户 appuser；带 curl/ca-certificates/tzdata；拷入两个二进制与 `configs/`；EXPOSE 8080/9090；HEALTHCHECK 打 `/health`；默认 CMD 跑 api |

### 基础 compose 服务表（docker/docker-compose.yml）

| 服务 | 镜像 | 宿主端口（全部绑 127.0.0.1） | 依赖/健康检查 | 备注 |
|---|---|---|---|---|
| postgres | postgres:16-alpine | 127.0.0.1:5432 | pg_isready 健康检查 | 库名/用户/密码均 my_go_study/postgres/postgres，数据卷 postgres_data |
| redis | redis:7-alpine | 127.0.0.1:6379 | redis-cli ping | |
| rabbitmq | rabbitmq:3.13-management-alpine | 127.0.0.1:5672 + 15672 | rabbitmq-diagnostics ping | 管理台 15672 |
| elasticsearch | elasticsearch:8.15.3 | 127.0.0.1:9200 | curl 9200，start_period 30s | 单节点、关安全、512m 堆 |
| app | 本地 Dockerfile 构建 | 8080 + 9090（对所有接口暴露） | depends_on pg/redis/rabbitmq healthy、ES started；自身打 /health | `restart: unless-stopped`（注释说明 RabbitMQ 首启慢拍时自愈）；挂 `../logs:/app/logs` |
| worker | 同一镜像 | 无 | depends_on redis/rabbitmq healthy、app started | `command: ["/app/worker"]`，显式 `healthcheck: disable`（worker 无 HTTP） |
| prometheus | prom/prometheus:v2.54.1 | 127.0.0.1:9091 | depends_on app | 保留 24h、开 lifecycle reload |
| alertmanager | prom/alertmanager:v0.27.0 | 127.0.0.1:9093 | | |
| feishu-adapter | python:3.12-alpine | 无对外端口 | | 跑 deploy 下的 feishu_adapter.py，仅网络内被 Alertmanager 调 |
| grafana | grafana/grafana:11.2.0 | 127.0.0.1:3000 | depends_on prometheus | 账号密码取 GRAFANA_ADMIN_*，默认 admin/admin |

容器内固定走服务名：`DATABASE_HOST=postgres`、`REDIS_ADDR=redis:6379`、`RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/`、ES 默认 `http://elasticsearch:9200`；其余值通过 `${VAR:-默认}` 从宿主机环境注入（宿主机环境由 docker-compose.sh source 三个 env 文件提供）。

### LAN overlay（docker-compose.lan.yml）

通俗理解：基础文件是「只给本机用的开发栈」（DB/Redis 只绑 127.0.0.1）；overlay 是叠加在它上面的「真机联调补丁」——API 仍只开 8080/9090，但认证切成 local、WS 对外宣告的主机名换成 Mac 的真实局域网 IP。

| 差异点 | 基础 compose | LAN overlay 覆盖 |
|---|---|---|
| APP_ENV | `${APP_ENV:-dev}` | 固定 `lan` |
| AUTH_PROVIDER | `${AUTH_PROVIDER:-}`（空） | `${AUTH_PROVIDER:-local}` |
| QUEUE_ENABLED | `${QUEUE_ENABLED:-true}` | 固定 `"true"` |
| REALTIME_PUBLIC_WS_HOST | 不注入 | `${REALTIME_PUBLIC_WS_HOST:?...}`，必填，占位值会被 lan-compose.sh 拦截 |
| 白名单/测试号 | 透传同名变量 | 同名透传（AUTH_SESSION_WHITELIST_*、AUTH_DEV_TEST_*） |
| DB/Redis 端口 | 127.0.0.1 绑定 | overlay 不额外暴露 DB/Redis（注释明确） |
| worker | dev 环境变量 | 同步覆盖为 lan/local/QUEUE_ENABLED=true |

LAN 启动链路：`make lan-up` → 校验 Docker → `lan-compose.sh`（sync IP → source `.env.lan` → 校验 IP）→ `docker compose -f docker-compose.yml -f docker-compose.lan.yml up -d --build`。无 Docker 时走 `make lan-run`（本机进程 + Homebrew PG/Redis）。

## 学习要点回顾

1. go.work 是「本地多模块免发版」机制：业务四模块 + thunder 共 5 个 `use`，模块间用源码解析，app/go.mod 因此没有本地 require。
2. 依赖方向是单向的：app → common/core/model；core → common/model；common 与 model 是叶子，不反向依赖。写代码时把「端口接口」放进 model、把「驱动实现」放进 core，就是为了维持这个方向。
3. 工作区根不是 module：所有 go 命令要么 `go -C app ...`，要么在根用 `go build all`；Makefile 已全部封装好。
4. 配置分三层来源：YAML（configs/）、shell env（.env 系列）、代码默认值；加载细节见 [02-配置体系](02-配置体系.md)。
5. 六个 cmd 共享同一套 config.Load 起步，但只有 api/worker 是常驻进程，其余四个是一次性任务命令；seed-home-todo 没有 make target。
6. Docker 栈分两层文件：基础 compose 绑 127.0.0.1 仅供本机，lan overlay 切 local Auth 并注入真实局域网 IP 供真机联调。
7. worker 镜像与 api 同一个，仅靠 `command` 区分；worker 无 HTTP 端点，所以必须关闭镜像默认 healthcheck。
8. thunder 是参考用的独立框架，不进业务依赖图，也不进 `make test`/`tidy` 的模块清单。

## 可动手验证

1. 在仓库根执行 `go build all`，再到 app 目录执行 `go build ./...`，对比「工作区根不是 module」带来的差异。
2. 打开 [app/go.mod](file:///Users/qpy/Desktop/github/my_go_study/app/go.mod) 确认没有 `require common`，然后 `go -C app list -m all | head`，观察 go.work 如何让 `common`/`core`/`model` 以本地路径出现。
3. `grep -rn "mszlu521/thunder" app core common model`（应无结果），验证 thunder 与业务代码隔离。
4. `make -n run` 与 `make -n lan-up` 预览命令展开，对照本篇的脚本链路。
5. 执行 `./scripts/load-env.sh env | grep -E 'APP_ENV|SUPABASE'` 观察三个 env 文件的加载结果（注意会显示密钥，别录屏）。
6. `docker compose -f docker/docker-compose.yml config --services` 列出基础栈全部服务，再加上 `-f docker/docker-compose.lan.yml` 对比 overlay 后的 environment。
