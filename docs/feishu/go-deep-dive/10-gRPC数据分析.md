# 10 gRPC 数据分析：proto 契约、metadata 鉴权与图表种子数据

> 一句话定位：数据分析是与 HTTP :8080 **同进程双端口**的 gRPC 试验服务（:9090），走 protobuf 契约 + unary interceptor 鉴权，读本地 Postgres `analytics_records` 宽表；属于「gen 契约 → grpc delivery → usecase → postgres」垂直链路，阅读前置：懂 SessionAuth 业务 token+session 两套认证（见 [04-HTTP路由与中间件](04-HTTP路由与中间件.md)）、懂 [09-Realtime与WebSocket](09-Realtime与WebSocket.md) 里同一套 DeviceSession 的概念。

## 0. 通俗理解

HTTP JSON 像「写信」：每次都用纸笔把字段名抄一遍，啰嗦但谁都看得懂。gRPC 像「发电报」：双方先约好一本密码本（`.proto`），线上只传编号和紧凑二进制，又快又有严格 schema。

这个试验服务只有两封电报：**拉一页分析记录列表**、**按 id 取一条详情**。鉴权方式和 HTTP 一模一样，只是车票从 header 换进了 gRPC 的「随身口袋」metadata：`authorization`（Bearer token）、`x-session-id`、`x-device-id`。

表里的数据也不是随便造的：种子生成器刻意保证「PV ≥ UV ≥ 点击 ≥ 转化」的漏斗递减、ROI=收入/成本、评分夹在 0–100，好让 Flutter 端的转化环、迷你柱、雷达图有东西可画。

## 1. 源码地图

| 层 | 文件（相对路径） | 职责 |
|----|------------------|------|
| 契约 | `app/api/proto/analytics/v1/analytics.proto` | 唯一真相源：service + 5 个 message |
| 生成物 | `app/api/gen/go/analytics/v1/analytics.pb.go` | 消息结构/getter/序列化（make proto） |
| 生成物 | `app/api/gen/go/analytics/v1/analytics_grpc.pb.go` | Client/Server 接口、注册函数、handler |
| 服务端 | `app/internal/delivery/grpc/server.go` | 建 grpc.Server、挂拦截器、注册服务、reflection、Listen |
| 服务端 | `app/internal/delivery/grpc/analytics_server.go` | 两个 RPC 实现 + entity→proto 转换 |
| 鉴权 | `app/internal/delivery/grpc/interceptor/auth.go` | metadata 取 token/session/device，对齐 HTTP SessionAuth |
| 用例 | `app/internal/usecase/analytics_usecase.go` | 分页/详情 + 48 条图表种子数据生成 |
| 仓储接口 | `model/repository/analytics_repository.go` | 6 个方法的端口 |
| 实体 | `model/entity/analytics_record.go` | 宽表实体（35 列）+ Filter |
| 仓储实现 | `app/internal/repository/postgres/analytics_repo.go` | GORM 分页/详情/种子维护 |
| 种子命令 | `app/cmd/seed-analytics/main.go` | 独立可执行：清 seed → 重灌 |
| 种子 SQL | `scripts/seed_analytics_records_chart_v1.sql` | 手工最小 4 条样例 |
| 装配 | `app/cmd/api/main.go` L430-437、L544-571 | EnsureSeedData + gRPC Server 启停 |

## 2. Proto 契约

源码锚点：`app/api/proto/analytics/v1/analytics.proto`

- `syntax = "proto3"`；`package analytics.v1`；`option go_package = "app/api/gen/go/analytics/v1;analyticsv1"`。

### 2.1 RPC 表

| RPC | 请求 message | 响应 message | 语义 |
|-----|--------------|--------------|------|
| `ListAnalyticsRecords` | `ListAnalyticsRecordsRequest` | `ListAnalyticsRecordsResponse` | 分页列表，`page` 从 1 开始，`page_size` 默认 10、最大 50（注释约定，实际兜底在仓储层） |
| `GetAnalyticsRecord` | `GetAnalyticsRecordRequest` | `GetAnalyticsRecordResponse` | 按 id 取详情 |

### 2.2 Message 字段表

