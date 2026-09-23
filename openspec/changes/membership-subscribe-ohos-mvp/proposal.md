## Why

Flutter `features/pay` 会员续费页已有 SVIP / AI SVIP 双 Tab 与微信/支付宝预支付，但套餐仍是 Mock，支付成功不落会员权益；鸿蒙端还缺余额支付与华为 IAP 订阅。需要把「开通/续费」做成可联调闭环：固定 1 月 / 6 月 / 1 年套餐，四渠道在鸿蒙可用，权益以服务端为准。

## What Changes

- **保留**现有会员续费页与 SVIP / AI SVIP 双 Tab；套餐改为每档 **1 个月 / 6 个月 / 1 年**（开通与续费同一页，按当前权益状态切换文案）
- 鸿蒙端支付渠道：**微信、支付宝、余额、华为内购**（语义分叉见 design：C）
  - 微信 / 支付宝 / 余额：一次性买断时长（到期日 +N）
  - 华为内购：AGC **自动续期订阅**（`AUTORENEWABLE`），续扣与到期由华为侧驱动
- Go BFF：会员权益真相源（查状态、买断开通/续期、余额扣款、华为订阅验单/确权/补单钩子）；支付成功后必须写权益，不再只 toast
- Flutter（`my_ai_project`）：改会员页套餐与渠道；鸿蒙上接 IAP Kit（或 MethodChannel）；余额走现有钱包；非鸿蒙可先保持微信/支付宝（余额可选）
- 不做：商城 `payment_channel=4` 真验签、苹果 IAP、会员权益细粒度功能开关、趣豆 Mock 抵扣保留为视觉可删/可关

## Capabilities

### New Capabilities

- `membership-subscribe`: 会员档位与 1/6/12 月套餐、开通/续费、买断渠道与华为自动续期订阅、权益查询与幂等确权

### Modified Capabilities

- （无既有 main specs）

## Impact

- **Go BFF**：新 `wys_membership_*`（或等价）表 + usecase/controller/router；复用 `PaymentUsecase`（微信/支付宝）、`CashWalletUsecase`（余额）；新增华为 IAP 订阅 JWT/JWS/查询/确认发货客户端；`.env` 增 AGC 密钥项
- **Flutter**（`my_ai_project`）：`module_pay` membership 套餐与 `PaymentMethodType`；gateway 扩展余额与华为；OHOS 原生桥；入口仍走现有会员路由
- **AGC**：配置 SVIP/AI SVIP × 1/6/12 自动续期商品与沙盒账号
- **非目标**：改商城本地模拟支付语义；拆掉双 Tab；上架合规最终裁剪（调试阶段四渠道全开）
