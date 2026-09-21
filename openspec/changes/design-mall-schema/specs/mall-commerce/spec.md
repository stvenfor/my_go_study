## Purpose

门店商城共用一套商品目录，用种类区分实体与虚拟履约。支付渠道用统一枚举；任意渠道在本地点支付均可成功落库，不调用渠道 SDK。

## ADDED Requirements

### Requirement: Shared catalog for physical and virtual goods
When `auth.provider=local`, the system SHALL store mall categories, products, and SKUs in store-scoped tables. A product MUST be either physical or virtual, and every SKU under that product MUST share that kind. Price MUST be `numeric(10,2)` stored without floating-point types in the API layer. Physical SKUs MUST carry stock on the SKU row. Virtual SKUs MUST use redeem-code or content-url delivery. Specs MUST be stored as JSONB on the SKU.

#### Scenario: Create a physical product with stock on SKU
- **WHEN** a store staff member creates a physical product and a SKU with stock quantity 10 and price 19.90
- **THEN** the system stores one product and one SKU of kind physical
- **AND** the SKU stock is 10

#### Scenario: Create a virtual redeem-code SKU
- **WHEN** a store staff member creates a virtual product with deliver type redeem-code and seeds unused codes
- **THEN** the system stores the SKU without sellable stock_qty
- **AND** unused codes exist for that SKU

### Requirement: Store staff write their own catalog
A caller with permission `mall.catalog.write` for a store SHALL create and update that store's categories, products, SKUs, stock, and redeem codes. The system MUST reject catalog writes for a store lacking that permission. The system MUST NOT add a new role.

#### Scenario: Staff writes own store
- **WHEN** a `store_staff` member of store A with `mall.catalog.write` creates a product for store A
- **THEN** the system persists the product on store A

#### Scenario: Staff cannot write another store
- **WHEN** a staff member of store A creates a product for store B
- **THEN** the system rejects the request

### Requirement: Buyers see on-shelf SKUs only
An authenticated buyer SHALL list only on-shelf products and SKUs for the requested store. Draft and off-shelf products MUST NOT appear.

#### Scenario: On-shelf product is listed
- **WHEN** an authenticated buyer lists products for a store that has one on-shelf SKU
- **THEN** the response includes that product and SKU with kind, specs, and price

### Requirement: Cart holds SKU quantities for the buyer
The system SHALL store one cart row per user and SKU with positive quantity. The cart MUST NOT be readable by another user.

#### Scenario: Add then update quantity
- **WHEN** a buyer adds a SKU with quantity 1 and later sets quantity 3 for the same SKU
- **THEN** the buyer has one cart line with quantity 3

### Requirement: Order snapshots catalog fields and receiver
Creating an order MUST snapshot per line: kind, product title, cover URL, specs, unit price, quantity. The order MUST snapshot receiver name, phone, and address when any line is physical. A new order MUST start unpaid with null `payment_channel`. Creating an order MUST NOT decrement stock or issue codes. The client MUST supply an idempotency key; repeating the same key for the same buyer MUST return the existing order without creating another.

#### Scenario: Physical order requires address
- **WHEN** a buyer checks out a physical SKU without receiver fields
- **THEN** the system rejects the request

#### Scenario: Idempotent create
- **WHEN** a buyer creates an order twice with the same idempotency key
- **THEN** only one order row exists
- **AND** both responses refer to that order

### Requirement: Local mock pay succeeds for any channel
Marking an unpaid order paid SHALL accept `payment_channel` values 1 (Alipay), 2 (WeChat), 3 (Apple IAP), or 4 (Huawei IAP) and MUST succeed without calling an external payment provider. In one transaction the system MUST insert a successful payment row, set the order to paid with that channel, decrement physical stock and/or assign virtual codes or content URLs, append order_log and audit_log. A second pay for an already-paid order MUST return success without a second successful payment or a second stock decrement. Invalid channel values MUST be rejected. Insufficient stock or codes MUST roll back the whole transaction and leave the order unpaid.

#### Scenario: Pay with WeChat writes success locally
- **WHEN** the buyer pays an unpaid order with payment_channel 2
- **THEN** a payment row exists with channel 2 and status success
- **AND** the order status is paid with payment_channel 2
- **AND** no external HTTP call to WeChat is made

#### Scenario: Any of the four channels succeeds the same way
- **WHEN** the buyer pays unpaid orders with channels 1, 2, 3, and 4 respectively
- **THEN** each order becomes paid with that channel written on both order and payment

#### Scenario: Repeat pay is idempotent
- **WHEN** the buyer pays an already-paid order again with any valid channel
- **THEN** the response succeeds
- **AND** stock is unchanged from after the first pay
- **AND** only one successful payment row exists for that order

#### Scenario: Invalid channel rejected
- **WHEN** the buyer pays with payment_channel 9
- **THEN** the system rejects the request
- **AND** the order stays unpaid

### Requirement: Cancel unpaid only
Cancelling an unpaid order MUST set status cancelled and MUST NOT change stock or codes. Paid orders MUST NOT be cancelled by this capability.

#### Scenario: Cancel unpaid
- **WHEN** the buyer cancels their unpaid order
- **THEN** the order status is cancelled
- **AND** stock is unchanged

### Requirement: Order reads scoped to buyer
Reading an order SHALL require the authenticated matching `user_id` to equal the order's buyer. Another user MUST NOT read it.

#### Scenario: Cross-user read rejected
- **WHEN** user B requests user A's order
- **THEN** the system rejects the request

### Requirement: Mall routes follow local user_id rule
Mall routes SHALL require SessionAuth and a matching `user_id`. The system MUST NOT register mall routes when `auth.provider` is not `local`.

#### Scenario: Supabase mode has no mall routes
- **WHEN** `auth.provider=supabase`
- **THEN** mall routes are not registered
