## 1. Schema & scripts

- [x] 1.1 Add migration: drop legacy BIGSERIAL `users`, create UUID `users` (including unique string `user_id`) from `auth_users` ⟕ `profiles`, backfill `user_id` from `id`, point business owner keys (including `transactions`) at `users.user_id`, rewire refresh-token FK to `users.id` under a non-colliding column name, drop `auth_users`/`profiles`
- [x] 1.2 Rewrite `scripts/navicat_local_schema.sql` for greenfield UUID `users` with `user_id`, business FKs, and refresh tokens
- [x] 1.3 Update seed / `cmd/import-supabase` to upsert one `users` row and set `user_id`

## 2. Domain & persistence

- [x] 2.1 Replace `AuthUser` + local `Profile` with one entity mapped to `users`, including `user_id`
- [x] 2.2 Point postgres profile and business repositories at `users` / string `user_id`; keep supabase profile repo on Cloud `profiles`
- [x] 2.3 Update `cmd/api` AutoMigrate; remove legacy uint `User` migrate

## 3. Usecase & HTTP

- [x] 3.1 Register/login/OTP write the unified row; register ignores client `user_id`; register and login responses include server `user_id`
- [x] 3.2 Require `user_id` on non-exempt routes and reject missing or session-mismatched values; exempt register, login, refresh, phone OTP send, and phone OTP verify
- [x] 3.3 Scope profile and transaction reads/writes by the matching string `user_id`
- [x] 3.4 Remove legacy Profile/List routes (`/api/v1/user/profile`, `/api/v1/user/list`)

## 4. Docs & verify

- [x] 4.1 Update `docs/local-auth-postgres.md` for `users.user_id` and the request rule
- [x] 4.2 Smoke local: register without `user_id` → login returns `user_id` → profile/transaction with match succeeds → mismatch and missing `user_id` fail → refresh without `user_id` still works → logout with `user_id`
