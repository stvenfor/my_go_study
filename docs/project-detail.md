# my_go_study Go 服务端项目详细文档

## 1. 文档说明

| 项 | 内容 |
|----|------|
| **分析范围** | 本仓库 `go.work` 编排的 4 个业务模块（`app` / `common` / `core` / `model`）+ 本地三方库 `thunder/`；配套资产 `configs/`、`migrations/`、`supabase/migrations/`、`docker/`、`deploy/`、`scripts/`、`Makefile`。**不含** Flutter 仓 `my_ai_project` 的代码，不含 `docs/` 下的既有叙述性文档（仅在结论冲突时引用）。 |
| **生成时间** | 2026-10-29（分支 `main`） |
| **证据规则** | 每条结论后附 `路径:行号` 或 `文件#函数/类型`。凡写「未找到」均在 12.4 附对应检索方式，便于复核。 |
| **待确认项** | 全部集中在 **第 12 节**，正文中用 `⚠️ 待确认` 标记并回链。 |

复核用的公共前提：

- workspace 定义 `go.work:1-9`：`go 1.25.0`，`use ( ./common ./core ./model ./app ./thunder )`。
- 全仓 Go 源码**没有任何 `//go:build` 构建标签**（`grep -rn "//go:build" --include="*.go" .` → 0 命中），即不存在按环境裁剪的编译期分支。

---

## 2. 项目概览

**定位**：与 Flutter 客户端 `my_ai_project` 配对的 Go BFF（Backend For Frontend）。单进程同时对外提供 HTTP 与 gRPC 两个端口，向下聚合 PostgreSQL / Redis / RabbitMQ / Elasticsearch / Supabase。

| 维度 | 事实 | 证据 |
|------|------|------|
| 框架 | Gin + gRPC + GORM + go-redis + amqp091 + Clean Architecture | `app/go.mod:6,15,17,20` |
| 模块结构 | go.work 多模块（4 业务模块 + 1 本地三方库） | `go.work:3-9` |
| 对外端口 | HTTP `:8080`、gRPC `:9090`（同进程） | `configs/config.yaml:4,9`；`docker/docker-compose.yml:110-111` |
| 可执行入口 | **6 个**，全部在 `app/cmd/` 下 | `app/cmd/api/main.go:76`、`app/cmd/worker/main.go:35`、`app/cmd/import-supabase/main.go`、`app/cmd/scheduler-trigger/main.go`、`app/cmd/seed-analytics/main.go`、`app/cmd/seed-home-todo/main.go` |
| 唯一可执行模块 | 只有 `app` 有 `cmd/`；`common`/`core`/`model` 为纯库 | `app/go.mod:1`（`module app`） |
| 路由风格 | 按域拆分：23 个 router 文件；15 个域已改垂直切片自挂路由，其余仍集中在 `delivery/http/router/` | `app/internal/delivery/http/router/router.go:22-31,117-121` |
| 三方聚合 | Eino（客服 Agent）、Supabase gotrue/postgrest、融云、极光、微信/支付宝/华为 IAP/Apple IAP | `app/go.mod:5,13,14`；`common/config/config.go:710-718` |

### 2.1 直接依赖

**`app`（14 个直接依赖，`app/go.mod:5-20`）**

| 依赖 | 版本 | 用途 | 使用证据 |
|------|------|------|----------|
| `gin-gonic/gin` | v1.10.0 | HTTP 框架 | `app/internal/delivery/http/router/router.go:12` |
| `google.golang.org/grpc` | v1.71.0 | gRPC server | `app/internal/delivery/grpc/server.go:47` |
| `google.golang.org/protobuf` | v1.36.6 | proto 运行时 | `app/api/gen/go/analytics/v1/` |
| `gorm.io/gorm` | v1.25.12 | ORM（**注意：driver 不在 app，在 core**） | `app/cmd/api/main.go:622` |
| `redis/go-redis/v9` | v9.7.0 | Redis（业务 repo 直连） | `app/internal/repository/redis/session_repo.go` |
| `github.com/cloudwego/eino` | v0.9.21 | 客服 Agent 的 Tool 编排 | `app/internal/usecase/cs_agent_usecase.go` |
| `supabase-community/gotrue-go` | v1.2.0 | Supabase Auth | `app/internal/usecase/` |
| `supabase-community/postgrest-go` | v0.0.11 | Supabase 数据读写 | `app/internal/repository/supabase/` |
| `gorilla/websocket` | v1.5.3 | Realtime WS | `app/internal/delivery/ws/client.go:118` |
| `golang-jwt/jwt/v5` | v5.2.1 | 自建 JWT | `app/internal/delivery/http/middleware/local_session_auth.go:24` |
| `go.uber.org/zap` | v1.27.0 | 日志 | `app/cmd/api/main.go:86` |
| `golang.org/x/crypto` | v0.42.0 | bcrypt | `app/internal/usecase/local_auth_usecase.go` |
| `google/uuid` | v1.6.0 | 主键 / ticket / request id | `model/entity/community.go:18` |
| `prometheus/client_model` | v0.6.1 | 指标模型 | `common/metrics/metrics.go` |

**`common`（5 个，`common/go.mod` require 块）**：`golang-jwt/jwt/v5 v5.2.1`、`prometheus/client_golang v1.20.5`、`spf13/viper v1.19.0`、`go.uber.org/zap v1.27.0`、`gopkg.in/natefinch/lumberjack.v2 v2.2.1`。

**`core`（8 个，`core/go.mod` require 块）**：`google/uuid v1.6.0`、`rabbitmq/amqp091-go v1.10.0`（`core/go.mod:7`）、`redis/go-redis/v9 v9.7.0`、`supabase-community/gotrue-go v1.2.0`、`supabase-community/supabase-go v0.0.4`、`go.uber.org/zap v1.27.0`、`gorm.io/driver/postgres v1.5.11`、`gorm.io/gorm v1.25.12`。

**`model`（1 个，`model/go.mod:3`）**：`github.com/google/uuid v1.6.0`。其余 import 全是标准库（`time` / `fmt` / `strings` / `encoding/json` / `math/big` / `strconv`）—— 领域层近乎零依赖。

**`thunder`（`thunder/go.mod:1`）**：`module github.com/mszlu521/thunder`，`go 1.25.0`。

> **关键事实**：`thunder` 在 `go.work:8` 被 `use`，但 4 个业务模块的 `.go` 与 `go.mod` 中**没有任何一处 import 或 require 它**（`grep -rn "mszlu521/thunder" app common core model` → 0 命中），也没有 `replace` 指向它（`grep replace app/go.mod` → 空）。它只被 `docker/Dockerfile:31` COPY（因为 `go build all` 要求 workspace 成员目录存在）。→ 见 12.2。

> **各模块 Go 版本不一致**：`app` = 1.24.6（`app/go.mod:3`）、`common`/`core`/`model` = 1.22.0、`thunder` = 1.25.0、workspace = 1.25.0（`go.work:1`）。实际生效取 workspace/最高值。

---

## 3. 目录结构与职责

### 3.1 规模（`find` + `wc -l` 于 2026-10-29 实测）

| 模块 | `.go` 文件 | 其中 `_test.go` | 非测试代码行 | 目录数 |
|------|-----------|----------------|-------------|--------|
| `app` | 237 | 55 | 26,896 | 57 |
| `model` | 42 | 2 | 2,750 | 4 |
| `core` | 17 | 4 | 1,337 | 6 |
| `common` | 11 | 5 | 1,438 | 6 |

`model/entity/` 共 30 个 `.go` 文件（`ls model/entity/*.go | wc -l`）。

### 3.2 顶层

```text
go.work                    # 5 模块编排；★ 未入库（见 12.1）
Makefile                   # 唯一工程入口，26 个目标（见 11.1）
app/                       # 可执行模块：cmd + api(proto) + internal(全部业务)
common/                    # config / logger / jwt / auth / metrics —— 零项目内依赖
core/                      # database / queue / supabase / jpush / rongcloud —— 外部系统驱动
model/                     # entity / repository / provider —— 领域层
thunder/                   # 本地三方库（业务未接入）
configs/                   # config.{yaml,dev,lan,prod}.yaml + supabase.env
migrations/                # golang-migrate 格式 22 对 SQL
supabase/migrations/       # 仅 2 个 Supabase 侧 RLS 脚本
docker/                    # docker-compose.yml + lan overlay + Dockerfile
deploy/observability/      # prometheus / alerts / alertmanager / grafana / feishu_adapter
scripts/                   # 22 个 shell + feishu_doc_sync/ Python 包
docs/                      # 既有叙述文档 + 飞书同步配置与源
.harness/ .cursor/ .qoder/ # Agent 编排与 IDE 规则
```

### 3.3 `app/` 内部

| 路径 | 职责 | 证据 |
|------|------|------|
| `app/cmd/api/main.go` | 唯一的 DI / 装配 / 启动点，45 步 | `:76-596` |
| `app/cmd/worker/main.go` | RabbitMQ 消费者进程，自建第二条 AMQP 连接 | `:35-137` |
| `app/cmd/{import-supabase,seed-analytics,seed-home-todo,scheduler-trigger}/` | 一次性运维/种子工具 | 各 `main.go` |
| `app/api/proto/analytics/v1/analytics.proto` | gRPC 契约唯一真相源 | `:8` |
| `app/api/gen/go/analytics/v1/` | `make proto` 产物 | `Makefile:17-25` |
| `app/internal/delivery/http/{handler,controller,middleware,router,dto,authctx}` | HTTP 适配层 | — |
| `app/internal/delivery/grpc/` | gRPC server + unary 鉴权 interceptor | `server.go:47`、`interceptor/auth.go:47` |
| `app/internal/delivery/ws/` | WS Hub / Client / Handler | `hub.go:26` |
| `app/internal/delivery/worker/` | RabbitMQ 任务分发器 | `handler.go:71` |
| `app/internal/usecase/` | **横向**用例层（19 个带测试的文件） | — |
| `app/internal/repository/{postgres,redis,supabase,es,llm,csagent,queue}` | **横向**适配器 | — |
| `app/internal/<15 个域>/` | **垂直切片**：`router.go` + `handler.go` + `service.go` + `repo_postgres.go` + `service_test.go` | 例 `app/internal/mall/`、`app/internal/wallet/` |
| `app/internal/shared/` | 跨域小工具：`http.go`（SessionAuthChain）、`errors.go`、`dberr.go`、`resource_error.go`、`domain_helpers.go`、`im_sync.go` | `ls app/internal/shared/` |
| `app/internal/testsupport/` | 测试替身 | `access_stub.go` |

**15 个垂直切片域**（`ls -d app/internal/*/`）：`access` `after_sales` `community` `deal_invoice` `home` `mall` `membership` `new_car_follow` `payment` `points` `purchase_calculator` `short_video` `transaction` `used_car` `wallet`。

> 迁移期两套装配并存是**有意为之**，`router.go:24-31` 的头注释明确写了「全部域切完后，`*Controller` 字段与 `registerXxxRoutes` 一起删除」。→ 见 5.1。

---

## 4. 启动与初始化时序

### 4.1 文字时序

`cmd/api/main.go` 是单函数式的长流程（`main` 起于 `:76`）。整体分五段：

1. **基础基建（失败即退出）**：`APP_ENV` → `config.Load` → `logger.Init` → PostgreSQL 连接 + `autoMigrate` → Redis 连接。这一段是唯一"硬失败"路径，任何一步出错 `return err` → `os.Exit`。证据 `:81-119`。
2. **本地表结构准备**：~~在 `Auth.IsLocalProvider()` 分支里逐域调用 `<domain>.EnsureSchema(db)`~~ —— 2026-11-03（parity-p0-1）已收口到 `app/internal/schemaboot.Boot()`（`main.go:121`，见 4.2 步骤 8 与 6.3）：会员 / 成交发票 / 二手车 / 新车跟进 / 售后 / 社区索引 / 推送设备 / IM 共 8 步作为 BestEffort（失败仍 Warn-continue），整体受 `database.auto_migrate` 守卫，`APP_ENV=lan|prod` 强制为 0 条 DDL。种子（`EnsureSeed` / `EnsurePackingDemo`）仍在原分支，由 `config.IsLabAppEnv` 守卫。
3. **业务能力装配（按配置开关，可选依赖）**：交易服务（按 provider 二选一 PG / Supabase）→ 可选 ES → Realtime（Hub → Ticket/Sync/Push/Presence → WS + Controller）→ 可选队列（RabbitMQ 入队 + Pub/Sub 订阅）→ 可选 SSE → 可选客服 Agent。`:258-410`。
4. **HTTP 服务**：`handler.NewUserHandler`（`:424`）→ 构造 SessionAuth 中间件（`:441-448`）→ `router.Setup(Options{...})`（`:511`）→ `http.Server{ReadHeaderTimeout: 10s}`（`:531-535`）→ `go server.ListenAndServe()`（`:565`）。
5. **gRPC + 信号**：`grpc.NewServer(UnaryInterceptor(...))`（`:545-551`）→ `go Listen(:9090)`（`:560`）→ 阻塞在 `SIGINT/SIGTERM`（`:573-575`）→ 10s 超时 ctx（`:578`）→ **gRPC `GracefulStop`（`:581-584`）→ RabbitMQ `Close`（`:586-590`）→ HTTP `Shutdown`（`:592-594`）** → `defer` 逆序释放 `logger.Sync`(`:90`) / Redis(`:115`) / sqlDB(`:105`)。

**注意**：AGENTS.md 里描述的顺序（"9. router.Setup；10. SIGTERM → HTTP Shutdown + gRPC GracefulStop"）与实际代码不完全一致 —— 实际停机是 **gRPC 先停、HTTP 后停**。→ 见 12.3。

### 4.2 步骤表（API 进程）

| # | 位置 | 动作 | 失败处理 |
|---|------|------|----------|
| 1 | `main.go:78-80` | 读 `APP_ENV`（默认 `dev`） | — |
| 2 | `main.go:81-84` | `config.Load("configs", env)` | `log.Fatalf` 级别（返回 err） |
| 3 | `main.go:86-89` | `logger.Init(cfg.Log)` | 返回 err |
| 4 | `main.go:90` | `defer logger.Sync()` | — |
| 5 | `main.go:93` | Supabase 未启用时打 Warn 提示 | Warn-continue |
| 6 | `main.go:96-100` | `database.NewPostgres(cfg.Database)` | 返回 err |
| 7 | `main.go:102-105` | `sqlDB` + `defer sqlDB.Close()` | 返回 err |
| 8 | `main.go:121-140` | **`schemaboot.Boot(db, Options{...})`**：`auto_migrate` 开关守卫下跑 `autoMigrate`（Required，失败拒启）+ 8 个域内建表（BestEffort，失败 Warn），随后做 `schema_migrations` 版本校验（落后 / dirty / 空库 → 拒启） | Required 与版本校验返回 err；BestEffort Warn-continue |
| 9 | `main.go:143-151` | `database.NewRedis(cfg.Redis)` + `defer Close` | 返回 err |
| 10 | ~~`membership.EnsureSchema`~~ | 已并入 #8 的 BestEffort（原 Warn-continue 语义保持） | — |
| 11 | `main.go:~228` | `home.EnsurePackingDemo(db)`（经 `seedIfLab`，仅 lab APP_ENV） | Warn-continue |
| 12-15 | ~~`deal_invoice` / `used_car` / `new_car_follow` / `after_sales` 的 `EnsureSchema`~~ | 均已并入 #8 的 BestEffort | — |
| 16 | `main.go:248` | `purchase_calculator.EnsureSeed(db)` | Warn-continue |
| 17 | `main.go:258 / 279` | 交易仓储：local → `NewPostgresRepository`；否则 → `NewSupabaseRepository` | 分支选择 |
| 18 | `main.go:287-288` | ES `EnsureIndex` | Warn-continue（"写路径降级，搜索可能 503"） |
| 19 | `main.go:295-329` | Realtime：Hub → ticket/sync/push/presence → `ws.Handler` → `RealtimeController` | — |
| 20 | `main.go:312-324` | 若 `queue.enabled`：`PushEnqueuer` + `FanoutSubscriber.Run(ctx, ...)`（订阅 goroutine 用裸 `context.Background()`，见 `:321`） | Warn-continue |
| 21 | ~~`main.go:336 / 339`~~ | 社区 trigram 索引已并入 #8（`community.EnsureSearchIndexes`）；`:339` 附近仅剩 `seedIfLab` 种子 | Warn-continue |
| 22 | `main.go:352` | `MemoryRateLimiter`（SSE 按用户滑动窗口） | — |
| 23 | ~~`main.go:374 / 395`~~ | `EnsurePushDeviceSchema` / `EnsureImSchema` 已并入 #8；极光/融云客户端装配仍在原处 | — |
| 24 | `main.go:382` | JPush `apns_production` 选项 | — |
| 25 | `main.go:392 / 420` | JPush / 融云未配置 → 跳过真实下发 | Warn-continue |
| 26 | `main.go:410` | SSE / 客服 Agent provider 装配（mock / openai_compatible） | — |
| 27 | `main.go:424` | `handler.NewUserHandler(...)` | — |
| 28 | `main.go:434` | analytics 表种子 | Warn-continue |
| 29 | `main.go:441-448` | 构造 `sessionAuth`：`IsLocalProvider()` → `LocalSessionAuth`；否则 `SupabaseSessionAuth` | — |
| 30 | `main.go:511-530` | `router.Setup(Options{...})`（含 `DomainRouters` 遍历挂载） | — |
| 31 | `main.go:531-535` | `http.Server{Addr, Handler, ReadHeaderTimeout: 10 * time.Second}` | — |
| 32 | `main.go:540` | 端口占用等启动前错误 | **`log.Fatal`（跳过所有 defer）** |
| 33 | `main.go:545-551` | `grpc.NewServer(grpc.UnaryInterceptor(authInter))` | — |
| 34 | `main.go:556-560` | `grpc.Listen(":9090")` + `go Serve` | — |
| 35 | `main.go:565` | HTTP `go ListenAndServe` 错误分支 | **`log.Fatal`（跳过所有 defer）** |
| 36 | `main.go:569` | gRPC 缺鉴权后端 → 跳过 | Warn-continue |
| 37 | `main.go:573-578` | `signal.Notify(SIGINT,SIGTERM)` + `context.WithTimeout(10s)` | — |
| 38 | `main.go:581-584` | `grpcServer.GracefulStop()` | — |
| 39 | `main.go:586-590` | `queueClient.Close()` | — |
| 40 | `main.go:592-594` | `server.Shutdown(ctx)` | — |
| 41 | `main.go:635-676` | `autoMigrate`（现由 `schemaboot` 的 Required 步骤调用，`auto_migrate=false` 时整段不执行）：遗留 transactions 改名 → `ConsolidateLocalUsers` → `db.AutoMigrate(13 模型)` | 返回 err |

