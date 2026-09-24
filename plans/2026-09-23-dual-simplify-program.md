# Program — 双仓行为不变精简（Simplify × Ponytail）

- 日期：2026-09-23
- 对照：`docs/module-parity-go-flutter.md`
- 报告：`docs/simplify-reports/`
- Epic：`plans/epics/dual-simplify-parity.md`

## 里程碑

1. 对照表 Paired 行全部有报告 Done/Noop
2. R21/R22 共享层收口
3. iOS 烟测：登录 `13400000000` / `123456` 无异常

## 不做

- 改业务逻辑 / 契约 / 权限口径
- 补 Deferred/Gap 功能
- 大重构、新依赖、新抽象
- commit/push（等人指令）

## 队首

R01 Auth — Brief 预批（人一次确认连续执行）

## 验收闸门

- 机跑：Go 定向 `go test` + `go build ./cmd/api`；Flutter `dart analyze` 触及包
- 关键轮后 iOS 模拟器烟测登录
- 报告 §1/§2/§3 齐全
