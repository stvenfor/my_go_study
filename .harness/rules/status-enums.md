# Rule — status enums

合法 `status`（仅此五态）：

`Migrated` | `Degraded` | `Placeholder` | `Deferred` | `Removed`

- 能编译 / `/health` 通 ≠ Migrated
- Degraded / Placeholder / Deferred **必须**有 `note`
- 未登记却声称「已完成」= 静默消失（禁止）

机检：Brief / acceptance-record 人工核对；有 manifest 时再挂校验脚本