### 4.3 Worker 进程时序（`app/cmd/worker/main.go`）

| # | 位置 | 动作 | 与 API 的差异 |
|---|------|------|--------------|
| 1 | `:35-42` | 同套 `config.Load` + `logger.Init` | 独立进程、独立 logger |
| 2 | `:44-46` | `queue.enabled == false` → **拒绝启动** | API 只是 Warn-continue |
| 3 | — | **不执行 `autoMigrate`** | 表结构只由 API 负责 |
| 4 | `:50-60` | 独立 PG 连接 + **第二条独立 AMQP 连接** | 与 API 不共享 |
| 5 | `:66-67` | `FanoutPublisher` → `NewDeliveryPushUsecase` | 走 Pub/Sub 扇出，不直投本地 Hub |
| 6 | `:115-129` | `Consumer.Run` + `StartHourlyTicker` 挂同一 ctx | — |
| 7 | `:131-136` | 信号 → `cancel()` | **未等待 goroutine 退出**，无 `WaitGroup` |

`cmd/scheduler-trigger/main.go:37` 入队的是**空 `BroadcastNotifyPayload{}`**（无 `ScheduleSlot`），消费侧回退用 `time.Now()` 当 slot（`worker/handler.go:115-120`）→ 风险见 12.1。

### 4.4 API 与 Worker 的共享 / 独立矩阵

| 资源 | API | Worker | 是否共享实例 |
|------|-----|--------|------------|
| PostgreSQL 连接池 | `main.go:96` | `worker/main.go:50` | ❌ 各自独立 |
| Redis | `main.go:112` | 经 usecase 间接 | ❌ |
| AMQP 连接 | `main.go:312`（生产） | `worker/main.go:50-60`（消费） | ❌ 两条连接 |
| WS Hub | 进程内 `main.go:295` | ❌ 无 Hub | 靠 Redis Pub/Sub 跨进程 |
| `autoMigrate` | ✅ `:107` | ❌ | 单侧责任 |

---

## 5. 架构分层与依赖图

### 5.1 风格判定

**结论：这是一个「Clean Architecture 的进行中混合体」，不是教科书式实现。** 依赖方向这条最硬的红线**守住了**，但端口位置、模块边界、接口契约纪律三处都有明确偏离。

已验证的红线（全部为 0 命中，反向依赖不存在）：

| 检查 | 命令 | 结果 |
|------|------|------|
| `model` 依赖任何兄弟模块 | `grep -rn '"app/\|"core/\|"common/' model --include="*.go"` | **0** |
| `common` 依赖 `app` 或 `core` | `grep -rn '"app/\|"core/' common --include="*.go"` | **0** |
| `core` 依赖 `app` | `grep -rn '"app/' core --include="*.go"` | **0** |

正向依赖（引用文件数，排除 `_test.go`）：`app→model` 97 文件、`app→common` 35 文件、`app→core` 14 文件、`core→common` 6 文件、`core→model` **1 文件**。

7 条有据可查的偏离：

| # | 偏离 | 证据 | 影响 |
|---|------|------|------|
| 1 | **端口不在最内层私有目录，而在共享内核 `model/repository/`** | `model/repository/session_repository.go:18` 等 15 个接口 | 任何模块都能看见端口，符合 Go 社区"共享内核"变体，但不是 Uncle Bob 原图 |
| 2 | **`app` 同时持有 delivery 与 repository 适配器**（适配层没有独立模块） | `app/internal/repository/`（7 个子包）与 `app/internal/delivery/`（4 个子包）同级 | 单模块内"外层包"与"内层包"混在一起，靠目录约定而非编译边界约束 |
| 3 | **基建反向依赖领域**：`core` 唯一一处 import `model/entity` | `core/queue/pubsub_broadcaster.go:10` | infra→domain 耦合，`core` 无法作为纯驱动层复用 |
| 4 | **贫血实体**：`model/entity` 绝大多数只有 `TableName()` 与枚举常量，行为在 `app/internal/<域>/service.go` | 例 `model/entity/mall.go`（结构体 + const，无方法业务逻辑）；业务规则在 `app/internal/mall/service.go` | "领域层"实际只是数据层；Domain Rule 在 usecase/service |
| 5 | **横向 `usecase/` + 垂直 `<domain>/` 两套装配并存**（迁移未完） | `app/internal/delivery/http/router/router.go:24-31` 头注释自述；`app/internal/` 下同时有 `usecase/`、`repository/` 与 15 个域目录 | 新增代码要先判断"该写横向还是垂直" |
| 6 | **`usecase` 直接 import 具体基建，破坏 DIP** | `app/internal/usecase/` 下 3 处：`core/jpush:9`、`core/supabase:22`、`phone_otp:9` | 对比：`rongcloud` **是**被抽象的（`app/internal/usecase/im_group_usecase.go:52` 走接口）→ 同一层内纪律不一致 |
| 7 | **`thunder` 声明在 workspace 但零接入** | `go.work:8` vs `grep "mszlu521/thunder" app common core model` → 0 | 目录存在是为了 `go build all` / `docker/Dockerfile:31` 的 COPY 不失败 |

### 5.2 分层职责

| 层 | 模块 / 包 | 职责 | 不得做 | 关键证据 |
|----|-----------|------|--------|----------|
| Frameworks & Adapters（入向） | `app/internal/repository/*`、`app/internal/delivery/*` | GORM/Redis/ES/PostgREST 落地；Gin/gRPC/WS/AMQP 接线 | 不写业务规则 | `app/internal/repository/redis/session_repo.go:16` |
| Application（UseCase） | `app/internal/usecase/`、`app/internal/<域>/service.go` | 编排、事务边界、权限判定、错误映射 | 不 import Gin/GORM 类型 | `app/internal/usecase/device_session_usecase.go:81` |
| Domain（模型） | `model/entity`、`model/repository`、`model/provider` | 实体、枚举、值对象、端口、不变量 | 不 import 兄弟模块、不 import 任何三方（除 `google/uuid`） | `model/go.mod:3` |
| Infrastructure（外部系统驱动） | `core/{database,queue,supabase,jpush,rongcloud}` | 连接、客户端封装、协议细节 | 不该 import `model`（**目前有一处违例**） | `core/queue/pubsub_broadcaster.go:10` |
| Foundations（通用底座） | `common/{config,logger,jwt,auth,metrics}` | 配置、日志、JWT、token 校验、Prometheus registry | 不 import `app`/`core` | 0 命中（见 5.1 表） |

### 5.3 关键抽象接口

**全量口径**：4 个业务模块的非测试代码中共 **56 个** `type … interface {` 声明（`grep -rn "^type .* interface {" --include="*.go" app common core model | grep -v _test | wc -l`），其中位于 `model/`（端口层）的只有 **15 个**：`model/repository/` 14 个 + `model/provider/` 1 个。

**编译期契约断言严重不足**：全仓只有 **5 处** `var _ I = (*T)(nil)`：

| 断言 | 位置 |
|------|------|
| `var _ provider.StreamProvider = (*OpenAICompatibleProvider)(nil)` | `app/internal/repository/llm/openai_compatible_provider.go:213` |
| `var _ provider.StreamProvider = (*MockStreamProvider)(nil)` | `app/internal/repository/llm/mock_stream_provider.go:121` |
| `var _ repository.ConversationRepository = (*ConversationRepository)(nil)` | `app/internal/repository/redis/conversation_repo.go:98` |
| `var _ repository.RealtimeEventRepository = (*RealtimeEventRepository)(nil)` | `app/internal/repository/redis/realtime_event_repo.go:86` |
| `var _ repository.ConversationRepository = (*MemoryConversationRepository)(nil)` | `app/internal/usecase/completion_usecase.go:453` |

→ 51 个接口没有任何显式断言（含 `SessionRepository`：定义 `model/repository/session_repository.go:18`，实现 `app/internal/repository/redis/session_repo.go:16`，实现体没有 `var _` 行；契约破坏只能靠调用点报错）。→ 见 12.2。

**15 个端口清单**：

| 接口 | 定义 | 实现 | 主要调用方 |
|------|------|------|-----------|
| `SessionRepository` | `model/repository/session_repository.go:18` | `app/internal/repository/redis/session_repo.go:16` | `app/internal/usecase/device_session_usecase.go:81,112,177` |
| `ProfileRepository` | `model/repository/profile_repository.go:11` | **双实现**：`app/internal/repository/postgres/profile_repo.go`（local）+ `app/internal/repository/supabase/profile_repo.go:25,43`（Cloud） | `ProfileController` |
| `WSTicketRepository` | `model/repository/ws_ticket_repository.go:16` | `app/internal/repository/redis/ws_ticket_repo.go` | `realtime_ticket_usecase.go:56`；`ws/handler.go:114` |
| `RealtimeEventRepository` | `model/repository/realtime_event_repository.go:10` | `app/internal/repository/redis/realtime_event_repo.go` | `realtime_push_usecase.go:126,151` |
| `PresenceRepository` | `model/repository/presence_repository.go:6` | `app/internal/repository/redis/presence_repo.go` | `RealtimeController` |
| `ConversationRepository` | `model/repository/conversation_repository.go:15` | `app/internal/repository/redis/conversation_repo.go` + 内存版 `app/internal/usecase/completion_usecase.go:453` | SSE usecase |
| `AnalyticsRepository` | `model/repository/analytics_repository.go:10` | `app/internal/repository/postgres/` | `AnalyticsUsecase` → gRPC |
| `AddressRepository` | `model/repository/address_repository.go:17` | `app/internal/repository/postgres/address_repo.go` | `app/internal/wallet`? 否 → `AddressController` |
| `PushDeviceRepository` | `model/repository/push_device_repository.go:10` | `app/internal/repository/postgres/push_device_repo.go` | `jpush_usecase.go` |
| `StoreStatsRepository` | `model/repository/store_stats_repository.go:15` | `app/internal/repository/postgres/` | `ProfileController`（`main.go:276`） |
| `ImFriendRepository` | `model/repository/im_repository.go:19` | `app/internal/repository/postgres/im_repo.go` | `ImController` |
| `ImGroupRepository` | `im_repository.go:31` | `im_repo.go:149-235` | `im_group_usecase.go:52-194` |
| `ImBackupRepository` | `im_repository.go:43` | `im_repo.go` | `im_controller.go` |
| `ImUserLookup` | `im_repository.go:48` | `repository/supabase/`（跨库查用户） | `im_*` |
| `StreamProvider` | `model/provider/stream_provider.go:20` | `repository/llm/openai_compatible_provider.go:213` + `mock_stream_provider.go:121` | `CompletionUsecase`、`CsAgentUsecase` |

**多实现接口（可替换性验证）**：`ProfileRepository`（postgres / supabase）、`ConversationRepository`（redis / memory）、`StreamProvider`（openai_compatible / mock）—— 这三组是仓库里唯一真正做到"同一端口多个适配器 + 运行时按配置选"的地方。

### 5.4 对外入口

| 类型 | 入口 | 监听/注册 | 鉴权 |
|------|------|----------|------|
| HTTP API | `:8080` | `main.go:531-535` → `router.Setup` `router.go:53` | 全局 4 层 + 分组 SessionAuth（见 9.1） |
| gRPC | `:9090` | `main.go:545-560`；`server.go:47` | unary interceptor `interceptor/auth.go:47` |
| WebSocket | `GET /realtime/v1/connect`（可配 `realtime.ws_path`） | `router.go:106-112` | **首帧 `auth` + 一次性 Redis ticket**（`ws/handler.go:74,98-125`） |
| SSE（AI 小石头） | `POST /api/v1/sse/completions` | `router/sse_routes.go:11-14` | SessionAuth + 内存限流（`completion_usecase.go:311-320`） |
| SSE（客服 Agent） | `POST /api/v1/cs-agent/chat` | `router/cs_agent_routes.go:10-13` | SessionAuth |
| 指标 | `GET /metrics` | `router.go:66` | **无**（注释自述靠内网可达） |
| 健康 | `GET /health` | `router.go:62-64` | 无 |
| AMQP 消费 | `queue.<rabbitmq.queue>` | `core/queue/consumer.go:45` | — |
| 定时 | 每分钟 ticker（整点触发） | `core/queue/hourly_ticker.go:30` | — |
| CLI | `import-supabase` / `seed-analytics` / `seed-home-todo` / `scheduler-trigger` | 各自 `main.go` | 人工执行 |

### 5.5 依赖图

实线 = 已验证的 import（附命中文件数）；虚线 = 异常/未接入。

```mermaid
graph TD
  subgraph WS["go.work（5 模块）"]
    direction TB

    subgraph OUTER["Frameworks & Adapters（app，唯一可执行）"]
      DLV["delivery/http<br/>handler·controller·middleware·router·dto"]
      GRP["delivery/grpc<br/>server + auth interceptor"]
      WS2["delivery/ws<br/>Hub·Client·Handler"]
      WKR["delivery/worker<br/>任务分发器"]
      UC["usecase/（横向 19）"]
      DOM["&lt;15 个垂直切片域&gt;<br/>router·handler·service·repo_postgres"]
      REPO["repository/<br/>postgres·redis·supabase·es·llm·csagent·queue"]
    end

    MODEL["model<br/>entity（55 表）· repository（14 端口）· provider（1 端口）<br/>外部依赖：仅 google/uuid"]
    CORE["core<br/>database·queue·supabase·jpush·rongcloud"]
    COMMON["common<br/>config（Viper）· logger（zap）· jwt · auth · metrics"]
    THUNDER["thunder<br/>module github.com/mszlu521/thunder"]

    DLV --> UC
    GRP --> UC
    WS2 --> UC
    WKR --> UC
    DOM --> UC
    UC --> REPO
    DLV --> REPO
    DOM --> REPO

    MODEL -.->|"0 命中（红线守住）"| OUTER
    COMMON -.->|"0 命中（红线守住）"| OUTER
    CORE -.->|"0 命中（红线守住）"| DLV

    appToModel["app → model（97 文件）"]:::hidden
    REPO --> MODEL
    UC --> MODEL
    DOM --> MODEL
    GRP --> MODEL

    REPO --> CORE
    UC --> CORE
    CORE --> COMMON
    appToCommon["app → common（35 文件）"]:::hidden
    UC --> COMMON
    DLV --> COMMON

    CORE -.->|"★ 唯一违例 1 文件<br/>core/queue/pubsub_broadcaster.go:10"| MODEL
    UC -.->|"具体基建未抽象<br/>core/jpush:9 · core/supabase:22 · phone_otp:9"| CORE

    goWork["go.work:8 use"] -.->|"零 import<br/>仅 docker/Dockerfile:31 COPY"| THUNDER

    MAIN["cmd/api/main.go:76<br/>唯一 DI 装配点"] --> DLV
    MAIN --> GRP
    MAIN --> WS2
    MAIN --> UC
    MAIN --> REPO
    WMAIN["cmd/worker/main.go:35<br/>第二条独立 AMQP 连接"] --> WKR
  end

  classDef hidden display:none;
  style MODEL fill:#2d6a4f,color:#fff
  style COMMON fill:#3d405b,color:#fff
  style CORE fill:#4a5b7a,color:#fff
  style THUNDER stroke-dasharray: 5 5
```

> 说明：`appToModel` / `appToCommon` 两个隐形节点仅用于承载"命中文件数"标注，不参与渲染布局。

**读图要点**：
1. 圆心是 `model`，没有任何出向实线 —— 依赖倒置在这里是成立的。
2. 唯一的虚线违例（`core → model`）是**基建认识领域**，修法是给 `core/queue` 定义一个 `Broadcaster` 端口、把 `entity` 参数换成 `core` 自己的载荷结构，由 `app` 侧做映射。
3. `usecase → core` 的三条具体依赖（jpush / supabase / phone_otp）与同层已抽象的 rongcloud 并存 —— 说明这不是设计选择，是遗漏。

---

## 6. 数据模型与存储

### 6.1 实体与表全量清单

`model/entity/` 共 30 个 `.go` 文件、101 个 struct，其中 **55 个带 `TableName()`（即 GORM 表实体）**，分布在 22 个文件（`grep -rn "TableName() string" model/entity | wc -l` = 55）。另有 **53 个表名常量**（`grep -rn 'Table = "' model/entity`）。

**账号 / 认证**

| 表 | 实体 | 实体声明 | `TableName()` | 表名常量 | 要点 |
|----|------|---------|--------------|---------|------|
| `users` | `User` | `user.go:16` | `user.go:36` | `user.go:6` | **字符串主键** `user_id;size:64`（`:17`）；`status int16 default:0` + 枚举 0/1/2（`:10-12,22`）；`DeletedAt *time.Time`（`:33`，**不是 `gorm.DeletedAt`** → 不触发 GORM 软删语义）；`PasswordHash json:"-"`（`:21`） |
| `auth_refresh_tokens` | `AuthRefreshToken` | `auth_user.go:9` | `:17` | `:6` | `TokenHash uniqueIndex`（`:12`） |
| `profiles` | `Profile` | `profile.go:10` | **无** | `profile.go:7` | 纯读模型；文件头 `:1-2` 注明「本地不建 profiles 表，`ProfilesTable` 仅供 Supabase Cloud 查询」 |
| `auth_refresh_tokens` 的签发/撤销 | — | — | — | — | 见 `app/internal/usecase/`（refresh token 流程） |

**交易 / 分析**

| 表 | 实体 | 声明 | `TableName()` | 要点 |
|----|------|------|--------------|------|
| `transactions` | `Transaction` | `transaction.go:11` | `:23`（const `:9`） | `UserID *string size:64 index`（`:13`）；`Date string size:32` 存 `YYYY-MM-DD`（`:17` 注释）；`Amount float64` |
| `transaction_records` | `TransactionRecord` | `transaction_record.go:9` | `:22-24`（**字面量，无常量**） | `:21` 注释自标「遗留 uint 交易表（已弃用）」；`UserID uint`（`:11`）与 `Transaction` 冲突 → 启动时改名（6.3） |
| `analytics_records` | `AnalyticsRecord` | `analytics_record.go:7` | `:45` | 宽表指标 `MetricPV/UV/Click/Convert int64` + `MetricRevenue/Cost/ROI/BounceRate float64`（`:21-28`）；**所有时间字段为 Unix 秒 `int64`**（`:37-42`，文件头 `:1` 注释） |

