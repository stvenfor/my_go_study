# 阶段 3：app/internal 垂直切片（vertical slice）——「文件 → 域」映射表

> 出表日期：2026-09-29
> 前置：阶段 0/1/2 已完成（5 模块 go.work 落地，见 `.workbuddy/memory/2026-09-29.md`）
> 本表用途：**动手前的对账清单**。确认无误后按「分批执行顺序」逐批搬。

---

## 0. 前置状态与一个待办

### 0.1 阶段 0/1/2 已完成并验证

| 验收项 | 结果 |
|---|---|
| `go build all` | exit 0 |
| `go -C <mod> vet ./...`（4 模块） | 4/4 OK |
| `make build` | `bin/api` 47MB + `bin/worker` 36MB |
| `make test` | 19 包 ok / 0 FAIL |
| `docker build -f docker/Dockerfile .` | exit 0（4m17s），容器内 `/app/{api,worker,configs}` 齐备 |

### 0.2 ⚠️ 有 17 个文件未提交，且不是可丢的尾巴

`git status` 显示 17 个 modified，**HEAD 提交（`c55751a 组件化微服务`）只包含了目录搬迁 + 229 个 .go 的 import 重写**，下面这些「让新布局真正能用」的同步还在工作区：

```
.air.toml          Makefile           docker/Dockerfile      AGENTS.md
app/api/proto/analytics/v1/analytics.proto
app/cmd/{api,worker,import-supabase,seed-analytics,seed-home-todo}/main.go
docs/{startup-guide,coding-playbook-dual-repo,local-auth-postgres}.md
docs/grpc/analytics-flutter-trial.md
scripts/{lan-run,test_realtime_ws,test_scheduled_notify}.sh
```

实证：`git show HEAD:Makefile | grep -c 'go -C'` = **0**，工作区 = **10**。即 HEAD 版 Makefile 仍是旧写法 `go build ./cmd/api`，在新布局下**必然失效**。

**结论：开阶段 3 之前先 commit 这 17 个文件。** 否则阶段 3 的 diff 会和它们混在一起，回滚粒度被污染。

---

## 1. 目标形状：从 mashenGOAI 实测解码

没有照抄文档，是把 `goProject/mashenGOAI` 拆开看的。**三个反直觉点先说清**：

### 1.1 `model` 模块是「保留」的（不是被掏空）

```
mashenGOAI/model/          ← 9 个 .go，一域一文件
  base.go  users.go  agents.go  knowledges.go  subscriptions.go
  tools.go  market.go  workflows.go  llms.go
```

`model` 是**跨域实体层**（GORM 实体 + 共享 `BaseModel`），按域分文件但仍集中在 `model` 模块。
→ **我们不用掏空 model**。而且 `my_go_study/model/entity/*.go` 本来就已经是「一域一文件」（29 个），**基本零改动**。

### 1.2 域内 `<域>/model.go` 装的不是实体，是**仓储实现**

```
mashenGOAI/app/internal/llms/
  handler.go     ← HTTP 绑定（handler struct + NewHandler()）
  service.go     ← 业务逻辑（service struct，依赖 repository 接口）
  repository.go  ← 私有 interface repository（小写，域内可见）
  model.go       ← ✳ models struct{ db *gorm.DB }，实现 repository 接口 —— 是数据访问实现，不是实体
  req.go / res.go ← 请求 / 响应 DTO
```

命名反直觉，但这是它的实态。`core/` 里**只有 `ai/`**（Eino 基础设施），不是仓储实现层。

### 1.3 路由是「每域一个 Router struct」

```go
// app/internal/router/auths.go
type AuthRouter struct{}
func (u *AuthRouter) Register(engine *gin.Engine) {
    g := engine.Group("/api/v1/auth")
    h := auths.NewHandler()
    g.POST("/register", h.Register)
    ...
}
```

外加 `app/internal/inits/inits.go` 统一装配，`app/shared/` 放跨域共享。

### 1.4 目标目录（对齐后）

