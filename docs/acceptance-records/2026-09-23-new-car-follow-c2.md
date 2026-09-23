# Acceptance Record — new-car-follow-c2

Date: 2026-09-23
Decision: **Partial**（Go 机跑绿；Flutter 详情流水已挂，待人证）

## Verification

```text
go test ./internal/usecase/ -count=1 -run 'Follow|NewCarFollow'
ok

go build ./cmd/api/
ok
```

## Checklist evidence

| Row | Evidence |
|-----|----------|
| 流水倒序 | TestNewCarFollowCreateLogAndOrder |
| 写后 last_follow + 客户 next | 同上 |
| 空正文拒绝 | ErrNewCarFollowBadLogBody |
| 越权 404 | CreateLog other → NotFound |
| 路由 | GET/POST `/:file_id/logs` |
| Flutter | 详情「写一条跟进」+ 时间线 |

## Deferred

- 人证真机写流水
- 外呼/企微
