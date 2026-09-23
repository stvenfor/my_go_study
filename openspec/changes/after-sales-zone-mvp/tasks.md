## 1. Go — schema & domain

- [x] 1.1 Migration：`wys_after_sales_record`（含 `appointment_id` 可空唯一、`customer_id`/`customer_user_id`、`service_kind`、索引）+ down
- [x] 1.2 Entity + repository 接口/实现：按店分页列表、按 `customer_user_id` 分页列表、按 id 取、事务创建并可选将预约标 done

## 2. Go — usecase / HTTP

- [x] 2.1 Usecase：成员写/店内读、客户只读分支、创建校验（kind、客户身份、appointment 同店 pending）；usecase 测试锁住「带 appointment 建档 → done」与「客户不可建」
- [x] 2.2 Controller + Router：`GET/POST /api/v1/after-sales/records`、`GET .../records/:id`、`GET .../pending-appointments`；SessionAuth；DI 进 `main.go`
- [x] 2.3 文档：`docs/after-sales-zone-api.md`；`CONTEXT.md` 补「售后专区」术语（与售后预约区分）

## 3. Flutter — 入口与 API（`my_ai_project`）

- [x] 3.1 `RoutePath` + module 注册列表 / 详情 / 新建；Mine `after_sales` 登录后跳转（去掉「开发中」toast）
- [x] 3.2 API client + 模型：records 列表/详情/新建、pending-appointments；成员 vs 客户由服务端分支，客户端按「能否建档」隐藏 FAB（成员判定可用当前店成员信息或 create 403 回退）

## 4. Flutter — 页面

- [x] 4.1 列表页：记录列表 + 成员侧待处理预约入口；空态/下拉刷新；点进详情
- [x] 4.2 详情页：展示 kind/标题/日期/客户/车牌/内容/关联预约
- [x] 4.3 新建页：表单必填项；从预约进入时预填并带 `appointment_id`；`customer_user_id` 无现成选人组件则本切片不暴露 UI（仍可由 API/种子写入）

## 5. 联调验证

- [x] 5.1 店员：pending 预约 → 专区新建记录 → 预约 done、首页待办 count 下降、列表/详情可见（usecase：`TestAfterSalesCreateWithAppointmentMarksDone` + 店隔离列表）
- [x] 5.2 客户：仅 `customer_user_id=自己` 的记录可读；无新建入口；不可读他人/不可建档（usecase：`TestAfterSalesCustomerCannotCreate` / `TestAfterSalesCustomerListOwnOnly`；Flutter `can_create` 隐藏新建）