```
my_go_study/
├── go.work                      （已就绪，不变）
├── model/                       ← 保留：跨域实体（entity/*.go，一域一文件）
├── common/                      ← 保留：config/logger/jwt/auth/metrics
├── core/                        ← 保留：database/queue/jpush/rongcloud/supabase（基础设施）
├── thunder/                     ← 保留
└── app/
    ├── cmd/                     （6 个入口，不变）
    └── internal/
        ├── <域>/                 ← 新增：23 个域
        │   ├── handler.go        ← 原 delivery/http/controller/<域>_controller.go
        │   ├── service.go        ← 原 usecase/<域>_usecase.go
        │   ├── repository.go     ← 仓储接口（从 model/repository/ 下沉，小写）
        │   ├── repo_*.go         ← 仓储实现（从 internal/repository/ 拆入，见决策 2）
        │   ├── req.go / res.go   ← 原 dto/request|response 中该域的部分
        │   └── routes.go         ← 原 delivery/http/router/<域>_routes.go
        ├── router/               ← 每域一个 Router struct（照 1.3）
        ├── inits/                ← 装配（从 app/cmd/api/main.go 抽出）
        └── shared/               ← 跨域：middleware / resource_error / grpc interceptor
```

**现状规模**：`app/internal` 共 **204 个 .go / 30,430 行**，其中测试 **50 个**。

---

## 2. 完整映射表（23 个域）

记号：`svc`=usecase→service，`hdl`=controller→handler，`if`=仓储接口，`impl`=仓储实现，`ent`=实体，`rt`=路由

### 1) `auth` — 账号 / 登录 / 会话 / OTP / 三方登录　`/api/v1/user`
- svc ← `usecase/{account_login, user_usecase, phone_otp_usecase, local_auth_usecase, supabase_auth_usecase, wechat_auth_usecase, huawei_auth_usecase, device_session_usecase}.go`
- hdl ← `delivery/http/handler/user_handler.go`（+ `user_handler_account_test.go`）
- req ← `dto/request/user_request.go`　res ← `dto/response/user.go`
- if ← `model/repository/session_repository.go`（单设备登录会话）
- impl ← `repository/redis/session_repo.go`、`repository/postgres/consolidate_users.go`
- ent ← `model/entity/{auth_user.go, user.go}`
- rt ← `router/user_routes.go`

### 2) `profile` — 用户资料 + 收货地址　`/profiles` `/me` `/user/addresses`
- svc ← `usecase/{profile_usecase, address_usecase}.go`
- hdl ← `controller/{profile_controller, address_controller}.go`
- res ← `dto/response/profile.go`（+ `profile_store_list_test.go`）
- if ← `model/repository/{profile_repository, address_repository}.go`
- impl ← `postgres/{profile_repo, address_repo}.go`、`supabase/profile_repo.go`
- ent ← `model/entity/{profile.go, address.go}`
- rt ← `router/profile_routes.go`
- ✳ 注：`address` 的路由现挂在 `user_routes.go` 里，建议一并挪到本域

### 3) `access` — 门店 / 成员 / 角色 / 权限　`/stores` `/roles` `/me/permissions`
- svc ← `usecase/{access_usecase.go, store_scope.go}`
- hdl ← `controller/access_controller.go`
- if ← `model/repository/{access_repository, store_stats_repository}.go`
- impl ← `postgres/{access_repo.go, store_stats_repo.go}`
- ent ← `model/entity/{access.go, store.go, store_test.go}`
- rt ← `router/access_routes.go`

### 4) `home` — 首页待办 / 加盟申请　`/home` `/stores/:id/join-applications`
- svc ← `usecase/home_todo_usecase.go`　hdl ← `controller/home_todo_controller.go`
- if ← `model/repository/home_todo_repository.go`
- impl ← `postgres/{home_todo_repo.go, home_todo_packing_demo.go}`
- ent ← `model/entity/home_todo.go`　rt ← `router/home_todo_routes.go`

### 5) `mall` — 商城 / SKU / 购物车 / 订单　`/mall`
- svc ← `usecase/mall_usecase.go`　hdl ← `controller/mall_controller.go`
- if ← `model/repository/mall_repository.go`　impl ← `postgres/{mall_repo.go, mall_seed.go}`
- ent ← `model/entity/mall.go`　rt ← `router/mall_routes.go`