**权限 / 门店**

| 表 | 实体 | `TableName()` | 常量 | 要点 |
|----|------|--------------|------|------|
| `wys_store` | `WysStore` | `access.go:41` | `:25` | **数字主键** `StoreID int` |
| `wys_store_member` | `WysStoreMember` | `:52` | `:26` | **复合主键** `UserID string` + `StoreID int`（`:45-46`）；`Position int16`（`:47`）枚举见 `store.go:10-12` |
| `role` | `Role` | `:61` | `:27` | `Code string primaryKey`；scope `platform`/`store`（`:11-12`） |
| `permission` | `Permission` | `:69` | `:28` | 能力码常量 `:14-21` + `mall.go:50` |
| `role_permission` | **无实体** | — | `:29` | 只由 SQL 建：`app/internal/repository/postgres/sql/access_control.sql:32` |
| `user_role` | `UserRole` | `:80` | `:30` | `StoreID *int`（平台行为空，`:71` 注释） |
| `wys_user_store_stats` | `WysUserStoreStats` | `store.go:43` | `:6` | 复合主键 + `Role int16 default:0`（`:34`）+ 4 个计数列（`:35-38`） |

**社区（5 张表全部用字面量、无表名常量）**：`wys_topics`（`community.go:25`）、`wys_posts`（`:48`）、`wys_user_follows`（`:62`）、`wys_post_likes`（`:71`）、`wys_post_comments`（`:84`）。均使用 **`uuid.UUID` 主键**（`:18,29,76`），`WysPost.ImageURLs []byte type:jsonb default:'[]'`（`:33`），计数列 `LikeCount/CommentCount/Heat`（`:39-42`，热度公式 `PostHeat` `:51`）。

**短视频**：`wys_short_videos`（`short_video.go:37`，const `:11`；`Status` 0=审核中/1=正常 `:14-15`；`Duration string default:'0:15'`、`AspectRatio float64 default:1.25`）、`wys_short_video_likes`（`:46`，复合主键）。

**首页待办**：`wys_store_join_application`（`home_todo.go:40`）、`wys_store_customer`（`:53`）、`wys_after_sales_appointment`（`:66`，`AppointmentDate gorm:"type:date"` `:60`）、`wys_store_review_order`（`:78`）；`wys_home_todo_packing_demo` **无实体**，只在 `sql/home_todo_schema.sql:58` 与 `app/internal/home/repo_seed.go:62` 出现。

**积分 / 现金钱包 / 会员**：`wys_points_wallet`（`points.go:37`）、`wys_points_ledger`（`:50`，`Delta/Balance int64` + `Reason` 枚举 `:11-16`）、`wys_check_ins`（`:61`，复合主键 `UserID`+`Day`，`yyyy-MM-dd Asia/Shanghai` `:55` 注释）、`wys_task_claims`（`:72`，**三元复合主键** `UserID`+`Day`+`TaskCode` `:65-67`）；`wys_cash_wallet`（`cash_wallet.go:34`，**金额以「分」为单位** `BalanceFen int64`）、`wys_cash_ledger`（`:47`）、`wys_bank_card`（`:61`，`:49` 注释「不存完整卡号」，仅 `CardLast4` `:54`）；`wys_membership_entitlement`（`membership.go:34`，复合主键 `UserID`+`Tier`；IAP 字段 `:28-30`）、`wys_membership_order`（`:55`）。
> `MembershipPlan`（`membership.go:58`）与 `MembershipCatalog()`（`:72-81`）是**内存目录、非表**：6 个套餐价格硬编码。

**商城（10 实体 + 1 常量无实体，表名常量集中在 `mall.go:53-65`）**：`wys_mall_category`（`:152`）、`wys_mall_product`（`:169`）、`wys_mall_sku`（`:222`）、`wys_mall_virtual_code`（`:234`）、`wys_mall_cart_item`（`:244`，复合主键 `UserID`+`SKUID`）、`wys_mall_order`（`:266`，**幂等键 `IdempotencyKey`** `:252`）、`wys_mall_order_item`（`:287`）、`wys_mall_order_log`（`:299`，状态迁移审计）、`wys_mall_payment`（`:317`，`ChannelPayload json.RawMessage type:jsonb` `:311`）、`wys_mall_audit_log`（`:330`，`Detail jsonb` `:326`）；`wys_mall_refund`（常量 `:63`）**无 Go 实体**，表由 `sql/mall_schema.sql:184` 建立。
> **金额精度处理**：`WysMallSKU.Price string gorm:"type:numeric(10,2)"`，`:210` 注释「金额用字符串承载 numeric，避免 float」；同时保留 `PricePoints int64`（积分价）与 `Version int`（乐观锁）。支付窗口 `MallOrderPayTTL = 15 * time.Minute`（`:34`），惰性判定 `MallOrderPayDeadline`（`:123`）/`MallOrderPayExpired`（`:132`）。

**汽车业务**：`wys_deal_invoice`（`deal_invoice.go:68`）、`wys_used_car_order`（`used_car_order.go:148`，**全仓唯一用 `float64` 存业务金额** `:138` ⚠️ 待确认 12.2）、`wys_new_car_follow_file`（`new_car_follow.go:141`，`FollowLevel string size:1` 枚举 H/A/B/E `:24-29`；`Stage` 0..5 `:15-20`）、`wys_new_car_follow_log`（`:214`）、`wys_after_sales_record`（`after_sales_record.go:32`）、`wys_auto_finance_product`（`auto_finance.go:31`，**BPS 基点表**：`AnnualRateBPS int`（`:18` 注释 `450 = 4.50%`）、`MinDownPaymentBPS`（2000=20%）；金额一律 `...Fen int64`；`Enabled bool default:true` `:26`）。

**IM（6 张）**：`wys_im_friend_request`（`im.go:32`，**字符串主键** `:24`；`Status string default:pending` `:13-15`）、`wys_im_friendship`（`:41`，`:34` 注释「user_a < user_b 字典序唯一」）、`wys_im_free_group`（`:53`）、`wys_im_free_group_member`（`:62`）、`wys_im_store_group`（`:72`，`StoreID string` 且 `GroupID uniqueIndex`——一店一群 `:67`；⚠️ 此处 `store_id` 是 **string**，而其他域是 `int`，见 12.2）、`wys_im_message_backup`（`:88`，`MessageUID uniqueIndex` 作幂等键 `:74` 注释；`Payload string type:text` **不是 jsonb**）。

**其他**：`wys_user_address`（`address.go:26`，`:8` 注释「订单只快照字段，不存 address_id」）、`wys_push_devices`（`push_device.go:20`，`Platform` 枚举注释 `:12`，`Mock bool` `:15`）。

**非表实体（读模型 / 契约对象，无 `TableName()`）**：`RealtimeEnvelope`（`realtime_envelope.go:14`，WS 包络）、`SSEFrame/SSEEventData/SSEErrorBody/ChatTurn`（`sse_event.go:36,42,55,67`）、`SysNotifyAction/SysNotifyPayload/ScheduledSysNotifyOpts`（`sys_notify_payload.go:10,18,37`）、`AuthorProfile`（`author_profile.go:8`，`:3-7` 注释说明是跨域读模型），以及各域 DTO（`WalletSummary`、`PointsStatus`、`HomeTodoCard`、`MallOrderDetail`、`AddressView`、`NewCarFollowFileDTO`、`UsedCarOrderDTO`、`DealInvoiceDTO`、`FinanceProductDTO`、`ShortVideoStats`、`UserStoreStats`、`Profile`/`UpdateProfileInput`）。

### 6.2 `wys_` 前缀规范与例外

规范出处：`AGENTS.md` / `docs/local-auth-postgres.md` §3。实测：**49 张表带 `wys_` 前缀，10 张不带**。

不带前缀的例外：`users`（`user.go:6`）、`profiles`（`profile.go:7`）、`auth_refresh_tokens`（`auth_user.go:6`）、`transactions`（`transaction.go:9`）、`transaction_records`（`transaction_record.go:23`）、`analytics_records`（`analytics_record.go:4`）、`role`（`access.go:27`）、`permission`（`:28`）、`role_permission`（`:29`）、`user_role`（`:30`）。

另外 `wys_` 前缀内部**单复数混用**：多数单数（`wys_store`/`wys_mall_order`/`wys_points_wallet`），复数的有 `wys_check_ins`（`points.go:8`）、`wys_task_claims`（`:9`）、`wys_push_devices`（`push_device.go:5`）、`wys_short_videos`/`wys_short_video_likes`（`short_video.go:11-12`）与社区 5 张（`community.go`）。

历史遗留：`migrations/20260921000000_create_wys_store_stats.up.sql:2,10,21` 曾建 `wys_stores`/`wys_store_members`/`wys_store_customers`，后被 `...160000_access_control` 的 `wys_store`/`wys_store_member` 和 `20260922160000_home_todo.up.sql:19` 的 `wys_store_customer` 取代。

### 6.3 建表路径：四条 → 一条闸门（2026-11-03 已收口，parity-p0-1）

历史上**同一个库里的表由四种机制建立**，每次启动全跑一遍，多实例并发会撞 DDL 锁与 42701。现在全部收进 `app/internal/schemaboot.Boot()`，由单一开关 `database.auto_migrate` 守卫（`APP_ENV=lan|prod` 由 `common/config.NoDDLAppEnv`（`config.go:278`）强制关闭）：

| 原路径 | 机制 | 现状 |
|--------|------|------|
| **A. GORM `AutoMigrate`**（13 模型） | `main.go:635 autoMigrate` → `main.go:648 ConsolidateLocalUsers` + `db.AutoMigrate(...)` | 变成 `schemaboot` 的 **Required** 步骤（`main.go:127`）：`auto_migrate=true` 才跑，失败即拒启（原来也拒启） |
| **B. 域内 `EnsureSchema`**（8 处，失败仅 Warn） | 会员 / 成交发票 / 二手车 / 新车跟进 / 售后 / 社区索引 / 推送设备 / IM | 变成 `schemaboot` 的 **BestEffort** 步骤（`main.go:130-137`），同样受开关守卫；正式环境不再执行 |
| **C. `ConsolidateLocalUsers`（10 段 embedded SQL）** | access/mall/address/points/wallet/membership/home_todo/new_car_follow | 仍在 A 里（`postgres/consolidate_users.go:41-73`）；这些表已由 `migrations/20260923167500_align_gorm_only_schema` + 既有迁移覆盖 |
| **D. `migrations/`** | golang-migrate CLI | **权威真相**：25 对 up/down，空库端到端首次跑通（此前跑到 `20260923170000` 即断） |

Worker 走同一个闸门（`app/cmd/worker/main.go:90`，BestEffort 只有极光设备表）。

`autoMigrate` 里仍有一步**破坏性前处理**：若 `transactions.user_id` 的 `data_type` 是 `bigint`/`integer`，直接 `ALTER TABLE transactions RENAME TO transactions_legacy_uint`（`main.go:644`），旧数据即脱离服务路径；探测 SQL 的错误被 `_ =` 丢弃。该步现在同样只在 `auto_migrate=true` 时发生。

无 Go 实体、只靠 SQL 存在的表：`role_permission`（`access_control.sql:32`）、`wys_mall_refund`（`mall_schema.sql:184`）、`wys_home_todo_packing_demo`（`home_todo_schema.sql:58`，另在 `app/internal/home/repo_seed.go:62` 有一份 `CREATE TABLE IF NOT EXISTS` —— 种子链路属 P0-5，本片白名单外，登记为残留）。

### 6.4 Migrations

| 目录 | 格式 | 数量 | 说明 |
|------|------|------|------|
| `migrations/` | golang-migrate（`.up.sql` / `.down.sql` 成对） | **50 个文件（25 对，down 全覆盖）** | 版本 `20240101000000` ~ `20260924110000`；`20260923167500_align_gorm_only_schema` 补 AutoMigrate-only 的 14 张表，`20260924110000_align_column_drift` 对齐 146 处列定义 |
| `supabase/migrations/` | 裸 SQL，注释要求「在 Supabase Dashboard → SQL Editor 执行」 | 2 个 | `003_transactions_user_id_rls.sql`、`004_fix_transactions_rls_policies.sql` |
| `app/internal/repository/postgres/sql/` | embedded，运行时执行 | 13 个 | 由 `ConsolidateLocalUsers` 跑（6.3 路径 C），同样受 `auto_migrate` 开关守卫 |

`make migrate-up` / `migrate-down` / `migrate-status` / `migrate-baseline` / `migrate-force`（`Makefile:68-105`）调 golang-migrate CLI；CLI 由 `go install` 按需构建到 `bin/migrate`（版本 pin `v4.15.2`，**必须带 `-tags "postgres pgx"`**，否则报 `unknown driver postgres`）。DSN 不再硬编码，改从 `.env` / `.env.local` 的 `DATABASE_*` 拼装（收掉 H11 点名的 `postgres:postgres` 残留）。`migrate-down` 每次只回退 **1** 个版本。

`migrations/` 中**未找到**任何 `CREATE POLICY` / RLS 语句（`grep -rln "row level security\|CREATE POLICY" migrations/` → 空）。

### 6.5 RLS（仅 Supabase 侧）

本地 Postgres 无 RLS；RLS 只作用于 Supabase 的 `public.transactions`：

| 语句 | 位置 | 内容 |
|------|------|------|
| 加列 | `supabase/migrations/003_..._rls.sql:5-6` | `user_id uuid references auth.users(id) on delete cascade` |
| 索引 | `:8` | `transactions_user_id_idx` |
| 启用 | `:11` | `enable row level security` |
| 清场 | `:14-24` | 匿名块遍历 `pg_policies` **删除 transactions 上全部既有策略** |
| 4 条策略 | `:26-45` | `transactions_{select,insert,update,delete}_own`，TO `authenticated`，USING/WITH CHECK `auth.uid() = user_id` |
| 自检 | `:47-54` | 期望 `rls_enabled = true`、`policy_count = 4` |

`004_fix_transactions_rls_policies.sql` 存在的原因是 **`:2` 记录的真实故障**：「PostgreSQL 多条 SELECT 策略是 **OR** 关系，有一条放行即可见全部数据」—— 实测 `policy_count=8` 时数据泄露。修法：先列全部策略（`:5-9`）→ 全量 drop（`:11-22`）→ 重新 enable（`:25`）→ 只重建 4 条 own 策略（`:28-47`）→ 双重验证（`:49-62`）。

校验脚本 `scripts/check_transactions_rls.sh`（`make check-rls`，`Makefile:155-156`）做**两层对照**：① 用户 access token 直连 PostgREST（`:63-66`）测数据库层，`:76-79` 判 `count==0` ⇒ RLS 生效；② 经 Go 后端 `/api/v1/transactions`（`:70-72`）测应用层过滤，`:84-89` 判定。凭据要求 `:12-16,25-28`。

应用层配合：`app/internal/transaction/repo_supabase.go:5-8` 注释「绝不能用 service_role 替代，否则会绕过 RLS」；`:42`「双保险：`Eq user_id` + RLS」。

### 6.6 Redis Key 清单

Redis 客户端在 `core/database/redis.go:14-29`（`Ping` 超时 5s `:21-26`）。**不存在 `core/redis` 包**；业务 key 全在 `app/internal/repository/redis/`。

| Key | 定义 | 用途 | TTL 策略 |
|-----|------|------|---------|
| `auth:session:{userID}` | `session_repo.go:14` | 单设备会话（`repository.DeviceSession` JSON） | `ttl<=0` → `Set(key,raw,0)` **永不过期**（`:45-46`）；否则 `Set(...,ttl)`（`:48`）。ttl 来自 `cfg.Auth.SessionTTL()`（`device_session_usecase.go:81,124,198`）。删除 `Del`（`:52`）。`ListActiveUserIDs` 用 `Scan(match="auth:session:*", count=100)`（`:56-78`） |
| `ws:ticket:{ticket}` | `ws_ticket_repo.go:22` | WS 一次性鉴权票据 | `Set(...,ttl)`（`:38`），ttl = `cfg.Realtime.TicketTTL()`（默认 120s，`config.go:518-523`；`config.yaml:51`）；**`Consume` 用 `GetDel` 原子取删**（`:43`，`:5-8` 头注释解释防两 WS 并发复用），`redis.Nil` → 「ticket 无效或已过期」（`:45`） |
| `sse:conv:{userID}:{conversationID}` | `conversation_repo.go:15`，拼装 `:27-29` | SSE 停留会话（`[]entity.ChatTurn`，会话 ID 形如 `conv_<uuid>` `:33`） | 三处均有 `ttl<=0 → 30min` 兜底（`Create :38-40`、`SaveTurns :75-77`、`Touch :84-86`）；`Touch` 用 `Expire` **滑动续期**（`:88`），`!ok` → `ErrConversationNotFound`（`:93`） |
| `presence:online` | `presence_repo.go:22` | 在线用户 Set | — |
| `presence:user:{userID}` | `presence_repo.go:23` | 单用户连接集合 | — |
| `realtime:seq:{userID}` | `realtime_event_repo.go:14` | 事件单调序号（`NextSeq`） | 不过期 |
| `realtime:events:{userID}` | `realtime_event_repo.go:15` | 事件流（供 `/realtime/sync` 补拉），裁剪到 `event_retention` | 不过期，靠 `LTRIM` |
| `scheduler:dedup:{taskID}` | `app/internal/delivery/worker/handler.go:28,135-143` | 整点广播持久去重（`SetNX`） | **TTL 2h**（`:136`）；入队失败回删（`:154`） |
| `realtime:fanout`（Pub/Sub 频道，非 key） | `common/config/config.go:406`，默认 `:434-439` | 多实例 WS 扇出 | — |

### 6.7 Elasticsearch

| 项 | 值 | 证据 |
|----|----|------|
| 索引名 | `transactions`（常量 `TransactionsIndex`） | `app/internal/transaction/repo_es.go:20` |
| mapping | 内联 JSON 字面量，`PUT /{index}` | `repo_es.go:39-71`（mapping 起 `:41`） |
| 文档写入 | `PUT /transactions/_doc/{id}`，`id` = 交易主键字符串 | `repo_es.go:74-115`（URL `:100`） |
| 删除 | `DELETE /transactions/_doc/{id}` | `:120` |
| 搜索 | `POST /transactions/_search` | `:180` |
| 重索引工具 | `scripts/reindex_transactions_es.sh`（首行自标 "Partial…Deferred"） | — |
| 索引失败可观测 | `es_transaction_index_failures_total` Counter | `common/metrics/metrics.go:31-36`；打点 `app/internal/transaction/service.go:98,110` |
| 社区搜索索引 | `community.EnsureSearchIndexes` 现为 `schemaboot` BestEffort（失败仅 Warn） | `main.go:135`；trigram 迁移 `migrations/20260923170000_community_search_trgm`（正式环境只走迁移） |

