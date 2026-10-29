# 04-HTTP 路由与中间件

> 一句话定位：本文讲清 BFF 的「门卫与楼层分布图」——Gin 引擎怎么装配、146 条路由每条挂在哪、请求要过哪几道中间件、统一响应与错误长什么样。所属模块：app（`delivery/http` legacy 层 + 各垂直切片域 `router.go`）。阅读前置：知道 Gin 的 `Engine/Group/HandlerFunc/中间件 c.Next()/c.Abort()` 基本概念即可。

## 通俗理解

把整个 HTTP 服务想成一栋写字楼：

- **路由表**就是楼层导览牌：访客报一个「方法 + 路径」（如 `POST /api/v1/user/login`），前台把他领到对应窗口（handler）。
- **全局中间件**是大楼的四道闸机，人人必过：`Recovery`（有人晕倒先急救，别全楼瘫痪）、`CORS`（浏览器跨域放行单）、`PrometheusHTTP`（数人头、掐秒表）、`RequestLogger`（登记来访日志）。
- **SessionAuth 中间件**是办公区门禁：刷 access token + 校验「你是不是这台设备上的当前会话」，过了才把身份塞进 `gin.Context`，业务窗口只认门禁写入的身份，不要客户端自己报姓名。
- **RequireActiveAccount** 是门禁后的第二道查验：账号已注销/停用的人，手里就算还有有效门卡也拦下。
- **两套响应信封**：统一信封 `{code,message,data,timestamp}` 给管理端/新接口用；Flutter App 历史原因吃的是裸 JSON，错误时 `{error, code}`，由 `backend.go` 兼容。

全楼目前共 **146 条路由**（含 `/health`、`/metrics`、1 条 WebSocket），其中 100 条属于已切成垂直切片的 15 个业务域，46 条仍挂在 legacy 集中装配里。

源码主锚点：`app/internal/delivery/http/router/`、`app/internal/delivery/http/middleware/`、`app/internal/delivery/http/authctx/authctx.go`、`app/internal/shared/{http,errors,resource_error,dberr}.go`、`app/internal/delivery/http/dto/`。

---

## 1. 引擎装配：router.Setup

源码锚点：`app/internal/delivery/http/router/router.go#L54-L124`

`Setup(opts Options) *gin.Engine` 是唯一装配入口，由 `app/cmd/api/main.go#L511-L530` 调用。装配顺序固定：

1. `gin.New()`（注意**没有**用 `gin.Default()`，日志器用自己的 `RequestLogger`）。
2. 全局四道：`gin.Recovery()` → `middleware.CORS()` → `middleware.PrometheusHTTP()` → `middleware.RequestLogger(log)`。
3. 裸路由：`GET /health`（内联 200）、`GET /metrics`（Prometheus 拉取口，注释明确「不经 SessionAuth，仅本机/内网可达」）。
4. 建 `/api/v1` 路由组。
5. 决定业务认证开关与 sessionAuth 实现（见下）。
6. 调 legacy `registerXxxRoutes` 注册 user/profile/realtime/push/im/sse/cs-agent。
7. WebSocket：`GET {cfg.Realtime.WsPath}`（空则默认 `/realtime/v1/connect`）→ `WSHandler.ServeWS`，**不带 SessionAuth**（它在升级时自己用一次性 ticket 鉴权，见 realtime 相关文档）。
8. 遍历 `opts.DomainRouters`，逐个 `domain.Register(r)`。

### 1.1 Options 字段表

源码锚点：`router.go#L25-L51`

| 字段 | 类型 | 职责 | 备注 |
| --- | --- | --- | --- |
| Log | `*zap.Logger` | 给 RequestLogger | 可为 nil 时 gin 外日志缺失（装配处必传） |
| Mode | `string` | gin 模式 | `debug/release/test`，来自 `cfg.Server.Mode` |
| JWTManager | `*jwtmanager.Manager` | 本地 JWT 解析 | sessionAuth 为 nil 时 Setup 用它自建 LocalSessionAuth |
| UserHandler | `*handler.UserHandler` | 用户域 7+2+6 条路由 | registerUserRoutes 必调 |
| ProfileController | `*controller.ProfileController` | 资料 6 条 | nil 则不注册 |
| AddressController | `*controller.AddressController` | 收货地址 6 条 | 仅 sbAuth != nil 且非 nil 时挂在 /user/addresses |
| RealtimeController | `*controller.RealtimeController` | ws-ticket/sync/push | businessAuth + 非 nil + sessionAuth 非 nil |
| JPushController | `*controller.JPushController` | 设备登记/推送 | 同上 |
| ImController | `*controller.ImController` | 融云 IM 15 条 | 同上 |
| SseController | `*controller.SseController` | SSE 补全 1 条 | 额外要求 `cfg.SSE.Enabled` |
| CsAgentController | `*controller.CsAgentController` | 客服 Agent 1 条 | 额外要求 `cfg.CsAgent.Enabled` |
| WSHandler | `*wshandler.Handler` | WebSocket 升级 | nil 则不挂 wsPath |
| Config / Supabase | `config.Config` / `config.SupabaseConfig` | 认证分支与开关 | `Auth.IsLocalProvider()` / `Supabase.Enabled()` |
| DeviceSessionUC | `*usecase.DeviceSessionUsecase` | 单设备会话校验 | 自建 sessionAuth 的必要条件 |
| AccountGate | `gin.HandlerFunc` | 账号状态门 | 装配处传 `middleware.RequireActiveAccount(localAuth)`；Supabase 分支为 nil（main.go 中该变量只在 local 分支赋值） |
| SessionAuth | `gin.HandlerFunc` | 业务会话中间件 | 为 nil 时 Setup 按 Config 自建，兼容旧调用方 |
| DomainRouters | `[]shared.Registrable` | 15 个切片域路由 | 各自 `Register(engine)` |

### 1.2 sessionAuth 的选择逻辑

`router.go#L70-L78`，三选一：

