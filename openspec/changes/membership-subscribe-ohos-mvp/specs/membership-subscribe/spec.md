## Purpose

Lets users open or renew SVIP / AI SVIP membership with 1-, 6-, or 12-month plans, using prepaid buyout channels or Huawei auto-renewable IAP, with server-owned entitlement as the source of truth.

## ADDED Requirements

### Requirement: Dual-tier catalog with three durations

The system SHALL expose membership tiers `svip` and `ai_svip`, each with plans for 1 month, 6 months, and 12 months. Plan identifiers and display titles MUST be stable for clients. Catalog responses MUST include price (fen or decimal yuan consistently documented) and whether the plan is buyout-eligible and/or mapped to a Huawei auto-renew `productId`.

#### Scenario: Catalog lists six buyout plans

- **WHEN** an authenticated client requests membership catalog
- **THEN** the response includes both tiers and three durations per tier with prices

#### Scenario: Dual tabs unchanged on client

- **WHEN** the user opens the membership renew page
- **THEN** SVIP and AI SVIP tabs remain available and selecting a tab shows that tier’s 1/6/12 plans

### Requirement: Entitlement status for open vs renew

The system SHALL persist per-user per-tier entitlement with an `expires_at` timestamp. Status is active when `expires_at` is strictly after server now; otherwise inactive/expired. Clients MUST use this status to label the primary CTA as open or renew without inventing local entitlement truth.

#### Scenario: Expired user sees open

- **WHEN** the user has no active entitlement for the selected tier
- **THEN** membership status for that tier is inactive and the CTA semantics are「开通」

#### Scenario: Active user sees renew

- **WHEN** the user has `expires_at` in the future for the selected tier
- **THEN** membership status for that tier is active and the CTA semantics are「续费」

### Requirement: Prepaid buyout via WeChat, Alipay, or balance

For buyout channels, a successful payment SHALL set `expires_at` to `max(server_now, current_expires_at) + plan_duration`. Balance channel MUST debit the cash wallet atomically with entitlement write and MUST reject when balance is insufficient. WeChat and Alipay MUST use existing prepay issuance; entitlement MUST only extend after a confirmed successful payment bound to the authenticated user and matching plan amount.

#### Scenario: Balance buyout extends expiry

- **WHEN** an authenticated user with sufficient balance buys `svip` 1-month via balance
- **THEN** wallet balance decreases by the plan price and `svip` `expires_at` increases by one month from the max of now and previous expiry

#### Scenario: Insufficient balance rejected

- **WHEN** the user selects balance buyout without enough funds
- **THEN** the system rejects the purchase and does not change entitlement

#### Scenario: WeChat or Alipay confirm after client success

- **WHEN** the client completes WeChat or Alipay payment for a buyout plan and calls confirm with the bound trade reference
- **THEN** the system extends entitlement for that tier and plan duration once (idempotent on repeat confirm)

### Requirement: Huawei auto-renewable subscription verify

For Huawei IAP, the client SHALL purchase an auto-renewable product mapped to a tier+duration plan. The server MUST verify purchase authenticity (JWS and/or Huawei subscription query with server JWT), upsert entitlement from verified subscription state, and acknowledge delivery so the client can finish the purchase. The server MUST NOT trust client-only claims without verification.

#### Scenario: First subscribe grants entitlement

- **WHEN** the client submits a verified Huawei subscription purchase for a mapped `productId`
- **THEN** the server marks delivery acknowledged and the user’s tier entitlement is active according to Huawei subscription validity

#### Scenario: Invalid token rejected

- **WHEN** the client submits a forged or mismatched purchase payload
- **THEN** the server rejects the request and entitlement is unchanged

#### Scenario: Restore unfinished purchases

- **WHEN** the OHOS app starts and finds an unfinished Huawei subscription purchase
- **THEN** the client re-submits verification and finishes purchase only after server success

### Requirement: HarmonyOS channel matrix

On HarmonyOS builds, the membership payment UI MUST offer WeChat, Alipay, balance, and Huawei IAP. Selecting Huawei IAP MUST use auto-renew semantics and copy; selecting the other three MUST use buyout semantics and copy.

#### Scenario: Four methods visible on OHOS

- **WHEN** the membership page runs on an OHOS build
- **THEN** the payment method list includes wechat, alipay, balance, and huawei

### Requirement: Independent tiers

SVIP and AI SVIP entitlements SHALL be independent. Purchasing or renewing one tier MUST NOT clear or require inactivity of the other.

#### Scenario: Both tiers active

- **WHEN** the user buys out SVIP and separately buys out AI SVIP
- **THEN** both entitlements can be active concurrently with their own `expires_at`