### 6.8 双实现与运行时选择

| 端口 | local 实现 | Supabase 实现 | 选择点 |
|------|-----------|--------------|--------|
| `ProfileRepository` | `app/internal/repository/postgres/profile_repo.go:1`（「本地资料读写统一 `users` 表」） | `app/internal/repository/supabase/profile_repo.go:1`（PostgREST 打 `profiles`） | `app/cmd/api/main.go:275` |
| 交易仓储 | `app/internal/transaction/repo_postgres.go` | `app/internal/transaction/repo_supabase.go` | `main.go:258`（local）/ `:279`（supabase） |
| 鉴权 token 校验 | `common/jwt.Manager.ParseUUID` | `common/auth.ValidateAccessToken` → `GET {SUPABASE_URL}/auth/v1/user` | 中间件二选一：`main.go:441-448` |
| 门店统计 | **只有 Postgres** | — | `main.go:276`（即使 supabase 模式，storeStatsRepo 仍是 Postgres → 单请求跨两库） |

开关：`auth.provider`（`common/config/config.go:167-169`，env `AUTH_PROVIDER` `:660`）+ `SupabaseConfig.Enabled()`（`url` 与 `anon_key` 均非空，`config.go:360-376`）。判定入口 `router.go:69`（`businessAuth := opts.Config.Auth.IsLocalProvider() || opts.Supabase.Enabled()`）。

---

## 7. 对外契约清单

**总量**：145 条硬编码路由注册 + 1 条路径可变（WS）= **146 个 HTTP 端点**；gRPC 1 服务 2 方法；WS 6 类入站帧；AMQP 4 类任务。

### 7.1 HTTP 路由总表

图例：`SA` = SessionAuth 链（`sessionAuth` + 可选 `accountGate`，见 9.1）；`pub` = 无鉴权。

**管理与传输（`app/internal/delivery/http/router/router.go`）**

| 方法 | 路径 | 处理 | 注册 | 鉴权 |
|------|------|------|------|------|
| GET | `/health` | 内联 `200 {status:ok}` | `:62-64` | pub |
| GET | `/metrics` | `gin.WrapH(metrics.Handler())` | `:66` | **pub**（注释自述「仅本机 / Docker 内网可达即可」） |
| GET | `{realtime.ws_path}`（默认 `/realtime/v1/connect`） | `ws.Handler.ServeWS` | `:106-112` | ticket（首帧） |

**用户与账号（`router/user_routes.go`，组 `/api/v1/user` `:13`）**

| 方法 | 路径 | 处理 | 行 | 鉴权 |
|------|------|------|----|------|
| POST | `/register` `/login` `/refresh` | `UserHandler.{Register,Login,Refresh}` | `:15,16,17` | pub |
| POST | `/phone/otp/send` `/phone/otp/verify` | `SendPhoneOTP` / `VerifyPhoneOTP` | `:18,19` | pub |
| POST | `/wechat/login` `/huawei/login` | `LoginWithWechat` / `LoginWithHuawei` | `:20,21` | pub |
| POST | `/logout` `/deactivate` | `Logout` / `Deactivate` | `:25,26`（`append(chain, …)` —— **逐路由挂**） | SA |
| GET/POST | `/addresses` | `AddressController.{List,Create}` | 组 `:28`，`:29,30` | SA |
| GET/PATCH/POST/DELETE | `/addresses/:address_id`（含 `/default`） | `Get/Update/SetDefault/Delete` | `:31,32,33,34` | SA |

**资料（`router/profile_routes.go`）** — `/api/v1/profiles`（`:13`，`Use :14`）：`GET /me`、`PATCH /me`、`POST /me/store`（`:16,17,18`）；`/api/v1/me`（`:21`，`Use :22`）：`GET /profile`、`PATCH /profile`、`GET /stores`（`:24,25,26`）← 新旧两套路径并存。

**Realtime（`router/realtime_routes.go`，`/api/v1/realtime` `:11`，`Use :12`）**：`POST /ws-ticket`（`:14`）、`POST /sync`（`:15`）、`POST /push`（`:16`）。

**推送（`router/jpush_routes.go`，`/api/v1/push` `:13`，`Use :14`）**：`POST /devices`（`:15`）、`POST /send`（`:16`）。

**IM（`router/im_routes.go`，`/api/v1/im` `:13`，`Use :14`；15 条 `:16-30`）**：`POST /session`、`GET /users/search`、`GET /users/profile`、`GET /friends`、`GET /friends/requests`、`POST /friends/requests`、`POST /friends/requests/:id/respond`、`GET /private/admission`、`POST /groups/free`、`POST /groups/free/:id/{invite,kick,quit,dismiss}`、`POST /groups/store/sync`、`POST /messages/backup`。

**流式**：`POST /api/v1/sse/completions`（`sse_routes.go:11,12,14`，**仅当 `cfg.SSE.Enabled`** —— `router.go:93`）；`POST /api/v1/cs-agent/chat`（`cs_agent_routes.go:10,11,13`，**仅当 `cfg.CsAgent.Enabled`** —— `router.go:97`）。

**已切片域（每域一个 `Router.Register(engine)`，由 `router.go:117-121` 遍历 `opts.DomainRouters` 挂载）**

| 组前缀 | 定义 | 链挂载 | 端点（行号） |
|--------|------|--------|-------------|
| `/api/v1/transactions` | `transaction/router.go:43` | `:44` | `GET ""`（`:46`，无 `page` 时走 Flutter 格式）、`POST ""` `CreateLegacy`（`:47`）、`GET /search`（`:48`，**注释「须在 /:id 之前」**）、`GET/PUT/DELETE /:id`（`:49,50,51`） |
| `/api/v1/transactions/manage` | `:55` | `:56` | `GET/POST ""`（`:58,59`）、`GET/PUT/DELETE /:id`（`:60,61,62`） |
| `/api/v1/mall` | `mall/router.go:39` | `:40` | 12 条：`POST /products`（`:42`）、`POST /products/:product_id/skus`（`:43`）、`POST /skus/:sku_id/codes`（`:44`）、`GET /stores/:store_id/products[/:product_id]`（`:46,47`）、`POST/GET /cart`（`:49,50`）、`GET/POST /orders`（`:52,53`）、`GET /orders/:order_id`（`:54`）、`POST /orders/:order_id/{pay,cancel}`（`:55,56`） |
| `/api/v1/community` | `community/router.go:37` | `:38` | 12 条：`GET /topics`（`:40`）、`/topics/search`（`:41`）、`/search`（`:42`）、`POST/GET /posts`（`:44,45`）、`DELETE /posts/:id`（`:46`）、`POST/DELETE /posts/:id/like`（`:47,48`）、`GET/POST /posts/:id/comments`（`:49,50`）、`POST/DELETE /users/:id/follow`（`:52,53`） |
| `/api/v1/short-videos` | `short_video/router.go:37` | `:38` | 8 条（`:40-47`）：`POST/GET ""`、`GET /profile`、`GET/DELETE /:id`、`POST/DELETE /:id/like`、`POST /:id/view` |
| `/api/v1/new-car-follow-files` | `new_car_follow/router.go:36` | `:37` | 8 条（`:39-46`）：`/summary`、`/customers`、`""`(GET/POST)、`/:file_id`(GET/PATCH)、`/:file_id/logs`(GET/POST) |
| `/api/v1/deal-invoices` | `deal_invoice/router.go:36` | `:37` | 6 条（`:39-44`）：`/summary`、`/customers`、`""`、`/:invoice_id`、`PATCH /:invoice_id/resubmit` |
| `/api/v1/home` | `home/router.go:33` | `:34` | 7 条（`:36-42`）：`/todo-cards`、`/join-applications`、`POST /join-applications/:application_id/{approve,reject}`、`/follow-up-customers`、`/after-sales-appointments`、`/store-review-orders` |
| `/api/v1/stores`（home 侧） | `home/router.go:45` | `:46` | `POST /:store_id/join-applications`（`:48`） |
| `/api/v1/stores`（access 侧） | `access/router.go:37` | `:38` | `POST ""`、`POST /:store_id/members`、`DELETE /:store_id/members/:user_id`（`:40,41,42`） |
| `/api/v1/roles` | `access/router.go:45` | `:46` | `POST/DELETE /assignments`（`:48,49`） |
| `/api/v1/me`（access 侧） | `access/router.go:52` | `:53` | `GET /permissions`（`:55`） |
| `/api/v1/wallet` | `wallet/router.go:30` | `:31`（`shared.SessionAuthChain`） | 7 条（`:32-38`）：`""`、`/ledger`、`GET/POST /cards`、`POST /cards/:card_id/default`、`DELETE /cards/:card_id`、`/recharge` |
| `/api/v1/points` | `points/router.go:30` | `:31`（同上） | `GET /status`、`POST /check-in`、`GET /tasks`、`POST /tasks/:code/claim`（`:32-35`） |
| `/api/v1/membership` | `membership/router.go:39` | `:40` | 6 条（`:41-45,47`）：`GET /me`、`POST /buyout`、`/buyout/confirm`、`/huawei/verify`、`/apple/verify`、`/huawei/notifications` |
| `/api/v1/used-car-orders` | `used_car/router.go:37` | `:38` | 5 条（`:40-44`）：`/summary`、`/customers`、`""`、`/:order_id` |
| `/api/v1/after-sales` | `after_sales/router.go:36` | `:37` | `GET /pending-appointments`、`GET/POST /records`、`GET /records/:record_id`（`:39-42`） |
| `/api/v1/payments` | `payment/router.go:36`（**建组即带链** `chain...`） | 同左 | `POST /prepay`（`:37`） |
| `/api/v1/purchase-calculator` | `purchase_calculator/router.go:24` | **无链**（`:19-24` 注释「公开，无需 SessionAuth」） | `GET /products`（`:25`）、`POST /quote`（`:26`） |

> 注：`/api/v1/me` 与 `/api/v1/stores` 各被**两个不同域**分别 `Group`（`profile_routes.go:21` 与 `access/router.go:52`；`home/router.go:45` 与 `access/router.go:37`）。Gin 允许同前缀不同子路径，但意味着「同一 URL 前缀的鉴权与语义由两个模块分别负责」⚠️ 待确认 12.2。

### 7.2 响应信封与业务错误码

**两套响应格式并存**（Flutter 兼容性的直接后果）：

| 格式 | 结构 | 使用方 | 定义 |
|------|------|--------|------|
| **统一信封** | `{code, message, data, timestamp}` | 新建 / 已切片域 | `dto/response/common.go:29-34`；写出 `write()` `:125-132`；`Success` `:86`、`SuccessWithMessage` `:94`、`SuccessCreated` `:102`、`SuccessList` `:110`、`Error` `:121` |
| **Backend 直出** | 裸 JSON；错误为 `{error, code}` | 交易、认证等 Flutter 对标接口 | `dto/response/backend.go:11-15`；`BackendJSON` `:18`、`BackendError` `:23`、`BackendErrorWithCode` `:28`、`BackendNoContent` `:51` |

**分页约定**（仅统一信封）：`ListData[T]{list, pagination}`（`common.go:45-48`）、`Pagination{page,size,total,totalPages}`（`:37-42`）、`ParsePageQuery`（`:57-64`，**`size` 上限硬制 100** `:60-62`）、`Offset()`（`:67-69`）。

**业务错误码全量（`common.go:12-26`）**

| 码 | 常量 | 语义（含注释原文要点） |
|----|------|----------------------|
| 0 | `CodeSuccess` | 成功 |
| 10001 | `CodeInvalidParams` | 参数非法 |
| 10002 | `CodeUnauthorized` | `:15` 注释「登录凭证错误（如密码错误）；**勿与 token 失效混用**」 |
| 10003 / 10004 | `CodeForbidden` / `CodeNotFound` | 无权 / 不存在 |
| **10021** | `CodeSessionReplaced` | `:18` 注释「单设备 session：与登录 10002（密码错误）区分，**客户端按 code 分支勿依赖文案**」 |
| **10022** | `CodeSessionInvalid` | session 无效 |
| **10023** | `CodeTokenInvalid` | `:21` 注释「access JWT 无效 / 未带 Authorization；客户端 tryRefresh → 失败再清会话」 |
| **10024** | `CodeInvalidOTP` | `:23` 短信 / 邮箱验证码错误或过期 |
| 50000 | `CodeInternalError` | 服务端错误 |

**Backend 直出格式的 HTTP→业务码默认映射**：`backend.go:32-46`（401→`CodeTokenInvalid`、403→`CodeForbidden`、404→`CodeNotFound`、400→`CodeInvalidParams`、≥500→`CodeInternalError`、其余→`CodeInvalidParams`）。

**Supabase 错误归一**：`mapSupabaseAuthError` + `UserHandler.handleUsecaseError`（契约见 `AGENTS.md` §Agent 修改规范 4）。设备会话错误统一出口：`middleware/device_session_error.go:14-22`（401 + 10021/10022）；注意该文件用 stdlib `log.Printf` 而非 zap（`:17,20`）。

### 7.3 gRPC 契约

**唯一 proto**：`app/api/proto/analytics/v1/analytics.proto`

| 项 | 值 | 行 |
|----|----|----|
| package | `analytics.v1` | `:3` |
| go_package | `app/api/gen/go/analytics/v1;analyticsv1` | `:5` |
| service | `AnalyticsService` | `:8` |
| rpc | `ListAnalyticsRecords(ListAnalyticsRecordsRequest) returns (ListAnalyticsRecordsResponse)` | `:9` |
| rpc | `GetAnalyticsRecord(GetAnalyticsRecordRequest) returns (GetAnalyticsRecordResponse)` | `:10` |
| message | `ListAnalyticsRecordsRequest` / `…Response` / `GetAnalyticsRecordRequest` / `…Response` / `AnalyticsRecord` | `:13,18,25,29,34` |

**服务端与鉴权**：`app/internal/delivery/grpc/server.go:47-48`（`grpc.UnaryInterceptor(...)`）；`app/internal/delivery/grpc/interceptor/auth.go` —— `Authenticator` `:29`、`NewLocal` `:37`（内部 `jwtMgr.ParseUUID` `:68`）、`NewSupabase` `:42`（`ValidateAccessToken` `:74`）、`UnaryServerInterceptor` `:47-96`（含 `sessionUC.Validate` `:84-89`）；bearer 前缀解析 `:109`。装配 `main.go:546-551`。

**metadata 键**：`authorization` / `x-session-id` / `x-device-id`（与 HTTP SessionAuth 同一套会话；见 `AGENTS.md` 全局约束 6）。生成物 `app/api/gen/`，重生成靠 `make proto`（`Makefile:17-25`，含 protoc 依赖检查 `:18-20`）。

### 7.4 WebSocket 协议

**端点**：`GET {realtime.ws_path}`（`router.go:106-112`；配置 `configs/config.yaml:50`，回退默认 `/realtime/v1/connect` 在 `common/config/config.go:743-745`）。

**入站帧类型（`app/internal/delivery/ws/handler.go`）**

| `type` | 行 | 行为 |
|--------|----|------|
| `auth` | `:74` | 携 ticket 换连接；成功回 `auth_ok`（`:141`） |
| `ping` | `:76` | 回 `pong`（`:80`） |
| `sub` | `:83` | 订阅主题 |
| `unsub` | `:87` | 退订 |
| `event` | `:91` | 客户端上报事件 |
| 其他 | `:103`、`:118` | 回 `error` |

**鉴权与票据**：ticket 由已登录的 `POST /api/v1/realtime/ws-ticket`（`realtime_routes.go:14` → `controller/realtime_controller.go:35`）签发；消费端 `handler.go:98-125`，**`GetDel` 原子消费防重放**（`:114-115`；仓 `app/internal/repository/redis/ws_ticket_repo.go:43`）。

**出站帧类型**：`pong`（`:80`）、`error`（`:103,118`）、`auth_ok`（`:141`）、`ack`（`:200`），以及 Hub 广播的事件包络（`model/entity/realtime_envelope.go:14`）。

**传输限制**：单帧上限 64KB（`ws/client.go:33` `maxMsgSize`，`SetReadLimit` `:118`）；**发送缓冲 32，满则丢弃 + Warn、不阻塞**（`client.go:103-109`，缓冲构造 `:61`）→ 一致性含义见 12.1。

### 7.5 SSE 端点

| 端点 | 处理 | 门控 | 契约实体 |
|------|------|------|----------|
| `POST /api/v1/sse/completions`（AI 小石头） | `sse_routes.go:14` → `SseController.Completions` | `router.go:93`：`businessAuth && SseController != nil && sessionAuth != nil && cfg.SSE.Enabled` | `SSEFrame/SSEEventData/SSEErrorBody`（`model/entity/sse_event.go:36,42,55`）；请求 `dto/request/sse_completion.go`（含 `clientRequestId` `:7`） |
| `POST /api/v1/cs-agent/chat`（客服 Agent） | `cs_agent_routes.go:13` → `CsAgentController.Chat` | `router.go:97`（`cfg.CsAgent.Enabled`） | 同上帧格式；`clientRequestId` 缺省时服务端自生成 uuid（`usecase/cs_agent_usecase.go:55-56`） |

两路由**强制分开**（`AGENTS.md` 修改规范 9），配置亦独立（`configs/config.yaml:87-109`）。

### 7.6 CLI 契约（非网络入口）

| 命令 | 作用 | 前置 |
|------|------|------|
| `make import-supabase` / `import-supabase-dry` | 从 Supabase 导入存量数据 | 需 `service_role`（`Makefile:132` 注释；代码 `app/cmd/import-supabase/main.go:64`） |
| `make seed-analytics` | 埋点示例数据 | `app/cmd/seed-analytics/main.go` |
| `make trigger-hourly-notify` | 手工入队整点广播 | `Makefile:179-180` → `app/cmd/scheduler-trigger/main.go:37` |
| `seed-home-todo` | 首页待办种子 | **未在 Makefile 暴露目标** ⚠️ 12.2 |

---

## 8. 配置体系

### 8.1 加载顺序（`common/config/config.go:625-660` + `:724-730`）