| 条件 | 结果 |
| --- | --- |
| `opts.SessionAuth` 已显式传入 | 直接用（main.go 实际走这条：`main.go#L441-L448` 统一构造一次，legacy 路由与切片域共用） |
| 未传 + local provider + DeviceSessionUC 非 nil | `middleware.LocalSessionAuth(JWTManager, DeviceSessionUC)` |
| 未传 + Supabase 启用 + DeviceSessionUC 非 nil | `middleware.SupabaseSessionAuth(Supabase, DeviceSessionUC)` |
| 两种 provider 都没启用 | sessionAuth == nil，所有受保护路由/域整体不注册，启动日志告警「未启用业务认证」 |

`businessAuth := cfg.Auth.IsLocalProvider() || cfg.Supabase.Enabled()` 是所有 legacy 受保护组的总闸门。

---

## 2. 全量路由总表

### 2.1 怎么读这张表

- **全局链**：每条路由实际执行顺序都是 `Recovery → CORS → PrometheusHTTP → RequestLogger → （组级链）→ handler`，下表「组级中间件」列只写 `/api/v1` 组内追加的部分。
- `SA` = SessionAuth（local 模式是 `LocalSessionAuth`，云模式是 `SupabaseSessionAuth`，二选一，全进程一致）。
- `Gate` = `RequireActiveAccount`（仅 local provider 下装配；Supabase 模式为 nil，链自动缩短）。
- 受保护链统一由两个同构 helper 生成：legacy 域用 `router.sessionAuthChain`（`router/session_auth_chain.go#L6-L12`），切片域用 `shared.SessionAuthChain`（`shared/http.go#L10-L16`），逻辑完全一样：`[SA]`，Gate 非 nil 再 append Gate。
- handler 列中 `X·handler.F` 表示 `app/internal/X` 包内未导出类型 `handler` 的方法；`controller.X` 表示 `delivery/http/controller` 包。

### 2.2 顶层裸路由（3 条）

| 方法 | 路径 | 组级中间件 | handler | 鉴权要求/备注 |
| --- | --- | --- | --- | --- |
| GET | `/health` | 无 | 内联 `c.JSON(200,{"status":"ok"})` | 公开；PrometheusHTTP 对其跳过计数 |
| GET | `/metrics` | 无 | `gin.WrapH(metrics.Handler())` | 公开（靠网络层隔离）；跳过自身指标计数 |
| GET | `/realtime/v1/connect`（默认，可被 `cfg.Realtime.WsPath` 覆盖） | 无 | `(*ws.Handler).ServeWS`（`delivery/ws/handler.go#L58`） | 不走 SA；升级时用一次性 ws-ticket 鉴权；WSHandler 为 nil 时不注册 |

### 2.3 用户与地址组 `/api/v1/user`（15 条）

源码锚点：`router/user_routes.go`。7 条登录注册类公开；`sbAuth != nil` 时才追加 8 条受保护路由。

| 方法 | 路径 | 组级中间件 | handler | 鉴权要求 |
| --- | --- | --- | --- | --- |
| POST | `/api/v1/user/register` | 无 | `handler.UserHandler.Register` | 公开 |
| POST | `/api/v1/user/login` | 无 | `UserHandler.Login` | 公开 |
| POST | `/api/v1/user/refresh` | 无 | `UserHandler.Refresh` | 公开（凭 refresh_token） |
| POST | `/api/v1/user/phone/otp/send` | 无 | `UserHandler.SendPhoneOTP` | 公开 |
| POST | `/api/v1/user/phone/otp/verify` | 无 | `UserHandler.VerifyPhoneOTP` | 公开 |
| POST | `/api/v1/user/wechat/login` | 无 | `UserHandler.LoginWithWechat` | 公开 |
| POST | `/api/v1/user/huawei/login` | 无 | `UserHandler.LoginWithHuawei` | 公开 |
| POST | `/api/v1/user/logout` | SA + Gate | `UserHandler.Logout` | 会话有效；身份取自上下文 user_id |
| POST | `/api/v1/user/deactivate` | SA + Gate | `UserHandler.Deactivate` | 同上；local 实现软删账号，supabase 实现无该方法时 handler 回 400（详见 05 篇 §7）。profile_routes.go 头注释「本地模式再拒绝注销或停用账号」指的是组链上的 Gate，不是本路由 |
| GET | `/api/v1/user/addresses` | SA + Gate | `controller.AddressController.List` | 会话有效；AddressController 非 nil 才注册 |
| POST | `/api/v1/user/addresses` | SA + Gate | `AddressController.Create` | 同上 |
| GET | `/api/v1/user/addresses/:address_id` | SA + Gate | `AddressController.Get` | 同上 |
| PATCH | `/api/v1/user/addresses/:address_id` | SA + Gate | `AddressController.Update` | 同上 |
| POST | `/api/v1/user/addresses/:address_id/default` | SA + Gate | `AddressController.SetDefault` | 同上 |
| DELETE | `/api/v1/user/addresses/:address_id` | SA + Gate | `AddressController.Delete` | 同上 |

### 2.4 Profile 组（6 条）

源码锚点：`router/profile_routes.go`。businessAuth + ProfileController 非 nil + sessionAuth 非 nil 才注册，整组 `SA + Gate`。

| 方法 | 路径 | handler | 备注 |
| --- | --- | --- | --- |
| GET | `/api/v1/profiles/me` | `ProfileController.GetMe` | camelCase 统一格式 |
| PATCH | `/api/v1/profiles/me` | `ProfileController.UpdateMe` | |
| POST | `/api/v1/profiles/me/store` | `ProfileController.SwitchStore` | 切换当前门店 |
| GET | `/api/v1/me/profile` | `ProfileController.GetMeLegacy` | Flutter snake_case 直出 |
| PATCH | `/api/v1/me/profile` | `ProfileController.UpdateMeLegacy` | |
| GET | `/api/v1/me/stores` | `ProfileController.ListMyStores` | 可切换经销商列表 |

### 2.5 Realtime / Push / SSE / CsAgent 组（7 条）

