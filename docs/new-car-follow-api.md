# 新车跟进档案 — 接口说明

> 状态：C1+C2+C4 **Go 已实现**；Flutter UI/流水/店管展示已挂（人证）  
> 术语：[CONTEXT.md](../CONTEXT.md)「新车跟进档案」  
> Briefs：`ui-parity` · `c2-follow-log` · `c4-store-admin-scope`

## 已决

| # | 决策 |
|----|------|
| 1 | 表 `wys_new_car_follow_file`；客户复用 `wys_store_customer` |
| 2 | `follow_level` ∈ {A,B,E,H}；读模型派生 `intent_band`：H/A→高，B→中，E→低 |
| 3 | 销售：`current_store` + 本人 `owner_user_id`；**店管**（`role.assign_store`）：当前店全员 |
| 4 | 写 `next_follow_up_at` 时事务回写客户表（驱动首页待跟进） |
| 5 | 流水表 `wys_new_car_follow_log`；`GET/POST .../:file_id/logs` |
| 6 | 附件：**假上传**（客户端本地/占位），本模块无 OSS |

鉴权：SessionAuth。无当前店 → 400。响应壳 `{ code, message, data, timestamp }`，snake_case。分页 `page`+`size`。

## 四格 summary

| 字段 | 含义 |
|------|------|
| `active` | stage ∈ 新建…已报价 |
| `overdue` | 未关闭且 `next_follow_up_at <= now` |
| `high_intent` | 未关闭且级别 H∪A |
| `lost` | stage=战败 |

店管四格按**全店**计；销售按本人。

## 路由

前缀 `/api/v1/new-car-follow-files`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/summary` | 顶栏 |
| GET | `/customers` | 选客户 |
| GET | `` | 列表：`follow_level` / `intent_band` / `stage` / `overdue=1` |
| POST | `` | 建档：`follow_level` 必填；`customer_id` 或 `display_name`+`phone` |
| GET | `/:file_id` | 详情 |
| PATCH | `/:file_id` | 改级别/阶段/下次跟进等 |
| GET | `/:file_id/logs` | 流水，按 `created_at` 倒序 |
| POST | `/:file_id/logs` | 写流水：`body` 必填；可选 `follow_level`、`next_follow_up_at` |

列表/详情额外字段：`owner_user_id`、`owner_display_name`。

### POST logs body

| 字段 | 说明 |
|------|------|
| `body` | 必填正文 |
| `follow_level` | 可选；写入流水并更新档案级别 |
| `next_follow_up_at` | 可选（含 `null`）；出现则回写档案+客户 |

写成功会更新档案 `last_follow_at`；若档案仍为「新建」则升为「跟进中」。

## 错误

| 条件 | HTTP |
|------|------|
| 级别非法 | 400 跟进级别无效 |
| 流水正文空 | 400 跟进内容不能为空 |
| 无当前店 | 400 |
| 不可见档案 | 404 |
| 同客户未关闭重复建档 | 409 |
