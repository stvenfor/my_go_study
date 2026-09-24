# R17 — Membership + prepay（Go × Flutter）

| 字段 | 值 |
|------|-----|
| Round | R17 |
| 对照 | [module-parity-go-flutter.md](../module-parity-go-flutter.md) |
| Ponytail | 已过滤 |
| 状态 | Done |

## 1. Before：`module_pay` 脚手架 Calculator；Prepay 取 user 后丢弃。
## 2. Goals：G1 清空脚手架；G2 Prepay 仅鉴权不绑定未用变量。
## 3. After：`module_pay.dart` library only；`payment_controller.go` `_, _, ok`。

