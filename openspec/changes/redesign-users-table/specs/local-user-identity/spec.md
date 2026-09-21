## Purpose

Local accounts live on one `users` row keyed by a unique `user_id`, including name, email, profile fields, and account status. Registration, profile edits, lockout, and deactivation write that row. This capability does not grant or check permissions.

## ADDED Requirements

### Requirement: One users row keyed by user_id
When `auth.provider=local`, the system SHALL store each account in a single Postgres table named `users`. The primary key SHALL be a unique, non-null string column `user_id`. The table MUST NOT have a second account identifier such as a separate UUID `id`. The same row SHALL hold `user_name`, `email`, `phone`, `password_hash`, `status`, `email_verified_at`, `phone_verified_at`, `failed_login_count`, `locked_until`, `password_changed_at`, `last_login_at`, `avatar_url`, `current_store_id`, `created_at`, `updated_at`, and `deleted_at`. The system MUST NOT keep a `profiles` table or an `auth_users` table for local mode. API JSON MUST NOT include `password_hash`, `failed_login_count`, or `locked_until`.

#### Scenario: Register inserts one users row
- **WHEN** a client successfully registers via `POST /api/v1/user/register` without sending `user_id`
- **THEN** the system inserts exactly one `users` row
- **AND** that row's `user_id` is a non-empty server-generated string
- **AND** `user_name` and lowercase `email` are stored on that row
- **AND** `status` is 0, `failed_login_count` is 0, and `deleted_at` is null
- **AND** no row is written to `profiles` or `auth_users`

#### Scenario: Client-supplied user_id is not stored
- **WHEN** a client sends `user_id` on `POST /api/v1/user/register`
- **THEN** the system does not persist that client value as the account's `user_id`
- **AND** the stored `user_id` is the server-generated value

#### Scenario: Duplicate user_id cannot be inserted
- **WHEN** a second account would reuse an existing `user_id`
- **THEN** the database rejects the insert

#### Scenario: Email unique only among open accounts
- **WHEN** an open account already has an email and another open account is inserted with the same email
- **THEN** the database rejects the insert
- **AND** a closed account (`deleted_at` set) with that email does not by itself cause the rejection

### Requirement: Auth and profile responses share the users row
Register, login, and profile read responses SHALL expose the same account fields from that `users` row: `user_id`, `user_name`, `email`, and `status`, plus `phone` and `avatar_url` when present. If a response also includes `id`, it MUST equal `user_id`. The system MUST NOT return a profile payload whose identity differs from the account `user_id`.

#### Scenario: Login returns the users row fields
- **WHEN** a client successfully logs in via `POST /api/v1/user/login`
- **THEN** the user object includes `user_id`, `user_name`, `email`, and `status` from that `users` row
- **AND** the body does not include `password_hash`, `failed_login_count`, or `locked_until`

#### Scenario: Profile read returns the same row
- **WHEN** an authenticated client calls `GET /api/v1/profiles/me` with a matching `user_id` for an open active account
- **THEN** the response `user_id` equals that account's `user_id`
- **AND** `user_name`, `email`, `status`, `phone`, and `avatar_url` come from the same `users` row

### Requirement: Profile updates write the users row
Updating profile data SHALL update columns on the caller's `users` row and SHALL NOT write a separate profile table. The canonical write field for the display name is `user_name`. A request field `display_name` or `username`, when sent, MUST be applied to `user_name` on that same row. Profile update MUST NOT change `user_id`, `email`, `status`, `deleted_at`, `password_hash`, `failed_login_count`, or `locked_until`.

#### Scenario: Patch user_name updates users
- **WHEN** an authenticated client sends `PATCH /api/v1/profiles/me` with a matching `user_id` and a new `user_name`
- **THEN** the system updates `users.user_name` for that `user_id`
- **AND** a following `GET /api/v1/profiles/me` with the same `user_id` returns the new `user_name`

#### Scenario: Patch display_name alias updates user_name
- **WHEN** an authenticated client sends `PATCH /api/v1/profiles/me` with a matching `user_id` and `display_name` and without `user_name`
- **THEN** the system stores that value in `users.user_name`
- **AND** the next profile read returns it as `user_name`

#### Scenario: Profile patch cannot disable the account
- **WHEN** an authenticated client sends `PATCH /api/v1/profiles/me` with a matching `user_id` and `status` other than the current value
- **THEN** `users.status` is unchanged

### Requirement: Login enforces account state
The system SHALL reject `POST /api/v1/user/login` when the matching open account has `deleted_at` set, `status = 1`, `locked_until` in the future, or `status = 2` with `locked_until` null or in the future. A wrong password for an existing open account SHALL increment `failed_login_count`. When that count reaches 5, the system SHALL set `status = 2` and `locked_until` to 15 minutes after the failure. A successful login after `locked_until` has passed SHALL set `status = 0`, `failed_login_count = 0`, `locked_until` null, and `last_login_at` to the current time. Unknown email MUST remain distinguishable from a wrong password.

