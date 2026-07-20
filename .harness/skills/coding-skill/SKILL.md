# Skill — coding-skill

## When

**executor** 主用；`agent-pre` 已绿（或 Brief 已人批）；Brief 为实现类（非只读审计）。  
范围不清 → 停，交回 `.harness/agents/planner.md`。

## Do

1. 严格文件白名单（角色合同：`.harness/agents/executor.md`）
2. 新增能力按：entity → repository → usecase → controller/handler → router
3. 遵守 `.harness/rules/module-boundary.md`、status enums、`auth-token-hygiene.md`
4. 栈细节：仓库根 `/AGENTS.md` + 相关 `docs/`
5. Supabase 业务：`WithUserToken`；transactions 带 `user_id`
6. 在本仓库根目录执行 `go` / `make`（勿在父目录 `my_code_study`）

## Don't

- 一口吞整模块
- 假 Migrated
- 改 Brief「不做」中的模块
- 无人批准扩大 ONLY
- 把 `service_role` 写入可入库文件

## Out

可跑测的 diff；准备交给 `unit-test-write` / `unit-test-ci`；收口用 `acceptance-record-writer`。