```text
1) AddConfigPath(configPath) / SetConfigName("config") / SetConfigType("yaml")   :627-629
2) ReadInConfig()；失败即返回 err（config.yaml 缺失 = 启动失败）                  :631
3) 若 env 非空 → SetConfigName("config."+env) 再读一次（存在则覆盖同名键）
   ⚠️ 第二次 ReadInConfig 的错误被 `_ =` 丢弃 → config.dev.yaml 语法错误会静默      :636-641
4) AutomaticEnv() + 68 条显式 BindEnv（优先级最高）                                :643-718
5) Unmarshal(&cfg)                                                               :721-724
6) applySupabaseDefaults：读 configs/supabase.env，只填空字段                       :726,862-882
7) applyLocalEnvDefaults：读根目录 .env 与 .env.local，只填空字段                   :728,884-905
8) 校验与兜底：jwt.secret 非空、连接池/gRPC 端口/ws_path/ticket TTL/… 默认值         :732-760+
```

**关键设计**：第 6/7 步「只填空字段」意味着 **yaml / env 显式值 > supabase.env / .env(.local)**，而后两者是"团队常量 + 本地覆盖"的载体。`make run` 还额外用 `./scripts/load-env.sh` 把 `.env` + `.env.local` source 进进程环境（`Makefile:27-31`）。

### 8.2 配置结构

根结构 `Config`（`common/config/config.go:16-38`），18 个字段：

| yaml 键 | 字段 / 类型 | 定义 |
|---------|------------|------|
| `-`（来自 `APP_ENV`） | `AppEnv string` | `:17-18`（注释明确「与 gin server.mode 无关」） |
| `server` | `ServerConfig` | `:120` |
| `grpc` | `GRPCConfig` | `:126` |
| `database` | `DatabaseConfig` | `:132` |
| `redis` | `RedisConfig` | `:145` |
| `jwt` | `JWTConfig` | `:152` |
| `auth` | `AuthConfig` | `:164`（含 `SessionTTL()` `:190-196`、`IsSessionExempt` `:200-218`、`DevTestSessionExempt` `:221-226`、`IsLabAppEnv` `:239`、`DevBypassEnabled` `:292-296`） |
| `log` | `LogConfig` | `:380` |
| `supabase` | `SupabaseConfig` | `:360`（`Enabled()`、`DBKey()` `:371-376`） |
| `realtime` | `RealtimeConfig` | `:508`（`TicketTTL()` `:518-523`） |
| `queue` | `QueueConfig` / `QueueRabbitMQConfig` / `QueuePubSubConfig` | `:390 / :398 / :405` |
| `scheduler` | `SchedulerConfig` / `HourlyNotifyConfig` / `HourlyNotifyActionConfig` | `:453 / :460 / :471` |
| `sse` | `SSEConfig` / `SSEOpenAIConfig` | `:545 / :559` |
| `community` | `CommunityConfig` | `:102` |
| `short_video` | `ShortVideoConfig` | `:107` |
| `third_party` | `ThirdPartyConfig` | `common/config/third_party.go:7` |
| `elasticsearch` | `ElasticsearchConfig` | `:87` |
| `cs_agent` | `CsAgentConfig` | `:39` |

### 8.3 能力开关矩阵

| 开关 | yaml 键 | env | 默认 | 门控什么 | 判定点 |
|------|---------|-----|------|---------|--------|
| 业务总开关 | `auth.provider` / `supabase.url`+`anon_key` | `AUTH_PROVIDER` / `SUPABASE_URL` `:660,657` | local / 空 | `businessAuth`，决定几乎所有业务路由是否注册 | `router.go:69`；`main.go:93` |
| gRPC | `grpc.enabled` | `GRPC_ENABLED` `:647` | **未显式配置则默认 true**（`config.go:738-741`） | `:9090` 监听 | `main.go:545-571` |
| 队列 | `queue.enabled` | `QUEUE_ENABLED` `:667` | `false`（`config.yaml:59`）/ dev、lan、prod 为 `true`（`config.dev.yaml:29` 等） | API 侧入队器 + Pub/Sub 订阅；**Worker 拒绝启动** | `main.go:312`；`worker/main.go:44-46` |
| Push 异步 | `queue.push_async` | `QUEUE_PUSH_ASYNC` `:668` | dev/lan = `false` | 同步直投本地 Hub vs 入队 | `usecase/realtime_push_usecase.go:86-96` |
| ES | `elasticsearch.url` | `ELASTICSEARCH_URL` `:673` | `""` = 关闭（`config.yaml:68`） | 索引写 + 搜索 | `main.go:287-288`；`transaction/handler.go:309`（未就绪→**HTTP 503**） |
| 调度 | `scheduler.enabled` + `scheduler.hourly_notify.enabled` | `SCHEDULER_ENABLED` `:674` / `SCHEDULER_HOURLY_NOTIFY_ENABLED` `:675` | 总 `false`（`config.yaml:73`）；dev/lan 打开 `enabled` 但 `hourly_notify.enabled: false`（`config.dev.yaml:38-41`） | 整点广播 ticker | `core/queue/hourly_ticker.go` |
| SSE | `sse.enabled` | `SSE_ENABLED` `:677` | `true`（`config.yaml:88`） | `/api/v1/sse/completions` 是否注册 | `router.go:93` |
| 客服 Agent | `cs_agent.enabled` | `CS_AGENT_ENABLED` `:689` | `true`（`config.yaml:104`） | `/api/v1/cs-agent/chat` 是否注册 | `router.go:97` |
| 三方未配置降级 | `third_party.*` 各 key | 见 8.4 | 空 / 含 `PLACEHOLDER` | JPush / 融云 返回 `ErrNotConfigured`，usecase 吞掉继续 | `core/jpush/client.go:29-33,72-78`；`core/rongcloud/client.go:27-33`；`main.go:392,420` |
| 测试号 bypass | `auth.dev_test_*` | `AUTH_DEV_TEST_*` `:664-666` | dev 有值（`config.dev.yaml:18-20`）、lan 空 | OTP bypass + 多端豁免，**须 `APP_ENV≠prod`** | `config.go:221-226,292-296` |
| Session TTL | `auth.session_ttl_hours` | `AUTH_SESSION_TTL_HOURS` `:661` | `0` = **永不过期**（`config.yaml:32`） | Redis session 写入 TTL | `config.go:190-196`；`session_repo.go:45-46` |
| WS 多端上限 | `realtime.max_connections_per_user` | — | 兜底 3（`config.go:746-748`） | Hub 注册 | `configs/config.yaml:53` |

### 8.4 环境变量（68 条 `BindEnv`，`common/config/config.go:646-718`）

按用途分组（左列 = env 名，右列 = yaml 键）：

| 组 | 变量 |
|----|------|
| 服务 | `SERVER_PORT`(646) · `GRPC_ENABLED`(647) · `GRPC_PORT`(648) |
| 存储 | `DATABASE_HOST/PORT/USER/PASSWORD/DBNAME`(649-653) · `REDIS_ADDR/PASSWORD`(654-655) · `ELASTICSEARCH_URL`(673) |
| 鉴权 | `JWT_SECRET`(656) · `SUPABASE_URL`(657) · `SUPABASE_ANON_KEY` + 别名 `SUPABASE_KEY`(658) · `SUPABASE_SERVICE_ROLE_KEY`(659) · `AUTH_PROVIDER`(660) · `AUTH_SESSION_TTL_HOURS`(661) · `AUTH_SESSION_WHITELIST_USER_IDS`(662) · `AUTH_SESSION_WHITELIST_EMAILS`(663) · `AUTH_DEV_TEST_PHONE/OTP/PASSWORD`(664-666) |
| 队列 | `QUEUE_ENABLED`(667) · `QUEUE_PUSH_ASYNC`(668) · `RABBITMQ_URL/QUEUE/PREFETCH`(669-671) · `QUEUE_PUBSUB_CHANNEL`(672) |
| 调度 / 实时 | `SCHEDULER_ENABLED`(674) · `SCHEDULER_HOURLY_NOTIFY_ENABLED`(675) · `REALTIME_PUBLIC_WS_HOST`(676) |
| SSE | `SSE_ENABLED` · `SSE_PROVIDER` · `SSE_MAX_PROMPT_BYTES` · `SSE_MAX_TOKENS` · `SSE_KEEPALIVE_SECONDS` · `SSE_REQUEST_TIMEOUT_SECONDS` · `SSE_RATE_LIMIT_PER_USER_PER_MINUTE` · `SSE_CONVERSATION_TTL_SECONDS` · `SSE_MAX_TURNS` · `SSE_OPENAI_BASE_URL/MODEL/API_KEY`（`:677-688`） |
| Agent / 社区 | `CS_AGENT_ENABLED/PROVIDER/CHUNK_DELAY_MS/MODEL`(`:689-692`) · `COMMUNITY_ASK_EVERYONE_INVITE_USER_IDS`(`:693`) |
| 三方登录 | `WECHAT_APP_ID/APP_SECRET/MCH_ID/MCH_API_KEY/MCH_NOTIFY_URL`(`:694-698`) · `ALIPAY_APP_ID/PRIVATE_KEY/NOTIFY_URL`(`:699-701`) · `HUAWEI_CLIENT_ID/CLIENT_SECRET/REDIRECT_URI`(`:710-712`) |
| IAP | `HUAWEI_IAP_APP_ID/ISSUER_ID/KEY_ID/PRIVATE_KEY_PEM/SUBSCRIPTION_ROOT_URL`(`:702-706`) · `APPLE_IAP_SHARED_SECRET/BUNDLE_ID`(`:707-708`) |
| 推送 / IM | `JPUSH_APP_KEY/MASTER_SECRET/APNS_PRODUCTION`(`:713-715`) · `RONGCLOUD_APP_KEY/APP_SECRET/API_BASE_URL`(`:716-718`) |

另有两处**不经 BindEnv** 的直接读取：`APP_ENV`（`main.go:78`）与 `os.Getenv("GRPC_ENABLED")`（`config.go:739`，用于区分"未配置"与"配置为 false"）。

**兜底默认值（无 yaml 值时）**：AMQP `amqp://guest:guest@localhost:5672/`（`config.go:755`，队列名 `my_go_study.tasks` `:418-423`，prefetch 10 `:426-431`，Pub/Sub 频道 `realtime:fanout` `:434-439`）。

### 8.5 配置文件与 git 状态

| 文件 | 入库 | 内容 | 证据 |
|------|------|------|------|
| `configs/config.yaml` | ✅ | 基线（`git ls-files configs/` 命中） | `:1-109` 全部键见 8.3 |
| `configs/config.dev.yaml` | ✅ | 本机 `make run`：`log.level=debug`、测试号 OTP 三连（`:18-20`）、`queue.enabled=true push_async=false`（`:28-30`）、`scheduler.hourly_notify.enabled=false`（`:38-41`） | — |
| `configs/config.lan.yaml` | ✅ | `auth.provider=local`（`:11`）、测试号留空（`:13-15`）、队列开、整点关 | — |
| `configs/config.prod.yaml` | ✅ | `mode=release`、`log=info`、`sslmode=require`、`queue/scheduler/hourly_notify` 全开（`:3-18`） | — |
| `configs/supabase.env` | ✅ | **`SUPABASE_URL`（len 40）与 `SUPABASE_ANON_KEY`（len 46）均有实值** → `git ls-files configs/` 命中 | 与 `AGENTS.md` 配置表一致 |
| `configs/supabase.env.example` | ✅ | 模板 | — |
| `.env` | ✅ **入库且有实值** | 9 个键全非空：`APP_ENV SERVER_PORT DATABASE_HOST/PORT/USER/PASSWORD DBNAME REDIS_ADDR JWT_SECRET`。实测 `JWT_SECRET` == `config.yaml:28` 的弱默认 `change-me-in-production`，`DATABASE_PASSWORD` == compose 默认 `postgres`。入库是有意的（commit `afbc29f`「将 .env 纳入仓库，作为共享的本地开发默认配置」） | ⚠️ 见 12.1 |
| `.env.example` `.env.lan.example` `.env.local.example` | ✅ | 模板 | `git ls-files` |
| `.env.local` `.env.*.local` `.env.lan` | ❌ | `service_role` / 真实 IP | `.gitignore:5-8` |
| `go.work` `go.work.sum` | ❌ **不入库** | workspace 编排 | `.gitignore:32-34` → 致命性见 12.1 |
| `bin/ tmp/ logs/ .idea/ .vscode/ .codegraph/` | ❌ | 产物 | `.gitignore:1-3,10-23` |

> ⚠️ `make check-secrets`（`scripts/check-secrets.sh:13`）**只匹配 `sb_secret_*` 与 `SUPABASE_SERVICE_ROLE_KEY=sb_`**，因此 `.env` 里的 JWT / DB 口令不在拦截范围。详见 11.5。

### 8.6 Docker / compose

| 文件 | 用途 | 关键行 |
|------|------|--------|
| `docker/docker-compose.yml` | 全栈：PG + Redis + RabbitMQ + ES + app + worker + 观测 | 见下表 |
| `docker/docker-compose.lan.yml` | LAN overlay | `:11-40` 注入环境变量 |
| `docker/Dockerfile` | 镜像构建；`:31` COPY `thunder/go.mod`（workspace 成员必须存在） | — |
| `scripts/docker-compose.sh` / `lan-compose.sh` / `lan-run.sh` | 加载 env 后调 compose / 无 Docker 本机跑 | `Makefile:74-130` |

| 服务 | 镜像 | 宿主端口 | 行 |
|------|------|---------|----|
| `postgres` | `postgres:16-alpine` | `127.0.0.1:5432` | `:2-11` |
| `redis` | `redis:7-alpine` | `127.0.0.1:6379` | `:20-24` |
| `rabbitmq` | `rabbitmq:3.13-management-alpine` | `127.0.0.1:5672` + `15672` | `:31-36` |
| `elasticsearch` | `elasticsearch:8.15.3` | `127.0.0.1:9200` | `:43-52` |
| `app` | 本地构建 | **`8080:8080` + `9090:9090`（对全网卡）** | `:62,110-111` |
| `worker` | 本地构建 | 无 | `:121` |
| `prometheus` | `prom/prometheus:v2.54.1` | `127.0.0.1:9091` | `:158-172` |
| `alertmanager` | `prom/alertmanager:v0.27.0` | `127.0.0.1:9093` | `:174-184` |
| `feishu-adapter` | `python:3.12-alpine` | 无 | `:186-187` |
| `grafana` | `grafana/grafana:11.2.0` | `127.0.0.1:3000` | `:198-213` |

**端口暴露原则**：基础 compose 把 DB/Redis/ES 全绑 `127.0.0.1`（仅本机）；LAN overlay **不额外向局域网暴露 DB/Redis**，只让 API 的 `8080`（HTTP/WS）+ `9090`（gRPC）出网（`docker-compose.lan.yml:5-6` 注释）。⚠️ `/metrics` 亦因此对局域网可达，见 12.1。

---

## 9. 横切能力

### 9.1 中间件链

中间件全部在 `app/internal/delivery/http/middleware/`，**均为返回 `gin.HandlerFunc` 的构造函数，没有 struct 形式**。

| 构造函数 | 定义 | 职责 |
|----------|------|------|
| `Auth`（遗留 JWT） | `auth.go:20` | `jwtManager.Parse`（legacy uint claims `:36`）→ 注入 `userID`/`username`（`:43-44`）。**全仓无任何调用点** → 9.2 / 12.2 |
| `LocalSessionAuth` | `local_session_auth.go:15` | `jwtMgr.ParseUUID`（`:24`）+ `DeviceSessionUsecase.Validate`（`:39`） |
| `SupabaseSessionAuth` | `supabase_session_auth.go:20` | `authctx.ValidateAccessToken`（`:29`）+ 同一 session 校验（`:39`）→ 注入 user/token（`:45-46`） |
| `RequireActiveAccount` | `require_active_account.go:17` | 账号停用/注销门（`AccountChecker.AllowsRequest` `:29`；**checker 为 nil 时直接放行** `:19-22`） |
| `CORS` | `cors.go:11` | `Access-Control-Allow-Origin: *`（`:13`）+ `Allow-Credentials: true`（`:17`）；OPTIONS 直返 204（`:19-21`） |
| `RequestLogger` | `logger.go:12` | zap 访问日志：status/method/path/query/ip/latency/user_agent（`:22-30`）；5xx→Error、4xx→Warn、其余 Info（`:35-43`） |
| `PrometheusHTTP` | `prometheus.go:12` | Counter + Histogram；跳过 `/metrics`、`/health`（`:15-18`）；route 用 `c.FullPath()` 模板防高基数（`:22`） |
| `abortDeviceSessionError` | `device_session_error.go:14` | session 失效/互踢统一 401 + 业务码（`:16-22`）；**用 stdlib `log.Printf` 而非 zap**（`:17,20`） |
| `supabase_auth.go` | `supabase_auth.go:14-37` | 仅 `authctx` 的转发层；文件头 `:3-5` 自述「批 2~4 切完后可删除」 |

**全局链（全仓唯一一处 `r.Use` 连续注册）** —— `app/internal/delivery/http/router/router.go:56-60`：

```text
gin.New()
  1. gin.Recovery()            :57
  2. middleware.CORS()         :58
  3. middleware.PrometheusHTTP() :59
  4. middleware.RequestLogger(log) :60
```

**SessionAuth 从不全局注册**，只按组/按路由挂载：
- 构造一次于 `app/cmd/api/main.go:441-448`（local → `LocalSessionAuth`，否则 `SupabaseSessionAuth`），经 `router.Options.SessionAuth`（`:528`）传入；`router.go:71-78` 有 nil 时自行构造的兜底。
- 组合函数 `sessionAuthChain` = `[sessionAuth, accountGate?]`：`router/session_auth_chain.go:6-12`；域版本 `app/internal/shared/http.go:10`（`SessionAuthChain`）。
- 组级 `group.Use(chain...)` 全量清单（28 处 `Use`）：`transaction/router.go:44,56`、`home/router.go:34,46`、`access/router.go:38,46,53`、`after_sales/router.go:37`、`new_car_follow/router.go:37`、`deal_invoice/router.go:37`、`points/router.go:31`、`wallet/router.go:31`、`mall/router.go:40`、`community/router.go:38`、`membership/router.go:40`、`short_video/router.go:38`、`used_car/router.go:38`、`profile_routes.go:14,22`、`cs_agent_routes.go:11`、`realtime_routes.go:12`、`sse_routes.go:12`、`jpush_routes.go:14`、`im_routes.go:14`；`payment/router.go:36` 是**建组时直接带链**。
- 逐路由级：`user_routes.go:25,26`（`append(chain, userHandler.Logout)`）。
- 白名单（无 SessionAuth）：`/health`（`router.go:62`）、`/metrics`（`:66`）、WS（`:111`）、注册/登录/刷新/OTP/微信/华为（`user_routes.go:15-21`）、`/api/v1/purchase-calculator`（`purchase_calculator/router.go:19-24` 注释「公开」）。

