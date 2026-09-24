# R07 — Home todo（Go × Flutter）

| 字段 | 值 |
|------|-----|
| Round | R07 |
| 对照 | [module-parity-go-flutter.md](../module-parity-go-flutter.md) |
| Ponytail | 已过滤 |
| 状态 | Noop（第二轮复查仍无安全项） |

## 1. 优化前问题（Before）

### 1.1 结构 / 对应
- Go：`home_todo_*`（controller / usecase / postgres + packing demo schema）
- Flutter：`home_todo_api` / `home_todo_packer` / `home_todo_pages`
- 配对完整；`ApplyToStore` 等为公开 HTTP 面。

### 1.2–1.4
- packing/demo 双 schema 为有意演示数据，非死代码。
- Flutter packer 有单测，逻辑与展示耦合紧，不宜在零业务变更下拆。

### 1.5 SKIP
- 删除 packing demo、合并 schema、改卡片聚合规则 — 会改行为或演示能力。

## 2. 优化目标（Goals）

- 本轮无目标（YAGNI：无可在零业务变更下落地的 diff）。

## 3. 优化后结果（After）

- 无代码变更。
- 第二轮深挖结论与首轮一致：StillNoop。

### 3.3 Ponytail 跳过项
| 项 | 阶梯 | 原因 |
|----|------|------|
| 删 packing demo | 1 YAGNI | 有意演示 |
| 抽 After-sales 式骨架基类 | 1 | 收益不足 |
| 改 todo-cards 聚合 | skip | 业务逻辑 |
