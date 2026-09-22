# 新车成交发票 — 接口与接入方案

> 状态：**已实现（Go BFF + Flutter 对接）**  
> 术语：[CONTEXT.md](../CONTEXT.md)「新车成交」节  
> ADR：[0011-new-car-deal-invoice.md](./adr/0011-new-car-deal-invoice.md)  
> 对齐客户端：`my_ai_project/features/settings/lib/deal_invoice/`（本切片不迁包）

## 0. 已决摘要

| # | 决策 |
|---|------|
| 1 | 读：列表 + 摘要 + 详情；写：提交 / 同单重提；**不做**审核 API/UI |
| 2 | 购车客户 = 扩展后的 `wys_store_customer`（加必填 `phone`） |
| 3 | 发票图：客户端相册/拍摄本地预览；服务端 `image_url` 可空，不接对象存储 |
| 4 | 顶栏：一次 `GET .../summary`（人设 + 四格统计） |
| 5 | 范围：`current_store` + **仅本人上传** |
| 6 | 审核态对齐 Flutter 四态；重提 = 同 id 改回待审 |
| 7 | `rating_stars` 只读展示（种子）；不做评分写接口 |
| 8 | 首页文案「新车关注」→「新车成交」；路由仍 `dealInvoiceDemo` |
| 9 | 客户选择器：当前店全量分页，非仅逾期跟进 |

---

## 1. 目标与边界

| 做 | 不做（本切片） |
|----|----------------|
| 首页入口改名并进入现有成交 UI | 迁 `deal_invoice` 到 `module_home` |
| 列表 Tab 筛选 + 下拉刷新 + 上拉更多 | 门店全员可见他人发票 |
| 顶栏人设 + 四格统计接真数据 | 真实 CDN / OSS 上传 |
| 选客户（当前店客户列表） | 新建独立客户表 |
| 选图仅本地展示后提交元数据 | OCR、发票验真 |
| 提交 → 待审核；驳回后同单重提 | 通过/驳回/评分写接口 |

鉴权：全部 **SessionAuth**。无当前店 → 业务接口 400（与 Mine/待办一致）。

统一响应壳：`{ code, message, data, timestamp }`。JSON **snake_case**。分页：`page` + `size`（默认 10，最大 50）。

---

## 2. 状态与统计

### 2.1 审核态（库 `smallint`）

| 值 | 码名 | UI |
|----|------|-----|
| 0 | `pending_review` | 待审核 |
| 1 | `approved_pending_rating` | 已通过（待评） |
| 2 | `rated` | 已通过（已评，可带星） |
| 3 | `rejected` | 未通过 |

Tab 映射：

| Tab | 条件 |
|-----|------|
| 全部发票 | 全部 |
| 待审核 | status = 0 |
| 已通过 | status ∈ {1, 2} |
| 未通过 | status = 3 |

### 2.2 四格统计（本人 × 当前店）

| 字段 | 含义 |
|------|------|
| `uploaded` | 全部条数 |
| `pending_review` | status = 0 |
| `approved` | status ∈ {1, 2} |
| `rejected` | status = 3 |

---

## 3. 表设计

### 3.1 扩展 `wys_store_customer`

```sql
ALTER TABLE wys_store_customer
  ADD COLUMN IF NOT EXISTS phone varchar(32) NOT NULL DEFAULT '';

-- 新数据要求非空 trim；种子补真实号。应用层创建/提交时校验 phone 非空。
```

读模型字段：`customer_id`, `store_id`, `display_name`, `phone`。

> 与首页「待跟进客户」共用此表；跟进接口可继续只返回逾期子集。成交选择器走专用 list（见下）。

### 3.2 新建 `wys_deal_invoice`

| 列 | 类型 | 说明 |
|----|------|------|
| invoice_id | bigserial PK | |
| store_id | integer NOT NULL | 当前店快照 |
| uploader_user_id | varchar(64) NOT NULL | 会话 user_id |
| customer_id | bigint NOT NULL | → wys_store_customer |
| customer_phone | varchar(32) NOT NULL | 提交时快照 |
| customer_name | varchar(128) NOT NULL | 提交时快照 |
| status | smallint NOT NULL DEFAULT 0 | 0–3 |
| image_url | text | 可空；本切片常空 |
| reject_reason | text | 驳回时有值 |
| rating_stars | smallint | 1–5，仅 rated 有意义 |
| submitted_at | timestamptz NOT NULL | 首次/重提更新 |
| created_at / updated_at | timestamptz | |

索引：`(store_id, uploader_user_id, status, submitted_at DESC)`；`(store_id, uploader_user_id, submitted_at DESC)`。

约束：`status IN (0,1,2,3)`；`rating_stars IS NULL OR rating_stars BETWEEN 1 AND 5`。

---

## 4. HTTP 契约