### 6) `points` — 积分 / 签到 / 任务　`/points`
- svc ← `usecase/points_usecase.go`　hdl ← `controller/points_controller.go`
- if ← `model/repository/points_repository.go`
- impl ← `postgres/{points_repo.go, points_task_progress.go}`
- ent ← `model/entity/points.go`　rt ← `router/points_routes.go`

### 7) `wallet` — 现金钱包 / 银行卡 / 充值　`/wallet`
- svc ← `usecase/cash_wallet_usecase.go`　hdl ← `controller/cash_wallet_controller.go`
- if ← `model/repository/cash_wallet_repository.go`　impl ← `postgres/cash_wallet_repo.go`
- ent ← `model/entity/{cash_wallet.go, cash_money.go}`　rt ← `router/cash_wallet_routes.go`

### 8) `membership` — 会员 / 买断 / IAP 校验　`/membership`
- svc ← `usecase/{membership_usecase.go, apple_iap_verifier.go, huawei_iap_verifier.go}`
- hdl ← `controller/membership_controller.go`
- if ← `model/repository/membership_repository.go`　impl ← `postgres/membership_repo.go`
- ent ← `model/entity/membership.go`　rt ← `router/membership_routes.go`

### 9) `payment` — 支付预下单　`/payments`
- svc ← `usecase/payment_usecase.go`　hdl ← `controller/payment_controller.go`
- rt ← `router/payment_routes.go`　（无自有仓储/实体）

### 10) `transaction` — 交易管理 / 全文检索　`/transactions` `/transactions/manage`
- svc ← `usecase/transaction_usecase.go`　hdl ← `controller/transaction_controller.go`
- req ← `dto/request/transaction_request.go`　res ← `dto/response/transaction.go`
- if ← `model/repository/{transaction_repository, transaction_search}.go`
- impl ← `postgres/transaction_repo.go`、`supabase/transaction_repo.go`、`es/transaction_store.go`
- ent ← `model/entity/{transaction.go, transaction_record.go}`　rt ← `router/transaction_routes.go`

### 11) `used_car` — 二手车订单　`/used-car-orders`
- svc ← `usecase/used_car_order_usecase.go`　hdl ← `controller/used_car_order_controller.go`
- if ← `model/repository/used_car_order_repository.go`　impl ← `postgres/used_car_order_repo.go`
- ent ← `model/entity/used_car_order.go`　rt ← `router/used_car_order_routes.go`

### 12) `new_car_follow` — 新车跟进档案　`/new-car-follow-files`
- svc ← `usecase/new_car_follow_usecase.go`　hdl ← `controller/new_car_follow_controller.go`
- if ← `model/repository/new_car_follow_repository.go`　impl ← `postgres/new_car_follow_repo.go`
- ent ← `model/entity/new_car_follow.go`　rt ← `router/new_car_follow_routes.go`

### 13) `after_sales` — 售后专区　`/after-sales`
- svc ← `usecase/after_sales_zone_usecase.go`　hdl ← `controller/after_sales_zone_controller.go`
- if ← `model/repository/after_sales_zone_repository.go`　impl ← `postgres/after_sales_zone_repo.go`
- ent ← `model/entity/after_sales_record.go`　rt ← `router/after_sales_zone_routes.go`

### 14) `deal_invoice` — 成交单据　`/deal-invoices`
- svc ← `usecase/deal_invoice_usecase.go`　hdl ← `controller/deal_invoice_controller.go`
- if ← `model/repository/deal_invoice_repository.go`　impl ← `postgres/deal_invoice_repo.go`
- ent ← `model/entity/deal_invoice.go`　rt ← `router/deal_invoice_routes.go`

### 15) `purchase_calculator` — 购车计算器　`/purchase-calculator`（公开，无鉴权）
- svc ← `usecase/purchase_quote_usecase.go`　hdl ← `controller/purchase_calculator_controller.go`
- if ← `model/repository/auto_finance_repository.go`　impl ← `postgres/auto_finance_repo.go`
- ent ← `model/entity/auto_finance.go`　rt ← `router/purchase_calculator_routes.go`