| 方法 | 路径 | 组级中间件 | handler | 注册条件 |
| --- | --- | --- | --- | --- |
| POST | `/api/v1/realtime/ws-ticket` | SA + Gate | `RealtimeController.WSTicket` | businessAuth + controller 非 nil |
| POST | `/api/v1/realtime/sync` | SA + Gate | `RealtimeController.Sync` | 同上 |
| POST | `/api/v1/realtime/push` | SA + Gate | `RealtimeController.Push` | 同上（开发环境推送调试） |
| POST | `/api/v1/push/devices` | SA + Gate | `JPushController.RegisterDevice` | JPushController 非 nil |
| POST | `/api/v1/push/send` | SA + Gate | `JPushController.Send` | 同上 |
| POST | `/api/v1/sse/completions` | SA + Gate | `SseController.Completions` | 额外要求 `cfg.SSE.Enabled` |
| POST | `/api/v1/cs-agent/chat` | SA + Gate | `CsAgentController.Chat` | 额外要求 `cfg.CsAgent.Enabled` |

### 2.6 IM 组 `/api/v1/im`（15 条）

源码锚点：`router/im_routes.go`。整组 `SA + Gate`，ctrl/sessionAuth 任一为 nil 整组不注册。

| 方法 | 路径 | handler | 业务含义 |
| --- | --- | --- | --- |
| POST | `/api/v1/im/session` | `ImController.CreateSession` | 取融云连接 token |
| GET | `/api/v1/im/users/search` | `ImController.SearchUsers` | 搜用户 |
| GET | `/api/v1/im/users/profile` | `ImController.GetUserProfiles` | 批量用户资料 |
| GET | `/api/v1/im/friends` | `ImController.ListFriends` | 好友列表 |
| GET | `/api/v1/im/friends/requests` | `ImController.ListFriendRequests` | 好友申请列表 |
| POST | `/api/v1/im/friends/requests` | `ImController.RequestFriend` | 发起好友申请 |
| POST | `/api/v1/im/friends/requests/:id/respond` | `ImController.RespondFriend` | 同意/拒绝 |
| GET | `/api/v1/im/private/admission` | `ImController.CheckPrivateAdmission` | 私聊准入校验 |
| POST | `/api/v1/im/groups/free` | `ImController.CreateFreeGroup` | 建自由群 |
| POST | `/api/v1/im/groups/free/:id/invite` | `ImController.InviteFreeGroup` | 邀请 |
| POST | `/api/v1/im/groups/free/:id/kick` | `ImController.KickFreeGroup` | 踢人 |
| POST | `/api/v1/im/groups/free/:id/quit` | `ImController.QuitFreeGroup` | 退群 |
| POST | `/api/v1/im/groups/free/:id/dismiss` | `ImController.DismissFreeGroup` | 解散 |
| POST | `/api/v1/im/groups/store/sync` | `ImController.SyncStoreGroup` | 门店群同步 |
| POST | `/api/v1/im/messages/backup` | `ImController.BackupMessages` | 消息业务备份 |

### 2.7 垂直切片域（100 条）

所有切片域形态一致：`Router{handler, sessionAuth, accountGate}` + `NewRouter(svc, sessionAuth, accountGate)` + `Register(engine)`；Register 开头判断 `r == nil || r.handler == nil || r.sessionAuth == nil` 就整体跳过（唯一例外 purchase_calculator 不持有 sessionAuth）。下表「链」列均为 `SA + Gate`（purchase_calculator 为「无」）。路由基线被 `router/domain_routes_test.go` 三个测试逐字固化，切片丢路由会直接挂测试。

**批 1：access / home / points / wallet（25 条）**

| 方法 | 路径 | handler（app/internal/…） |
| --- | --- | --- |
| POST | `/api/v1/stores` | access·handler.CreateStore |
| POST | `/api/v1/stores/:store_id/members` | access·handler.UpsertMember |
| DELETE | `/api/v1/stores/:store_id/members/:user_id` | access·handler.RemoveMember |
| POST | `/api/v1/stores/:store_id/join-applications` | home·handler.ApplyToStore |
| POST | `/api/v1/roles/assignments` | access·handler.AssignRole |
| DELETE | `/api/v1/roles/assignments` | access·handler.RevokeRole |
| GET | `/api/v1/me/permissions` | access·handler.ListMyPermissions |
| GET | `/api/v1/home/todo-cards` | home·handler.ListTodoCards |
| GET | `/api/v1/home/join-applications` | home·handler.ListPendingJoinApplications |
| POST | `/api/v1/home/join-applications/:application_id/approve` | home·handler.ApproveJoinApplication |
| POST | `/api/v1/home/join-applications/:application_id/reject` | home·handler.RejectJoinApplication |
| GET | `/api/v1/home/follow-up-customers` | home·handler.ListOverdueCustomers |
| GET | `/api/v1/home/after-sales-appointments` | home·handler.ListPendingAppointments |
| GET | `/api/v1/home/store-review-orders` | home·handler.ListPendingReviewOrders |
| GET | `/api/v1/points/status` | points·handler.GetStatus |
| POST | `/api/v1/points/check-in` | points·handler.CheckIn |
| GET | `/api/v1/points/tasks` | points·handler.ListTasks |
| POST | `/api/v1/points/tasks/:code/claim` | points·handler.ClaimTask |
| GET | `/api/v1/wallet` | wallet·handler.GetSummary |
| GET | `/api/v1/wallet/ledger` | wallet·handler.ListLedger |
| GET | `/api/v1/wallet/cards` | wallet·handler.ListCards |
| POST | `/api/v1/wallet/cards` | wallet·handler.BindCard |
| POST | `/api/v1/wallet/cards/:card_id/default` | wallet·handler.SetDefaultCard |
| DELETE | `/api/v1/wallet/cards/:card_id` | wallet·handler.DeleteCard |
| POST | `/api/v1/wallet/recharge` | wallet·handler.Recharge |

**批 2：汽车业务 5 域（25 条，其中 23 条 `SA + Gate`，2 条公开）**

