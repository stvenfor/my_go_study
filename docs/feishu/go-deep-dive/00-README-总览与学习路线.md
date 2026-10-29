# my_go_study BFF 深度学习 · 总览与学习路线

> 一句话定位：这是一套「Flutter/KMP 客户端 ↔ Go BFF ↔ Postgres/Redis/RabbitMQ/ES」的实战后端，Gin + gRPC 同进程双端口、Clean Architecture 与垂直切片并存。本篇是 14 篇函数级深度文档的总入口，建议按下方路线阅读。
>
> 前置要求：会 Go 基础语法、了解 HTTP/JSON；不要求提前懂 gRPC、WebSocket、RabbitMQ（相关篇章都有通俗解释）。

## 一、这个项目到底是什么（通俗版）

可以把它想象成一家汽车生活 App 的「**后厨中央厨房**」：

- 前台是手机 App（Flutter `my_ai_project` / KMP 客户端），只负责点菜（发 HTTP 请求）和看直播（WebSocket 推送）。
- 后厨就是本仓库：接单窗口（Gin HTTP `:8080`）、数据分析专线（gRPC `:9090`）、广播喇叭（WebSocket）、外卖配送（RabbitMQ + Worker）、食材仓库（Postgres/Redis/Elasticsearch）。
- 一个进程同时开两扇门（HTTP 8080 + gRPC 9090），另有一个独立的 Worker 进程专吃异步任务。

专业一句话：**Go 编写的 BFF（Backend For Frontend）聚合层**，对客户端屏蔽 Supabase/本地 Auth、Postgres、Redis、RabbitMQ、ES、各类三方（微信/华为/支付宝/融云/JPush）的差异，对上提供统一的 REST + WebSocket + gRPC 接口。

## 二、整体架构（一图流）

```text
Flutter / KMP 客户端
   │  HTTP JSON :8080（业务 REST + SSE）
   │  WebSocket  :8080（Realtime，ticket 鉴权）
   │  gRPC       :9090（数据分析，metadata 鉴权）
   ▼
app/cmd/api（同进程）
   ├─ middleware：CORS/Logger/Prometheus → SessionAuth/JWT
   ├─ delivery/http：legacy 控制器 + 15 个垂直业务域（router→handler→service→repository）
   ├─ delivery/ws：Hub/Client（连接管理 + Redis Pub/Sub 跨实例扇出）
   └─ delivery/grpc：AnalyticsService
                    │
        ┌───────────┼───────────────┬──────────────┐
        ▼           ▼               ▼              ▼
   Postgres     Redis          RabbitMQ       Elasticsearch
  (wys_* 表)  (session/      (异步任务)       (交易全文)
              ticket/pubsub)     │
                                 ▼
                          app/cmd/worker（独立进程）
                          消费任务 → 推送/Pub-Sub 广播/定时通知
```

配套的可交互架构图（含源码跳转）已同步在本目录上级「go学习」中：文件节点 `my_go_study-BFF整体架构图.html`，下载后浏览器打开。

## 三、go.work 四模块（先记住依赖方向）

| 模块 | 角色 | 一句话 |
|------|------|--------|
| `app/` | 唯一可执行模块 | cmd/api、cmd/worker + 全部业务代码 |
| `common/` | 叶子底座 | config（Viper）、logger（zap）、jwt、auth、metrics |
| `core/` | 外部系统驱动 | database（PG/Redis）、queue（RabbitMQ/PubSub）、supabase、jpush、rongcloud |
| `model/` | 领域层 | entity（表结构）、repository（仓储端口）、provider（LLM 等端口） |

依赖铁律：`app → {common, core, model}`、`core → {common, model}`；common/model 不许反向依赖。另有本地三方库 `thunder/`（replace 引入，当前业务代码未实际 import）。

## 四、14 篇文档地图

### 第一层：看懂骨架（建议按序）

| 序号 | 文档 | 解决的问题 |
|------|------|-----------|
| 01 | [工程结构与 go-work 多模块](01-工程结构与go-work多模块.md) | 仓库长什么样、Makefile/Docker 怎么跑、多模块怎么协作 |
| 02 | [配置体系](02-配置体系.md) | 配置从哪来、18 个配置段每个字段什么意思、env 怎么覆盖 |
| 03 | [启动流程与依赖注入](03-启动流程与依赖注入.md) | main.go 680 行干了什么、15 个装配阶段、Worker 与 4 个工具命令 |

### 第二层：看懂请求怎么进出

| 序号 | 文档 | 解决的问题 |
|------|------|-----------|
| 04 | [HTTP 路由与中间件](04-HTTP路由与中间件.md) | 146 条路由总表、9 个中间件逐个函数、响应/错误封装 |
| 05 | [认证与会话](05-认证与会话.md) | 登录/OTP/微信/华为、单设备互踢、两套 Token、白名单与 TTL=0 |
| 09 | [Realtime 与 WebSocket](09-Realtime与WebSocket.md) | WS 连接、ticket 鉴权、同步/异步广播、在线状态 |
| 10 | [gRPC 数据分析](10-gRPC数据分析.md) | proto 契约、metadata 鉴权、图表数据怎么组装 |
| 12 | [SSE 与客服 Agent](12-SSE与客服Agent.md) | AI 流式输出、Eino Tool 调 ES 搜索、mock/真模型切换 |

