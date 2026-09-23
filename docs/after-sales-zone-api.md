# 售后专区 — 维修保养记录 API

> 状态：Go 已实现；Flutter 专区页同切片  
> 术语：`CONTEXT.md`「售后专区」≠ 首页「售后预约」

## 已决

| # | 决策 |
|---|------|
| 1 | 表 `wys_after_sales_record`；可选 `appointment_id`（唯一） |
| 2 | 写：当前店**成员**；读：成员看本店，客户看 `customer_user_id=自己` |
| 3 | 带 `appointment_id` 建档事务内将预约标 done |
| 4 | 本切片无编辑/删除 |

鉴权：SessionAuth + `auth.provider=local`。响应 `{ code, message, data, timestamp }`。

## 路由

前缀 `/api/v1/after-sales`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/records` | 列表；`page`/`size`；data 含 `list`/`pagination`/`can_create` |
| GET | `/records/:record_id` | 详情 |
| POST | `/records` | 成员新建 |
| GET | `/pending-appointments` | 成员：待处理且预约日 ≥ 今天 |

### POST body

| 字段 | 必填 | 说明 |
|------|------|------|
| `service_kind` | 是 | `0`/`repair` 或 `1`/`maintenance` |
| `title` | 是 | |
| `service_date` | 是 | `YYYY-MM-DD` |
| `customer_id` 或 `customer_name`+`customer_phone` | 是 | 可从预约/门店客户补全 |
| `appointment_id` | 否 | 同店 pending |
| `plate_no` / `mileage` / `content` / `customer_user_id` | 否 | |

## 错误

| 条件 | HTTP |
|------|------|
| 无当前店（写） | 400 |
| 非成员（写 / pending） | 403 |
| kind/客户/日期/预约无效 | 400 |
| 记录不可见 | 404 |
| 预约已有记录 | 409 |