### 9.2 两套鉴权体系（勿混用）

| 维度 | Session Auth（现行） | Legacy JWT `Auth` |
|------|---------------------|-------------------|
| 实现 | `local_session_auth.go:15` / `supabase_session_auth.go:20` | `auth.go:20` |
| token 解析 | `jwtMgr.ParseUUID`（`local_session_auth.go:24`）**或** HTTP `GET {SUPABASE_URL}/auth/v1/user`（`common/auth/supabase.go:33-45`，apikey=anon + Authorization=用户 token `:39-45`） | `common/jwt.Manager.Parse`（`auth.go:36`，**uint UserID claims**） |
| session 二次校验 | `DeviceSessionUsecase.Validate`（`local_session_auth.go:39`、`supabase_session_auth.go:39`） | **无** |
| 上下文键 | `ContextSupabaseUserKey` / `ContextAccessTokenKey`（`supabase_session_auth.go:45-46` → `authctx`） | `ContextUserIDKey` / `ContextUsernameKey`（`auth.go:43-44`） |
| 用户 ID 类型 | UUID string | uint |
| 挂载路由 | 除 9.1 白名单外全部业务路由 | **未找到任何调用点**（`grep -rn 'middleware\.Auth(' app common core model` → 0 命中） |
| gRPC 等价 | `delivery/grpc/interceptor/auth.go:47`（`NewLocal :37` / `NewSupabase :42`） | 无 |
| WS 等价 | 第三套：首帧 `auth` + 一次性 Redis ticket（`ws/handler.go:98-125`） | 无 |

> ⚠️ `user_routes.go:12` 仍接收 `jwtManager` 参数但函数体内**从未使用**（`:12-38` 无引用）→ 死参数。

### 9.3 Session 生命周期（单设备登录）

| 环节 | 位置 |
|------|------|
| Redis key | `auth:session:<userID>`（`repository/redis/session_repo.go:14`） |
| 写（登录签发） | `usecase/device_session_usecase.go:81` `IssueOnLogin → sessions.Save`；调用点 `delivery/http/handler/user_handler.go:310` |
| 写（refresh 续期） | `device_session_usecase.go:150-187 RenewOnRefresh` → `saveDeviceSession :189-202`；调用点 `user_handler.go:112` |
| 写（device_id 升级） | `device_session_usecase.go:121-127`（同 session 换 deviceID 允许覆写） |
| 读 | `session_repo.go:24-37`；`Validate` 读点 `device_session_usecase.go:112`；`RenewOnRefresh` 读点 `:177` |
| 删（logout） | `RevokeOnLogout :88-97` → `sessions.Delete :96`；repo `session_repo.go:51-53`；调用点 `user_handler.go:140`（远端会话错误被 `_ =` 吞掉 `:147`） |
| 扫描（广播用） | `ListActiveUserIDs` SCAN（`session_repo.go:56-78`）；worker 消费点 `delivery/worker/handler.go:122` |
| **单设备强制** | 新登录 `Save` 覆盖同 key（`session_repo.go:44-48`）→ 旧设备 `Validate` 命中 `stored.DeviceID != deviceID` → `ErrSessionReplaced`（`device_session_usecase.go:130-131`）→ 401 + code 10021（`middleware/device_session_error.go:16-18`） |
| 白名单豁免 | 配置 `auth.session_whitelist_user_ids` / `session_whitelist_emails`（`config.go:167-169`，env `:662-663`）→ `IsSessionExempt`（`config.go:200-218`）；测试号 `DevTestSessionExempt`（`:221-226`，须 `APP_ENV≠prod`，`DevBypassEnabled :292-296`）；生效点 `DeviceSessionUsecase.IsExempt :46-51`，**豁免账号 `IssueOnLogin` 只发 uuid、不落 Redis**（`:69-73`） |
| **TTL=0 永不过期** | `AuthConfig.SessionTTL()`：`SessionTTLHours<=0 → return 0`（`config.go:190-196`，注释原文「0 表示永不过期（仅 logout / 互踢删除）」）；`Save` 中 `ttl<=0 → Set(ctx,key,raw,0)`（`session_repo.go:45-46`）；实际配置 `configs/config.yaml:32` |

### 9.4 日志

- 初始化：`logger.Init(cfg.Log)`（`main.go:86-89`；worker `cmd/worker/main.go:48`）+ `defer logger.Sync()`（`main.go:90`）。
- 实现：`common/logger/logger.go:18-56` —— **双写**：console（文本→stdout）+ 可选 lumberjack 文件（JSON，轮转参数 `:37-51`）；`zap.AddCaller()`（`:53`）；级别解析 debug/info/warn/error（`:75-88`）。
- 配置：`configs/config.yaml:41-47`（`level: info`、`file: logs/app.log`、`max_size:100`、`max_backups:7`、`max_age:30`、`compress:true`）。
- **request-id / trace-id：未找到**。`RequestLogger` 字段列表（`middleware/logger.go:22-30`）不含任何关联 ID；`grep -i "RequestID|X-Request-Id|request.id"` 于 `middleware/logger.go` 与 `common/logger` 无命中。仅有业务级 `clientRequestId`（客户端传入或服务端自生成，`dto/request/sse_completion.go:7`、`usecase/cs_agent_usecase.go:55-56`），**不在中间件层、不覆盖普通请求** → 跨服务/跨日志无法串联系，见 12.2。

### 9.5 指标（Prometheus）

Registry：`common/metrics/metrics.go:13`（**独立 `prometheus.NewRegistry()`**，不用默认 registry），`init()` 注册 Go collector + Process collector + 3 个自定义指标（`:39-47`）；暴露 `metrics.Handler()`（`:50-52`）。

| 指标 | 类型 / 标签 | 定义 | 打点 |
|------|------------|------|------|
| `http_requests_total` | CounterVec `{method, route, status_class}` | `metrics.go:16-22` | `middleware/prometheus.go:28` |
| `http_request_duration_seconds` | HistogramVec `{method, route}`（buckets `metrics.go:27`） | `metrics.go:23-30` | `prometheus.go:29` |
| `es_transaction_index_failures_total` | Counter | `metrics.go:31-36` | `app/internal/transaction/service.go:98,110` |

抓取与告警：`deploy/observability/prometheus.yml:13-19`（job `app`，target `app:8080`，`metrics_path: /metrics`，`scrape_interval: 15s` `:2`）；`deploy/observability/alerts.yml`（`APIHigh5xxRate :4`、`APITargetDown :17-18`、`APIHighLatencyP99 :26`）；`alertmanager.yml` + 飞书 webhook 适配 `deploy/observability/feishu_adapter.py:1-2`；Grafana provisioning `deploy/observability/grafana/`；容器编排 `docker/docker-compose.yml:158-213`。

### 9.6 存在 / 缺失的能力矩阵

| 能力 | 状态 | 证据 |
|------|------|------|
| panic recovery | ✅ 仅 Gin 默认 | `router.go:57`（**无自定义 Sentry/日志桥接**） |
| HTTP 读头超时 | ✅ 10s | `main.go:531-535` |
| HTTP 整体 Read/Write/Idle 超时 | ❌ 未设 | `main.go:531-535` 只有 `ReadHeaderTimeout`；`grep "TimeoutHandler|gin.Timeout"` → 0 |
| 下游 client 超时 | ⚠️ **不一致** | jpush/融云 `http.Client{Timeout: 10s}`（`core/jpush/client.go:64`、`core/rongcloud/client.go:44`）；supabase admin `adminLookupTimeout = 5s`（`core/supabase/auth_admin.go:16`）；**但 `ValidateAccessToken` 用 `http.DefaultClient.Do`（无超时）**（`common/auth/supabase.go:47`） |
| 限流 | ⚠️ **仅 SSE/客服** | `MemoryRateLimiter`（`usecase/completion_usecase.go:311-320`，装配 `main.go:352`）；**无全局限流中间件** |
| CORS | ✅ 但配置有硬冲突 | `cors.go:11-25`；见 12.1 |
| 请求体大小限制 | ❌ HTTP 侧无 | `grep "MaxBytesReader|MaxMultipartMemory"` → 0；**仅 WS 单帧 64KB**（`ws/client.go:33,118`） |
| 分布式 tracing | ❌ | go.mod 无 otel/jaeger/zipkin 依赖；`.go` 命中均为 `QuotelLine` 之类误匹配 |
| 熔断 / 重试 / 幂等框架 | ❌ | 幂等仅靠业务字段（`WysMallOrder.IdempotencyKey` `model/entity/mall.go:252`、`WysImMessageBackup.MessageUID uniqueIndex` `im.go:74`） |
| `atomic.*` / `singleflight` | ❌ 全仓未用 | `grep` 于 app 0 命中 |

### 9.7 优雅停机

**实际顺序**（`app/cmd/api/main.go`）：

```text
signal.Notify(SIGINT, SIGTERM)      :573-575
ctx, cancel := WithTimeout(10s)     :578
  ① grpcServer.GracefulStop()      :581-584
  ② queueClient.Close()            :586-590
  ③ server.Shutdown(ctx)           :592-594
defer 逆序：logger.Sync(:90) ← Redis Close(:115) ← sqlDB Close(:105)
```

**两个破坏 defer 的路径**：`main.go:540` 与 `main.go:565` 用 `log.Fatal` —— `log.Fatal` 内部 `os.Exit(1)`，**跳过所有 defer**（Redis、sqlDB、logger.Sync 均不释放）。

**上下文传播缺口**：API 侧 Pub/Sub 订阅 goroutine 用裸 `context.Background()`（`main.go:321`），且 `fanoutSub` 变量（声明 `main.go:164`）之后**再无任何 Stop/引用** → 停机不取消该 `Run`（`core/queue/pubsub_broadcaster.go:72-90`）；`BroadcastToUser` 同样硬编码 `context.Background()`（`pubsub_broadcaster.go:39`）。Worker 侧 `cancel()` 后**未等待 goroutine 退出**（`cmd/worker/main.go:131-136`，无 `WaitGroup`）。

---

## 10. 异步与调度

### 10.1 RabbitMQ 拓扑

| 项 | 事实 | 证据 |
|----|------|------|
| Exchange | **不使用自定义 exchange**，全部走 AMQP **默认 exchange 直投同名队列** | `core/queue/client.go:64`（`PublishWithContext(ctx, "", c.queue, ...)`）；`grep "ExchangeDeclare|QueueBind"` → 0 命中 |
| 队列名 | 配置 `queue.rabbitmq.queue`（`common/config/config.go:400`），空则回退 `"my_go_study.tasks"`（`:418-423`） | env `RABBITMQ_QUEUE`（`:670`） |
| 声明 | `QueueDeclare(durable=true, args=nil)` | `client.go:32`、`consumer.go:45` |
| 持久化 | `DeliveryMode: amqp.Persistent`、`MessageId = taskID` | `client.go:66-67` |
| Prefetch | `ch.Qos(prefetch, 0, false)`，默认 10 | `consumer.go:50-55`；`config.go:426-431`；env `RABBITMQ_PREFETCH` `:671` |
| **Publisher confirms** | ❌ **未找到**（`grep "Confirm()"` → 0）；publish 后不等 broker ack | `client.go:59-74` |
| **重试策略** | ❌ 无；失败 `Nack(requeue=true)` → **坏消息无限重投** | `consumer.go:100`；`grep "retry|Retry"` 于 core/queue + worker → 0 |
| **死信队列 / DLX** | ❌ **未找到**（`grep "x-dead-letter"` → 0；`QueueDeclare` 的 args 恒为 nil） | `client.go:32`、`consumer.go:45` |
| 连接数 | API 与 Worker 各持**一条独立 AMQP 连接** | `main.go:312`；`cmd/worker/main.go:50-60` |

### 10.2 任务类型与全链路

任务类型常量（`core/queue/tasks.go:11-16`）：`realtime:push_notify`、`scheduled:broadcast_notify`、`sms:send`、`jpush:register`。信封 `Envelope{type, task_id, payload}`（`:19-23`）；载荷 `PushNotifyPayload`（`:26-33`）、`SendSMSPayload`（`:36-39`）、`JPushRegisterPayload`（`:42-47`）、`BroadcastNotifyPayload`（`:50-52`）。

**链路 1 · `realtime:push_notify`（唯一完整闭环）**

```text
POST /api/v1/realtime/push            router/realtime_routes.go:16（链 :10-12）
→ RealtimeController.Push             controller/realtime_controller.go:96（:114 调 pushUC.PushToUser）
→ RealtimePushUsecase.PushToUser      usecase/realtime_push_usecase.go:81（UseAsyncPush 判定 :86-96；端口 PushEnqueuer :42-44）
→ repository/queue.PushEnqueuer       app/internal/repository/queue/push_enqueuer.go:37-39（装配 main.go:312）
→ core/queue.Client.EnqueuePushNotify client.go:81-91 → publish :59-74
〔Worker〕
queue.Consumer.Run → Dispatcher       core/queue/consumer.go:17-19,85-105
→ worker.Handler.Dispatch             app/internal/delivery/worker/handler.go:71-88（case :77）
→ handlePushNotify                    :90-107 → DeliverPush
→ RealtimePushUsecase.DeliverPush     realtime_push_usecase.go:109
    NextSeq :126 → events.Append :151 → hub.BroadcastToUser :155 → 离线则 jpush :157-168
〔API 实例扇出〕
FanoutSubscriber.Run                  main.go:313-324 → pubsub_broadcaster.go:72-90
→ handleMessage :92-113 → hub.BroadcastToUser
```

**链路 2 · `scheduled:broadcast_notify`** — 生产者：整点 ticker（`core/queue/hourly_ticker.go:50`）或 `app/cmd/scheduler-trigger/main.go:37`；消费者 `worker/handler.go:79-80` → `handleBroadcastNotify :109-167`：`ListActiveUserIDs`（`:122`）→ 豁免账号跳过（`:130`）→ `SetNX` 去重（`:136`）→ **再入队 `EnqueuePushNotifyWithTaskID`（`:146`）**。

**链路 3 · `sms:send`** — 消费者 `handler.go:81-82` → `handleSendSMS :169-178`，**只打日志** `"sms task stub — integrate SMS provider in production"`（`:174`）。生产者 `EnqueueSendSMS`（`core/queue/client.go:120`）**无任何调用方**（`grep` 仅命中定义处）；`:119` 注释自认「生产 OTP 预留」。

**链路 4 · `jpush:register`** — 消费者 `handler.go:83-84` → `handleJPushRegister :180-207`（**真调** `jpushUC.RegisterDevice` `:192`）。生产者 `EnqueueJPushRegister`（`client.go:133`）同样**无调用方**。

### 10.3 Redis Pub/Sub 扇出

| 项 | 值 | 证据 |
|----|----|------|
| 频道 | 配置 `queue.pubsub.channel`，默认 `realtime:fanout` | `config.go:406,434-439`；env `QUEUE_PUBSUB_CHANNEL` `:672` |
| 发布者 | `redis.Publish`（**硬编码 `context.Background()`** `:39`；marshal/publish 错误被吞、`return 0`） | `core/queue/pubsub_broadcaster.go:36-42` |
| Worker 装配 | `NewFanoutPublisher` → `NewDeliveryPushUsecase` | `cmd/worker/main.go:66-67`；`worker/handler.go:210-216` |
| 订阅者 | `Subscribe` + `Run` | `pubsub_broadcaster.go:72-90`（`Subscribe :73`）；API 启动 `main.go:313-324`（channel 名 `:316`，回调 `hub.BroadcastToUser :317`） |

### 10.4 推送通道真实 / 占位判定

| 通道 | 判定 | 证据 |
|------|------|------|
| **JPush（极光）** | ✅ 真实：REST v3 + Basic auth | `core/jpush/client.go:76-148`（默认 URL `:16`）；未配密钥 → `Send` 返回 `ErrNotConfigured`（`:72-78`），usecase 吞错继续（`app/internal/usecase/jpush_usecase.go:100`）；含 `PLACEHOLDER` 的 key 判为未配置（`client.go:29-33`）；`main.go:392` 日志「JPush 服务端未配置（跳过真实下发）」 |
| **RongCloud（融云）** | ✅ 真实：Server API + SHA1 签名 | `core/rongcloud/client.go:17`（defaultAPIBase）、`:27-33`（同款 PLACEHOLDER 判断）；未配置时 `/im/session` 返回未配置（`main.go:420`） |
| **APNs** | ⚠️ **无独立通道** —— iOS 走极光透传 | `grep -i "apns"`（排除 jpush）→ 0 命中；仅 `apns_production` 选项（`core/jpush/client.go:133-135`，配置项 `JPushAPNsProduction`，装配 `main.go:382`） |
| **SMS** | ❌ **stub** | `app/internal/delivery/worker/handler.go:174-176` 只打日志；`core/queue/client.go:119-120` 注释「预留」且无调用方。OTP 实际靠开发固定码：`configs/config.dev.yaml:19 dev_test_otp: "123456"`，校验点 `usecase/local_auth_usecase.go:207`、`phone_otp_usecase.go:55`；bypass 条件 `config.go:292-296`（须非 prod） |

### 10.5 定时整点广播

| 环节 | 位置 | 细节 |
|------|------|------|
| ticker | `core/queue/hourly_ticker.go:15-58` | 每分钟 `time.NewTicker(time.Minute)`（`:30`）；`now.Minute()!=0 → continue`（`:40-42`）；cron 小时匹配 `hourMatchesCron`（`:61-83`）；**内存去重** `slot == lastSlot`（`:32,47-49,54`）；入队 `:50` |
| 时区 | `scheduler.timezone: Asia/Shanghai` | `configs/config.yaml:74`；`now.In(loc)` `hourly_ticker.go:40` |
| cron | `"0 10-19 * * *"` | `configs/config.yaml:77` |
| 持久去重 | Redis `SetNX scheduler:dedup:<taskID>`，TTL 2h | `app/internal/delivery/worker/handler.go:28,135-143`（TTL `:136`）；taskID = `campaignID:userID`（`usecase/hourly_notify_usecase.go:62-67`）；入队失败回删 key（`worker/handler.go:154`） |
| 文案模板 | `title_template` / `body_template`（`{{hour}}`）/ `default_message` / `expires_minutes: 120` / `action{type:deeplink, route:/home}` | `configs/config.yaml:78-84`；载荷 `model/entity/sys_notify_payload.go:10,18,37` |
| 手工触发 | `make trigger-hourly-notify` | `Makefile:179-180` → `app/cmd/scheduler-trigger/main.go:37` |

