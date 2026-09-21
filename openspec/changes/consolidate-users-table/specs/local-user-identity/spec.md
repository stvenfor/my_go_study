## Purpose

Defines local-mode user identity: one `users` table with a server-issued string `user_id`, a separate refresh-token table, and a rule that authenticated requests carry a matching `user_id` except registration-class endpoints.

## ADDED Requirements

### Requirement: Unified local users table
When `auth.provider=local`, the system SHALL store each account's credentials and profile fields in a single Postgres table named `users` with UUID primary key `id`. The system MUST NOT maintain separate `auth_users` or `profiles` tables for local mode.

#### Scenario: Register creates one user row
- **WHEN** a client successfully registers via `POST /api/v1/user/register`
- **THEN** the system persists exactly one `users` row containing email, password hash, and initial display name (and other profile defaults as applicable)
- **AND** no row is written to `auth_users` or `profiles`

#### Scenario: Profile read uses same row
- **WHEN** an authenticated client calls `GET /api/v1/profiles/me` with a valid matching `user_id`
- **THEN** the system returns display_name, avatar_url, phone (and related profile fields) from that user's `users` row

### Requirement: Public user_id on users
The `users` table SHALL include a string column `user_id` that is unique and NOT NULL. On registration the system MUST generate `user_id` as the canonical string form of `id`. The client MUST NOT supply `user_id` on register. Successful register and login responses MUST include this `user_id`.

#### Scenario: Register returns server-issued user_id
- **WHEN** a client successfully registers without sending `user_id`
- **THEN** the response includes a non-empty string `user_id`
- **AND** that value equals the string form of the new row's `id`
- **AND** a second row with the same `user_id` cannot be inserted

#### Scenario: Client-supplied user_id on register is ignored or rejected
- **WHEN** a client sends `user_id` on `POST /api/v1/user/register`
- **THEN** the system does not persist that client value as the account's `user_id`
- **AND** the stored `user_id` is the server-generated value

### Requirement: Requests require matching user_id
Except the exempt routes listed below, every API request SHALL include `user_id` (query or JSON body). When the caller is authenticated, that `user_id` MUST match the authenticated account. The system MUST reject the request when `user_id` is missing or does not match. Exempt routes are: `POST /api/v1/user/register`, `POST /api/v1/user/login`, `POST /api/v1/user/refresh`, `POST /api/v1/user/phone/otp/send`, `POST /api/v1/user/phone/otp/verify`.

#### Scenario: Authenticated call without user_id is rejected
- **WHEN** an authenticated client calls `GET /api/v1/profiles/me` without `user_id`
- **THEN** the system rejects the request with a client-visible failure
- **AND** no profile data is returned

#### Scenario: user_id that does not match the session is rejected
- **WHEN** an authenticated client calls `PATCH /api/v1/profiles/me` with a `user_id` that is not the session account's `user_id`
- **THEN** the system rejects the request
- **AND** the `users` row is not updated

#### Scenario: Exempt register does not require user_id
- **WHEN** a client calls `POST /api/v1/user/register` without `user_id`
- **THEN** the system does not reject the request for a missing `user_id`

### Requirement: Business rows associate by string user_id
Persisted business records owned by a user (including `transactions`) SHALL store the owner's string `user_id` and reference `users.user_id`. Reads and writes of those records MUST be scoped to the request's matching `user_id`.

#### Scenario: Transaction is stored under user_id
- **WHEN** an authenticated client creates a transaction with a matching `user_id`
- **THEN** the stored row's owner key is that string `user_id`
- **AND** a later list for a different `user_id` does not return that row

### Requirement: Refresh tokens stay in a separate table
The system SHALL store opaque refresh tokens in a dedicated table (current name `auth_refresh_tokens`, optionally renamed to `user_refresh_tokens`) with a foreign key to `users.id`. The system MUST NOT embed refresh token lists inside the `users` row. The refresh-token foreign-key column MUST NOT be confused with the public string `users.user_id`.

#### Scenario: Login issues refresh token
- **WHEN** a client successfully logs in via `POST /api/v1/user/login` under local auth
- **THEN** the system returns an access token, a refresh token, and `user_id`
- **AND** a hashed refresh token row exists linked to that user's internal `id`

#### Scenario: Logout revokes refresh tokens
- **WHEN** a client calls `POST /api/v1/user/logout` with a valid session and a matching `user_id`
- **THEN** the corresponding refresh token row(s) for that session are removed or invalidated
- **AND** the `users` row itself remains

### Requirement: Stable route paths
Under local auth, the system SHALL keep these route paths: `POST /api/v1/user/login|register|refresh|logout`, phone OTP routes, `GET|PATCH /api/v1/profiles/me`. Path renames are NOT required. Response envelopes stay compatible except that register and login MUST add `user_id`, and non-exempt routes MUST require `user_id`.

#### Scenario: Profile patch updates unified table
- **WHEN** an authenticated client sends `PATCH /api/v1/profiles/me` with a matching `user_id` and display_name and/or avatar_url
- **THEN** the system updates the corresponding fields on that `users` row
- **AND** a subsequent `GET /api/v1/profiles/me` with the same `user_id` reflects those values

### Requirement: Legacy uint users path removed
The system MUST NOT expose or depend on the legacy BIGSERIAL `users` table (username/email/password with integer id) for authentication or profile listing. Legacy routes that read that table (`GET /api/v1/user/profile`, `GET /api/v1/user/list`) MUST be removed or return gone/not-found consistent with deprecation.

#### Scenario: Legacy list endpoint unavailable
- **WHEN** a client calls `GET /api/v1/user/list` after this change
- **THEN** the endpoint is absent or explicitly returns a non-success indicating the legacy API is retired

### Requirement: Email uniqueness on users
The system SHALL enforce unique email on the unified `users` table for local auth.

#### Scenario: Duplicate register rejected
- **WHEN** a client attempts to register with an email that already exists in `users`
- **THEN** the system rejects the registration with a client-visible failure
- **AND** no second `users` row is created