### 第三层：看懂业务与数据

| 序号 | 文档 | 解决的问题 |
|------|------|-----------|
| 06 | [通用业务域](06-通用业务域.md) | 权限 RBAC、首页待办、支付预下单、钱包、积分任务、会员/IAP |
| 07 | [汽车业务域](07-汽车业务域.md) | 收支交易（PG/Supabase/ES 三仓储）、二手车、跟进 CRM、报价器、发票 |
| 08 | [电商与社区域](08-电商与社区域.md) | 商城（混合支付/补偿）、社区问大家、短视频过审、售后核销 |
| 13 | [数据层与存储](13-数据层与存储.md) | 30 个实体逐表、11 个仓储端口、GORM/AutoMigrate、PostgREST、ES 客户端 |

### 第四层：看懂异步与运维

| 序号 | 文档 | 解决的问题 |
|------|------|-----------|
| 11 | [消息队列与 Worker](11-消息队列与Worker.md) | RabbitMQ 收发语义、任务分发表、Pub/Sub 扇出、定时通知幂等 |
| 14 | [可观测性与外部集成](14-可观测性与外部集成.md) | Prometheus 指标/告警、JPush、融云 IM、Docker lan 联调 |

## 五、推荐学习路线（三条）

**路线 A · 后端新手（约 1 周，每天 2 小时）**
01 → 02 → 跑起 `make deps-up && make run`，curl `/health` → 03（对照进程启动日志看）→ 04（挑 `/api/v1/user/login` 一条路由跟到底）→ 05 → 06 的 wallet 域（最简单的 CRUD）→ 13。

**路线 B · 有后端经验，重点学架构（3-4 天）**
01 → 03 → 04 → 05（双轨认证是最绕的部分）→ 07 的 transaction 三仓储对比 → 09 + 11（实时+异步是本项目精华）→ 10 → 14。

**路线 C · 带着问题查（不用顺序读）**
- 「某接口 401/被踢下线」→ 05 + 04 的中间件链
- 「消息为什么没收到/重复收到」→ 09 + 11
- 「新增一张业务表/一个接口要改哪里」→ 06（照抄 wallet 五件套）+ 13 + 03 的装配阶段
- 「配置不生效」→ 02 的加载链与 env 覆盖表
- 「AI 回复从哪来」→ 12
- 「Docker/局域网联调」→ 01 + 14

## 六、阅读代码的统一方法（每个业务域都适用）

以 `wallet`（钱包）为例，五件套的阅读顺序：

1. `wallet/router.go`：先看「有哪些 URL」，建立外部视图；
2. `wallet/handler.go`：看请求怎么解析、响应怎么包（HTTP 适配层，不含业务）；
3. `wallet/service.go`：业务规则全在这里（事务、余额校验）；
4. `wallet/repository.go`：接口定义（Service 依赖抽象，不依赖 GORM）；
5. `wallet/repo_postgres.go`：接口的 GORM 实现，对照 `wys_*` 表。

请求链路固定为：

```text
Router（挂路由 + 中间件）
  → Handler（bind/validate DTO，从会话取 userID）
    → Service（业务逻辑 + 事务）
      → Repository 接口 → RepoPostgres（GORM）→ Postgres
  ← 统一 ResponseEnvelope（code/message/data）
```

## 七、关键数字与约定（速读卡）

| 项 | 值 | 出处 |
|----|----|------|
| HTTP 端口 | 8080 | config `server.port` |
| gRPC 端口 | 9090 | config `grpc.port` |
| 路由总数 | 146（顶层 3 + legacy 43 + 15 业务域 100） | 04 篇逐条统计 |
| 业务域数量 | 15 个垂直切片 | `app/internal/*/` |
| 数据表前缀 | `wys_`（新表强制） | 项目规范 |
| Session Redis key | `auth:session:{userID}` | DeviceSession |
| Pub/Sub 频道 | `realtime:fanout` | 跨实例 WS 扇出 |
| RabbitMQ 队列 | `my_go_study.tasks` | Worker 消费 |
| ES 索引 | `transactions`（user_id keyword 隔离） | transaction/repo_es.go |
| 测试手机号 | 13400000000 / 验证码 123456（dev） | local auth bypass |

## 八、文档维护说明

- 本系列 14 篇均为**函数级、源码锚点**文档，源文件在仓库 `docs/feishu/go-deep-dive/`（GFM Markdown），随代码演进用仓库同步脚本更新飞书副本。
- 代码引用格式为 `相对路径:行号`；行号会随代码变动漂移，函数名/表名/路由是更稳定的检索锚点。
- 文档基于撰写时的工作区代码；已知的占位/假实现（mock LLM、SMS 桩、JPush 占位等）在各篇「假实现边界」中显式列出。