| 方法 | 路径 | 链 | handler |
| --- | --- | --- | --- |
| GET | `/api/v1/used-car-orders` | SA+Gate | used_car·handler.List |
| POST | `/api/v1/used-car-orders` | SA+Gate | used_car·handler.Create |
| GET | `/api/v1/used-car-orders/:order_id` | SA+Gate | used_car·handler.Get |
| GET | `/api/v1/used-car-orders/customers` | SA+Gate | used_car·handler.ListCustomers |
| GET | `/api/v1/used-car-orders/summary` | SA+Gate | used_car·handler.Summary |
| GET | `/api/v1/new-car-follow-files` | SA+Gate | new_car_follow·handler.List |
| POST | `/api/v1/new-car-follow-files` | SA+Gate | new_car_follow·handler.Create |
| GET | `/api/v1/new-car-follow-files/:file_id` | SA+Gate | new_car_follow·handler.Get |
| PATCH | `/api/v1/new-car-follow-files/:file_id` | SA+Gate | new_car_follow·handler.Patch |
| GET | `/api/v1/new-car-follow-files/:file_id/logs` | SA+Gate | new_car_follow·handler.ListLogs |
| POST | `/api/v1/new-car-follow-files/:file_id/logs` | SA+Gate | new_car_follow·handler.CreateLog |
| GET | `/api/v1/new-car-follow-files/customers` | SA+Gate | new_car_follow·handler.ListCustomers |
| GET | `/api/v1/new-car-follow-files/summary` | SA+Gate | new_car_follow·handler.Summary |
| GET | `/api/v1/after-sales/pending-appointments` | SA+Gate | after_sales·handler.ListPendingAppointments |
| GET | `/api/v1/after-sales/records` | SA+Gate | after_sales·handler.ListRecords |
| POST | `/api/v1/after-sales/records` | SA+Gate | after_sales·handler.CreateRecord |
| GET | `/api/v1/after-sales/records/:record_id` | SA+Gate | after_sales·handler.GetRecord |
| GET | `/api/v1/deal-invoices` | SA+Gate | deal_invoice·handler.List |
| POST | `/api/v1/deal-invoices` | SA+Gate | deal_invoice·handler.Create |
| GET | `/api/v1/deal-invoices/:invoice_id` | SA+Gate | deal_invoice·handler.Get |
| GET | `/api/v1/deal-invoices/customers` | SA+Gate | deal_invoice·handler.ListCustomers |
| GET | `/api/v1/deal-invoices/summary` | SA+Gate | deal_invoice·handler.Summary |
| PATCH | `/api/v1/deal-invoices/:invoice_id/resubmit` | SA+Gate | deal_invoice·handler.Resubmit |
| GET | `/api/v1/purchase-calculator/products` | 无（公开域） | purchase_calculator·handler.ListProducts |
| POST | `/api/v1/purchase-calculator/quote` | 无（公开域） | purchase_calculator·handler.Quote |

purchase_calculator 是全项目唯一公开域：`router.go#L9-L27` 中 Router 只持有 handler，`NewRouter(svc)` 不收 auth 参数；但 Service 只在 local provider 分支构造，云模式下该域同样不存在。

**批 3：电商 + 社区 6 域（50 条，全部 `SA + Gate`）**

mall（12 条，`app/internal/mall/router.go`，仅 local 模式构造 Service）：

| 方法 | 路径 | handler |
| --- | --- | --- |
| POST | `/api/v1/mall/products` | mall·handler.CreateProduct |
| POST | `/api/v1/mall/products/:product_id/skus` | mall·handler.CreateSKU |
| POST | `/api/v1/mall/skus/:sku_id/codes` | mall·handler.AddVirtualCodes |
| GET | `/api/v1/mall/stores/:store_id/products` | mall·handler.ListProducts |
| GET | `/api/v1/mall/stores/:store_id/products/:product_id` | mall·handler.GetProduct |
| POST | `/api/v1/mall/cart` | mall·handler.UpsertCart |
| GET | `/api/v1/mall/cart` | mall·handler.ListCart |
| GET | `/api/v1/mall/orders` | mall·handler.ListOrders |
| POST | `/api/v1/mall/orders` | mall·handler.CreateOrder |
| GET | `/api/v1/mall/orders/:order_id` | mall·handler.GetOrder |
| POST | `/api/v1/mall/orders/:order_id/pay` | mall·handler.PayOrder |
| POST | `/api/v1/mall/orders/:order_id/cancel` | mall·handler.CancelOrder |

membership（6 条，`app/internal/membership/router.go`，仅 local 模式）：

| 方法 | 路径 | handler | 备注 |
| --- | --- | --- | --- |
| GET | `/api/v1/membership/me` | membership·handler.Me | |
| POST | `/api/v1/membership/buyout` | membership·handler.Buyout | |
| POST | `/api/v1/membership/buyout/confirm` | membership·handler.ConfirmBuyout | |
| POST | `/api/v1/membership/huawei/verify` | membership·handler.VerifyHuawei | |
| POST | `/api/v1/membership/apple/verify` | membership·handler.VerifyApple | |
| POST | `/api/v1/membership/huawei/notifications` | membership·handler.HuaweiNotifications | 华为服务端关键事件回调；注释写明「可不带用户 Session」，但 MVP 阶段仍挂在 SA+Gate 同组便于联调——**这是已知的占位取舍，上线前需要挪到公开组** |

community（12 条，`app/internal/community/router.go`，仅 local 模式）：

| 方法 | 路径 | handler |
| --- | --- | --- |
| GET | `/api/v1/community/topics` | community·handler.ListTopics |
| GET | `/api/v1/community/topics/search` | community·handler.SearchTopics |
| GET | `/api/v1/community/search` | community·handler.Search |
| POST | `/api/v1/community/posts` | community·handler.CreatePost |
| GET | `/api/v1/community/posts` | community·handler.ListPosts |
| DELETE | `/api/v1/community/posts/:id` | community·handler.DeletePost |
| POST | `/api/v1/community/posts/:id/like` | community·handler.LikePost |
| DELETE | `/api/v1/community/posts/:id/like` | community·handler.UnlikePost |
| GET | `/api/v1/community/posts/:id/comments` | community·handler.ListComments |
| POST | `/api/v1/community/posts/:id/comments` | community·handler.AddComment |
| POST | `/api/v1/community/users/:id/follow` | community·handler.FollowUser |
| DELETE | `/api/v1/community/users/:id/follow` | community·handler.UnfollowUser |

short_video（8 条，`app/internal/short_video/router.go`，仅 local 模式）：

