# Acceptance Record — new-car-follow-c1

Date: 2026-09-22
Decision: Partial（Go 机跑绿；Flutter 壳已挂路由，待人证联调）

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
| 建档须 follow_level；非法 → 400/usecase Err | TestNewCarFollowCreateRejectsBadLevel |
| intent_band H/A→高 B→中 E→低 | TestFollowLevelIntentBandMapping + CreateAndIntentFilter |
| 列表 intent_band=高 / follow_level=B | TestNewCarFollowCreateAndIntentFilter |
| 写 next_follow 回写客户 | Create + PatchSyncNextFollow |
| 他人档案 404 | TestNewCarFollowGetForbiddenOtherOwner |
| 无当前店 | TestNewCarFollowNoStore |

## Interaction script（人证）

1. `make run`（local Auth）
2. Flutter 首页点「新车跟进」→ 列表四格与 Tab
3. 建档选级别 H → 详情显示 高
4. 改级别 B → 意向变 中

## Deferred

- C2 跟进流水
- C3 待办深链
- 店管全店
