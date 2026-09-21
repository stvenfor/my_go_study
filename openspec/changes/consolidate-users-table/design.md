## Context

See proposal.md — Why. Local mode today: `auth_users` + `profiles` + `auth_refresh_tokens` + unused BIGSERIAL `users`. Business rows such as `transactions.user_id` are UUID. Supabase mode is out of the local schema merge.

## Goals / Non-Goals

**Goals:**

- One local table `users` (UUID `id`) holding credentials, profile fields, and public string `user_id`
- Keep refresh tokens 1:N, FK to internal `users.id`
- Scope business rows and non-exempt HTTP calls by string `user_id`
- Drop legacy BIGSERIAL `users` code paths

**Non-Goals:**

- Changing Supabase Cloud schema or GoTrue
- Merging refresh tokens into the user row
- Renaming route paths
- Letting the client choose `user_id`
- Production zero-downtime migration tooling (dev/LAN may rebuild DB)

## Decisions

### 1. Target table name: `users` (UUID PK)

- **Choice:** Reclaim name `users` for the unified UUID table; drop legacy BIGSERIAL `users`.
- **Why:** One account table; Flutter already treats identity as a string.
- **Alt:** Keep `auth_users` — rejected (two user-ish names remain).

### 2. Column set (local `users`)

| Column | Notes |
|--------|--------|
| `id` uuid PK | `gen_random_uuid()` on register; internal only |
| `user_id` varchar UNIQUE NOT NULL | public key; set to canonical string of `id` at insert; never taken from the client |
| `email` | UNIQUE NOT NULL |
| `phone` | one column; unique partial index if needed |
| `password_hash` | never in API JSON |
| `display_name` | single field |
| `avatar_url` | nullable |
| `created_at` / `updated_at` | |

### 3. Two different "user id" meanings

- **Public** `users.user_id` (string): request parameter and business-row owner key. FK from business tables targets `users(user_id)`.
- **Internal** `users.id` (uuid): refresh-token table FK. Rename that column if it is also called `user_id` (for example `account_id`) so it does not collide with the public string.
- **Alt:** Use string `user_id` as the only key and drop UUID `id` — rejected; refresh tokens and existing UUID joins stay on `id`.

### 4. Request rule

- **Choice:** Non-exempt handlers require `user_id` in query or JSON body and require it to equal the authenticated account's `user_id`. Missing or mismatch is a client error.
- **Exempt:** `POST /api/v1/user/register`, `login`, `refresh`, `phone/otp/send`, `phone/otp/verify`.
- **Why:** Callers explicitly bind the payload to an account; the session check stops cross-user writes.
- **Alt:** Trust JWT only and ignore a body `user_id` — rejected; the confirmed contract requires the field.

### 5. Refresh token table

- **Choice:** Keep a separate table; FK to `users.id` ON DELETE CASCADE.
- **Why:** 1:N revoke without rewriting the user row.
- **Alt:** JSONB on `users` — rejected.

### 6. Domain model and HTTP

- **Choice:** One entity mapped to `users`. Local profile and business repos read/write that row and filter by string `user_id`. Routes stay. Register and login responses add `user_id`. Remove legacy uint Profile/List routes.
- **Alt:** Keep `AuthUser` + `Profile` as two structs over one table — unnecessary.

### 7. Migration (dev/LAN)

- **Choice:** Drop or rename legacy BIGSERIAL `users` first; create UUID `users` with `user_id` backfilled from `id::text`; point `transactions.user_id` at the string key; drop `auth_users`/`profiles`. Update `navicat_local_schema.sql` for greenfield. Dirty local DBs may `DROP` and re-seed.
- **Alt:** Dual-write period — overkill for LAN-first usage.

## Risks / Trade-offs

- [Name collision during migrate] Legacy BIGSERIAL `users` vs new UUID `users` → Drop/rename the legacy table before creating the new one.
- [Public vs internal user id] Refresh-token `user_id` uuid vs `users.user_id` string → Rename the token FK column in the same migration.
- [Breaking clients] Existing Flutter calls omit `user_id` → Document the field; apply includes the client follow-up, not this planning pass.
- [Supabase mode] Local `users.user_id` vs Cloud profiles → Supabase repo stays on Cloud `profiles`; do not require the new column there in this change.
- [Phone drift] Prefer non-empty `profiles.phone`, else `auth_users.phone`.

## Migration Plan

1. Stop API; backup DB (optional for LAN).
2. Run up migration or rebuild from updated `navicat_local_schema.sql`.
3. Deploy Go with the new entity, request check, and AutoMigrate aligned to the schema.
4. Smoke: register (no `user_id` in, `user_id` out) → login (`user_id` out) → profile/transaction calls with matching `user_id` → mismatch rejected → refresh (exempt) → logout with `user_id`.
5. Rollback: restore DB dump + previous binary.

## Open Questions

- None blocking. Optional refresh-token table rename (`auth_refresh_tokens` → `user_refresh_tokens`) does not change the public contract.
