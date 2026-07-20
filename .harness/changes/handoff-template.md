# Handoff template（conductor）

conductor 每派一次角色，复制填空；并更新 [current.md](./current.md)。

```text
## Handoff
Task:
Decision: open planner | executor | reviewer | human
Slice: plans/slices/<id>.md | —
Why: （路由表：无 Brief / 已批待实现 / 待审 / Rework / Blocked / Approved）
Skill chain:
  - planner: request-analysis
  - executor: coding-skill → unit-test-* →（可选）api-smoke-verify
  - reviewer: expert-reviewer
Parallel: no | yes
  - file lock / single writer for 验收清单:
Blockers:
Do not: 不写业务；不批 Brief；不 commit；不假 Accept
Next human gate: 批 Brief | commit/push | 无
```

## current.md 字段约定（conductor 主写）

| 字段 | 含义 |
|------|------|
| Active slice | 当前合同路径或 — |
| Role | 正在值班的角色：`conductor` 调度后写成下一角色名 |
| Next | 下一动作一句话 |
| Notes | 并行锁、Blocked 原因、Handoff 摘要 |