| Message | 字段（序号: 类型 名） |
|---------|----------------------|
| `ListAnalyticsRecordsRequest` | 1:int32 page；2:int32 page_size |
| `ListAnalyticsRecordsResponse` | 1:repeated AnalyticsRecord items；2:int64 total；3:int32 page；4:int32 page_size |
| `GetAnalyticsRecordRequest` | 1:int64 id |
| `GetAnalyticsRecordResponse` | 1:AnalyticsRecord item |
| `AnalyticsRecord` | 见下，35 个字段，全部标量；时间字段为 Unix 秒（int64） |

`AnalyticsRecord` 字段（proto 名 snake_case → Go PascalCase）：

| 分组 | 字段 |
|------|------|
| 标识 | id(1,int64)、code(2)、title(3)、subtitle(4) |
| 分类 | category(5)、sub_category(6)、status(7)、priority(8,int32)、region(9)、channel(10) |
| 负责人 | owner_name(11)、owner_team(12)、source_system(13) |
| 漏斗指标 | metric_pv(14)、metric_uv(15)、metric_click(16)、metric_convert(17，均 int64) |
| 金额指标 | metric_revenue(18,double)、metric_cost(19,double)、metric_roi(20,double) |
| 质量指标 | metric_bounce_rate(21,double)、metric_avg_duration_sec(22,int32)、score_quality(23,double)、score_risk(24,double) |
| 标记 | tag_primary(25)、tag_secondary(26)、flag_featured(27,bool)、flag_anomaly(28,bool)、notes(29) |
| 时间（秒） | observed_at(30)、window_start(31)、window_end(32)、published_at(33)、created_at(34)、updated_at(35) |

## 3. 生成物职责（gen/go/analytics/v1，只列职责，勿手改）

> 通俗理解：这两个文件是 `make proto` 用 protoc 按密码本自动印出来的电报表格，人不编辑。

| 文件 | 内容 | 关键导出 |
|------|------|----------|
| `analytics.pb.go` | 5 个 message 的 Go struct、proto 反射描述符、每个字段的 `GetXxx()` getter（空安全）、Reset/String/ProtoMessage/ProtoReflect | `AnalyticsRecord`（含 35 个 Get 方法）、`ListAnalyticsRecordsRequest/Response`、`GetAnalyticsRecordRequest/Response` |
| `analytics_grpc.pb.go` | gRPC 胶水 | `AnalyticsServiceClient` 接口 + `NewAnalyticsServiceClient`；`AnalyticsServiceServer` 接口；`UnimplementedAnalyticsServiceServer`（前向兼容底座，业务 server 匿名内嵌它）；`RegisterAnalyticsServiceServer(s, srv)`；两个 `_AnalyticsService_*_Handler` 解码函数；service desc |

业务实现 `AnalyticsServer` 匿名内嵌 `UnimplementedAnalyticsServiceServer`——proto 将来加 RPC 时，老代码不实现也能编译通过（返回 Unimplemented）。

## 4. gRPC 服务端

### 4.1 `server.go` 函数表

| 函数/类型 | 职责 | 备注 |
|-----------|------|------|
| `Server` struct | 包 `grpcServer *grpc.Server`、listener、log、port | |
| `Options` struct | Port / Log / `*grpcauth.Authenticator` / `*usecase.AnalyticsUsecase` | 依赖注入袋 |
| `NewServer(opts) (*Server, error)` | 校验端口>0、Authenticator 非空、AnalyticsUC 非空；log 缺省 nop；`grpc.NewServer(grpc.UnaryInterceptor(auth.UnaryServerInterceptor()))`；注册 AnalyticsService；**`reflection.Register(gs)`**；`net.Listen("tcp", :port)` | reflection 让 grpcurl 等工具无需 stub 即可列服务 |
| `(s) Serve() error` | 阻塞 `grpcServer.Serve(listener)`，先打启动日志 | main.go 在 goroutine 中调 |
| `(s) GracefulStop()` | nil 安全后调 `grpcServer.GracefulStop()` | SIGTERM 时 main.go 调用 |

注意：只注册了 **unary** 拦截器；本服务没有 streaming RPC。

### 4.2 `analytics_server.go` 函数表

