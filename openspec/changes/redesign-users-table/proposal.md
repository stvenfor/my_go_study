## Why

上一版把本地账号收进 `users` 后，身份仍拆成 UUID `id` 和字符串 `user_id`，资料仍用单独的 Profile 模型，列名是 `display_name`。按上线标准，这一行还缺账号状态、验证时间、登录失败锁定和注销。本变更只补账号表和登录门禁，不做按角色拦截的权限系统。

## What Changes

- **BREAKING**：`users.user_id` 为唯一主键。去掉多余的 UUID `id`。注册时由服务端生成 `user_id`，客户端不得指定。
- 同一行包含 `user_name`、`email`、资料列（`phone`、`avatar_url`、`current_store_id`），以及账号列：`status`、`email_verified_at`、`phone_verified_at`、`failed_login_count`、`locked_until`、`password_changed_at`、`last_login_at`、`deleted_at`。不再有 `profiles` 表。
- **BREAKING**：`display_name` 改为 `user_name`。注册、登录、资料读取与更新都读写这一行。资料更新不得改 `status`、`deleted_at`、`password_hash`、失败次数或锁定时间。
- `email`、`phone` 仅在未注销行上唯一（`phone` 为空不占唯一）。邮箱入库前转小写。
- 登录拒绝：已注销、`status = 1`（停用）、或锁定未到期。连续失败默认 5 次后 `status = 2` 且 `locked_until` 为 15 分钟后；锁定到期后的成功登录恢复 `status = 0`。
- 用户注销只把本行 `deleted_at` 写上，并撤销该 `user_id` 的 refresh token。本变更不提供把 `status` 设为停用的管理接口。
- 响应不得带 `password_hash`、`failed_login_count`、`locked_until`。
- 保留 refresh token 表和 `wys_user_store_stats`（外键指向 `users.user_id`）。门店角色仍只用于统计展示，不在本变更里做权限校验。
- `auth.provider=supabase` 不改 Cloud schema。本变更只覆盖 **local**。

## Capabilities

### New Capabilities

- `local-user-identity`: 本地一张 `users` 表，以唯一 `user_id` 承载身份、资料和账号状态；注册、资料修改、锁定和注销都写这一行

### Modified Capabilities

- （无 `openspec/specs/` 主规格。本变更替代未归档的 `consolidate-users-table` 里「UUID `id` + 独立 Profile 语义」的约定。）

## Impact

- **DB**：`my_go_study` 本地库。`users` 以 `user_id` 为主键；部分唯一索引；开发环境可重建。
- **Go**：`entity.User` 为唯一账号模型；登录检查状态、锁定和注销；资料更新不能改安全列。
- **Flutter (`my_ai_project`)**：**BREAKING**。登录 user 与资料模型对齐 `user_id`、`user_name`、`email`。仍区分「账号未注册」和密码错误。
- **不含**：角色表、权限表、按门店角色拦截接口。
- **Supabase Cloud**：不改。
