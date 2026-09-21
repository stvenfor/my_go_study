## Context

See proposal.md — Why. Local `users` today has UUID `id` plus string `user_id`, and profile HTTP is still a separate type. Store role `0/1/2` on `wys_user_store_stats` is display-only. This change adds account lifecycle columns on `users`. It does not add RBAC.

## Goals / Non-Goals

**Goals:**

- One local `users` row; `user_id` is the only account key
- Account columns and login gates (disabled, locked, deleted) on that row
- Profile GET/PATCH update the same row and cannot change security columns
- Refresh tokens and store stats stay side tables, both referencing `users.user_id`

**Non-Goals:**

- Permission tables, role-to-permission maps, or rejecting APIs by store role
- An admin API that sets `status = 1`
- Supabase Cloud schema or GoTrue
- Folding refresh tokens or `wys_user_store_stats` into the user row
- Collapsing 「账号未注册」 and wrong-password into one error (Flutter branches on the former)
- Renaming existing HTTP paths
- Letting the client choose `user_id`
- Production zero-downtime tooling (dev/LAN may rebuild)

## Decisions

### 1. `user_id` is the primary key

- **Choice:** `users.user_id varchar(64)` PRIMARY KEY. Drop UUID `id`. Register sets `user_id` with `uuid.NewString()`.
- **Why:** One public key. The old pair was always the same value.
- **Alt:** Keep UUID `id` plus unique `user_id` — rejected.

### 2. Columns on `users`

| Column | Notes |
|--------|--------|
| `user_id` | PK, server-generated |
| `user_name` | NOT NULL; not unique; replaces `display_name` |
| `email` | NOT NULL, stored lowercase |
| `phone` | nullable |
| `password_hash` | NOT NULL; never in JSON |
| `status` | smallint NOT NULL default 0. `0` active, `1` disabled, `2` locked. CHECK in (0, 1, 2) |
| `email_verified_at` / `phone_verified_at` | nullable |
| `failed_login_count` | int NOT NULL default 0; never in JSON |
| `locked_until` | nullable; never in JSON |
| `password_changed_at` / `last_login_at` | nullable |
| `avatar_url` | nullable |
| `current_store_id` | nullable int |
| `created_at` / `updated_at` / `deleted_at` | `deleted_at` set means closed |

Partial unique indexes, both `WHERE deleted_at IS NULL`:

- `email`
- `phone` WHERE `phone IS NOT NULL`

Profile fields are columns, not a JSON blob and not a `profiles` table. Permissions are not columns on this row.

### 3. Login and lock

- Reject when `deleted_at` is set, or `status = 1`, or (`status = 2` and (`locked_until` is null or `locked_until > now`)), or `locked_until > now`.
- Wrong password on an existing row increments `failed_login_count`. At 5, set `status = 2` and `locked_until = now + 15 minutes`.
- Success after the lock has expired: set `status = 0`, `failed_login_count = 0`, `locked_until = null`, `last_login_at = now`.
- Unknown email stays 「账号未注册」. Wrong password stays a credential error. Do not merge them.
- Lock does not revoke an already issued session. Disable and delete do.

### 4. Who may change status

- Register writes `status = 0`, `failed_login_count = 0`, `deleted_at` null.
- Login writes lock, unlock, and `last_login_at`.
- `POST /api/v1/user/deactivate` with a matching `user_id` sets `deleted_at` and revokes every refresh token for that `user_id`. Same email may register again as a new `user_id`.
- No HTTP API sets `status = 1` in this change. If a row is already `1`, login and later authenticated calls fail.
- `PATCH /api/v1/profiles/me` may change `user_name`, `avatar_url`, and `phone` only. `display_name` and `username` write `user_name`. Ignore or reject `status`, `deleted_at`, `user_id`, `email`, `password_hash`, `failed_login_count`, `locked_until`.

### 5. Authenticated calls

Non-exempt routes still require `user_id` matching the session. They also MUST reject the call when that row is deleted or `status = 1`. Exempt routes stay: register, login, refresh, phone OTP send/verify. Refresh of a deleted or disabled account MUST fail and MUST NOT issue tokens.

### 6. Response and side tables

Auth and profile JSON include `user_id`, `user_name`, `email`, `status`, and when set `phone` and `avatar_url`. If `id` is present it equals `user_id`. Never return `password_hash`, `failed_login_count`, or `locked_until`.

Refresh-token FK → `users(user_id)` ON DELETE CASCADE. `transactions.user_id` and `wys_user_store_stats.user_id` reference `users(user_id)`. Store `role` stays on the stats table and is not checked by handlers in this change.

### 7. Flutter

Login user and profile parsing read `user_id`, `user_name`, and `email`. Keep accepting `id` / `display_name` / `username` as aliases. Keep the 「账号未注册」 branch.

## Risks / Trade-offs

- [No admin disable API] `status = 1` can be set only in the database until a later permission change → Login still rejects it.
- [Lock vs open session] A locked account can keep using an existing access token until it expires → Acceptable; lock stops new logins. Disable and delete revoke refresh tokens.
- [Email reuse after delete] Partial unique index allows a new row → Old `user_id` stays on historical business rows.
- [Flutter field names] Clients that only read `display_name` → Response includes `user_name`; PATCH still accepts `display_name`.
- [Supabase] Cloud `profiles` unchanged → Do not add these columns there in this change.

## Migration Plan

1. Stop API. Optional DB backup.
2. Up migration: backfill `user_id`, rename `display_name` to `user_name`, add account columns with defaults (`status = 0`), swap PK to `user_id`, add partial unique indexes, retarget FKs. Greenfield uses updated `navicat_local_schema.sql`.
3. Deploy Go and Flutter together.
4. Smoke: register → login returns `user_id`, `user_name`, `email`, `status = 0` → five bad passwords lock → login rejected until `locked_until` → profile patch cannot set `status` → deactivate sets `deleted_at` and blocks login → same email can register a new `user_id`.
5. Rollback: restore dump and previous binaries.

## Open Questions

- None. Lock threshold is 5 failures and 15 minutes, not a later choice.
