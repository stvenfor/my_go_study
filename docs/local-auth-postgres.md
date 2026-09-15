# 本地 Auth + Postgres（可迁回 Supabase）

> 目标：日常 **完全使用本机 Postgres + 自建 Auth**；保留 `auth.provider=supabase` 以便后期切回 Cloud。  
> 与 [lan-backend-host.md](./lan-backend-host.md) 的关系：**升级后的 `lan` = 本地全栈 + 局域网暴露**（不再 LAN+Cloud）。  
> 状态：**实现中已可编译**（`auth.provider=local`、lan 默认本地 Auth；Cloud 导入后置）。

---

## 0. 已决摘要

| # | 决策 | 选择 |
|---|------|------|
| 1 | 离开 Cloud 的方式 | 本机 Postgres + 代码实现等价能力（不自托管 GoTrue/PostgREST） |
| 2 | 以后迁 Supabase | 保留 `auth.provider=supabase` + 现有 PostgREST 仓储；表/用户 ID 用 **UUID** 对齐 |
| 3 | Auth | **双模式** `local` \| `supabase`；`lan` 默认 `local` |
| 4 | 数据 | 先空库；Cloud 导入后置 |
| 5 | 表与隔离 | `auth_users` / `profiles` / `transactions(user_id uuid)`；应用层按 user_id 过滤；本地暂不上 RLS |
| 6 | 环境 | `dev` 仍可连 Cloud；**升级 `lan`** 为本地全栈+局域网 |
| 7 | lan 与 Cloud | 不做 LAN+Cloud；Cloud 用 `dev` + `make run` |
| 8 | Flutter | API 形状不变：`token` / `refresh_token` / `session_id` / `user.id`(UUID 字符串) |

---

## 1. 目标架构

```text
Flutter（真机 / 模拟器）
  │  同一套 HTTP / WS 契约
  ▼
Go BFF
  ├─ auth.provider=local（lan 默认）
  │    Auth → auth_users + 本地 JWT(sub=UUID) + refresh 表
  │    Data → GORM/SQL profiles & transactions（忽略 accessToken）
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
| `auth_users` | `id UUID PK`, email, phone, password_hash, display_name, timestamps |
| `profiles` | `id UUID PK` = user id；display_name, avatar_url, phone |
| `transactions` | `id BIGSERIAL`, `user_id UUID` FK, type/category/amount/date/note |
| `auth_refresh_tokens` | opaque refresh → user_id + expires |

**冲突处理：** 停止对遗留 `TransactionRecord`（uint `user_id`）的 AutoMigrate；旧 `users` uint 表可保留给遗留 `/user/list`，与 Flutter 主路径无关。

迁移：`migrations/` 新增 up/down SQL；lan/local 启动时 migrate 或 AutoMigrate 新 entity。

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

1. 导出本地 `auth_users` / `profiles` / `transactions`（用户密码需按 GoTrue 要求另行处理或请用户重置）  
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
| `--default-password` | 写入 `auth_users` 的 bcrypt 临时密码（≥6） |
| `--dry-run` | 只拉取统计 |
| `--skip-users` / `--skip-profiles` / `--skip-transactions` | 跳过对应表 |

导入后用该临时密码登录本地；**Cloud 原密码不会生效**。幂等：按用户 UUID / profile id / transaction id upsert。

---

## 6. 相关 ADR

- `docs/adr/0004-local-auth-postgres-dual-provider.md`
- `docs/adr/0005-supabase-cloud-import.md`