| 方法签名 | 职责 | 备注/坑 |
|----------|------|---------|
| `AnalyticsServer` struct | 内嵌 `UnimplementedAnalyticsServiceServer`，持 `*AnalyticsUsecase` | |
| `NewAnalyticsServer(uc)` | 构造 | |
| `(s) ListAnalyticsRecords(ctx, req) (*...Response, error)` | ① `grpcauth.UserFromContext` 二次校验，失败 `codes.Unauthenticated`；② 取 page/pageSize 调 `uc.ListPage`；③ **之后**才归一化 page（<1→1）、pageSize（≤0→10，>50→50）；④ 逐条 `toProto`；⑤ 返回 items/total/page/page_size | 坑/细节：归一化发生在查询调用**之后**；查询安全性靠仓储层再次兜底（见 §7），响应里的 page/page_size 是归一化后的值 |
| `(s) GetAnalyticsRecord(ctx, req)` | 同样先验 ctx 用户；`uc.Get(req.GetId())`；错误一律 `codes.NotFound`（含 id≤0 的业务错误与不存在） | 成功转 proto 返回 |
| `toProto(item *entity.AnalyticsRecord) *analyticsv1.AnalyticsRecord` | nil→nil；35 字段逐一手写映射（Go entity 的 `MetricPV/MetricROI` 等 PascalCase → proto 的 `MetricPv/MetricRoi`） | 枚举没有，全是标量直拷；时间 int64 秒原样透传 |

错误码约定：未登录 `Unauthenticated`；列表内部错误 `Internal`（「列表查询失败: …」）；详情任何失败 `NotFound`。

## 5. metadata 鉴权：如何与 HTTP SessionAuth 对齐

源码锚点：`app/internal/delivery/grpc/interceptor/auth.go`

**通俗理解**：gRPC 没有 HTTP 头，但有个叫 metadata 的等价口袋。拦截器就是 gRPC 版的 Gin 中间件——每个 RPC 进门先搜身，口袋里票对不上就直接打回，对得上就把用户信息塞进 ctx，业务方法再从 ctx 取。

### 5.1 函数/类型表

| 函数/类型 | 职责 | 备注 |
|-----------|------|------|
| `Authenticator` struct | `local bool`、`jwtMgr *jwt.Manager`、`supabase config.SupabaseConfig`、`deviceSessionUC *DeviceSessionUsecase` | 二选一模式 |
| `NewLocal(jwtMgr, sessionUC)` | 本地 Auth：自签 JWT（UUID subject） | `cfg.Auth.IsLocalProvider()` 时 main.go 选用 |
| `NewSupabase(cfg, sessionUC)` | Supabase：远程校验 access token | |
| `(a) UnaryServerInterceptor() grpc.UnaryServerInterceptor` | 核心，见 §5.2 步骤 | 全局唯一拦截器 |
| `UserFromContext(ctx) (pkgauth.SupabaseUser, bool)` | 业务方法取已鉴权用户 | analytics_server 两个 RPC 都二次断言 |
| `bearerFromMD(md)` | 取 authorization，剥 `bearer ` 前缀（EqualFold）；没前缀也整串返回 | |
| `firstMD(md, key)` | metadata.Get(key) 第一个值 + TrimSpace；gRPC 键名自动小写，故用 `x-session-id` | |

metadata 键常量：`authorization`、`x-session-id`、`x-device-id`（与 Flutter 调用侧、HTTP 头逐字对齐）。

### 5.2 拦截步骤（请求链路第 1 段）

```text
1. metadata.FromIncomingContext → 没带 metadata：Unauthenticated「缺少 metadata」
2. bearerFromMD 取 token → 空：Unauthenticated「未提供 Authorization」
3. 验 token：
   - local：jwtMgr.ParseUUID(token) 失败 → Unauthenticated「token 无效」；
            成功 → user{ID: claims.Subject, Email: claims.Email}
   - supabase：common/auth.ValidateAccessToken(ctx, cfg, token) 远程 /auth/v1/user 校验
4. 取 x-session-id / x-device-id
5. deviceSessionUC.Validate(userID, email, sessionID, deviceID)：
   - ErrSessionReplaced → Unauthenticated(MsgSessionReplaced)（他机登录被顶）
   - 其他错误          → Unauthenticated(MsgSessionInvalid)
   （deviceSessionUC 为 nil 时跳过第 5 步）
6. ctx 注入 ContextUserKey（SupabaseUser）与 ContextAccessTokenKey（原始 token）→ handler
```

对齐关系：与 HTTP `middleware.SupabaseSessionAuth` / `LocalSessionAuth` 是**同一套 token + Redis 单设备 session**，区别仅载体（header ↔ metadata）与错误形态（302/401 JSON ↔ gRPC status code）。