| 方法 | 路径 | handler |
| --- | --- | --- |
| POST | `/api/v1/short-videos` | short_video·handler.Create |
| GET | `/api/v1/short-videos` | short_video·handler.List |
| GET | `/api/v1/short-videos/profile` | short_video·handler.Profile |
| GET | `/api/v1/short-videos/:id` | short_video·handler.Get |
| DELETE | `/api/v1/short-videos/:id` | short_video·handler.Delete |
| POST | `/api/v1/short-videos/:id/like` | short_video·handler.Like |
| DELETE | `/api/v1/short-videos/:id/like` | short_video·handler.Unlike |
| POST | `/api/v1/short-videos/:id/view` | short_video·handler.View |

transaction（11 条，`app/internal/transaction/router.go`，两种 provider 都注册）。两组风格并存是**历史兼容要求，注释明确勿合并**：

| 方法 | 路径 | handler | 格式 |
| --- | --- | --- | --- |
| GET | `/api/v1/transactions` | transaction·handler.List | Flutter legacy（无 page 参数走旧格式） |
| POST | `/api/v1/transactions` | transaction·handler.CreateLegacy | legacy |
| GET | `/api/v1/transactions/search` | transaction·handler.Search | ES 不可用时 503；必须注册在 `/:id` 之前 |
| GET | `/api/v1/transactions/:id` | transaction·handler.GetLegacy | legacy |
| PUT | `/api/v1/transactions/:id` | transaction·handler.UpdateLegacy | legacy |
| DELETE | `/api/v1/transactions/:id` | transaction·handler.DeleteLegacy | legacy |
| GET | `/api/v1/transactions/manage` | transaction·handler.ListPage | 统一信封 + page/size 分页 |
| POST | `/api/v1/transactions/manage` | transaction·handler.Create | 统一格式 |
| GET | `/api/v1/transactions/manage/:id` | transaction·handler.Get | 统一格式 |
| PUT | `/api/v1/transactions/manage/:id` | transaction·handler.Update | 统一格式 |
| DELETE | `/api/v1/transactions/manage/:id` | transaction·handler.Delete | 统一格式 |

payment（1 条，`app/internal/payment/router.go`，两种 provider 都注册）：

| 方法 | 路径 | handler |
| --- | --- | --- |
| POST | `/api/v1/payments/prepay` | payment·handler.Prepay |

> 注册顺序坑：Gin 的 httprouter 树要求静态段（`/search`、`/profile`）先于参数段（`/:id`）注册，transaction 与 short_video 都在注释里固化了这一点。

### 2.8 路由总数对账

3（顶层）+ 15（user/address）+ 6（profile/me）+ 3（realtime）+ 2（push）+ 1（sse）+ 1（cs-agent）+ 15（im）+ 100（切片域：25+25+50）= **146 条**。其中公开路由 12 条（health/metrics/ws + user 7 条登录注册 + calculator 2 条），其余 134 条要求 SA 链（且 local 模式额外过 Gate）。条件开关导致进程内实际条数会少于 146（如 SSE/CsAgent 关闭、Supabase 模式下无 mall/membership/community/short_video 等）。

---

## 3. 中间件函数级清单

源码锚点：`app/internal/delivery/http/middleware/`。非测试文件共 9 个（auth/cors/device_session_error/local_session_auth/logger/prometheus/require_active_account/supabase_auth/supabase_session_auth）。

### 3.1 全局四件套

| 函数签名 | 职责 | 备注（调用方/关键逻辑/坑） |
| --- | --- | --- |
| `CORS() gin.HandlerFunc`（`cors.go#L11`） | 写 5 个 CORS 响应头；OPTIONS 预检直接 204 Abort | **坑**：同时设了 `Allow-Origin: *` 和 `Allow-Credentials: true`，按 Fetch 规范带凭证请求通配源会被浏览器拒绝；当前 App 走纯 Bearer 头不带 cookie 才没暴露问题。Allow-Headers 固定白名单（含 Authorization），自定义头 `X-Session-ID/X-Device-ID` **不在**该头内——预检对自定义头理论上需要放行（⚠️ 待确认：Flutter 原生 HTTP 客户端不发 CORS 预检，故暂未踩中；Web 端接入会失败） |
| `PrometheusHTTP() gin.HandlerFunc`（`prometheus.go#L12`） | 计数 + 延迟观测 | `/metrics`、`/health` 直接放行不计数；route 标签取 `c.FullPath()`（模板路径，低基数），未匹配路由记 `unmatched`；status 归并成 `2xx/4xx` 类。指标细节见 [14-可观测性与外部集成](14-可观测性与外部集成.md) |
| `RequestLogger(log *zap.Logger) gin.HandlerFunc`（`logger.go#L12`） | 请求结束后打一条结构化访问日志 | 先 `c.Next()` 再记录，字段：status/method/path/query/ip/latency/user_agent，有 `c.Errors` 时附 errors；≥500 Error 级、≥400 Warn 级、其余 Info 级 |
| `gin.Recovery()`（Gin 内置） | panic 兜底 500 | Setup 第一个 Use，保证 handler panic 不会带垮进程 |

### 3.2 会话鉴权两件套（当前唯一生效的鉴权中间件）

| 函数签名 | 职责 | 备注 |
| --- | --- | --- |
| `LocalSessionAuth(jwtMgr *jwt.Manager, sessionUC *usecase.DeviceSessionUsecase) gin.HandlerFunc`（`local_session_auth.go#L15`） | 本地 provider：Bearer token → `jwtMgr.ParseUUID` → 组装 `pkgauth.SupabaseUser{ID: claims.Subject, Email}` → `sessionUC.Validate(uid,email,X-Session-ID,X-Device-ID)` → 上下文写身份 | 失败统一走 `BackendErrorWithCode`（裸 JSON 信封）：无 token/token 无效 → 401 + 10023；会话失败由 abortDeviceSessionError 映射。sessionUC 为 nil 时跳过设备校验（防御性分支）。写入键与 Supabase 路径完全一致 |
| `SupabaseSessionAuth(cfg config.SupabaseConfig, sessionUC *usecase.DeviceSessionUsecase) gin.HandlerFunc`（`supabase_session_auth.go#L20`） | 云 provider：Bearer token → `pkgauth.ValidateAccessToken`（联网/本地验签 Supabase JWT）→ 同样的 sessionUC.Validate → 写上下文 | 结构与 Local 版逐行对称；header 常量 `HeaderSessionID="X-Session-ID"`、`HeaderDeviceID="X-Device-ID"`（`supabase_session_auth.go#L14-L17`） |
| `abortDeviceSessionError(c, err)`（`device_session_error.go#L14`） | 单设备会话失败的统一拒绝出口（未导出） | `errors.Is(err, ErrSessionReplaced)` → 401 + code 10021 + `MsgSessionReplaced`，打 log `[auth] session replaced`；其他错误 → 401 + 10022 + `MsgSessionInvalid`；最后 `c.Abort()`。客户端按 code 分支、勿依赖文案 |