**三条重复 / 漏发风险（均有行级证据）** → 汇总到 12.1：
1. `lastSlot` **仅进程内**（`hourly_ticker.go:32`）：多开 worker 各自入队；worker 重启后清零，同小时重启会再次入队同一 slot。用户级唯一防线是 Redis `SetNX`（`worker/handler.go:136`）。
2. 分钟 ticker 若卡顿跨过整点分钟窗口会**漏发**（`hourly_ticker.go:40` 要求**恰好** `Minute()==0`），无任何补偿。
3. `scheduler-trigger` 入队**空 payload**（`cmd/scheduler-trigger/main.go:37`）→ 消费侧回退用 `time.Now()` 当 slot（`worker/handler.go:115-120`），slot 取决于**消费时刻**：入队 09:59、消费 10:00 后会被归到错误 slot；且手动 trigger 与 ticker 同小时并发时二者 slot 不同 → **各推一轮**（dedup taskID 不同，兜不住）。

### 10.6 并发安全

| 对象 | 机制 | 证据 |
|------|------|------|
| WS Hub | `sync.RWMutex`；Register/Unregister 写锁；广播**先拷贝再解锁** | `app/internal/delivery/ws/hub.go:26`、`:45`、`:62`、`:84-90`；Unregister `close(client.send)` `:77` |
| WS Client | 订阅表独立 `topicsMu sync.RWMutex`；readPump/writePump 分 goroutine | `client.go:52`；文件头注释 `:10-13` |
| WS 发送 | **非阻塞**：缓冲 32 满则丢弃 + Warn | `client.go:103-109`（缓冲构造 `:61`） |
| WS ticket | Redis `GetDel` 原子消费 | `repository/redis/ws_ticket_repo.go:43`（`:5-7` 注释说明动机） |
| SSE 限流器 | 两处 `sync.Mutex` | `usecase/completion_usecase.go:314,353` |
| 各域 service 测试 mock | `sync.Mutex`（如 `membership/service_test.go:15`） | — |
| `atomic` / `singleflight` | ❌ 全仓未使用 | `grep` 于 app → 0 命中 |

**事件不丢的设计**：`DeliverPush` **先 `events.Append`（`:151`）再 `hub.BroadcastToUser`（`:155`）** —— 广播失败（`client.go:103-108` 丢弃）时 Redis 已有事件而端上未收，依赖 `/realtime/sync` 补拉。该设计在 `realtime_push_usecase.go` 文件头注释 `:5-8` 明确写出。

---

## 11. 测试与工程化

### 11.1 Makefile（唯一工程入口，26 个目标）

| 目标 | 行 | 实际执行 | 性质 |
|------|----|---------|------|
| `proto` | `17-25` | `protoc` 生成 analytics gRPC（`:18-20` 依赖检查） | 真实 |
| `run` / `run-worker` | `27-31` | `./scripts/load-env.sh go -C app run ./cmd{api,worker}` | 脚本包装 |
| `build` / `build-worker` | `33-40` | `go -C app build -o bin/` | 真实 |
| `build-all` | `43-44` | **`go build all`**（workspace 全模块；`:42` 注释说明 `./...` 已失效） | 真实 |
| `test-all` | `47-52` | `go build all` + 逐模块 `go test ./... -count=1`（`MODULES = model common core app` `:8`） | 真实 |
| `test` | `54-58` | 逐模块 `go test`（不含 build） | 真实 |
| `tidy` | `60-63` | 逐模块 `go mod tidy` —— `:60-61` 有升级警告注释，对应 `AGENTS.md` 全局约束 9 | 真实 |
| `air` | `65-66` | `air -c .air.toml` | 包装 |
| `migrate-up` / `migrate-down` | `68-72` | `migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/my_go_study?sslmode=disable" {up,down 1}` | 真实（**DSN 硬编码** `:69,72`） |
| `docker-build/up/down` | `74-99` | `scripts/docker-compose.sh` | 脚本包装 |
| `deps-up` | `85-96` | brew 装 PG16/Redis + `CREATE ROLE postgres PASSWORD 'postgres'`（`:93`） | 真实 |
| `sync-lan-ip` | `105-107` | `scripts/sync-lan-ip.sh` | 脚本包装 |
| `lan-up` / `lan-down` / `lan-run` / `lan-run-worker` | `113-130` | `scripts/lan-compose.sh` / `lan-run.sh` | 脚本包装 |
| `import-supabase` / `-dry` | `135-141` | `go run ./cmd/import-supabase`（`:132` 注释：需 service_role） | 真实 |
| `seed-analytics` | `144-145` | `go run ./cmd/seed-analytics` | 真实 |
| `clean` | `147-148` | `rm -rf bin tmp logs` | 真实 |
| `test-transactions` | `151-152` | `scripts/test_transactions_crud.sh` | e2e 包装 |
| `check-rls` | `155-156` | `scripts/check_transactions_rls.sh` | e2e 包装 |
| `test-realtime` | `159-160` | `scripts/test_realtime_ws.sh` | e2e 包装 |
| `test-single-device-login` | `162-163` | `scripts/test_single_device_login.sh` | e2e 包装 |
| `test-auth-refresh-logout` | `165-167` | `scripts/test_auth_refresh_logout.sh` | e2e 包装 |
| `test-phone-otp-login` | `169-171` | `scripts/test_phone_otp_login.sh` | e2e 包装 |
| `test-queue-push` | `173-176` | `scripts/test_queue_push.sh`（需 API + Worker + `queue.enabled`） | e2e 包装 |
| `trigger-hourly-notify` | `178-180` | `go run ./cmd/scheduler-trigger` | 真实 |
| `test-scheduled-notify` | `182-185` | `scripts/test_scheduled_notify.sh` | e2e 包装 |
| `push-notify-user` | `187-190` | `scripts/push_notify_user.sh` | e2e 包装 |
| `check-secrets` | `192-194` | `scripts/check-secrets.sh` | 脚本包装 |
| `feishu-push-auth` | `196-200` | `lark-cli docs +update`（变量 `LARK_CLI` `:13`、`FEISHU_AUTH_DOC` `:14`） | 真实 |

> `import-supabase-dry` **不在** `Makefile:15` 的 `.PHONY` 列表；`cmd/seed-home-todo` **完全没有对应目标**。⚠️ 12.2

### 11.2 测试分布与「活体依赖」

| 模块 | `_test.go` | 主要构成 |
|------|-----------|---------|
| `app` | 55 | **15 个域的 `service_test.go`**（access / after_sales / community / deal_invoice / home / mall / membership / new_car_follow / payment / points / purchase_calculator / short_video / transaction / used_car / wallet）、`app/internal/usecase/*_test.go` ×19、`delivery/http` ×8（controller / handler / middleware ×2 / router / dto ×3）、`delivery/ws` ×3、`repository`（postgres / redis / csagent / llm）、`shared` ×2 |
| `common` | 5 | `auth/supabase_test.go`、`config/auth_config_test.go`、`config/realtime_ws_url_test.go`、`jwt/jwt_uuid_test.go`、`metrics/metrics_test.go` |
| `core` | 4 | `jpush/client_test.go`、`queue/tasks_test.go`、`queue/pubsub_broadcaster_test.go`、`rongcloud/client_test.go` |
| `model` | 2 | `entity/store_test.go`、`entity/sys_notify_payload_test.go` |
| `thunder` | 1 | `thunder/tools/crypro/crypro_test.go`（与主工程无关） |

| 类别 | 说明 | 代表 |
|------|------|------|
| 纯单测（内存 mock） | 无外部依赖，`go test` 直接可跑 | `app/internal/usecase/device_session_usecase_test.go`、`app/internal/transaction/service_test.go`、各域 `service_test.go`（mock 内含 `sync.Mutex`，如 `membership/service_test.go:15`） |
| `httptest` 层测 | 起 Gin 路由但不联网 | 8 个文件；`middleware/supabase_session_auth_test.go:20`、`handler/user_handler_account_test.go:47`、`transaction/handler_test.go:79`、`controller/controller_test.go:76` |
| **需活体基础设施** | **全部有 `t.Skip` 守护 → 只会 skip，不会 fail** | Redis：`repository/redis/session_repo_test.go:17,56`、`delivery/ws/integration_test.go:21`、`usecase/realtime_usecase_test.go:19,62`；WS server + env：`delivery/ws/handler_test.go:14-18`（缺 `TEST_WS_URL`/`TEST_WS_TICKET` 即 skip）、`integration_test.go:46` |
| 需活体 ES | **没有** | `transaction/repo_es_test.go` 用 `httptest` 内存模拟 |
| 需活体 Postgres | **未找到任何一例** | `repository/postgres/profile_repo_test.go:9-30` 只测纯函数 `userProfileUpdates`，不连库 → **整个 Postgres 仓储层无测试**（12.1） |

集成冒烟由 shell 承担（11.3）。

### 11.3 `scripts/` 逐文件用途

| 路径 | 用途 |
|------|------|
| `scripts/load-env.sh` | 加载 `.env` + `.env.local` 后执行命令（`make run` 的包装） |
| `scripts/source-env.sh` | 被 source 的加载器：`configs/supabase.env` → `.env` → `.env.local` |
| `scripts/docker-compose.sh` | 加载 supabase/.env/.env.local 后执行 `docker compose`（基础文件） |
| `scripts/lan-compose.sh` | LAN 模式 compose（基础 + lan overlay） |
| `scripts/lan-run.sh` | 无 Docker 时本机跑 LAN 模式 API / Worker |
| `scripts/lan-ip.clients` | LAN 客户端清单（行首 `#` = 关闭） |
| `scripts/sync-lan-ip.sh` | 按当前网卡 IPv4 刷新 Go / 各客户端局域网地址 |
| `scripts/install-githooks.sh` | 把 `.githooks/*` **复制**到 `.git/hooks`（`:5-13`，非 `core.hooksPath`） |
| `scripts/check-secrets.sh` | 推送前阻止密钥入库（11.5） |
| `scripts/check_transactions_rls.sh` | 检查 Supabase `transactions` RLS 是否在数据库层生效 |
| `scripts/test_transactions_crud.sh` | e2e：transactions CRUD（需 `SUPABASE_ACCESS_TOKEN` 或 service_role） |
| `scripts/test_realtime_ws.sh` | e2e：Realtime WS（需后端 + Redis + 登录） |
| `scripts/test_single_device_login.sh` | e2e：设备 B 登录后设备 A 旧 session 应 401 |
| `scripts/test_auth_refresh_logout.sh` | e2e：登录 → refresh → 业务 API → logout → 401 |
| `scripts/test_phone_otp_login.sh` | e2e：固定测试号 `13400000000` + `123456` 登录 |
| `scripts/test_queue_push.sh` | e2e：RabbitMQ + Pub/Sub 异步 push（需 API + Worker） |
| `scripts/test_scheduled_notify.sh` | e2e：定时广播 + sync 验证 |
| `scripts/push_notify_user.sh` | 向指定 userId 推 `sys.notify` |
| `scripts/reindex_transactions_es.sh` | 仅 PUT transactions ES 索引 mapping（**首行自标 "Partial…Deferred"**） |
| `scripts/sync_docs_to_feishu.sh` + `scripts/feishu_doc_sync/*.py` | 飞书文档同步管线（11.6） |
| `scripts/navicat_*.sql`、`scripts/seed_analytics_records_chart_v1.sql` | 种子 / 查询 SQL |

### 11.4 CI / Lint / Hooks / 规则

| 项 | 状态 | 证据 |
|----|------|------|
| GitHub Actions / GitLab CI / Jenkins / Travis | ❌ **未找到** | `ls .github` → No such file；`find` `.gitlab-ci.yml` `Jenkinsfile` `.travis.yml` → 空；`deploy/` 下只有 `observability/` |
| `golangci-lint` | ❌ 未找到 | `find . -maxdepth 3 -name ".golangci*"` → 空 |
| `gofmt` / `go vet` 自动化 | ❌ | `grep -E "golangci|gofmt|go vet"` 于 `Makefile` 与 `.githooks/` → 空 |
| Git hooks | ✅ 3 个（**本地钩子，非 CI**） | `.githooks/pre-push`（`exec "$root/scripts/check-secrets.sh"`）；`.githooks/commit-msg` + `prepare-commit-msg`（Python：剥离 Cursor `Co-authored-by` 行 + **强制 commit 首行含中文**，`commit-msg:20` `re.search(r"[\u4e00-\u9fff]", subject)` 失败 `sys.exit(1)` `:22`）。⚠️ **靠手工复制安装**（`scripts/install-githooks.sh:5-13`），新克隆不安装即失效 |
| IDE / Agent 规则 | ✅ | `.cursor/rules/ponytail.mdc:1-6`（`alwaysApply: true`）；`.qoder/settings.local.json`（权限，不影响代码）；根 `AGENTS.md`；`.harness/`（多角色编排合同） |

### 11.5 `make check-secrets` 的实现与盲区

`scripts/check-secrets.sh`：

- **匹配模式**：`sb_secret_[A-Za-z0-9_-]+` 与 `SUPABASE_SERVICE_ROLE_KEY=sb_`（`:13`），仅匹配非注释行。
- **扫描范围**：固定文件清单 `configs/supabase.env(.example)`、`.env(.example)`、`configs/config*.yaml`（`:32`）且必须被 git 跟踪（`git ls-files --error-unmatch` `:25`）+ 暂存区里的 `*.env` / `*.env.*` / `configs/*.yaml`（`:37-43`）。
- **失败模式**：命中则 `FAILED=1` → 打印指引 → `exit 1`（`:45-50`）；由 `.githooks/pre-push` 挂到推送前。
- **盲区（实测）**：只查 service_role 形态。因此不拦：`.env` 中的 `JWT_SECRET`（实测值 == `configs/config.yaml:28` 的弱默认 `change-me-in-production`）、`DATABASE_PASSWORD`（实测 == compose 默认 `postgres`）、`Makefile:69,72` 内嵌的 DSN、`configs/config.dev.yaml:18-20` 的测试凭据。⚠️ 见 12.1。

### 11.6 文档 / 飞书同步管线

| 环节 | 实现 |
|------|------|
| 入口 | `scripts/sync_docs_to_feishu.sh` —— 要求 `lark-cli`（`:7-10`），缺 `pyyaml` 自动 `pip install`（`:15-17`），`bootstrap|sync|dry-run` 转发 `python3 scripts/feishu_doc_sync/main.py`（`:19-33`） |
| 配置发现 | 默认 `docs/feishu-sync.config.yaml`（`main.py:44-47 _resolve_config_path`）；另有 6 套变体 `docs/feishu-sync-{code,daily,godeep,study,tool,tool-docker}.config.yaml` |
| 配置键 | `wiki.space_name`、`wiki.project_folder`、`sync.include`（glob：`docs/**/*.md` + `AGENTS.md`）、`sync.exclude`、`paths.manifest`、`paths.repo_root` |
| 文档发现 | `scanner.py:67-68` 对 `include` 做 `repo_root.glob(pattern)`；manifest 路径解析 `scanner.py:50`（默认 `docs/feishu-sync.manifest.json`） |
| 状态文件 | `manifest.py` 的 `Manifest.load/save`（`main.py:69-72,86,100,135`），记录 `wiki_space_id` / `project_folder_node_token` / 每文件 hash 条目（`:102-131`） |
| 兜底 | 环境变量 `FEISHU_WIKI_SPACE_ID`（`main.py:102`） |
| 富文本页 | md 管线**画不了板/高亮块/栅格** → 走 DocxXML + `make feishu-push-auth`（`Makefile:196-200`），源在 `docs/feishu/`。限制与坑见 `docs/FEISHU_SYNC.md` §三 |

> 本文档落在 `docs/project-detail.md`，匹配 `docs/**/*.md`，因此下一次 `./scripts/sync_docs_to_feishu.sh sync` 会自动纳入。⚠️ **唯一例外**：5.5 的 Mermaid 依赖图在 md 管线里会作为代码块落地（飞书 md 导入不渲染 mermaid）；要图就在飞书侧改用 whiteboard，见 `docs/FEISHU_SYNC.md` §三。

---

## 12. 风险与待确认项

### 12.1 高（会导致错误结果或数据丢失）

