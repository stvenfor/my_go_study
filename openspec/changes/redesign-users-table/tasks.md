## 1. Schema

- [x] 1.1 Add up/down migration: `users.user_id` primary key, `display_name` renamed to `user_name`, account columns (`status`, verification times, `failed_login_count`, `locked_until`, `password_changed_at`, `last_login_at`, `deleted_at`), drop UUID `id`, partial unique indexes on open `email` and non-null `phone`, retarget FKs to `users(user_id)`
- [x] 1.2 Update `scripts/navicat_local_schema.sql` and seed SQL to the same column set and checks (`status` in 0, 1, 2)
- [x] 1.3 Align GORM `entity.User` and AutoMigrate with that table (no second id, no `profiles` table)

## 2. Local reads and writes

- [x] 2.1 Register creates one `users` row: server `user_id`, lowercase `email`, `user_name` from register username (else email local-part), `status = 0`, `failed_login_count = 0`, `deleted_at` null
- [x] 2.2 Profile get and patch load and update that row by `user_id`; `user_name` is canonical; `display_name` and `username` write the same column; patch must not change `status`, `deleted_at`, `email`, `user_id`, `password_hash`, `failed_login_count`, or `locked_until`
- [x] 2.3 Login rejects deleted, `status = 1`, and unexpired locks; fifth consecutive wrong password sets `status = 2` and `locked_until` 15 minutes ahead; success after expiry clears the lock and sets `last_login_at`; unknown email stays 「账号未注册」
- [x] 2.4 `POST /api/v1/user/deactivate` with a matching `user_id` sets `deleted_at` and revokes that account's refresh tokens; do not add an API that sets `status = 1`
- [x] 2.5 Refresh-token rows reference `users.user_id`; logout revokes the session token and leaves `deleted_at` null; refresh fails for deleted or disabled accounts
- [x] 2.6 Reject missing or mismatched `user_id` on non-exempt routes, and reject authenticated calls when the row is deleted or `status = 1`

## 3. HTTP contract

- [x] 3.1 Login, register, and `GET /api/v1/profiles/me` return `user_id`, `user_name`, `email`, and `status`, plus `phone` and `avatar_url` from the same row; if `id` is present it equals `user_id`; omit `password_hash`, `failed_login_count`, and `locked_until`
- [x] 3.2 Keep existing route paths; add deactivate; do not restore legacy integer user list or `GET /api/v1/user/profile`; do not enforce store role

## 4. Flutter (`my_ai_project`)

- [x] 4.1 Parse login user and profile from `user_id`, `user_name`, `email`, and `status`; keep `id` / `display_name` / `username` as aliases; keep the 「账号未注册」 branch

## 5. Check

- [x] 5.1 Go tests cover register ignoring client `user_id`, profile patch not changing `status`, fifth-failure lock, expired lock login, deactivate, and mismatched `user_id`
- [x] 5.2 Note the column set and smoke path in `docs/local-auth-postgres.md`: register → login → lock → profile patch → deactivate → re-register same email
