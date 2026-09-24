# CROSS — Flutter API _guarded 统一

| 字段 | 值 |
|------|-----|
| ID | CROSS |
| Program | Round-2 deep simplify |
| 状态 | Done |
| 硬约束 | 业务/页面语义不变 |

## 1. Before
多 feature *api.dart 重复 ensureInitialized + try/catch。
## 2. Goals
各文件内 _guarded/_data（错误文案表不动）。
## 3. After
home/mall/wallet/settings/deal_invoice/address/points/todo/new_car/after_sales/membership 等；dart analyze 无 error。~−200 净行。