| # | 风险 | 机理 | 证据 | 建议验证方式 |
|---|------|------|------|------------|
| H1 | **新克隆无法构建** | `go.work` / `go.work.sum` 被 gitignore，而 4 个模块的 `go.mod` **互相无 require、无 replace**（module 名就叫 `app`/`model`/`core`/`common`，非公共路径）→ 无 workspace 时跨模块 import 全部解析失败，`go build all` / `make test-all` 亦失效 | `.gitignore:32-34`；`go.work:3-9`；`app/go.mod:1-3`；`Makefile:7,42-44` | 在临时目录 `git clone` 后直接 `make build-all`，观察报错 |
| H2 | **`log.Fatal` 跳过所有 defer** | `log.Fatal` 内部 `os.Exit(1)`，`sqlDB.Close` / `redisClient.Close` / `logger.Sync` 均不执行；文件日志缓冲区可能不落盘 | `app/cmd/api/main.go:540,565` vs defer `:90,105,115` | 故意占用 8080 端口启动，检查 `logs/app.log` 尾部是否截断 |
| H3 | **AMQP 无 DLX、坏消息无限重投** | 消费失败 `Nack(requeue=true)`，且 `QueueDeclare(args=nil)` 未配 `x-dead-letter-*` → 一条必然失败的消息会永久循环，阻塞该 prefetch 批次 | `core/queue/consumer.go:45,100`；`grep "x-dead-letter"` → 0 | 投一条 payload 非法的消息，观察 CPU/日志循环 |
| H4 | **无 publisher confirm** | `publish` 返回即视为成功，broker 未落盘也当已投递 → 队列侧静默丢任务 | `core/queue/client.go:59-74`（`grep "Confirm()"` → 0） | 关 RabbitMQ 后调用 push，看 API 是否报错 |
| H5 | **Pub/Sub 广播失败被吞且恒返回 0** | `marshal`/`publish` 错误全部忽略并 `return 0`，调用方无法区分「广播 0 个实例」与「广播失败」 | `core/queue/pubsub_broadcaster.go:36-42` | 断 Redis 后触发 push，检查返回值与日志 |
| H6 | **整点广播可重复 / 可漏发** | ① `lastSlot` 仅进程内，多 worker / 重启后失效；② ticker 要求**恰好** `Minute()==0`，一次卡顿即整小时漏发；③ `scheduler-trigger` 入队空 payload，slot 取决于**消费时刻** | `core/queue/hourly_ticker.go:32,40,47-49`；`app/cmd/scheduler-trigger/main.go:37`；`app/internal/delivery/worker/handler.go:115-120,135-143` | 同小时并跑 ticker + `make trigger-hourly-notify`，数收到的 `sys.notify` 条数 |
| H7 | **`transactions` 表可能被改名后旧数据脱管** | `autoMigrate` 检出旧 `user_id` 为整型即 `RENAME TO transactions_legacy_uint` 并新建空表；探测 SQL 的错误被 `_ =` 吞掉。**2026-11-03 缓解（parity-p0-1）**：`autoMigrate` 现在整体受 `database.auto_migrate` 守卫，`APP_ENV=lan|prod` 强制 false → 正式环境不再有机会触发改名，改表结构只能写迁移 | `app/cmd/api/main.go:635-646`（改名 `:644`）；闸门 `app/internal/schemaboot/boot.go` | 在测试库把 `transactions.user_id` 改为 `bigint` 后以 `auto_migrate=true` 启动，检查旧表与行数（本机逃生门路径） |
| H8 | **PG ↔ ES 双写无补偿** | 主写成功后 `bestEffortIndex/Delete` 失败仅计数 + Warn，不回滚不重试 → 列表读主库、搜索读 ES，两者可长期发散 | `app/internal/transaction/service.go:60-115`（打点 `:98,110`）；启动 ES 失败亦只 Warn（`main.go:287-288`） | 停 ES 后建/改交易，恢复 ES 后对比 `search` 与列表结果 |
| H9 | **`ValidateAccessToken` 无超时** | 用 `http.DefaultClient.Do`，Supabase 侧挂起会拖住**每一个**走该中间件的鉴权请求（仅受请求 ctx 约束） | `common/auth/supabase.go:47`（对照：`core/jpush/client.go:64` 有 10s） | 用 toxiproxy/iptables 丢包模拟 Supabase 慢响应，观察 p99 |
| H10 | **`/metrics` 无鉴权且随 lan overlay 对局域网暴露** | 注册时注释「不经 SessionAuth；仅本机 / Docker 内网可达即可」，但 app 服务映射 `8080:8080`（非 `127.0.0.1:`），`/metrics` 因此同端口可达 | `router/router.go:66`；`docker/docker-compose.yml:110-111` | 从同网段另一设备 `curl http://<mac-ip>:8080/metrics` |
| H11 | **入库的弱默认凭据**（✅ 2026-11-03 已缓解：`configs/config.yaml` 兜底密钥已删为 `secret: ""`；compose 改 `${JWT_SECRET:?}` 必填；`common/config/secrets.go` prod 硬校验（占位值/长度/字符集 + DB 口令 + service_role 来源）；`check-secrets.sh` 覆盖弱值/短值/高熵。**残留**：`Makefile:69,72,93` 本机迁移与建角色仍写死 `postgres:postgres`——只影响本机 dev 工具链，不在本片白名单，归 P0-1 迁移命令收口时一并处理。口径与轮换见 `docs/startup-guide.md` §13） | `.env` 被有意提交（commit `afbc29f`），其中 `JWT_SECRET` 实测等于 `configs/config.yaml:28` 的 `change-me-in-production`、`DATABASE_PASSWORD` 等于 `postgres`；`Makefile:69,72,93` 亦硬编码同一口令。`check-secrets` 模式不覆盖这些 | `.env`（tracked）；`configs/config.yaml:28`；`Makefile:69,72,93`；`scripts/check-secrets.sh:13` | 确认是否存在任何 prod / 共享环境复用同一 `JWT_SECRET` |

### 12.2 中（工程一致性 / 契约风险）

| # | 项 | 证据 | 说明 |
|---|----|------|------|
| M1 | **CORS 两处硬冲突** | `cors.go:13`（`Allow-Origin: *`）+ `:17`（`Allow-Credentials: true`）—— 该组合按 Fetch 规范浏览器会拒绝携带凭据；且 `:15` 白名单 `Origin, Content-Type, Authorization, Accept` **不含 `X-Session-ID` / `X-Device-ID`**，而两个 SessionAuth 中间件都必读它们（头名常量 `supabase_session_auth.go:15-16`，读取点 `supabase_session_auth.go:36-37` 与 `local_session_auth.go:36-37`） | 跨源 Web 客户端携带 session 头的预检会缺头授权。当前主要是 Flutter 客户端，故未暴露 ⚠️ 待确认是否有浏览器端调用方 |
| M2 | **51 个接口无编译期契约断言** | 全仓仅 5 处 `var _`（5.3 表）；`SessionRepository` 实现体无断言 | 端口方法改名/增删时，只有调用点报错；测试 mock 会一起静默漂移 |
| M3 | **遗留 JWT 全套死代码** | `middleware/auth.go:14-17,20,49-57` 定义存在但 `grep 'middleware\.Auth('` → 0 调用点；`user_routes.go:12` 接收 `jwtManager` 但函数体不用 | 与 `AGENTS.md` 「两套 Token 勿混用」叙述相互强化，实际已无人使用 ⚠️ 待确认：能否直接删除 |
| M4 | **`EnqueueSendSMS` / `EnqueueJPushRegister` 无生产者** | `core/queue/client.go:120,133`（`grep` 仅命中定义） | 两个任务类型只有消费者；`sms:send` 消费端还是 stub（`worker/handler.go:174`） |
| M5 | **`thunder` 声明但未接入** | `go.work:8`；`grep "mszlu521/thunder"` → 0；`docker/Dockerfile:31` COPY 其 `go.mod` | 删除 `thunder/` 会让 `go build all` 与镜像构建失败 —— 事实上不能删 ⚠️ 待确认是否要正式接入或从 workspace 移出 |
| M6 | **`core → model` 反向依赖** | `core/queue/pubsub_broadcaster.go:10` | 全仓唯一一处；使 `core` 无法作为纯驱动层复用。修法见 5.5 |
| M7 | **`usecase` 未抽象 3 个具体基建** | `app/internal/usecase/`：`core/jpush:9`、`core/supabase:22`、`phone_otp:9`（对照：`rongcloud` 已抽象，`im_group_usecase.go:52`） | 同层纪律不一致；这三条路径无法做无副作用单测 |
| M8 | **63 处 `_ =` 丢弃错误** | `grep -c "_ = " app --include="*.go"`（排除测试）= 63；重点：`realtime_controller.go:43`、`im_controller.go:49`、`deal_invoice/handler.go:118`（`_ = c.ShouldBindJSON(...)` → **绑定失败仍以零值继续**） | 前三处是可验证的用户可见行为：请求体格式错不会返回 400 ⚠️ 待确认 |
| M9 | **`/api/v1/me` 与 `/api/v1/stores` 被两个域各自 Group** | `profile_routes.go:21` + `access/router.go:52`；`home/router.go:45` + `access/router.go:37` | 同一前缀的鉴权与语义归属分散 |
| M10 | **`store_id` 类型跨域不一致** | `wys_store.StoreID int`（`access.go:34,41`）vs `wys_im_store_group.StoreID string`（`im.go:65,72`） | 融云侧要求字符串 group/store 标识；跨域传递需显式转换 ⚠️ 待确认是否有隐式 `fmt.Sprint` 到处散落 |
| M11 | **`wys_used_car_order.Amount float64`** | `used_car_order.go:138` | 全仓唯一用 float 存业务金额（商城已改 `string numeric(10,2)` `mall.go:210`、钱包已改 `...Fen int64` `cash_wallet.go:29`）→ 金额精度隐患 |
| M12 | **`entity.User.DeletedAt *time.Time` 不是 `gorm.DeletedAt`** | `user.go:33`；同样写法见 `community.go:43,80`、`short_video.go:32`、`mall.go:149,166,219` | GORM **不会**自动加 `deleted_at IS NULL`；每个查询必须手写过滤 ⚠️ 待确认：是否所有列表查询都带了该条件 |
| M13 | **Postgres 仓储层零测试** | `app/internal/repository/postgres/profile_repo_test.go:9-30` 只测纯函数；无活体 PG 测试（`grep "t.Skip|os.Getenv"` 无 PG 守护，因为根本没有） | 6.3 的建表路径已收口为单一闸门（P0-1），但仓储与真实 schema 的一致性仍只在运行时暴露 |
| M14 | **无 request-id / trace-id** | `middleware/logger.go:22-30` 字段列表；`common/logger` 无相关实现 | 单次请求的多行日志 + 队列异步链路无法串联 |
| M15 | **停机不取消 Pub/Sub 订阅；worker 不等 goroutine** | `main.go:164,313,321`（`fanoutSub` 声明后不再引用；订阅 goroutine 用 `context.Background()`）；`cmd/worker/main.go:131-136`（无 `WaitGroup`） | 滚动升级时尾部消息可能丢 |
| M16 | ~~**`autoMigrate` 无开关**~~（✅ 2026-11-03 已闭环，parity-p0-1） | `database.auto_migrate`（默认 false，`APP_ENV=lan|prod` 由 `common/config.NoDDLAppEnv` 强制 false）+ 唯一入口 `app/internal/schemaboot.Boot()`；API `main.go:121`、Worker `cmd/worker/main.go:90` | 多实例 / 只读库账号不再启动期 DDL；库版本落后、dirty、空库三种情况直接拒启（实机验证见 `docs/acceptance-records/2026-11-03-parity-p0-1-migration-gate.md`） |
| M17 | ~~**`config.<env>.yaml` 解析失败静默**~~（✅ 2026-11-03 已闭环，parity-p0-1） | `common/config/config.go:722-732`：语法/类型错误 `return err`（不再 `_ =` 丢弃）；只有「该环境没有覆盖文件」被容忍并 `log.Printf` 提示（`APP_ENV=production` 历来没有 `config.production.yaml`，硬失败会让它起不来） | 写错 dev 配置不再静默退回基线值 |
| M18 | **`seed-home-todo` 无 make 目标；`import-supabase-dry` 不在 `.PHONY`** | `app/cmd/seed-home-todo/` 存在但 `Makefile:15` 目标列表无对应项 | 工具入口不对称 |
| M19 | **`migrations/` 与 embedded SQL 双轨**（✅ 2026-11-03 权威轨已定） | `migrations/` 现为**单一真相**：空库端到端首次跑通（此前在 `20260923170000` 就断），并补 `20260923167500_align_gorm_only_schema`（AutoMigrate-only 的 14 张表）与 `20260924110000_align_column_drift`（146 处列定义）；`sql/mall_schema.sql` 一类 embedded 文件仍与迁移各存一份，但只经 `ConsolidateLocalUsers` 在 `auto_migrate=true` 的本机逃生门里执行 | 残留：embedded SQL 与 `migrations/` 内容重复，改一边不同步 —— 归 P0-5「假实现/重复机制清单化」后续收口 |
| M20 | ~~**`20260924100000_rongcloud_im` 缺 `.down.sql`**~~（✅ 2026-11-03 已补） | `migrations/` 现 25 对全部成对；`schemaboot.LoadMigrations` 会列出缺 down 的文件（Warn），重复版本号升级为**硬错误**（`20260923160000` 撞过） | `make migrate-down` 不再在该版本失败 |

### 12.3 文档 / 代码冲突（`AGENTS.md` 侧）

| # | `AGENTS.md` 叙述 | 代码事实 | 结论 |
|---|-----------------|---------|------|
| C1 | §应用启动 步骤 5：「`UserUsecase`（遗留 JWT 路由）+ `DeviceSessionUsecase`（Redis session）」（`AGENTS.md:319`） | `grep "UserUsecase|NewUserUsecase" app/**/*.go` → **0 命中**；实际是 `handler.NewUserHandler`（`main.go:424`）；遗留 JWT 中间件无任何调用点（M3） | 叙述过时，应删 `UserUsecase` |
| C2 | 步骤 10：「SIGTERM → HTTP Shutdown + gRPC GracefulStop」 | 实际是 **gRPC 先停（`:581`）→ RabbitMQ（`:586`）→ HTTP 后停（`:592`）** | 顺序描述与代码相反，且漏了 RabbitMQ |
| C3 | 「`/api/v1/transactions/search` 需 ES」易读成「无 ES 则路由不注册」 | 路由**无条件注册**（`app/internal/transaction/router.go:48`）；无 ES 时**运行时返回 503**（`transaction/handler.go:309`） | 客户端契约不同：路由存在、状态码 503，不是 404 |
| C4 | 配置表：「`.env` 入库，DB、Redis、JWT 等」 | 属实（tracked），但**值为实值且 JWT/DB 为弱默认** → 不只是「入库」，是「入库且弱」 | 建议补一句「`.env` 只放本机默认口令，切勿复用到共享环境」（见 H11） |
| C5 | 「新建业务表一律 `wys_` 前缀」 | 49 张遵守 / 10 张不遵守；遵守的里面单复数混用（6.2） | 规范只约束新表，旧表未回改；文档可补一句「例外清单见本文 6.2」 |
| C6 | 「`docs/feishu-sync.config.yaml` 的 `wiki.space_name: "项目 docs"`」 | 实际知识库列表中无该名字（仅 学习 / 日常 / 工作 / 金融 存在），靠 manifest 里的 node token 仍解析成功 | 配置为事实上的"过时名称"，不影响运行但会误导；见 `docs/FEISHU_SYNC.md` |

对照基线（无冲突，列出以证明检查覆盖）：`AGENTS.md:320-322` 的 Realtime 装配描述与 `main.go:295-329` 一致；步骤 1-4、7、8、9 与 `main.go:81-109,511,545-571` 一致。

### 12.4 待确认项汇总（需要人工回答）

| 编号 | 问题 | 为何重要 |
|------|------|---------|
| Q1 | 本仓库的**目标部署形态**是什么？（单实例 / 多实例 / 是否有只读库账号） | 决定 H1（go.work 入库）、M16（autoMigrate 无开关）是否必须立刻修 |
| Q2 | **是否存在浏览器端调用方**（而非仅 Flutter）？ | 若有，M1 的 CORS 组合是阻断性 bug |
| Q3 | `migrations/` 与 `app/internal/repository/postgres/sql/` **哪一轨是权威**？两者如何防漂移？ | M19 直接决定改表流程 |
| Q4 | `thunder/` 是要**正式接入**还是应当从 workspace 移出？ | M5 |
| Q5 | 遗留 JWT（`middleware.Auth` + `TransactionRecord` + `/me/profile` 旧路径）的**下线时间点**？ | M3、6.1（已标"已弃用"的表仍在 AutoMigrate） |
| Q6 | `configs/supabase.env` 内的 URL / anon_key 是否为**生产项目**凭据？anon_key 本身可公开，但要确认无混淆 | H11 的严重程度分级 |
| Q7 | 软删过滤（`deleted_at IS NULL`）是否有集中的保障（作用域 / 视图）？ | M12 —— 若无，列表接口会泄露已删数据 |
| Q8 | 整点广播的**正确性预期**：宁缺勿重，还是宁可重不可缺？ | H6 的修法方向（去重键放哪一层） |

---

## 附录 A：本文档的维护约定

| 项 | 约定 |
|----|------|
| 重新生成 | 代码有大版本重构后按节重写；**行号会变**，因此 4 / 6 / 7 / 8 节优先维护表格结构，行号当作"定位线索"而非契约 |
| 与既有文档的关系 | `docs/architecture-learning-guide.md`（教学向、讲分层优缺点）、`docs/startup-guide.md`（操作向）、`docs/supabase-integration.md`（能力矩阵）—— 本文档的定位是**带证据的机读事实清单**，冲突时以代码 + 本文行号为准 |
| 飞书同步 | 本文匹配 `docs/**/*.md`，`./scripts/sync_docs_to_feishu.sh sync` 自动纳入；5.5 的 Mermaid 图在 md 管线不会渲染（见 `docs/FEISHU_SYNC.md` §三） |
| 负面结论检索 | 见附录 B |

## 附录 B：「未找到」类结论的检索方式

| 结论 | 检索方式 |
|------|---------|
| 无构建标签 | `grep -rn "//go:build" --include="*.go" .` |
| 无分布式 tracing | `grep -iE "opentelemetry|otel|jaeger|zipkin"` 于 `*.go` 与 `go.mod`（`.go` 命中均为 `QuotelLine` 之类误匹配，如 `app/internal/purchase_calculator/service.go:99`） |
| 无 request-id | `grep -i "RequestID\|X-Request-Id\|request.id"` 于 `middleware/logger.go`、`common/logger/` |
| 无 exchange / DLX / confirm | `grep -E "ExchangeDeclare|QueueBind|x-dead-letter|Confirm\(\)"` 于 `core/` + `app/` |
| 无全局限流 / 超时 / 体积限制 | `grep -E "gin.Timeout|TimeoutHandler|MaxBytesReader|MaxMultipartMemory|rate\.Limit"` |
| 无独立 APNs 通道 | `grep -i "apns"` 于全部 `.go`（排除 `core/jpush/`） |
| `thunder` 未被引用 | `grep -rn "mszlu521/thunder" app common core model` |
| 遗留 JWT 未被挂载 | `grep -rn "middleware\.Auth("` 于 4 个业务模块 |
| `migrations/` 无 RLS | `grep -rlnE "row level security|CREATE POLICY|create policy" migrations/` |
| 无 `core/redis` 包 | `ls core/` = `database supabase queue rongcloud jpush`；`Glob core/redis/**` 为空 |
| 无 Postgres 活体测试 | `grep -rn "t.Skip|os.Getenv|testing.Short"` 于 4 模块（无 PG 连接守护，因为根本没有 PG 测试） |
| 无 TODO/FIXME/HACK | `grep -rn "TODO\|FIXME\|HACK" --include="*.go" app common core model` → **0 命中**；`XXX` 5 处均为二手车种子 VIN 占位串（`app/internal/used_car/repo_postgres.go:200,207,214,221,228`） |
| 无 CI | `ls .github`（No such file）；`find` `.gitlab-ci.yml` `Jenkinsfile` `.travis.yml`；`deploy/` 下仅 `observability/` |
| 无 golangci-lint | `find . -maxdepth 3 -name ".golangci*"`；`grep -E "golangci|gofmt|go vet"` 于 `Makefile`、`.githooks/` |







