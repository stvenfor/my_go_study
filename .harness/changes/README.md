# changes/ — 本轮活动指针

**不是** Slice 合同真相源。合同仍在：

`plans/slices/<id>.md`（若本仓尚未建 `plans/`，先由 planner 落盘再建指针）

本目录记录「现在跑哪一轮」，避免多 Agent 抢错上下文。

| 文件 | 用途 | 主写 |
|------|------|------|
| [current.md](./current.md) | active slice + Role + harness 摘要 + Next | **conductor**（Role/Next/Notes）；executor/planner 可补 pre/post |
| [handoff-template.md](./handoff-template.md) | conductor Handoff 填空模板 | conductor |

新任务 → 先开 `.harness/agents/conductor.md`。
