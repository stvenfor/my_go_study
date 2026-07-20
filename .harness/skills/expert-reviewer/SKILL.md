# Skill — expert-reviewer

## When

executor 已验证绿；人或第二会话要求审查。

## Do

按 `.harness/agents/reviewer.md`：

- 对照 Brief DoD + 验收行 + diff + harness-report / 测试输出
- 查边界 / 状态诚实 / 白名单越界 / 密钥
- 输出 `Approved | Rework | Blocked` + P0/P1 + **建议交回 conductor 再派**（executor | planner | 人）

## Don't

- 直接改业务代码
- 在验证未绿时 Approved
- 范围需重切时自己扩 Brief（交回 **conductor → planner**）

## Refs

- `.harness/agents/reviewer.md`
- `/AGENTS.md`
- `.harness/rules/`
