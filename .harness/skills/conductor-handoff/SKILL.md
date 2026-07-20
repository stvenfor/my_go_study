# Skill — conductor-handoff

## When

新任务未指定角色；或多 Agent 交接 / 并行裁决。角色合同：`.harness/agents/conductor.md`。

## Do

1. 读 `plans/README` 队首（若有）+ `.harness/changes/current.md`
2. 按 conductor 路由表决定下一角色
3. 输出 Handoff（模板：`.harness/changes/handoff-template.md`）
4. 更新 `changes/current.md` 的 Role / Next / Notes

## Don't

- 不写 `internal/*` / `cmd/*` 业务
- 不批 Brief、不 commit、不勾验收
- 不代替 executor 跑 `agent-pre`/`post`

## Out

Handoff 文本 + 更新后的 `current.md`；对话里明确「请打开角色合同：`.harness/agents/<role>.md`」。
