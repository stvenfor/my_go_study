# R04 — Realtime + WS（Go × Flutter）

| 字段 | 值 |
|------|-----|
| Round | R04 |
| 对照 | [module-parity-go-flutter.md](../module-parity-go-flutter.md) |
| Ponytail | 已过滤 |
| 状态 | Done |

## 1. Before
- `Handler.Hub()` 无调用方；Flutter `RealtimeTelemetryReporter` 仅一处使用可内联
## 2. Goals
- G1 删 Hub 访问器；G2 内联 telemetry report
## 3. After
| Goal | 改动 | 不变 | 验证 |
|------|------|------|------|
| G1 | `delivery/ws/handler.go` | 是 | go test ./internal/delivery/ws |
| G2 | `realtime_telemetry.dart`；删 reporter 文件 | 是 | dart analyze telemetry |
### 3.4 无

