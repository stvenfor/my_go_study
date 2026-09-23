## Why

「我的」里的「售后专区」目前只有入口文案（副标题：售后维修保养记录），点击仍是「开发中」；首页已有「售后预约」待办（`wys_after_sales_appointment`），但专区与预约未打通，店员无法落维修保养记录，客户也无法只读回看。需要把专区做成可联调的列表/详情/新建闭环，并与预约域对齐。

## What Changes

- 新增维修保养记录域（`wys_` 表 + Go BFF CRUD 最小集：列表、详情、新建；本切片不做编辑/删除）
- 与既有 `wys_after_sales_appointment` **打通**：记录可关联预约；从预约建档时可把预约标为已处理（status=done），避免首页待办与专区数据割裂
- 鉴权：SessionAuth + `users.current_store_id`；**当前店店员/店管可写**；**客户（非本店成员）仅可读与自己相关的记录**；无当前店时写路径拒绝
- Flutter：接通「我的」→「售后专区」；列表 / 详情 / 新建页；店员侧可从待处理预约发起建档；客户侧只读列表与详情（无新建入口）
- 补充短 API 文档与术语（与首页「售后预约」明确区分）

## Capabilities

### New Capabilities

- `after-sales-zone`: 售后专区维修保养记录（列表/详情/新建）、与售后预约关联打通、按角色读写隔离（店员写 / 客户读）

### Modified Capabilities

- （无。主库 `openspec/specs/` 尚未归档首页待办能力；预约表行为以本变更内「打通」需求描述，不单独改写已归档 spec）

## Impact

- **Go BFF**：新 entity / repository / usecase / controller / router；可选扩展 home-todo 预约写路径（完成预约）；`cmd/api/main.go` DI；migration
- **表**：新建 `wys_after_sales_record`（或等价命名）；预约表可增可选外键/关联字段，或仅由记录侧持有 `appointment_id`
- **Flutter**（兄弟仓库 `my_ai_project`）：`MineFunctionData` 的 `after_sales` 从 toast 改为路由；新增专区 feature/页面与 API client；`RoutePath` 增列表/详情/新建
- **鉴权**：与门店业务一致——`auth.provider=local` + SessionAuth；范围按 `current_store_id` + 成员身份判定写权限
- **非目标**：商城退换货售后、预约完整 CRUD UI、记录编辑/删除、跨店聚合、支付/工单排程系统
