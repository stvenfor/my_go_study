# R03 — Transactions（Go × Flutter）

| 字段 | 值 |
|------|-----|
| Round | R03 |
| 对照 | [module-parity-go-flutter.md](../module-parity-go-flutter.md) |
| Ponytail | 已过滤 |
| 状态 | Done |

## 1. 优化前问题（Before）
- 死代码：未挂载 `transaction_handler` + `jwtAuthContext`（handler/controller）
- 死代码：`ToTransaction`/`formatUintID`；Flutter `transaction_mock_data`、未用 `sourceLabel`
## 2. 优化目标（Goals）
- G1：删除上述死代码，保留 `TransactionRecord` 供 AutoMigrate
## 3. 优化后结果（After）
| Goal | 改动 | 不变 | 验证 |
|------|------|------|------|
| G1 | 删 handler/auth_context×2、mock；精简 entity/repository getter | 是 | go build + entity/usecase tests |
### 3.3 Skipped：现行 TransactionController 路径不动
### 3.4 无