## 6. Usecase：analytics_usecase.go

| 函数/类型 | 职责 | 备注 |
|-----------|------|------|
| `AnalyticsUsecase` struct / `NewAnalyticsUsecase(repo)` | 持 `domainrepo.AnalyticsRepository` | |
| `(u) ListPage(ctx, page, pageSize)` | 透传 repo，包成 `entity.AnalyticsListFilter` | 无用户维度过滤（试验数据全局共享） |
| `(u) Get(ctx, id)` | id≤0 返回「无效的记录 ID」；否则 repo.GetByID | |
| `(u) EnsureSeedData(ctx)` | ① `CountNotesContaining("seed_chart_v1")` >0 直接返回；② 否则 `DeleteBySourceSystem("seed")` 清旧种子（非 seed 手工数据保留）；③ `CreateBatch(buildAnalyticsSeedRecords(now))` | 启动时 main.go L433 调用，失败只 warn 不阻断启动 |
| `buildAnalyticsSeedRecords(now)` | 8 条手工场景 + 40 条程序化生成 = 48 条 | 见 §8 |
| `makeSeedRecord(now, index, pv, sc)` | 单条组装：漏斗、金额、ROI、时间偏移 | |
| `clamp01 / clamp100 / round1 / round2` | 数值护栏与精度工具 | bounce 钳 0–1；评分钳 0–100 |

常量 `analyticsSeedMarker = "seed_chart_v1"` 写进每条 notes，作为「种子版本」标记——版本升级只要换 marker，启动即自动替换整批种子。

## 7. 仓储：postgres/analytics_repo.go + entity

### 7.1 实体（`model/entity/analytics_record.go`）

- 表名：`analytics_records`（常量 `AnalyticsRecordsTable`）。**注意没有 `wys_` 前缀**——它是独立试验表，不属于业务表规范。
- 35 列与 proto 一一对应；GORM tag：`code/category/status/observed_at` 带 index；金额双精度 NOT NULL DEFAULT 0；`notes` text；时间全 int64 Unix 秒（不是 timestamptz）。
- `AnalyticsListFilter{Page, PageSize}` 仅分页，无条件过滤。

### 7.2 仓储接口（`model/repository/analytics_repository.go`）

| 方法 | 语义 |
|------|------|
| `ListPage(ctx, filter) ([]AnalyticsRecord, int64, error)` | 分页 + total |
| `GetByID(ctx, id) (*AnalyticsRecord, error)` | 详情 |
| `Count(ctx) (int64, error)` | 总数（seed 命令打印用） |
| `CountNotesContaining(ctx, marker) (int64, error)` | `notes LIKE '%marker%'` 计数，种子版本探测 |
| `DeleteBySourceSystem(ctx, sourceSystem) error` | 空串报错；按 source_system 批量删 |
| `CreateBatch(ctx, items) error` | 空切片直接返回 nil；否则一次 Create |

### 7.3 实现函数表（`analytics_repo.go`）

构造 `NewAnalyticsRepository(db) domainrepo.AnalyticsRepository`（返回接口类型）。

| 方法 | 关键逻辑 |
|------|----------|
| `ListPage` | **服务端再次归一化**：page<1→1、pageSize≤0→10、>50→50，offset=(page-1)*size；先 `Count` total（错误包「统计 analytics_records 失败」），再 `Order("observed_at DESC, id DESC").Offset.Limit.Find` |
| `GetByID` | `Where("id = ?").First`；`gorm.ErrRecordNotFound` 转中文「数据分析记录不存在」 |
| `Count` | Model.Count，错误包装 |
| `CountNotesContaining` | `notes LIKE ?` 参数化 `%marker%` |
| `DeleteBySourceSystem` | 防空串；Where Delete |
| `CreateBatch` | 空切片 no-op；`Create(&items)` 批量插入 |

排序固定「观测时间倒序、同时间 id 倒序」，与 Flutter 列表一致。

## 8. 图表数据是怎么「组装」出来的

**通俗理解**：Flutter 页面要画漏斗环（PV→UV→点击→转化逐级变窄）、收支柱、雷达图（质量/风险分）。随机数会画出「点击比浏览还多」的荒谬漏斗，所以种子不是纯随机，而是**先定一条 PV，再按比例逐层折算**，每层都做不反超的钳制。

