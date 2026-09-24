# Program — 双仓深度精简 Round-2

- 日期：2026-09-23
- 前置：R1 `docs/simplify-reports/`
- Epic：`plans/epics/dual-deep-simplify-r2.md`
- 报告：`docs/simplify-reports-r2/`

## 放松（相对 R1）

1. 结构债 2. 可测性 3. 性能 4. 重构（共享守卫/错误映射/胶水）

## 硬约束

- 对外 JSON/proto 字段、HTTP 码/message、权限/SQL/状态机语义不变
- Flutter 路由与产品页布局结构不变
- 中间不问；commit 等人；Ponytail（新抽象须 ≥2 调用点）

## 角色

conductor=父 · planner=父 · executor=subagent/父 · reviewer=父

## 队首

跨切面（Flutter API 守卫 / Go controller 胶水）→ P2-R01…R22
