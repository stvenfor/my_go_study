# R05 — Analytics gRPC（Go × Flutter）

| 字段 | 值 |
|------|-----|
| Round | R05 |
| 对照 | [module-parity-go-flutter.md](../module-parity-go-flutter.md) |
| Ponytail | 已过滤 |
| 状态 | Done（第二轮） |

## 1. 优化前问题（Before）

### 1.1 结构
- Go gRPC Analytics 与 Flutter AnalyticsGrpcApi 配对完整；服务端 page clamp 触及契约，不动。

### 1.2 Quality
- Flutter `AnalyticsListBinding` / `AnalyticsDetailBinding` 重复注册 `AnalyticsGrpcApi` + `AnalyticsRepository`。

### 1.5 SKIP
- gRPC 分页 clamp 双处（契约敏感）。

## 2. 优化目标（Goals）

- G1：同文件抽 `_ensureAnalyticsDeps()`，注册语义不变。

## 3. 优化后结果（After）

| Goal | 改动 | 不变 | 验证 |
|------|------|------|------|
| G1 | `features/home/.../analytics_binding.dart` | 是 | dart analyze 无 error |

### 3.3 Skipped
| 项 | 原因 |
|----|------|
| page clamp 合并 | 契约风险 |