### 16) `community` — 社区 / 帖子 / 关注　`/community`
- svc ← `usecase/community_usecase.go`　hdl ← `controller/community_controller.go`
- if ← `model/repository/community_repository.go`　impl ← `postgres/community_repo.go`
- ent ← `model/entity/community.go`　rt ← `router/community_routes.go`

### 17) `short_video` — 短视频　`/short-videos`
- svc ← `usecase/short_video_usecase.go`　hdl ← `controller/short_video_controller.go`
- if ← `model/repository/short_video_repository.go`
- impl ← `postgres/{short_video_repo.go, short_video_seed.go}`
- ent ← `model/entity/short_video.go`　rt ← `router/short_video_routes.go`

### 18) `im` — 即时通讯（好友 / 群 / 会话 / 备份）　`/im`
- svc ← `usecase/{im_friend_usecase, im_group_usecase, im_session_usecase, im_backup_usecase}.go`
- hdl ← `controller/im_controller.go`
- if ← `model/repository/{im_repository.go, conversation_repository.go}`
- impl ← `postgres/im_repo.go`、`redis/conversation_repo.go`
- ent ← `model/entity/im.go`　rt ← `router/im_routes.go`
- 基础设施 ← `core/rongcloud`

### 19) `realtime` — 实时通道（WS / 在线态 / 增量同步 / ticket）　`/realtime` `/ws`
- svc ← `usecase/{realtime_presence, realtime_push, realtime_sync, realtime_ticket}_usecase.go`
- hdl ← `controller/realtime_controller.go`　✳ ws ← `delivery/ws/{client.go, handler.go, hub.go}` + 3 个测试
- req ← `dto/request/realtime_request.go`　res ← `dto/response/realtime.go`
- if ← `model/repository/{presence_repository, realtime_event_repository, ws_ticket_repository}.go`
- impl ← `redis/{presence_repo, realtime_event_repo, ws_ticket_repo}.go`
- ent ← `model/entity/realtime_envelope.go`　rt ← `router/realtime_routes.go`

### 20) `notify` — 推送（JPush / 小时级通知 / worker 分发器）　`/push`
- svc ← `usecase/{jpush_usecase.go, hourly_notify_usecase.go}`
- hdl ← `controller/jpush_controller.go`　✳ worker ← `delivery/worker/handler.go`（业务分发器）
- if ← `model/repository/push_device_repository.go`
- impl ← `postgres/push_device_repo.go`、`repository/queue/push_enqueuer.go`
- ent ← `model/entity/{push_device.go, sys_notify_payload.go(+test)}`
- rt ← `router/jpush_routes.go`　基础设施 ← `core/{jpush, queue}`

### 21) `ai` — SSE 流式补全 + 智能客服 Agent　`/sse` `/cs-agent`
- svc ← `usecase/{completion_usecase.go, cs_agent_usecase.go}`
- hdl ← `controller/{sse_controller.go, cs_agent_controller.go}`
- req ← `dto/request/{sse_completion.go, cs_agent_chat.go}`
- impl ← `repository/llm/{mock_stream_provider.go, openai_compatible_provider.go}`、
  `repository/csagent/tx_search_tool.go`
- ent/if ← `model/entity/sse_event.go`、`model/provider/stream_provider.go`
- rt ← `router/{sse_routes.go, cs_agent_routes.go}`

### 22) `analytics` — 统计（gRPC，无 HTTP 路由）
- svc ← `usecase/analytics_usecase.go`　hdl ← `delivery/grpc/analytics_server.go`
- if ← `model/repository/analytics_repository.go`　impl ← `postgres/analytics_repo.go`
- ent ← `model/entity/analytics_record.go`

### 23) `shared` — 跨域 / 平台（**不切分，只挪目录**）
- `delivery/http/middleware/*`（11 文件：auth / cors / logger / prometheus / device_session_error /
  local_session_auth / require_active_account / supabase_auth / supabase_session_auth(+test)）
- `delivery/http/router/{router.go, session_auth_chain.go}`　→ 目标 `internal/router/`
- `delivery/http/controller/{resource_error.go, supabase_context.go}`（共享错误与上下文）
- `delivery/grpc/{server.go, interceptor/auth.go}`
- `dto/response/{common.go, backend.go, common_test.go}`
- `cmd/*`（6 个入口）

