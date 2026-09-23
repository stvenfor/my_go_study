# Slice — new-car-follow-c2-follow-log（草案，C1 批后升格）

## Slice Brief
- SOURCE_MODULE: 新车跟进档案
- TARGET_MODULE: Go BFF + Flutter 详情流水区
- 本轮 ONLY:
  - 表 `wys_new_car_follow_log`
  - `POST/GET /api/v1/new-car-follow-files/:file_id/logs`
  - 写流水可带 `next_follow_up_at` / 可选改 `follow_level`；事务更新档案+客户
  - 详情页时间线 UI
- 不做:
  - 外呼录音、企微、店管全店、待办深链
- 验收:
  - 流水按时间倒序；写后 `last_follow_at` 更新；改下次跟进仍驱动首页待办
- 文件白名单: （C1 同模块文件 + log 相关；待 C1 Accept 后锁定）
- 验证命令:
  - go test ./internal/usecase/ -count=1 -run Follow
- 证据: docs/acceptance-records/2026-09-22-new-car-follow-c2.md

## Context Card
- 依赖: C1 Accept
- 下一 Slice: C3 待办深链（可选）
