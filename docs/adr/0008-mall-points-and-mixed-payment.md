# Mall SKUs support CNY, points, and fixed mixed payment

Existing mall checkout is RMB-only (channels 1–4). Product decision: keep one mall catalog; each SKU carries `price_cny` and `price_points` (either may be zero). Modes are derived: CNY-only, points-only, or mixed. Mixed means the buyer must pay the full CNY price and the full points price together—no sliding points-as-discount. Points are a first-class payment channel with their own payment row; mixed orders need both a points payment and a CNY-channel payment to succeed.

Rejected for this phase: a separate points-only catalog (would fork browse/cart/order UX), and “points offset N yuan” (harder ledger + partial-pay edge cases before the points wallet exists).

Points already deducted are fully restored when payment fails or the buyer cancels before fulfillment; CNY follows the existing refund path. Points are deducted at the pay action (not at order create, not deferred until CNY webhook).

One order may only contain SKUs that share the same payment mode (all CNY, all points, or all mixed). Reject mixed-mode carts at order submit rather than inventing cross-mode partial settlement.
