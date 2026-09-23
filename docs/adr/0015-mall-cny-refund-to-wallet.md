# Mall CNY refunds credit the cash wallet

When a mall order has already taken CNY (Alipay/WeChat/IAP/balance) and payment fails or the buyer cancels, that CNY is credited to the account **cash wallet**, not returned to the original PSP channel. Local-sim has no real PSP reverse; routing refunds into the wallet keeps one clear ledger and matches how points already restore on cancel. Rejected for this phase: original-channel refunds and withdraw-to-card.
