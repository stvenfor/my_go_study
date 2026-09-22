## 1. Go — 买家订单列表 API

- [x] 1.1 Repository：按 `buyer_user_id` + 可选 `status` 分页查询订单，并批量加载对应 `order_item` 快照
- [x] 1.2 Usecase：`ListOrders`（会话 user id、校验 status、组装列表行）；补一条用例测试（仅自己的单、status 过滤）
- [x] 1.3 Controller + Router：`GET /api/v1/mall/orders`（注册在 `:order_id` 之前），`SuccessList` 响应；非法 status → 400
- [x] 1.4 `docs/local-auth-postgres.md` 商城段补列表 API 一行说明

## 2. Flutter — 路由与入口（`my_ai_project`）

- [x] 2.1 `RoutePath` + `mall_module` 注册订单列表页、订单详情页
- [x] 2.2 Mine「我的订单」快捷入口：登录校验后跳转列表（去掉「开发中」toast）

## 3. Flutter — 订单数据与页面

- [x] 3.1 `MallApi`：列表（page/size/status）、详情、支付、取消；模型对齐 snake_case
- [x] 3.2 订单列表页：状态 Tab（全部 / 待支付 / 已支付含履约 / 已取消含关闭）、下拉刷新、空态、卡片进详情
- [x] 3.3 订单详情页：状态、行商品、金额、收货信息、虚拟履约信息；待支付底栏「取消 / 去支付」并刷新

## 4. 联调验证

- [x] 4.1 local 下：下单 → 列表可见 → 详情一致 → 待支付取消/支付后状态与列表刷新正确；另一账号不可见
