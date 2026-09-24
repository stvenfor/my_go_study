# R22 — Queue / Worker（Go × Flutter）

| 字段 | 值 |
|------|-----|
| Round | R22 |
| 对照 | [module-parity-go-flutter.md](../module-parity-go-flutter.md) |
| Ponytail | 已过滤 |
| 状态 | Done |

## 1. Before：Client/Scheduler 内联 Redis opt，已有 `RedisClientOpt`。
## 2. Goals：G1 复用 helper。
## 3. After：`asynq_client.go`、`scheduler.go`；`go test ./pkg/queue` ok。
### 3.3 Skipped：sms/jpush stub handlers（有意占位）