前缀：`/api/v1/deal-invoices`（SessionAuth）。

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/summary` | 顶栏人设 + 四格统计 |
| GET | `/` | 分页列表；`status` 可选：`all`\|`pending_review`\|`approved`\|`rejected` |
| GET | `/:invoice_id` | 详情（本人） |
| POST | `/` | 新建提交 → status=0 |
| PATCH | `/:invoice_id/resubmit` | 仅 status=3 可重提 → 0，清 reject_reason，更新 submitted_at / 可选 image_url |

客户选择器（可挂同一 router 或 `/api/v1/stores/current/customers`）：

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/deal-invoices/customers` | 当前店客户分页；`q` 可选匹配 phone/name |

### 4.1 `GET /summary` → `data`

```json
{
  "display_name": "东东枪",
  "avatar_url": "",
  "position_label": "销售经理",
  "store_name": "[4S] 北京沃德龙鼎吉利",
  "stats": {
    "uploaded": 5,
    "pending_review": 2,
    "approved": 2,
    "rejected": 1
  }
}
```

人设来源：users 展示名/头像 + 当前店成员职务 → `StoreRoleLabel` + 店名。非成员职务文案空串。

### 4.2 列表项 / 详情 `data` 字段（对齐 Flutter `DealInvoiceItem`）

```json
{
  "invoice_id": "1",
  "phone": "13812345678",
  "customer_name": "小张女士",
  "status": "pending_review",
  "submitted_at": "2026-09-22T10:00:00Z",
  "reject_reason": null,
  "rating_stars": null,
  "image_url": null
}
```

列表：`data: { list, pagination: { page, size, total } }`。

### 4.3 `POST /` body

```json
{
  "customer_id": 12,
  "image_url": null
}
```

校验：`customer_id` 属于当前店；快照 phone/name；`uploader_user_id` 取会话。成功返回完整读模型。

### 4.4 `PATCH .../resubmit` body

```json
{ "image_url": null }
```

仅本人且 `rejected`；否则 404/409。

---

## 5. Flutter 接入

### 5.1 入口

- `home_repository`：`新车关注` → **`新车成交`**
- `HomeFeatureGrid`：点击 → `RoutePath.dealInvoiceDemo`（已接）
- 可选：与二手车一样 **登录后进入**（推荐一并加上）

### 5.2 替换 Mock

| 现 Mock | 真接口 |
|---------|--------|
| `DealInvoiceMockRepository.fetch` | `GET /api/v1/deal-invoices` |
| `DealInvoiceStats.demo` + 顶栏写死文案 | `GET /summary` |
| `DealInvoiceCustomer.mockList` | `GET .../customers` |
| `pickInvoiceImage` 假延迟 | `image_picker` 相册/拍摄 → 本地 `XFile` 预览；`hasInvoiceImage=true` |
| `submit` 假延迟 | `POST /` 或 `PATCH .../resubmit`；**不** multipart |

保留现有 Widget；ViewModel 换 repository。上传页详情态展示：优先本地文件，其次 `image_url`（可空则占位图）。

### 5.3 模块

暂留 `module_settings`；路由与 Binding 不变。Go 种子：当前店若干客户（含 phone）+ 四态各至少 1 条发票，便于 Tab/详情验收。

---

## 6. 实现分票

Tracker：[`.scratch/new-car-deal-invoice/`](../.scratch/new-car-deal-invoice/spec.md)

| 票 | 交付 | 阻塞 |
|----|------|------|
| [01](../.scratch/new-car-deal-invoice/issues/01-summary-and-list.md) | 首页「新车成交」+ 真摘要 + 真列表（含表/种子） | — |
| [02](../.scratch/new-car-deal-invoice/issues/02-customer-picker.md) | 上传页选购车客户 | 01 |
| [03](../.scratch/new-car-deal-invoice/issues/03-create-with-local-image.md) | 本地选图 + 新建提交 | 02 |
| [04](../.scratch/new-car-deal-invoice/issues/04-detail-and-resubmit.md) | 详情 + 驳回同单重提 | 03 |

```text
01 → 02 → 03 → 04
```

Frontier：只开 **01**。每票独立会话 `/implement`。

---

## 7. 与现有域边界

| 易混 | 边界 |
|------|------|
| 店务审核单 `wys_store_review_order` | 首页待办「订单待审核」；**不是**成交发票 |
| 待跟进客户 | 同表逾期子集；选择器用全量客户 |
| 门店统计 `total_customers` | 展示数字，不是客户主数据 |
| 商城订单 | 买家购物；无关 |

---

## 8. 联调前提

- `auth.provider=local`，用户有 `current_store_id` 且为该店成员  
- 种子客户含 phone；至少一条 `rejected` 便于测重提  
- Flutter 指向 LAN/本机 `:8080`，带齐 Session 头  
