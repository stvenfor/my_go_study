# Acceptance Record — new-car-follow-c4

Date: 2026-09-23
Decision: **Partial**（Go 机跑绿；Flutter 列表展示销售名，待店管账号人证）

## Verification

```text
go test ./internal/usecase/ -count=1 -run 'Follow|NewCarFollow'
ok
```

## Checklist evidence

| Row | Evidence |
|-----|----------|
| 销售看不到他人 | TestNewCarFollowGetForbiddenOtherOwner + StoreAdminSeesOthers |
| 店管看得到 + owner 名 | TestNewCarFollowStoreAdminSeesOthers |
| 店管 summary 全店 | 同上 Active=1 / list total=2 |
| 权限 | `role.assign_store`（非职务）；ADR 0016 |
| Flutter | 列表/详情 `owner_display_name` 行 |

## Deferred

- 转交 owner、跨店、真 OSS、C3 深链
- 店管账号真机人证
