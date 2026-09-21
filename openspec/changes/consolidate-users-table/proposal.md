## Why

本地 Postgres 同时存在 `auth_users`、`profiles`、遗留 `users` 三套用户数据，字段重叠、无 FK、读写分散。合并为单一 `users` 后，业务请求与落库仍缺少统一的字符串用户键。增加服务端生成的 `user_id`，让登录后的请求和业务数据都按该字段关联。

## What Changes

- 合并 `auth_users` + `profiles` → 统一表 `users`（UUID `id` 主键；含 email/phone/password_hash 与 display_name/avatar_url 等资料字段）
- **新增** `users.user_id`：字符串、唯一、非空；注册时由服务端生成（`id` 的规范字符串），客户端不得自造；注册与登录响应返回该值
- 业务数据（至少 `transactions`）以字符串 `user_id` 关联 `users.user_id`，不再只用内部 UUID
- **保留** refresh token 表（一对多；内部 FK 指向 `users.id`，列名避免与公开 `user_id` 混淆）
- **删除**遗留 BIGSERIAL `users` 及其 HTTP/仓储路径（`GET /api/v1/user/profile`、`GET /api/v1/user/list`）
- **BREAKING**：除注册、登录、刷新、手机 OTP 发送/校验外，请求必须携带 `user_id`，且必须与当前登录身份一致，否则拒绝
- 路由路径不改：`/api/v1/user/*`、`/api/v1/profiles/me` 仍在
- `auth.provider=supabase` 继续走 GoTrue + Cloud `profiles`；本变更聚焦 **local** 模式

## Capabilities

### New Capabilities

- `local-user-identity`: 本地统一用户表、公开 `user_id`、refresh token 表，以及请求必须携带并匹配 `user_id` 的约定

### Modified Capabilities

- （无既有 main specs；本仓库 openspec 初建）

## Impact

- **DB**：`my_go_study` 本地库；`users.user_id` 与业务表字符串外键；开发环境可 drop 重建
- **Go**：entity、local auth、profile、业务仓储的归属过滤、请求校验、AutoMigrate、import、Navicat SQL
- **Flutter (`my_ai_project`)**：**BREAKING**。登录后接口须带 `user_id`；注册/登录响应多返回 `user_id`
- **Supabase Cloud**：不改 Cloud schema