两中间件成功后都写入：`ContextSupabaseUserKey`（"supabaseUser"）→ `pkgauth.SupabaseUser`；`ContextAccessTokenKey`（"accessToken"）→ 原始 token 字符串（转发 PostgREST/SSE 时需要原 token）。完整链路见 [05-认证与会话](05-认证与会话.md)。

### 3.3 账号状态门

源码锚点：`require_active_account.go`

| 声明 | 职责 | 备注 |
| --- | --- | --- |
| `type AccountChecker interface { AllowsRequest(ctx, userID string) error }` | 账号可用性判定端口 | 实现者是 `*usecase.LocalAuthUsecase`（main.go 装配） |
| `RequireActiveAccount(check AccountChecker) gin.HandlerFunc`（L18） | SA 之后第二道门：取会话用户，空则 401「未登录」；`AllowsRequest` 报错则 403「账号不可用」 | check 为 nil 直接 Next（Supabase 模式即如此，链缩短为只有 SA）；注释明确「锁定中的已有会话仍放行」——即只拦注销/停用，不拦锁定。错误响应用 `response.BackendError`（裸 JSON） |

### 3.4 遗留 JWT 中间件（当前未接线）

源码锚点：`auth.go`

| 声明 | 职责 | 当前状态 |
| --- | --- | --- |
| `Auth(jwtManager *jwt.Manager) gin.HandlerFunc`（L20） | 老的自建 uint-ID JWT 鉴权：Authorization 头 → Bearer 切分 → `jwtManager.Parse`（非 UUID 版 claims）→ 写 `userID(uint)`/`username` | **全仓库已无调用方**（grep `middleware.Auth(` 0 命中），路由表中没有任何路由挂它。错误用统一信封 `response.Error`（与 SA 链的裸 JSON 不同），属两套体系并存的遗留物 |
| `GetUserID(c) (uint, bool)`（L50） | 读 `ContextUserIDKey` 并断言 uint | 同样仅文件内自引用，无外部调用方 |
| 常量 `ContextUserIDKey="userID"`、`ContextUsernameKey="username"` | 老上下文键 | 与新键 `"supabaseUser"` 不通用，迁移代码时别拿错 |

### 3.5 身份读写：authctx 叶子包与 middleware 转发层

源码锚点：`app/internal/delivery/http/authctx/authctx.go`

为什么单独成包（包注释原话总结）：middleware 依赖 usecase，而切片域 handler 要取身份；域若直接 import middleware 会形成 `usecase → 域 → middleware → usecase` 导入环。authctx 只依赖 gin + common/auth，两边都能安全依赖。

| 函数/常量 | 签名 | 职责 |
| --- | --- | --- |
| `ContextSupabaseUserKey` / `ContextAccessTokenKey` | `= "supabaseUser"` / `"accessToken"` | 两种 provider 共用的身份键 |
| `SetIdentity` | `(c, user pkgauth.SupabaseUser, token string)` | 写身份（供 SA 中间件用；当前两个 SA 实现是直接 c.Set 同名字段，等价） |
| `GetSupabaseUser` | `(c) (pkgauth.SupabaseUser, bool)` | 取会话用户 |
| `GetAccessToken` | `(c) (string, bool)` | 取原始 access token |
| `BearerToken` | `(header string) string` | 剥 `"Bearer "` 前缀；**无前缀时原样返回**（trim 空白），比 auth.go 里严格切分的老逻辑宽松 |
| `SupabaseAuthContext` | `(c) (pkgauth.SupabaseUser, string, bool)` | 域 handler 统一取身份入口：用户或 token 缺任一即 false |

`middleware/supabase_auth.go` 是转发层（不是中间件！）：`GetSupabaseUser/GetAccessToken/SupabaseAuthContext/bearerToken` 全部转调 authctx 同名函数，并重新导出两个上下文常量，仅供未切片的 legacy controller 使用，包注释写明切片全部完成后可删。

---

## 4. 统一响应与错误封装

### 4.1 两套并存的响应风格

源码锚点：`app/internal/delivery/http/dto/response/`

**统一信封**（`common.go`）——管理端风格与新域接口使用：

```go
type Response struct {
    Code      int         `json:"code"`      // 0 成功；非 0 见错误码表
    Message   string      `json:"message"`
    Data      interface{} `json:"data"`     // 无数据时为 {} 而非 null
    Timestamp int64       `json:"timestamp"` // time.Now().Unix() 秒
}
```

**Flutter 兼容裸 JSON**（`backend.go`）——成功直接 `c.JSON(payload)`，错误才包一层：

```go
type BackendErrorBody struct {
    Error string `json:"error"`
    Code  int    `json:"code,omitempty"`
}
```

会话中间件、IM/Profile 等 legacy controller 走裸 JSON 风格；切片域 handler 与老 `middleware.Auth` 走统一信封。接到前端反馈「为什么错误结构不一样」时先看该路由属于哪个体系。

### 4.2 响应辅助函数表

