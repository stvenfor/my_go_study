# agents — 角色合同索引

**新任务默认先找 conductor**，再进入 planner / executor / reviewer。  
**人**批准 Brief 与 commit/push。

| 角色 | 文件 | 可写代码？ | 主产出 |
|------|------|------------|--------|
| **conductor** 统筹 | [conductor.md](./conductor.md) | 否（仅 `changes/` 指针与 Handoff） | 路由决策、Handoff、`current.md` |
| **planner** 架构师 | [planner.md](./planner.md) | 仅 plans / 架构文档 | Program·Epic·Slice Brief、差距表、架构裁决 |
| **executor** 执行 | [executor.md](./executor.md) | 白名单内 | diff + post 绿 + Context Card |
| **reviewer** 审查 | [reviewer.md](./reviewer.md) | 否 | Approved / Rework / Blocked |

```text
新任务 → conductor（路由 + current.md）
  → planner（若需 Brief）→ 人批
  → executor + agent-pre → 实现 → agent-post
  → reviewer
  → 人 commit/push
  （Rework→conductor 再派 executor；越界→conductor 派 planner）
```

编排总览：[../README.md](../README.md)  
Handoff 模板：[../changes/handoff-template.md](../changes/handoff-template.md)
