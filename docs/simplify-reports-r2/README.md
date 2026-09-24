# Simplify Round-2 报告索引

Program：`plans/2026-09-23-dual-deep-simplify-r2-program.md`

四条放松：结构债 / 可测性 / 性能 / 重构；**业务与页面语义不变**。

| ID | 报告 | 状态 |
|----|------|------|
| CROSS | [CROSS-go-store-scope.md](./CROSS-go-store-scope.md) | Done |
| CROSS | [CROSS-go-resource-error.md](./CROSS-go-resource-error.md) | Done |
| CROSS | [CROSS-go-topics-batch.md](./CROSS-go-topics-batch.md) | Done |
| CROSS | [CROSS-flutter-api-guard.md](./CROSS-flutter-api-guard.md) | Done |
| P2-R01 | [P2-R01-auth.md](./P2-R01-auth.md) | Done |
| P2-R02 | [P2-R02-profile.md](./P2-R02-profile.md) | Done |
| P2-R03 | [P2-R03-transactions.md](./P2-R03-transactions.md) | Done |
| P2-R04 | [P2-R04-realtime.md](./P2-R04-realtime.md) | Done |
| P2-R05 | [P2-R05-analytics.md](./P2-R05-analytics.md) | Done |
| P2-R06 | [P2-R06-used-car.md](./P2-R06-used-car.md) | Done |
| P2-R07 | [P2-R07-home-todo.md](./P2-R07-home-todo.md) | Done |
| P2-R08 | [P2-R08-new-car-follow.md](./P2-R08-new-car-follow.md) | Done |
| P2-R09 | [P2-R09-after-sales.md](./P2-R09-after-sales.md) | Done |
| P2-R10 | [P2-R10-deal-invoice.md](./P2-R10-deal-invoice.md) | Done |
| P2-R11 | [P2-R11-address.md](./P2-R11-address.md) | Done |
| P2-R12 | [P2-R12-community.md](./P2-R12-community.md) | Done |
| P2-R13 | [P2-R13-short-video.md](./P2-R13-short-video.md) | Done |
| P2-R14 | [P2-R14-mall.md](./P2-R14-mall.md) | Done |
| P2-R15 | [P2-R15-points.md](./P2-R15-points.md) | Done |
| P2-R16 | [P2-R16-wallet.md](./P2-R16-wallet.md) | Done |
| P2-R17 | [P2-R17-membership.md](./P2-R17-membership.md) | Done |
| P2-R18 | [P2-R18-purchase-calc.md](./P2-R18-purchase-calc.md) | Done |
| P2-R19 | [P2-R19-jpush.md](./P2-R19-jpush.md) | Done |
| P2-R20 | [P2-R20-sse-ai.md](./P2-R20-sse-ai.md) | Done |
| P2-R21 | [P2-R21-http-di.md](./P2-R21-http-di.md) | Done |
| P2-R22 | [P2-R22-queue.md](./P2-R22-queue.md) | Done |

## 验证

- Go：`go build ./cmd/api`；`go test ./internal/usecase ./internal/delivery/http/controller` 绿
- Flutter：触及 API `dart analyze` 无 error
- OTP 烟测：`13400000000` / `123456` → success + token

## 量级摘要

- Go：门店作用域共享、CRUD 错误映射共享、Topics 批量查询、R1 死代码清理叠加
- Flutter：十余个 *api.dart `_guarded` 统一、HttpConfig 对齐、脚手架/mock 删除（约 −200 净行）

**P2 Program 结束。** commit/push 等人指令。
