# Acceptance — Cash Wallet C1–C3

**Date:** 2026-09-23  
**Spec:** `.scratch/cash-wallet/spec.md`

## Paths checked (code / unit)

1. **充值**：`CashWalletUsecase` 边界 + 成功入账流水（`go test ./internal/usecase/ -run CashWallet`）
2. **余额支付**：渠道 6 扣分账、不足拒绝、本地支付失败回滚（`go test ./internal/usecase/ -run MallBalance`）
3. **取消退回钱包**：已支付取消后余额恢复（同 MallBalance 测试）
4. **Flutter**：`module_wallet` 入口；商城详情渠道含余额（静态接线，需真机/LAN 联调目视）

## Manual LAN（可选）

```bash
# API 已 make run / lan-up 且 SessionAuth 登录后
curl -s "$HOST/api/v1/wallet/recharge" -H "..." -d '{"amount":"10.00","channel":1}'
curl -s "$HOST/api/v1/mall/orders/$OID/pay" -H "..." -d '{"payment_channel":6}'
curl -s "$HOST/api/v1/mall/orders/$OID/cancel" -H "..." -X POST
curl -s "$HOST/api/v1/wallet"
```

期望：充值后余额增加 → 余额支付成功余额减少 → 取消后余额回到充值后水平（人民币部分）。