### 8.1 8 个手工场景（seedScenario）

| 场景（title） | 刻意制造的图表形态 |
|---------------|--------------------|
| 双十一主会场漏斗 | featured、高转化正向样例（uv 0.42 / click 0.38 / convert 0.12、收支系数 8.5/2.2） |
| 异常：投放点击虚高 | anomaly 红条：clickRatio 0.72 但 convert 0.015、收支系数 0.6/4.8 成本倒挂、risk 86 |
| 精选：小程序裂变 | featured 琥珀条、convert 0.22 健康环 |
| 零点击冷启动页 | click/convert=0，转化环需显示「—」 |
| 高客单线下门店 | revMul 18 vs costMul 5.5，收入柱远高于成本 |
| 成本倒挂活动 | anomaly；rev 1.2 < cost 7.5，雷达风险偏高 |
| 留存召回短信 / 华北品牌搜索 | 中等形态补充品类分布 |

每条场景带：category/subCat/status/region/channel、三级比例（uvRatio 对 PV、clickRatio 对 UV、convertRatio 对点击）、revMul/costMul（相对 PV 千次的收入/成本系数）、bounce(0–1)、quality/risk(0–100)、featured/anomaly/priority/tagA/tagB。

### 8.2 40 条程序化数据

- 手工 8 条的 pv 为 `12000 + i*1800`；程序 40 条 `pv = 5000 + j*317`；比例按 j 取模循环：`uvR=0.28+(j%6)*0.04`、`clickR=0.18+(j%5)*0.05`、`convR=0.03+(j%9)*0.025`。
- 每 13 条（j%13==0）强制 click=convert=0，制造零点击样本。
- category/region/channel 各按 5 元素、status 按 4 元素取模轮转，保证图表维度有色块；title 从 8 个标题循环加序号。
- featured `j%9==0`、anomaly `j%11==0`。

### 8.3 `makeSeedRecord` 的折算公式（核心）

```go
uv      = round(pv * clamp01(uvRatio));      if uv > pv      { uv = pv }
click   = round(uv * clamp01(clickRatio));   if click > uv   { click = uv }
convert = round(click * clamp01(convertRatio)); if convert > click { convert = click }
unit    = pv / 1000.0
rev     = round2(unit * revMul * 120)
cost    = round2(unit * costMul * 80)
roi     = cost > 0 ? round2(rev / cost) : 0
```

- 三级 clamp 保证物理真实：**PV ≥ UV ≥ 点击 ≥ 转化**（SQL 种子注释里也写明该约束）。
- 收入/成本用「每千次 PV 单价 × 系数」线性放大；ROI 现算而不存假比例，cost=0 时 ROI=0 避免除零。
- bounce 钳 0–1，quality/risk 钳 0–100。
- 时间：`observed = now - index*3600`（每条隔 1 小时），窗口 = observed 前推 7 天，published = observed 前 1 小时；created/updated 以分钟错位。
- code `AN-0001` 定宽；owner 分析师 01–08、增长组 1–3 轮转；`source_system="seed"`；notes 前缀 `seed_chart_v1`。

### 8.4 两个播种入口

| 入口 | 行为 |
|------|------|
| API 启动（main.go L433） | `EnsureSeedData`：marker 已在则跳过，否则自动替换 |
| `cmd/seed-analytics`（`./scripts/load-env.sh go -C app run ./cmd/seed-analytics`） | 先显式 `DeleteBySourceSystem("seed")` 再 `EnsureSeedData`，打印 total/seed_chart_v1 计数；强制刷新 |
| `scripts/seed_analytics_records_chart_v1.sql` | 纯 SQL 手工最小集：事务内先删 seed 再插 4 条（AN-0001..0004），时间用 `EXTRACT(EPOCH FROM NOW())`；注释说明完整集约 48 条由 Go 写入 |

## 9. 装配与请求链路总览

### 9.1 main.go 关键节点

```text
L431-437 analyticsRepo := postgres.NewAnalyticsRepository(db)
         analyticsUC := NewAnalyticsUsecase(repo)
         EnsureSeedData(ctx)   // 失败仅 Warn
L545 条件：cfg.GRPC.Enabled && analyticsUC != nil && deviceSessionUC != nil && businessEnabled
L547-551 按 auth provider 选 grpcauth.NewLocal / NewSupabase（两者都不可用 → Warn 跳过启动）
L553-567 NewServer(Options{Port: cfg.GRPC.Port(默认 9090), Log, Authenticator, AnalyticsUC})
         go grpcServer.Serve()
关机：grpcServer.GracefulStop()
```

