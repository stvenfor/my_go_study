# Skill — request-analysis

## When

**planner** 主用（常由 **conductor** 派来）；开 Slice 前对齐 Program 队首、写/核 Brief、确认 ONLY/不做。  
executor 仅在「Brief 已存在但字段可疑」时轻量复核，不替代 planner。

## Do

1. 读 `plans/*-program.md` 队首（若有；勿硬编码 Epic）
2. 对照架构：`.harness/rules/module-boundary.md`、`auth-token-hygiene.md`、`/AGENTS.md`
3. 落盘或核对 `plans/slices/<id>.md`（字段见 `.harness/rules/slice-contract.md`）
4. 输出：
   - 本轮角色交接（完成后回 **conductor** → 人批 → executor → reviewer）
   - 建议 skill 链
   - 风险 / 架构裁决（若有）
   - 机跑验证 vs 人证清单是否已拆分

## Don't

- 不改业务代码
- 不勾验收、不自称 Accept
- 不把整模块写成一个 Slice

## Refs

- `.harness/agents/planner.md`
- `/AGENTS.md`
- `~/.cursor/skills/migration-os-harness/prompts.md`
