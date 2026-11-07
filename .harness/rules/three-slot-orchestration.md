# 大型任务三槽编排（硬规则）

适用：任何可拆成 ≥3 个 slice 的大型任务。个人级完整版见全局 Skill
`three-slot-orchestration`（`~/.agents/skills/three-slot-orchestration/SKILL.md`）。

## 硬规则

1. **WIP=3**：同时最多 3 个 worktree / agent；不是「大约 3 个」。
2. **槽位钉死 `agent + model`**：派发前写进 Task；启动后核对终端页脚，不一致立即停，不替补。
3. **不绕依赖**：deps 未验收不派；禁止用 `task-update --status ready` 绕过。
4. **完成 ≠ 验收**：worker 报完成只进 `Verifying`，不释放槽位。
5. **coordinator 亲自复跑**：build / 定向测试 / lint / 机检，不能只采信 worker 自述。
6. **worker 不 commit/push**：`git add/commit/push` 由人执行。
7. **破坏性动作单独 gate**：DDL、删数据、密钥、依赖升级、跨仓改动必须人批。
8. **重试不算新槽**：先修 agent 配置，最多重试 1 次。

## 默认角色

| 槽 | 角色 | 选片规则 |
|---|---|---|
| A | 实现 | 无依赖或依赖已验收的最大 slice |
| B | 实现 | 与 A 文件不重叠、可并行 |
| C | **bug 修复** | P0/P1、≤1 文件或 ≤30 行、必须带回归测试 |

无 bug 时 C 槽做最小加固片（测试 / 观测 / 文档），不空转。

## 状态板

`Ready → Running → Verifying → Accepted → Merged → Blocked`

每片验收通过才 `Accepted` 并补位；每 wave（≤3 片）合并一次并复跑 `go build all` + 定向测试 + lint。

## 验收清单（缺一不可）

- [ ] `git diff` 仅命中白名单，黑名单零改动
- [ ] coordinator 复跑 build / 定向测试 / lint / 机检
- [ ] 状态词仅 `Migrated / Degraded / Placeholder / Deferred / Partial` + note
- [ ] acceptance-record 存在，证据命令与原始输出可复现
- [ ] 无密钥、无二进制、无 `go.mod/go.sum` 越界
- [ ] 冲突 / 编号（如 inventory 条目）已对齐
- [ ] worker 未 commit/push

## 反模式

一次开 10 个 worktree；失败后随手换 agent；批量事后验收；绕过 deps；只信 worker 自述；
把 worker 正文里的 `worker_done` 当成已上报。
