# 钱包与余额支付 API

SessionAuth（`Authorization` + `X-Session-ID` + `X-Device-ID`）。本地模拟，无真实 PSP。

用语见 `CONTEXT.md` §钱包（现金）。人民币退回钱包见 [ADR 0015](./adr/0015-mall-cny-refund-to-wallet.md)。

## 钱包

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/wallet` | 余额 + 银行卡列表 |
| GET | `/api/v1/wallet/ledger?limit&offset` | 流水（倒序） |
| GET | `/api/v1/wallet/cards` | 卡列表 `{ items }` |
| POST | `/api/v1/wallet/cards` | 绑卡 `{ bank_name, card_last4, holder_name?, is_default? }` |
| POST | `/api/v1/wallet/cards/:card_id/default` | 设默认卡 |
| DELETE | `/api/v1/wallet/cards/:card_id` | 删卡 |
| POST | `/api/v1/wallet/recharge` | 充值 |

### 充值

```json
{ "amount": "10.50", "channel": 1, "card_id": null }
```

| channel | 含义 |
|---------|------|
| 1 | 支付宝（模拟） |
| 2 | 微信（模拟） |
| 3 | 银行卡（须传 `card_id`） |

金额：0.01–50000 元。成功后 `balance` / `balance_fen` 更新，流水 `reason=recharge`。

余额库内以 **分**（`balance_fen`）记账；JSON 另给两位小数 `balance`。

## 商城余额支付

`POST /api/v1/mall/orders/:order_id/pay`

```json
{ "payment_channel": 6 }
```

| channel | 含义 |
|---------|------|
| 1–4 | 既有人民币模拟渠道 |
| 5 | 积分 |
| **6** | **余额** |

规则：

- 人民币订单 / 混合单人民币腿可用 `6`；余额不足整笔拒绝。
- 纯积分单不会用余额付积分价（服务端把渠道落到积分）。
- 扣款发生在 pay；`PayOrderLocal` 失败则退回已扣余额/积分。
- 已支付未履约订单取消：人民币全额 **退回钱包**（`reason=mall_refund`），积分退回积分账。

## Flutter

- 「我的」→「我的钱包」→ `features/wallet`
- 商城订单详情待支付：可选支付宝 / 微信 / 余额
