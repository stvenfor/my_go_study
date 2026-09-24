# R21 — HTTP/DI/router 共享层（Go × Flutter）

| 字段 | 值 |
|------|-----|
| Round | R21 |
| 对照 | [module-parity-go-flutter.md](../module-parity-go-flutter.md) |
| Ponytail | 已过滤 |
| 状态 | Done |

## 1. Before：`reinitialize` 与 `initialize` 重复；Auth 首启应走 initialize。
## 2. Goals：G1 reinitialize→委托 initialize；G2 AuthHttpConfig 首启 initialize。
## 3. After：`app_http_bootstrap.dart`、`auth_http_config.dart`。dart analyze 无 error。
### 3.3 Skipped：router 条件注册块、Deferred manifest

