## Purpose

Store staff maintain repair and maintenance records for the current store, link them to after-sales appointments when applicable, and let linked customers read their own records without write access.

## ADDED Requirements

### Requirement: Authenticated access scoped to current store for staff writes
When `auth.provider=local`, all after-sales zone write endpoints MUST require SessionAuth. Creating a maintenance record MUST use the actor's `current_store_id` as the record's store scope. If the actor has no current store, or is not a member of that store, the system MUST reject the create request. The system MUST NOT trust a client-supplied `store_id` for scoping writes.

#### Scenario: Member creates record for current store
- **WHEN** a logged-in store member with a non-zero `current_store_id` creates a maintenance record
- **THEN** the record is stored under that store
- **AND** the creator is recorded as the writer

#### Scenario: No current store rejects create
- **WHEN** a logged-in user with no current store attempts to create a maintenance record
- **THEN** the system rejects the request

#### Scenario: Non-member cannot create
- **WHEN** a logged-in user whose current store membership is absent attempts to create a maintenance record
- **THEN** the system rejects the request

### Requirement: Staff list and detail for current store records
A current-store member SHALL list maintenance records for that store with pagination (`page`, `size`) sorted by `created_at` descending, and SHALL retrieve a single record by id when it belongs to the current store. A member MUST NOT read another store's records via these endpoints.

#### Scenario: Member lists store records
- **WHEN** store A has three records and store B has one
- **AND** a member of store A with current store A requests the list
- **THEN** only store A's three records appear

#### Scenario: Cross-store detail hidden
- **WHEN** a member of store A requests a record_id that belongs to store B
- **THEN** the system rejects the request as not found or forbidden

### Requirement: Customer read-only access to own records
A logged-in user who is **not** acting as a member of the record's store SHALL be able to list and detail only records where `customer_user_id` equals the session user. Such a user MUST NOT create, update, or delete maintenance records. Store members retain full list/detail for their current store as specified above.

#### Scenario: Customer lists own records only
- **WHEN** user C is linked as `customer_user_id` on two records and not a member of those stores
- **AND** user C requests the customer-facing list
- **THEN** only those two records appear

#### Scenario: Customer create rejected
- **WHEN** a non-member customer attempts to create a maintenance record
- **THEN** the system rejects the request

#### Scenario: Customer cannot read another customer's record
- **WHEN** user C requests detail for a record whose `customer_user_id` is user D
- **AND** user C is not a member of that record's store
- **THEN** the system rejects the request

### Requirement: Create maintenance record with required fields
Creating a record MUST accept at least: service category (`service_kind`: repair or maintenance), service title, service date, and customer identity via either `customer_id` (existing `wys_store_customer` in the current store) or snapshot fields `customer_name` + `customer_phone`. Optional fields MAY include vehicle plate, mileage, content/notes, `customer_user_id`, and `appointment_id`. Invalid `service_kind` or missing required identity MUST be rejected.

#### Scenario: Create with store customer id
- **WHEN** a store member creates a record with a valid current-store `customer_id`, `service_kind=maintenance`, title, and service date
- **THEN** the record is persisted
- **AND** customer name/phone snapshots are filled from the store customer when not supplied

#### Scenario: Create with name and phone snapshot
- **WHEN** a store member creates a record without `customer_id` but with `customer_name` and `customer_phone`
- **THEN** the record is persisted with those snapshots

#### Scenario: Invalid service kind rejected
- **WHEN** a store member creates a record with an unknown `service_kind`
- **THEN** the system rejects the request

### Requirement: Link to after-sales appointment and mark done
When create includes `appointment_id`, the appointment MUST belong to the same current store and MUST be pending. On successful create, the system MUST mark that appointment as done in the same operation so it no longer counts toward the home todo `after_sales_appointment` card. An appointment already done or belonging to another store MUST be rejected. Omitting `appointment_id` MUST leave appointments unchanged.

#### Scenario: Create from pending appointment completes it
- **WHEN** store A has a pending appointment for today or later
- **AND** a store member creates a maintenance record with that `appointment_id`
- **THEN** the record references the appointment
- **AND** the appointment status becomes done
- **AND** the home todo pending appointment count no longer includes it

#### Scenario: Foreign or done appointment rejected
- **WHEN** a store member creates a record with an appointment_id from another store or already done
- **THEN** the system rejects the request
- **AND** no new record is created

### Requirement: Staff can list pending appointments for zone flow
A current-store member SHALL list pending after-sales appointments for the current store whose appointment date is today or later (same calendar rule as the home todo card), so the Flutter zone can offer “新建记录” from an appointment. This list MUST be readable by store members who can create records, not only users with home-todo admin permission.

#### Scenario: Member sees pending appointments in zone
- **WHEN** the current store has two pending future-or-today appointments and one expired pending
- **AND** a store member requests the zone pending-appointments list
- **THEN** only the two future-or-today pending appointments appear

### Requirement: Flutter after-sales zone entry and screens
The Flutter app MUST navigate from Mine function id `after_sales` to an after-sales zone list when logged in (otherwise login with redirect). Store members MUST see list, detail, and create flows (including optional start-from pending appointment). Customer-only users MUST see list and detail of their own records without a create entry point.

#### Scenario: Mine entry opens zone when logged in
- **WHEN** a logged-in user taps「售后专区」on Mine
- **THEN** the app opens the after-sales zone list route

#### Scenario: Customer UI hides create
- **WHEN** the session user is not a member of the current store (or has no current store) and opens the zone as a customer
- **THEN** the create action is not offered
- **AND** only own linked records are shown
