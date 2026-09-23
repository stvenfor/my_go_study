# Acceptance Record — new-car-follow-ui-parity

Date: 2026-09-23
Decision: **Partial**（Flutter 代码落地；待人证并排对照「新车成交」）

## Verification

```text
cd my_ai_project/features/home && dart analyze lib/new_car_follow
# exit 0（仅 info：implementation_imports / prefer_const）
```

## Checklist evidence

| Row | Evidence |
|-----|----------|
| 列表对标成交骨架 | `NewCarFollowListPage`：ProfileHeader + Sticky Tab + 卡片行 + 底 FAB |
| 建档表单 + 假上传 | `NewCarFollowCreatePage` + `NewCarFollowFakeUpload`（本地/占位，不入 API） |
| 详情结构化 + 改级别 | `NewCarFollowDetailPage` Hero/Info/级别 Chip |
| CONTEXT 术语 | `CONTEXT.md`「新车跟进档案」节 |
| 未改 Go list/summary 语义 | 本 Slice 无 `internal/**` 业务 diff |

## Interaction script（人证）

1. 首页点「新车跟进」→ 顶栏四格 + 吸顶 Tab + 底「新建跟进档案」
2. 与「新车成交」并排：骨架同构（灰底/白卡/蓝主色）
3. 建档：选级别文案、假上传选图或占位 → 提交成功回列表
4. 详情：改级别后意向档颜色/文案更新

## Deferred（已批后续 Slice）

- C2 跟进流水
- C4 店管全店
- 真 OSS
