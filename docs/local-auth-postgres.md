# 本地 Auth + Postgres（可迁回 Supabase）

> 目标：日常 **完全使用本机 Postgres + 自建 Auth**；保留 `auth.provider=supabase` 以便后期切回 Cloud。  
> 与 [lan-backend-host.md](./lan-backend-host.md) 的关系：**升级后的 `lan` = 本地全栈 + 局域网暴露**（不再 LAN+Cloud）。  
> 状态：**已实现**（`auth.provider=local`、lan 默认本地 Auth；`make import-supabase` 已可用）。

---

## 0. 已决摘要

| # | 决策 | 选择 |
|---|------|------|
| 1 | 离开 Cloud 的方式 | 本机 Postgres + 代码实现等价能力（不自托管 GoTrue/PostgREST） |
| 2 | 以后迁 Supabase | 保留 `auth.provider=supabase` + 现有 PostgREST 仓储；本地主键是字符串 `user_id` |
| 3 | Auth | **双模式** `local` \| `supabase`；`lan` 默认 `local` |
| 4 | 数据 | 先空库；Cloud 导入后置 |
| 5 | 表与隔离 | 统一 `users.user_id` 主键；业务数据按该字段关联；登录后请求默认必带且须匹配 |
| 6 | 环境 | `dev` 仍可连 Cloud；**升级 `lan`** 为本地全栈+局域网 |
| 7 | lan 与 Cloud | 不做 LAN+Cloud；Cloud 用 `dev` + `make run` |
| 8 | Flutter | 路径不变。注册/登录多返回 `user.user_id`。受保护接口身份只认 Session（Authorization + X-Session-ID + X-Device-ID），**不必**再传 `user_id` |

---

## 1. 目标架构

```text
Flutter（真机 / 模拟器）
  │  同一套 HTTP / WS 契约
  ▼
Go BFF
  ├─ auth.provider=local（lan 默认）
  │    Auth → users（user_id 主键）+ 本地 JWT(sub=user_id) + refresh 表(user_id)
  │    Data → GORM/SQL users & transactions（按字符串 user_id 过滤）
  │    Session → 现有 Redis device session（userID=UUID 字符串）
  │
  └─ auth.provider=supabase（dev 可选）
       Auth → GoTrue Cloud
       Data → PostgREST + RLS（现有）
```

Compose：继续 `postgres` + `redis` + `app` + `worker`；**不**新增 GoTrue/PostgREST 容器。

---

## 2. 配置

```yaml
# configs/config.lan.yaml（升级语义）
auth:
  provider: local   # AUTH_PROVIDER=local|supabase
```

| 场景 | 建议 |
|------|------|
| `make lan-up` | `APP_ENV=lan` → provider=local；无需 Cloud 密钥即可注册业务路由 |
| `make run` + `dev` | 可继续 supabase.env 连 Cloud |
| 显式切 Cloud | `AUTH_PROVIDER=supabase` + URL/anon |

路由启用条件：`provider=local` **或** `Supabase.Enabled()`。

---

## 3. Schema（本地）

| 表 | 要点 |
|----|------|
| `users` | 主键 `user_id`。同一行：`user_name`、`email`、`phone`、`password_hash`、`status`（0 正常 / 1 停用 / 2 锁定）、验证时间、`failed_login_count`、`locked_until`、`password_changed_at`、`last_login_at`、`avatar_url`、`current_store_id`、`deleted_at`。`email` 与非空 `phone` 仅在 `deleted_at IS NULL` 时唯一。接口不返回 `password_hash`、`failed_login_count`、`locked_until` |
| `transactions` | `user_id varchar` → `users.user_id` |
| `auth_refresh_tokens` | `user_id` → `users.user_id`；不把 token 塞进用户行 |
| `wys_user_store_stats` | 每人每店一行展示数字。`role` 列遗留，不参与接口放行 |
| `wys_mall_*` | 门店商城：类目、SPU/SKU、兑换码、购物车、订单快照、支付/退款/审计。金额 `numeric(10,2)`。无物理外键。`payment_channel`：1 支付宝 / 2 微信 / 3 苹果内购 / 4 华为内购。本地 `POST /api/v1/mall/orders/:id/pay` 任意合法渠道直接成功落库，不调渠道 SDK。待支付订单自 `created_at` 起 **15 分钟**支付窗口；列表/详情/支付时惰性超时取消，响应含 `pay_deadline_at`。启动时门店 1 种子 ≥35 条在售；`GET /api/v1/mall/stores/:store_id/products?page=&size=` 默认每页 10；`GET .../products/:product_id` 返回在售 SPU 与上架规格（不含发放地址）；`GET /api/v1/mall/orders?page=&size=&status=` 买家自己的订单列表（`status` 可省略或 `0–4`，逗号多值如 `1,2`；含行快照） |
| `wys_user_address` | 用户收货地址簿。软删；每人至多一条默认（偏唯一索引）。订单只快照 `receiver_*`，不引用 `address_id`。`GET/POST/PATCH/DELETE /api/v1/user/addresses*`（local SessionAuth） |

