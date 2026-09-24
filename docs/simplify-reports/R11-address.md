# R11 — Address（Go × Flutter）

| 字段 | 值 |
|------|-----|
| Round | R11 |
| 状态 | Done（第二轮） |
| Ponytail | 已过滤 |

## 1. Before
- Flutter `AddressModel.toBody` 零调用（编辑页手写 body）；Go `GET /:id` 为公开面保留。

## 2. Goals
- G1：删除未使用的 `toBody`。

## 3. After
| Goal | 改动 | 不变 | 验证 |
|------|------|------|------|
| G1 | `address_model.dart` 删 `toBody` | 是 | dart analyze |

### 3.3 Skipped：Go GET by id（公开 API）
