## Context

See proposal.md — Why. Home todo already ships `wys_after_sales_appointment` (pending/done, count = pending ∧ date ≥ today Shanghai). Mine「售后专区」is toast-only. `wys_store_customer` has name/phone but no app `user_id`. OpenSpec apply root is `my_go_study`; Flutter work is in sibling `my_ai_project` and listed as separate tasks.

## Goals / Non-Goals

**Goals:**

- New `wys_after_sales_record` + list / detail / create APIs
- Transactional link: create with `appointment_id` → appointment done
- Role split: current-store **member** write + full store read; **customer** (`customer_user_id` match) read-only
- Flutter Mine entry → list / detail / create; customer hides create
- Short API doc + CONTEXT term for「售后专区」vs「售后预约」

**Non-Goals:**

- Record edit/delete, appointment full CRUD UI
- Mall returns/refunds
- Binding every store customer to an app user automatically
- Changing home todo card sizing/packing

## Decisions

### 1. Table: new record table; appointment stays source of “待办”

- **Choose**: `wys_after_sales_record` with optional `appointment_id` FK (nullable, unique optional so one appointment → at most one record). Appointment row stays; status flip to done drives home count.
- **Why**: Appointment is a lightweight todo; record is the durable work log. Merging into one table would break home-todo semantics.
- **Alt**: Only extend appointment with repair fields — rejected; pending todos and history logs have different lifecycles.

Suggested columns (MVP):

| Column | Notes |
|--------|--------|
| `record_id` | PK |
| `store_id` | from `current_store_id` |
| `appointment_id` | nullable FK → appointment |
| `customer_id` | nullable FK → `wys_store_customer` |
| `customer_user_id` | nullable; enables customer read |
| `customer_name` / `customer_phone` | snapshots |
| `plate_no` | optional |
| `mileage` | optional int |
| `service_kind` | `0=repair` / `1=maintenance` |
| `title` | required |
| `content` | optional text |
| `service_date` | date |
| `created_by` | writer user id |
| `created_at` / `updated_at` | timestamptz |

Index: `(store_id, created_at DESC)`, `(customer_user_id, created_at DESC)`, unique partial on `appointment_id` WHERE NOT NULL.

### 2. AuthZ matrix

| Actor | List | Detail | Create |
|-------|------|--------|--------|
| Current-store member (staff or admin) | all records in current store | same store | yes |
| Customer (`customer_user_id` = me, not using member path) | own records only | own only | no |
| No login | — | — | — |

- **Write gate**: membership in `current_store_id` (any position). Broader than home-todo’s `store_admin`-only appointment card so 店员 can log work.
- **Customer list**: dedicated query or same endpoint with server-side branch: if member of current store → store list; else → `customer_user_id` list (ignore store filter / or filter optional). Prefer **one** `GET /api/v1/after-sales/records` that branches on membership to keep Flutter simple.
- **Alt**: separate `/staff/...` and `/me/...` routes — clearer but more surface; OK if branching gets messy.

### 3. API shape

Prefix `/api/v1/after-sales` (SessionAuth, local provider gate like other store modules).

| Method | Path | Who | Notes |
|--------|------|-----|--------|
| GET | `/records` | member or customer | `page`/`size`; member=current store; customer=own |
| GET | `/records/:record_id` | member or customer | ownership rules above |
| POST | `/records` | member only | body fields per spec; optional `appointment_id` |
| GET | `/pending-appointments` | member only | same date rule as home todo; for “从预约新建” |

Response envelope: existing `{ code, message, data, timestamp }`; lists use `SuccessList` (`list` + `pagination`).

Create + appointment: single DB transaction — insert record, set appointment `status=1`, fail all if appointment missing/wrong store/not pending.

Reuse home `GET /api/v1/home/after-sales-appointments`? **No for zone** — that path is admin-gated today; zone needs member-readable pending list. Keep home route as-is; add `/after-sales/pending-appointments` to avoid widening home permission accidentally.

### 4. Layering (Go)

Mirror new-car-follow / used-car:

`entity` → `repository` → `AfterSalesZoneUsecase` (+ `AccessUsecase` for membership/current store) → `AfterSalesZoneController` → router → `main.go` DI.

One focused usecase test: create-with-appointment marks done + count drops; customer cannot create; member list isolated by store.

### 5. Flutter UI (no mock — skeletal but intentional)

Mine `after_sales` → login gate → `RoutePath.afterSalesRecords` (name flexible).

| Screen | Behavior |
|--------|----------|
| List | Member: records + top chip/section “待处理预约” → tap opens create prefilled; Customer: records only, no FAB |
| Detail | Title, kind, date, customer, plate, content, linked appointment if any |
| Create | Form: kind, title, date, customer (picker or name/phone), optional plate/mileage/content; if opened from appointment, lock `appointment_id` + prefill name |

Visual: align Mine / home list patterns (AppNavBar, list tiles, empty state) — no new design system. Package: prefer `features/home` or small `features/after_sales` to avoid settings depending on domain APIs; Mine only navigates.

### 6. Docs / glossary

- `docs/after-sales-zone-api.md` short contract (like new-car-follow-api)
- `CONTEXT.md`: **售后专区** = 维修保养记录模块；_Avoid_ conflating with 售后预约 home card / 商城售后

## Risks / Trade-offs

- [Store customer has no user_id] → Customer read depends on staff setting `customer_user_id` (or future phone-match). Document; optional later auto-link by phone. Mitigation: create form field / picker for linked app user when known.
- [Widening appointment read to all members] → Home card still admin-only; zone list is separate. Accept intentional asymmetry.
- [Flutter cross-repo] → Tasks split Go / Flutter; Go can land first with curl acceptance.
- [Unique appointment_id] → Prevents double logging from same appointment; if business needs multiple visits, drop unique later.

## Migration Plan

1. Apply migration for `wys_after_sales_record` (+ indexes).
2. Deploy Go routes; old clients unchanged.
3. Ship Flutter entry + screens.
4. Rollback: drop routes / Flutter pages; down-migration drops record table only (appointments untouched except status flips already done — do not auto-reopen on rollback).

## Open Questions

- Whether create UI offers picking an app user for `customer_user_id` in MVP: default **yes if a simple user search already exists**, else optional text/UUID field deferred and customer-read stays empty until set — implementers pick smallest existing picker; if none, omit UI field and leave column for API/seed only.
