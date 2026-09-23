# Membership subscribe OHOS + Apple IAP（2026-09-23）

## Scope

- Go BFF：`/api/v1/membership/*`（me / buyout / confirm / huawei verify / **apple verify**）
- Flutter：原会员页双 Tab；套餐 1/6/12
  - 鸿蒙：微信/支付宝/余额/华为
  - **iOS：微信/支付宝/余额/苹果**
- 原生：华为 OHOS 插件 + **iOS `AppleIapPlugin`（AppDelegate 内 debug 伪凭证）**；真 StoreKit / IAP Kit 待换

## Local checks

```bash
go test ./internal/usecase/ -run TestMembership -count=1
# 苹果验单（dev）
curl -s "$HOST/api/v1/membership/apple/verify" -H "..." \
  -d '{"product_id":"wys_svip_1m","transaction_id":"dev-1"}'
```

## Deferred

- App Store Connect / AGC 建自动续期商品
- StoreKit 2 / IAP Kit 真收银台
- Server Notifications V2（苹果）与华为关键事件完整处理