| 函数 | 签名/行为 | HTTP 状态 | 使用方 |
| --- | --- | --- | --- |
| `Success` | `(c, data)`，data 为 nil 填 `{}`，message `"success"` | 200 | 统一信封各 handler |
| `SuccessWithMessage` | 同上但自定义 message | 200 | |
| `SuccessCreated` | message 固定「创建成功」 | 201 | |
| `SuccessList[T]` | data 为 `{list, pagination}`；nil 切片转 `[]` | 200 | 所有分页列表 |
| `Error` | `(c, httpStatus, code, message)`，data 固定 `{}` | 自定义 | 统一信封错误出口 |
| `BackendJSON` | `(c, status, payload)` 裸 JSON | 自定义 | Flutter 成功响应 |
| `BackendError` | `(c, status, message)`，业务码由 `defaultBackendCode` 按状态推导 | 自定义 | RequireActiveAccount 等 |
| `BackendErrorWithCode` | `(c, status, code, message)` 显式业务码 | 自定义 | 两个 SA 中间件、设备会话错误 |
| `BackendNoContent` | 仅写 204 | 204 | |
| `FormatTimePtr` | `*time.Time` → UTC RFC3339 字符串指针 | — | Supabase 时间格式兼容 |

`defaultBackendCode`（`backend.go#L33-L49`）映射：401→10023、403→10003、404→10004、400→10001、≥500→50000、其他→10001。

### 4.3 业务错误码表（`common.go#L12-L26`）

| 常量 | 值 | 含义 | 客户端处理约定 |
| --- | --- | --- | --- |
| CodeSuccess | 0 | 成功 | |
| CodeInvalidParams | 10001 | 参数错误 | 提示文案 |
| CodeUnauthorized | 10002 | 登录凭证错误（如密码错误） | 注释强调：勿与 token 失效混用 |
| CodeForbidden | 10003 | 无权限 | |
| CodeNotFound | 10004 | 资源不存在 | |
| CodeSessionReplaced | 10021 | 单设备：会话被新设备顶替 | 弹「已在其他设备登录」并退登录 |
| CodeSessionInvalid | 10022 | 会话无效/缺失 | 重新登录 |
| CodeTokenInvalid | 10023 | access JWT 无效/未带头 | 先 tryRefresh，失败再清会话 |
| CodeInvalidOTP | 10024 | 短信/邮箱验证码错误或过期 | |
| CodeInternalError | 50000 | 内部错误 | 兜底提示 |

### 4.4 分页三件套

| 声明 | 职责 | 备注 |
| --- | --- | --- |
| `Pagination{Page,Size,Total,TotalPages}` | 列表分页元数据 JSON | |
| `ListData[T]{List, Pagination}` | 统一列表 data 形状（泛型） | |
| `ParsePageQuery(c, defaultSize) PageQuery` | 从 `?page=&size=` 解析 | page 从 1 开始；非法/非正数回落默认；**size 上限 100** |
| `(PageQuery).Offset()` | `(page-1)*size` | DB 层用 |
| `NewPagination(page,size,total)` | 算总页数（ceil），total=0 时 TotalPages=0 | |

### 4.5 跨域共享件（app/internal/shared）

| 函数/类型 | 源码 | 职责 | 备注 |
| --- | --- | --- | --- |
| `SessionAuthChain(auth, accountGate)` | `shared/http.go#L10` | 生成 `[SA]` 或 `[SA,Gate]` 链 | 与 router 包内私有 sessionAuthChain 同构（两份重复实现，迁移期产物） |
| `Registrable` 接口 | `shared/http.go#L22` | `Register(engine *gin.Engine)` | Setup 遍历 DomainRouters 的统一端口 |
| `WriteResourceCRUDError(c, err, invalid, notFound, forbidden)` | `shared/resource_error.go#L17` | 资源型 CRUD 同构错误映射 | invalid→400/10001 + err.Error()；notFound→404/10004「资源不存在」；forbidden→403/10003「无权限」；兜底→500/50000「操作失败」。文案要求与切片前逐字一致，community/short_video 共用 |
| `ErrSearchUnavailable` | `shared/errors.go#L13` | ES 未配置/不可用哨兵 | transaction 域 Search 回 503；cs-agent 把它当良性分支不向 SSE 抛错；域内保留别名 `ErrTransactionSearchUnavailable` |
| `IsUniqueViolation(err)` | `shared/dberr.go#L10` | 判唯一约束冲突 | 靠错误串小写包含 `duplicate key`/`unique constraint`，access/home/points/after_sales 仓储共用；基于字符串匹配是已知脆弱点（换驱动/语言环境需回归） |

### 4.6 DTO 目录函数/类型级清单

**请求 DTO**（`dto/request/`，binding 标签即 gin 校验规则，失败由 handler 翻 400）：

| 文件 | 类型 | 字段与校验要点 |
| --- | --- | --- |
| user_request.go | `LoginRequest` | username/password 必填；device_id 必填；platform `oneof=android ios` |
| | `RegisterRequest` | username 3-64；password 6-64；email 格式且 ≤128；device_id/platform 同上 |
| | `SendPhoneOTPRequest` | phone 必填 |
| | `VerifyPhoneOTPRequest` | phone/otp/device_id 必填；platform 枚举 |
| | `WeChatLoginRequest` | code（微信授权码）+ device_id + platform |
| | `HuaweiLoginRequest` | code（华为一键登录 authorization code）+ device_id + platform |
| | `RefreshTokenRequest` | refresh_token 必填；device_id/session_id 选填；platform `omitempty` 枚举 |
| transaction_request.go | `CreateTransactionRequest` | type/category/amount/date 必填；note 可空 |
| | `UpdateTransactionRequest` | 全字段指针（部分更新） |
| realtime_request.go | `RealtimeTicketRequest` | platform、connId（camelCase JSON） |
| | `RealtimeSyncRequest` | sinceSeq int64、topics []string |
| | `RealtimePushRequest` | userId/topic/title/body/name + extra map（调试推送） |
| sse_completion.go | `CompletionRequest` / `CompletionOptions` | prompt、conversationId 指针（nil=首句）、clientRequestId、model/temperature/maxTokens；方法 `ConversationIDValue()` 空指针安全取值 |
| cs_agent_chat.go | `CsAgentChatRequest` | prompt、clientRequestId |

**响应 DTO**（`dto/response/`）：

