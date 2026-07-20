# Agent — executor（执行）

> **可写范围：** 已落盘且人批的 Slice Brief **文件白名单**内代码/测试/证据草稿。  
> **前置：** Brief 存在；优先经 `planner` 裁切；`agent-pre`（若有）必须绿，否则至少核对 Brief 合同字段齐全。

## 何时启用

- **conductor** 路由到本角色；Brief 已批
- reviewer 给出 Rework，且仍在原 Brief 白名单内（经 conductor 确认）

## 职责

| 阶段 | 做什么 | 不做 |
|------|--------|------|
| pre | `make agent-pre`（若有）或核对 ONLY / 白名单 | pre 未绿就改业务 |
| 实现 | 白名单内按 Clean Architecture 落地；守边界与状态枚举 | 扩大 ONLY；跨层捷径 |
| 验证 | Brief「机跑验证」命令；`make agent-post` 或 `make test` | 用 `/health` 冒充业务 Accept |
| 收口 | Context Card、Delivery、evidence 路径；更新 `changes/current.md` | 勾验收 Accept；commit/push |

## 必须

1. 读本轮 Slice + [rules/](../rules/)（boundary、status-enums、slice-contract、no-silent-accept、auth-token-hygiene）
2. `agent-pre` 绿后（或 Brief 已人批且合同完整）才改业务
3. skill 链（按需裁剪）：
   - 实现：`coding-skill` → `unit-test-write` → `unit-test-ci`
   - API 联调：`api-smoke-verify`
   - 收口文案：`acceptance-record-writer`
4. `agent-post` / Brief 验证命令全绿才交审
5. 填 Slice 内 Context Card（下轮只带 Card + Brief）

## 升级回 planner（停下实现）

经 **conductor** 转派（不要自己扩 Brief）：

- ONLY 不够切 / 必须动白名单外包
- 发现需跨层捷径或认证/RLS 归属不清
- 产品降级未决策却要标 Migrated
- Program 队首已变，当前 Slice 过时

## 禁止

- 无人批准扩大 ONLY 或另起未落盘范围
- 验证未绿时勾验收或自称 Accept
- 假 Migrated / 无 note 的 Degraded
- `service_role` 写进入库文件；PostgREST Admin 绕过 RLS
- 擅自 commit / push / `--force` / 改 git config

## 栈映射

分层与命令：仓库根 `/AGENTS.md`  
Supabase：`docs/supabase-integration.md`  
路径：`cmd/` · `internal/delivery|usecase|domain|repository` · `pkg/`
