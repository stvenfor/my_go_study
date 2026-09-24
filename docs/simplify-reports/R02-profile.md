# R02 — Profile + 门店/统计（Go × Flutter）

| 字段 | 值 |
|------|-----|
| Round | R02 |
| 对照 | [module-parity-go-flutter.md](../module-parity-go-flutter.md) |
| Ponytail | 已过滤 |
| 状态 | Done |

## 1. 优化前问题（Before）
### 1.1–1.4
- Flutter `UserProfileApi` 四处重复 try/catch，与 R01 `UserAuthApi` 已统一模式不一致。
### 1.5 SKIP
- Profile 契约字段、权限口径不动。

## 2. 优化目标（Goals）
- G1：对齐 `_guarded` / `_requireData`，映射语义不变。

## 3. 优化后结果（After）
### 3.1
| Goal | 改动 | 不变 | 验证 |
|------|------|------|------|
| G1 | `features/auth/.../user_profile_api.dart` | 是 | dart analyze（无 error） |
### 3.2 约 1 文件
### 3.3 Skipped：无
### 3.4 无