GRPC 配置（`common/config`）：`grpc.enabled`（未配置时默认启用，可用 `GRPC_ENABLED=false` 关）、`grpc.port` 默认 9090，环境变量 `GRPC_ENABLED/GRPC_PORT`。与 HTTP 同进程双端口。

### 9.2 一次列表 RPC 的完整链路

```text
Flutter AnalyticsGrpcApi
  → TCP :9090，metadata: authorization / x-session-id / x-device-id
  → grpc.Server: UnaryServerInterceptor（验 token + 单设备 session；注入 user 到 ctx）
  → _AnalyticsService_ListAnalyticsRecords_Handler（gen 解码）
  → AnalyticsServer.ListAnalyticsRecords（ctx 用户二次断言 + 分页归一化）
  → AnalyticsUsecase.ListPage
  → postgres.analyticsRepository.ListPage（再次兜底分页 → COUNT + SELECT ... ORDER BY observed_at DESC,id DESC LIMIT/OFFSET）
  → analytics_records（本地 PG）
  → toProto 逐条映射 → protobuf 二进制响应
```

## 10. 边界与注意

- 列表/详情**不带用户隔离**：analytics 是试验性全局数据，任何有效会话都能读全表。
- proto 时间一律 Unix 秒 int64（与业务表常见的 timestamptz 不同）。
- 只有 unary、只有服务端反射；无 streaming、无拦截器日志/metrics 链（只挂一个 auth 拦截器）。
- 生成代码勿手改，改契约走 `make proto`。
- 分页默认/上限在两处各做一次（server 响应层 + repo 查询层），属有意冗余兜底。

## 11. 学习要点回顾

1. proto 是单一真相源：Go struct、客户端 stub、服务端接口都从它生成；`Unimplemented` 内嵌保前向兼容。
2. gRPC metadata ≈ HTTP header：小写键 `authorization/x-session-id/x-device-id`，拦截器与 SessionAuth 共用 DeviceSession 校验，两套入口一张身份。
3. 拦截器做认证、业务方法用 `UserFromContext` 再断言——纵深防御，且让 usecase 保持与传输层无关。
4. gRPC 错误要用 `status.Error(codes.X, msg)`，不能把 HTTP 状态码搬过来。
5. 同进程双端口（:8080 HTTP / :9090 gRPC）是试验期形态，reflection 开启方便 grpcurl 调试。
6. 种子数据是「产品需求」：漏斗递减钳制、ROI 现算、评分边界、零点击样本，都是为了客户端图表的边界态联调。
7. marker 版本位（notes 里的 `seed_chart_v1`）让「清旧种子+灌新种子」幂等且可升级；删除只按 `source_system='seed'`，不碰手工数据。

## 12. 可动手验证

```bash
# 1) grpcurl 列服务/方法（reflection 已开；:9090）
grpcurl -plaintext 127.0.0.1:9090 list
grpcurl -plaintext 127.0.0.1:9090 list analytics.v1.AnalyticsService

# 2) 带 metadata 调列表（先从 /api/v1/user/login 拿 token + session_id）
grpcurl -plaintext \
  -H "authorization: Bearer <token>" \
  -H "x-session-id: <sessionId>" \
  -H "x-device-id: rt-test-device" \
  -d '{"page":1,"page_size":5}' \
  127.0.0.1:9090 analytics.v1.AnalyticsService/ListAnalyticsRecords

# 3) 无 metadata 应返回 Unauthenticated
grpcurl -plaintext -d '{"id":1}' 127.0.0.1:9090 analytics.v1.AnalyticsService/GetAnalyticsRecord

# 4) 强制刷新种子并观察计数
./scripts/load-env.sh go -C app run ./cmd/seed-analytics

# 5) 读代码任务
#  - 对照 proto 字段序号与 analytics.pb.go 结构体
#  - 在 analytics_server.go 找到分页归一化与 repo 兜底的两处重复
#  - 用 usecase 单测验证漏斗：go -C app test ./internal/usecase/ -run Analytics -v
```
