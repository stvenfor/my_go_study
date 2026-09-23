# 新车跟进档案 — 接口说明（C1）

> 状态：**Go 已实现**；Flutter 壳待人证  
> 术语：[CONTEXT.md](../CONTEXT.md)「新车跟进档案」  
> Brief：`plans/slices/new-car-follow-c1-file-crud.md`

## 已决

| # | 决策 |
|---|------|
| 1 | 表 `wys_new_car_follow_file`；客户复用 `wys_store_customer` |
| 2 | `follow_level` ∈ {A,B,E,H}；读模型派生 `intent_band`：H/A→高，B→中，E→低 |
| 3 | 范围：`current_store` + **仅本人** `owner_user_id` |
| 4 | 写 `next_follow_up_at` 时事务回写客户表（驱动首页待跟进） |
| 5 | 本切片无流水 `/logs` |

鉴权：SessionAuth。无当前店 → 400。响应壳 `{ code, message, data, timestamp }`，snake_case。分页 `page`+`size`。

## 四格 summary

| 字段 | 含义 |
|------|------|
| `active` | stage ∈ 新建…已报价 |
| `overdue` | 未关闭且 `next_follow_up_at <= now` |
| `high_intent` | 未关闭且级别 H∪A |
| `lost` | stage=战败 |

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

## 错误

| 条件 | HTTP |
|------|------|
| 级别非法 | 400 跟进级别无效 |
| 无当前店 | 400 |
| 他人档案 | 404 |
| 同客户未关闭重复建档 | 409 |
