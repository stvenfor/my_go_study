# R20 — SSE / AI（Go × Flutter）

| 字段 | 值 |
|------|-----|
| Round | R20 |
| 状态 | Done（第二轮） |
| Ponytail | 已过滤 |

## 1. Before
- `AiChatMessage.copyWith` 从未使用（controller 直接改可变字段）。

## 2. Goals
- G1：删除未用 `copyWith`。

## 3. After
| Goal | 改动 | 不变 | 验证 |
|------|------|------|------|
| G1 | `features/ai/.../ai_chat_message.dart` | 是 | dart analyze |

### 3.3 Skipped：SSE mock provider（配置回退）
