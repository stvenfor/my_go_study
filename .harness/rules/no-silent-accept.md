# Rule — no silent Accept

以下任一成立 → **不得**勾验收、不得 Accept：

- 未跑或未过 `agent-post` / Brief「验证命令」
- harness-report 中 `post.ok !== true`（若使用 harness）
- 缺 acceptance-record / Context Card（实现类）
- 假 Migrated / 无 note 的 Degraded

commit/push：仅当人明确指令。
