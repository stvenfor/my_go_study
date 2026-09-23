# Current change (harness pointer)

| 字段 | 值 |
|------|-----|
| Active slice | new-car-follow（ui-parity + C2 + C4 均 Partial 落地） |
| Epic | `plans/epics/new-car-follow.md` |
| Role | **人**（人证 / commit） |
| Brief | 三份已批并已执行完毕 |
| agent:post | `go test … -run 'Follow\|NewCarFollow'` 绿；`go build ./cmd/api` 绿 |
| 验收 tick | Partial ×3 — 待真机人证 |
| Next | 人证后如需 commit 再下指令；C3 深链仍 Deferred |

## Notes

- Go：流水表/API；店管 `role.assign_store` 全店口径；owner 字段
- Flutter：成交对标 UI + 假上传 + 详情流水 + 列表销售名
- 证据：`docs/acceptance-records/2026-09-23-new-car-follow-{ui-parity,c2,c4}.md`
- 迁移：`migrations/20260923160000_new_car_follow_log.*`（或 EnsureSchema 兜底）