| 文件 | 类型/函数 | 职责 |
| --- | --- | --- |
| user.go | `UserItem` / `UserProfile` | 老 uint-ID 用户体系（id uint，camelCase） |
| | `AuthUserItem` | 认证接口用户体：同时给 `id` 与 `user_id`（同值）、user_name/username 双名、status int16、phone/avatar_url omitempty |
| | `LoginData` | token + refresh_token + session_id + user |
| | `RefreshTokenData` | token + refresh_token + session_id（omitempty） |
| | `FromSupabaseAuthUser` / `AuthUserFromOutput` | 两个组装工厂，后者补 status/phone/avatar |
| profile.go | `ProfileItem`（camelCase）/`ProfileLegacyBody`（snake_case） | 同一资料两种 JSON 风格；id 优先取 `Profile.UserID` 回退 `ID`（`accountID`），name 取 UserName→DisplayName（`accountName`） |
| | `ProfileStatsItem` | 门店统计：注意 JSON 键是稳定 snake_case（store_id/days_joined…），与外层 camelCase 混用 |
| | `UserStoreListBody`/`UserStoreListItem` | GET /me/stores：list + current_store_id，含 is_current |
| | `FromProfile/FromProfileLegacy/FromUserStoreStats/FromUserStoreList` | entity → DTO 转换函数；role 数字枚举 + role_label 文案，空 label 时用 `entity.StoreRoleLabel` 补 |
| realtime.go | `RealtimeTicketData` | ticket/wsUrl/expiresInSeconds/connId，对齐 Flutter WsTicketResult |
| | `RealtimeSyncData` | events + latestSeq |
| | `RealtimePushData` | envelope/delivered/queued/taskId |
| transaction.go | `TransactionItem` + `FromTransaction`/`FromTransactions` | camelCase 收支记录项；nil 实体给零值结构体 |

---

## 5. 请求过闸全流程（以受保护接口为例）

以 `GET /api/v1/wallet`（local provider）为例的编号时序：

1. TCP/HTTP 进入 Gin，`Recovery` 就位（panic 兜 500）。
2. `CORS` 写跨域头；OPTIONS 在此被 204 短路。
3. `PrometheusHTTP` 记开始时间，`/metrics`、`/health` 在此被跳过统计。
4. `RequestLogger` 记开始时间。
5. 路由树匹配到 wallet 组，执行组链 `LocalSessionAuth`：剥 Bearer → `ParseUUID`（失败 401/10023 裸 JSON 并 Abort）→ 读 `X-Session-ID/X-Device-ID` → `DeviceSessionUsecase.Validate` 查 Redis（被顶替 401/10021；无效 401/10022）。
6. `RequireActiveAccount`：`LocalAuthUsecase.AllowsRequest` 查账号状态（注销/停用 → 403）。
7. `wallet.handler.GetSummary`：经 `authctx.SupabaseAuthContext` 取 user_id（不再读客户端参数）→ `Service` → `PostgresRepository` → `wys_` 业务表。
8. handler 用 `response.Success`（统一信封）写回 200；或经 `WriteResourceCRUDError` 映射业务错误。
9. 出栈：RequestLogger 按状态选日志级别；PrometheusHTTP 用 `FullPath()`（`/api/v1/wallet`，不是实际路径）打点。

切片域典型请求链路：`router(Register) → handler(HTTP 适配) → service(业务) → repository(接口) → repo_postgres(GORM/DB)`；legacy 域则是 `router → controller → usecase → repository/{postgres,redis,supabase}`。

---

## 6. 学习要点回顾

1. **装配分两代**：legacy 集中式 `registerXxxRoutes`（router 包 9 个文件）与垂直切片 `DomainRouters.Register`（15 域）并存，靠 `sessionAuth == nil 跳过注册` 保持行为等价；`domain_routes_test.go` 用三批共 100 条基线防切片丢路由。
2. **中间件顺序即安全边界**：Recovery→CORS→指标→日志是全局层；`SA → Gate` 是组级层；身份只信中间件写入的上下文，handler 不接受客户端自报 user_id。
3. **两套认证中间件同构**：Local 与 Supabase 版本逐行对称，区别只在 token 解析（本地 `ParseUUID` vs `pkgauth.ValidateAccessToken`），之后共用同一个 Redis 单设备校验与同一组上下文键。
4. **两套响应信封并存**：统一信封 code=0 与 Flutter 裸 JSON/`{error,code}`，错误码体系对齐（10021/10022/10023 三个会话码最容易混，务必背下）。
5. **导入环倒逼出 authctx 叶子包**：这是 Go 分层里非常实用的一招——把「上下文身份读写」下沉到谁都不依赖的小包，middleware 和域都能 import。
6. **条件注册是功能开关**：provider 类型、controller 是否 nil、`SSE/CsAgent.Enabled`、WSHandler 是否构造，共同决定进程内实际路由数；排障「404 但路由明明写了」先查装配条件。
7. **低基数指标意识**：PrometheusHTTP 用 `c.FullPath()` 模板而非真实 URL，避免 user_id/order_id 把 label 打爆；未匹配统一记 `unmatched`。
8. **已知占位/坑清单**：`middleware.Auth` 老 JWT 中间件已无调用方；membership 华为回调暂挂鉴权组；CORS `*` + credentials 组合对 Web 不合法；`IsUniqueViolation` 靠错误串匹配；`shared.SessionAuthChain` 与 router 私有 chain 重复。

## 7. 可动手验证

1. 起服务后导出真实路由树：在 `app/` 下执行 `go run ./cmd/api`（或 `make run`），另开终端 `curl -s localhost:8080/health` 应得 `{"status":"ok"}`；`curl -s localhost:8080/metrics | grep http_requests_total`。
2. 读代码对账：`go test ./app/internal/delivery/http/router/ -run 'TestBatch' -v`，观察三批基线测试；自己从 15 个域 router.go 数一遍路由是不是 25+25+50。
3. 不带 token 打受保护接口：`curl -i -X POST localhost:8080/api/v1/push/devices`，核对返回是裸 JSON `{"error":...,"code":10023}` 而非统一信封；再带一个假 Bearer 对比文案变化但 code 不变。
4. 在 `router.Setup` 里临时打印 `len(r.Routes())`（或写个测试遍历 `engine.Routes()`），对照本文 146 条总数与当前配置下的差异，解释少了哪些组。
5. 用 `curl -i -X OPTIONS localhost:8080/api/v1/user/login -H 'Origin: http://x'` 观察 CORS 预检 204 与 5 个响应头。