---

## 3. 三个需要拍板的决策

### 决策 1：`model` 模块留不留？
- **A（推荐，= mashenGOAI 原样）**：`entity/*` 留 `model` 模块，仓储接口下沉到域内（小写 `repository`）。
  → `model` 模块保留，只剩 entity + provider。**改动小，实体零搬迁。**
- B：连 entity 也下沉 → `model` 模块清空、退化成 **4 模块**，且要回改 `go.work`/各 `go.mod`/Dockerfile/AGENTS。

### 决策 2：仓储实现放哪？（影响改动量最大的一项）
- **A（严格照搬 mashenGOAI）**：实现拆进各域 `model.go`（`models struct{db *gorm.DB}`）。
  → `postgres/` 25 个文件按域拆散。**最「正宗」，diff 最大。**
- **B（最省事）**：实现全部留在技术层 `app/internal/repository/*`，只切 handler/service/req/res。
  → 代码**半垂直**：交付层垂直、实现层仍横向。
- **C（推荐）**：**接口 + 实现一起进域** —— `<域>/repository.go`（接口）+ `<域>/repo_postgres.go`、`<域>/repo_redis.go`（实现）。
  → 完整的垂直切片，同时比把实现塞进 `model.go` 更可读；`internal/repository/` 目录消失。

### 决策 3：路由装配形式
- **A（推荐，= mashenGOAI）**：`internal/router/<域>.go` 一个 `XxxRouter struct` + `Register(engine)`，
  新增 `internal/inits/inits.go` 统一装配。
  → 代价：现在 `router.Options` 一个大 struct 集中注入，拆开后各域要自带构造函数，**装配代码量会上升**（换来的好处是域自治）。
- B：保留现有 `registerXxxRoutes(v1, ctrl, ...)` 函数形式，仅挪目录 + 改 package。

---

## 4. 分批执行顺序（每批结束跑 `go build all` + 定向测试）

| 批次 | 域 | 特点 |
|---|---|---|
| **批 1（试点）** | `home` `points` `wallet` | 各 5-7 文件，无跨域耦合，用来验证切片模板 |
| **批 2（汽车业务）** | `used_car` `new_car_follow` `after_sales` `deal_invoice` `purchase_calculator` | 结构高度同构（list/summary/customers/get/create） |
| **批 3（电商 + 社区）** | `mall` `membership` `payment` `community` `short_video` `transaction` | transaction 牵涉 ES + Supabase 双实现 |
| **批 4（重头）** | `auth` `profile` `access` `im` `realtime` `notify` `ai` `analytics` | 跨域耦合最多，放最后 |

每批固定五步：① 建目录搬文件 → ② 改 package 名 + import → ③ 接路由装配 → ④ `go build all` → ⑤ `go -C app test ./...` 定向绿。

**回滚点**：批 1 之前先打一次快照（`git commit` 那 17 个文件 + `tar`），每批结束 commit 一次，保证批次级回滚粒度。

---

## 5. 风险清单

1. **`router.Options` 巨型 struct 拆解**：现在 30+ 字段集中注入，切分后需为每域写构造函数，装配层是最容易出编译错的地方。
2. **`delivery/ws` 与 `delivery/worker` 归属**：按语义归 `realtime` / `notify`，但它们是「按传输形态建目录」的产物，搬进域后 `realtime` 域会明显偏大（12 文件 + 3 测试）。
3. **`internal/repository/queue/push_enqueuer.go`**：这是阶段 0 新建的**跨层适配器**（`usecase.RealtimePushInput` → `queue.PushNotifyPayload`），归 `notify` 域后需确认 `realtime_push` 侧不反向依赖。
4. **测试文件 50 个**：多数与被测文件同名同包，搬迁时必须同步 `_test.go`，否则 `go test` 会因 package 不一致直接编译失败。
5. **`git mv` 与未暂存状态**：上次踩过 —— `git mv` 撞「已删除未暂存」会失败，用普通 `mv` 让 git 自动识别重命名。
