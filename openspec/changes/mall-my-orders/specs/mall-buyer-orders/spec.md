## Purpose

Buyers can browse their own mall orders in a paginated list with optional status filters, open order detail with line snapshots, and pay or cancel unpaid orders using existing mall payment flows.

## ADDED Requirements

### Requirement: Buyer lists own orders with pagination
When `auth.provider=local`, an authenticated buyer SHALL list only orders whose `buyer_user_id` equals the session user. The list MUST be paginated with `page` (1-based) and `size` query parameters and MUST return `list` plus `pagination` in the same shape as other list APIs. Default sort MUST be `created_at` descending. Amounts MUST be decimal strings (`numeric(10,2)`), not floating-point JSON numbers.

#### Scenario: Buyer sees only own orders
- **WHEN** user A has two orders and user B has one order
- **AND** user A requests the order list
- **THEN** the response contains only user A's two orders
- **AND** user B's order is absent

#### Scenario: Newest first with pagination
- **WHEN** a buyer has more orders than one page
- **AND** the buyer requests page 1 with size 10
- **THEN** the response returns at most 10 orders sorted by created_at descending
- **AND** pagination reports total count and page metadata

### Requirement: Optional status filter on order list
The list endpoint MUST accept an optional `status` query parameter matching order status values `0` (unpaid), `1` (paid), `2` (fulfilled), `3` (cancelled), `4` (closed). When omitted, the system MUST return all statuses for that buyer. An invalid status value MUST be rejected.

#### Scenario: Filter unpaid
- **WHEN** a buyer has unpaid and paid orders
- **AND** the buyer lists with `status=0`
- **THEN** only unpaid orders appear

#### Scenario: Invalid status rejected
- **WHEN** a buyer lists with `status=9`
- **THEN** the system rejects the request

### Requirement: List row carries enough fields for a card
Each list row MUST include at least: `order_id`, `order_no`, `store_id`, `status`, `amount`, `created_at`, and a non-empty `items` array of line snapshots (product title, cover URL when present, qty, line amount, kind). The system MUST NOT expose another buyer's data. List rows MAY omit full payment history; detail remains the source for payments.

#### Scenario: List row shows primary line for UI
- **WHEN** a buyer lists an order that has two line items
- **THEN** the list row includes both snapshot lines with titles and amounts
- **AND** the order-level amount equals the order total

### Requirement: Buyer reads own order detail
An authenticated buyer SHALL retrieve a single order by `order_id` only when they are the buyer. The detail MUST include the order header, all line snapshots, and payment rows when present. Another user MUST NOT read it.

#### Scenario: Own detail succeeds
- **WHEN** the buyer requests their order by order_id
- **THEN** the response includes order, items, and any payments

#### Scenario: Cross-user detail rejected
- **WHEN** user B requests user A's order_id
- **THEN** the system rejects the request

### Requirement: Unpaid order actions remain available
From list or detail context, an unpaid order MUST remain payable and cancellable via the existing pay and cancel endpoints. Paid, fulfilled, cancelled, or closed orders MUST NOT become unpaid again through these actions.

#### Scenario: Cancel unpaid from buyer flow
- **WHEN** the buyer cancels their unpaid order
- **THEN** the order status becomes cancelled
- **AND** stock and codes are unchanged

#### Scenario: Pay unpaid from buyer flow
- **WHEN** the buyer pays their unpaid order with a valid payment_channel
- **THEN** the order becomes paid
- **AND** a successful payment row exists