注册、登录、刷新、`POST /api/v1/user/phone/otp/send`、`POST /api/v1/user/phone/otp/verify` **不要求**请求里的 `user_id`。其余已登录接口必须在 query 或 JSON body 带 `user_id`，且必须与当前会话一致，否则 400。账号已注销或 `status = 1` 时拒绝业务请求。连续 5 次密码错误锁定 15 分钟。`POST /api/v1/user/deactivate` 写 `deleted_at` 并撤销 refresh token，不删除行。没有把 `status` 设为停用的接口。

**冒烟：** 注册（请求不带 `user_id`，响应有 `user_id`、`user_name`、`email`、`status=0`）→ 登录 → 连续 5 次错误密码后锁定 → `PATCH /api/v1/profiles/me` 改 `user_name` 且不能改 `status` → 注销 → 同一邮箱可再注册出新的 `user_id`。

**商城冒烟（local）：** 店员写商品/SKU → 买家下单（实体须收货人）→ `pay` 带 `payment_channel` 1–4 任一值即成功并扣库存/发码 → 再 pay 幂等不重复扣库存。

**冲突处理：** 遗留 BIGSERIAL `users` 在迁移时改名为 `users_legacy_uint`。不再提供 `/api/v1/user/list` 与 `/api/v1/user/profile`。本地不再使用 `auth_users` / `profiles`。

### 表命名规范（`wys_` 前缀）

- **身份表** `role` / `permission` / `role_permission` / `user_role` 跟 `users` 一样不加 `wys_`。门店组织用 `wys_store`、`wys_store_member`。
- **Mine 统计**在 `wys_user_store_stats`，每人每店一行。职务在 `wys_store_member.position`（0 销售顾问 / 1 销售经理 / 2 总经理），不是统计卡上的 `role`，也不参与接口放行。
- **商城**表一律 `wys_mall_*`。能力码 `mall.catalog.write` 绑在 `store_admin` / `store_staff`（及平台管理员）。
- **已有无前缀表**（`users` / `transactions` / `auth_refresh_tokens` 以及上面的身份表）保持现名。新的业务表仍用 `wys_`。
- GORM `TableName()` / 迁移 SQL / Navicat 脚本三者表名必须一致。

迁移：`migrations/20260921120000_consolidate_users.up.sql`、`migrations/20260921140000_users_account_status.up.sql`、`migrations/20260921180000_mall_schema.up.sql`；启动时 `ConsolidateLocalUsers`（含 redesign、access、mall SQL）后再 AutoMigrate。

---

## 4. 实现切片（顺序）

1. Config `auth.provider` + BindEnv  
2. Entity + 迁移 + postgres profile/transaction（UUID）  
3. Local Auth usecase（register/login/refresh/logout）+ UUID JWT  
4. Middleware：local 验 JWT 后写入与现网相同的 `supabaseUser` 上下文（ID=UUID）  
5. `main` / router：local 接线；lan 默认 local  
6. 文档 / ADR；Flutter 通常零改（LAN 仍用 `.env.lan`）

---

## 5. 以后迁回 Supabase Cloud

1. 导出本地 `users` / `transactions`（用户密码需按 GoTrue 要求另行处理或请用户重置）  
2. 在 Cloud 建表 + 跑 `supabase/migrations` RLS  
3. `AUTH_PROVIDER=supabase` + `SUPABASE_*`  
4. Flutter 仍打同一 BFF

---

## 5.1 Cloud → 本地导入（已实现）

前提：本地 Postgres 可达；`.env.local` 有 `SUPABASE_SERVICE_ROLE_KEY`；`configs/supabase.env` 有 URL/anon。

```bash
# 只看数量（不写库；跳过用户因无需密码）
make import-supabase-dry

# 正式导入（Cloud 密码无法导出，所有导入用户统一临时密码）
make import-supabase DEFAULT_PASSWORD='ChangeMe123!'
```

等价：

```bash
./scripts/load-env.sh go run ./cmd/import-supabase --default-password='ChangeMe123!'
./scripts/load-env.sh go run ./cmd/import-supabase --dry-run --skip-users
```

| 标志 | 含义 |
|------|------|
| `--default-password` | 写入 `users.password_hash` 的 bcrypt 临时密码（≥6） |
| `--dry-run` | 只拉取统计 |
| `--skip-users` / `--skip-profiles` / `--skip-transactions` | 跳过对应表 |

导入后用该临时密码登录本地；**Cloud 原密码不会生效**。幂等：按用户 UUID / profile id / transaction id upsert。

---

## 6. 相关 ADR

- `docs/adr/0004-local-auth-postgres-dual-provider.md`
- `docs/adr/0005-supabase-cloud-import.md`
