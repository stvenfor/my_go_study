# 会员订阅 API

SessionAuth。本地 `auth.provider=local`。套餐为 SVIP / AI SVIP × 1 月 / 6 月 / 1 年。

语义（方案 C）：

| 渠道 | 含义 |
|------|------|
| `wechat` / `alipay` / `balance` | 时长买断：`expires_at = max(now, expires_at) + months` |
| `huawei` / `apple` | 商店自动续期订阅；客户端 IAP 后调 verify |

## 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/membership/me` | 两档权益 + 目录 |
| POST | `/api/v1/membership/buyout` | `{ plan_id, channel }` |
| POST | `/api/v1/membership/buyout/confirm` | `{ order_id }` 微信/支付宝付成功后确认 |
| POST | `/api/v1/membership/huawei/verify` | 华为订阅验单 |
| POST | `/api/v1/membership/apple/verify` | 苹果订阅验单 |
| POST | `/api/v1/membership/huawei/notifications` | 关键事件 stub |

### buyout

- `channel=balance`：同步扣钱包并开通；余额不足拒绝。
- `channel=wechat|alipay`：创建待支付单并返回 `prepay_params`（同 `/payments/prepay`）；客户端 SDK 成功后调 `buyout/confirm`。

### huawei/verify

```json
{
  "product_id": "wys_svip_1m",
  "purchase_token": "...",
  "purchase_order_id": "",
  "subscription_id": "",
  "jws_purchase_order": ""
}
```

`product_id` 须映射到目录中的 `huawei_product_id`。未配置 `HUAWEI_IAP_*` 且非 release 时走本地信任校验（仅测联调）。配置齐全时调华为订阅查询 API。

### apple/verify

```json
{
  "product_id": "wys_svip_1m",
  "transaction_id": "...",
  "original_transaction_id": "",
  "receipt_data": "<base64 App Receipt>"
}
```

`product_id` 须映射 `apple_product_id`。配置 `APPLE_IAP_SHARED_SECRET` 后走 `verifyReceipt`（生产失败 21007 自动回退 sandbox）；dev 未配置时可仅校验 product 映射 + transaction_id。

## 目录 plan_id（占位价）

| plan_id | 档 | 时长 | 分 |
|---------|----|------|----|
| svip_1m / svip_6m / svip_12m | svip | 1/6/12 | 3000/15000/28000 |
| ai_svip_1m / ai_svip_6m / ai_svip_12m | ai_svip | 1/6/12 | 4800/24000/39800 |

华为商品 ID：`wys_svip_1m` … `wys_ai_svip_12m`（须在 AGC 建同名自动续期商品）。
苹果商品 ID：同上 `apple_product_id`（须在 App Store Connect 建自动续期订阅）。

## Flutter

改 `my_ai_project` `features/pay` 会员页：双 Tab 保留；套餐 1/6/12；
鸿蒙四渠道（含华为）；**iOS 四渠道（含苹果内购）**。
