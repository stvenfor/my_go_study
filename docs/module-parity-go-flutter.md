# Go BFF ↔ Flutter 模块对照表

> Program：`plans/2026-09-23-dual-simplify-program.md`  
> 报告索引：[`simplify-reports/README.md`](./simplify-reports/README.md)  
> 状态枚举：`Paired` | `Deferred` | `Gap`

| Round | 能力 | Status | Go | Flutter |
|-------|------|--------|----|---------|
| R01 | Auth / Session | Paired | `internal/delivery/http/handler/user_*`, `usecase/*auth*`, `device_session*`, `middleware/*session*` | `features/auth`, `commons/network` 鉴权头 |
| R02 | Profile + 门店/统计 | Paired | `controller/profile_*`, store stats | `features/auth` profile + `features/settings` Mine |
| R03 | Transactions | Paired | `transaction_*` | `features/home/.../transaction_api.dart` |
| R04 | Realtime + WS | Paired | `realtime_*`, `delivery/ws` | `components/realtime` |
| R05 | Analytics gRPC | Paired | `delivery/grpc/analytics_*`, `api/proto/analytics` | `commons/network` AnalyticsGrpcApi + home analytics |
| R06 | Used-car orders | Paired | `used_car_order_*` | `features/home/.../used_car_order_*` |
| R07 | Home todo | Paired | `home_todo_*` | `features/home/.../home_todo_*` |
| R08 | New-car follow | Paired | `new_car_follow_*` | `features/home/.../new_car_follow` |
| R09 | After-sales | Paired | `after_sales_zone_*` | `features/home/.../after_sales` |
| R10 | Deal invoice | Paired | `deal_invoice_*` | `features/settings/.../deal_invoice` |
| R11 | Address | Paired | `address_*` | `features/settings/.../address` |
| R12 | Community | Paired | `community_*` | `features/community` |
| R13 | Short video | Paired | `short_video_*` | `features/video` |
| R14 | Mall | Paired | `mall_*` | `features/mall` |
| R15 | Points | Paired | `points_*` | `features/home/.../points_*` |
| R16 | Wallet | Paired | `cash_wallet_*` | `features/wallet` |
| R17 | Membership + prepay | Paired | `membership_*`, `payment_*` | `features/pay` |
| R18 | Purchase calculator | Paired | `purchase_calculator_*` | `features/settings/.../purchase_calculator` |
| R19 | JPush / devices | Paired | `jpush_*` | `components/linking` push + `wys_push` |
| R20 | SSE / AI | Paired | `sse_*`, `completion_*` | `features/ai` |
| R21 | HTTP/DI/router 共享层 | Paired | `middleware/*`, `cmd/api`, `router/` | `AppHttpBootstrap`, `module_manifest` |
| R22 | Queue / Worker | Paired | `pkg/queue`, `cmd/worker` | N/A（仅 Go） |

## Deferred / Gap（本 Program 不补功能）

| 项 | Status | 说明 |
|----|--------|------|
| Flutter `bfui`/`classroom`/`music`/`friend` | Deferred | 无 Go 产品面 |
| Flutter `chat` + RongCloud | Deferred | 非 BFF |
| Flutter `legacy/wanandroid` | Deferred | 第三方 |
| Go `access_*` stores/roles | Gap | 弱/无 Flutter 客户端 |
| Go legacy JWT `user/list` | Deferred | Flutter 勿用 |
| 支付 notify webhook | Deferred | 服务端回调 |
| Huawei 账号登录 | Gap | Go 有 API，Flutter 无入口 |