#### Scenario: Disabled account cannot log in
- **WHEN** a client logs in with the correct password for an account whose `status` is 1
- **THEN** the system rejects the login
- **AND** no new refresh token is issued

#### Scenario: Fifth failure locks the account
- **WHEN** the same open account receives a fifth consecutive wrong password
- **THEN** `failed_login_count` is 5, `status` is 2, and `locked_until` is about 15 minutes ahead
- **AND** a correct password before `locked_until` is rejected

#### Scenario: Expired lock allows login
- **WHEN** a client logs in with the correct password after `locked_until` has passed
- **THEN** the login succeeds
- **AND** `status` is 0, `failed_login_count` is 0, and `locked_until` is null

#### Scenario: Unknown email stays a distinct error
- **WHEN** a client logs in with an email that has no open account
- **THEN** the system reports that the account is not registered
- **AND** that response is not the same as a wrong password for an existing account

### Requirement: Deactivation writes deleted_at
An authenticated client SHALL be able to close the session account by `POST /api/v1/user/deactivate` with a matching `user_id`. The system SHALL set `deleted_at` on that `users` row and revoke refresh tokens for that `user_id`. The system MUST NOT delete the row. This change MUST NOT expose an API that sets `status` to 1.

#### Scenario: Deactivate closes the row
- **WHEN** an authenticated client calls `POST /api/v1/user/deactivate` with a matching `user_id`
- **THEN** `deleted_at` is set on that `users` row
- **AND** refresh tokens for that `user_id` are revoked
- **AND** a later login for that account is rejected

#### Scenario: Same email can register after deactivation
- **WHEN** an email belongs only to a row with `deleted_at` set and a client registers that email again
- **THEN** the system creates a new `users` row with a new `user_id`

### Requirement: Requests require matching user_id
Except the exempt routes listed below, every API request SHALL include `user_id` (query or JSON body). When the caller is authenticated, that `user_id` MUST match the authenticated account. The system MUST reject the request when `user_id` is missing or does not match, and when that account is deleted or `status = 1`. Exempt routes are: `POST /api/v1/user/register`, `POST /api/v1/user/login`, `POST /api/v1/user/refresh`, `POST /api/v1/user/phone/otp/send`, `POST /api/v1/user/phone/otp/verify`. Refresh for a deleted or disabled account MUST fail and MUST NOT issue tokens.

#### Scenario: Profile read without user_id is rejected
- **WHEN** an authenticated client calls `GET /api/v1/profiles/me` without `user_id`
- **THEN** the system rejects the request
- **AND** no user row is returned

#### Scenario: Mismatched user_id does not update the row
- **WHEN** an authenticated client calls `PATCH /api/v1/profiles/me` with a `user_id` that is not the session account
- **THEN** the system rejects the request
- **AND** the `users` row is not updated

#### Scenario: Disabled account cannot call profile
- **WHEN** an authenticated client calls `GET /api/v1/profiles/me` with a matching `user_id` whose `status` is 1
- **THEN** the system rejects the request

### Requirement: Owned records reference users.user_id
Persisted business records owned by a user, including `transactions`, SHALL store the owner's `user_id` and reference `users.user_id`. Reads and writes of those records MUST be scoped to the request's matching `user_id`. Refresh tokens SHALL stay in a separate table whose foreign key references `users.user_id`, and MUST NOT be stored inside the `users` row. Per-store stats SHALL stay in `wys_user_store_stats` keyed by the same string `user_id`. This change MUST NOT authorize or reject a request based on store role.

#### Scenario: Transaction owner is user_id
- **WHEN** an authenticated client creates a transaction with a matching `user_id`
- **THEN** the stored owner key is that string `user_id`
- **AND** a list for a different `user_id` does not return that row

#### Scenario: Logout does not delete the users row
- **WHEN** a client calls `POST /api/v1/user/logout` with a valid session and a matching `user_id`
- **THEN** the refresh token for that session is revoked
- **AND** the `users` row remains and `deleted_at` stays null

### Requirement: Route paths stay
Under local auth, the system SHALL keep `POST /api/v1/user/login|register|refresh|logout`, the phone OTP routes, and `GET|PATCH /api/v1/profiles/me`, and SHALL add `POST /api/v1/user/deactivate`. The system MUST NOT expose the legacy integer-id user list or legacy `GET /api/v1/user/profile` backed by a BIGSERIAL `users` table.

#### Scenario: Profile route still exists
- **WHEN** an authenticated client calls `GET /api/v1/profiles/me` with a matching `user_id` for an active account
- **THEN** the route is served from the `users` row described above
- **AND** the path is unchanged
