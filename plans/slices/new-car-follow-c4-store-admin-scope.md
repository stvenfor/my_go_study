# Slice — new-car-follow-c4-store-admin-scope

## Slice Brief
- SOURCE_MODULE: 新车跟进档案
- TARGET_MODULE: Go BFF 可见范围 + Flutter 列表/摘要口径
- Source entry: 人批「店管全店」纳入本轮；权限对齐首页待办店管（`PermRoleAssignStore` / store_admin）
- Target entry: summary + list + get + logs 在店管下看**当前店全员**档案；销售仍仅本人
- 本轮 ONLY:
  - usecase：有店管权限 → `store_id=current_store` 不限 `owner_user_id`；否则保持 C1 本人
  - summary：店管四格按全店计；销售仍本人
  - 列表/详情/流水：店管可读本店任意档案；写流水/PATCH 默认允许店管改本店档案（转交字段不做）
  - 响应带 `owner_user_id` + 可选 `owner_display_name`（列表识别归属销售）
  - 单测：销售看不到他人；店管看得到；无店管权限不变
  - 文档与 CONTEXT 补「店管全店可见」
  - Flutter：列表行展示归属销售（店管态）；非店管 UI 不变
- 不做:
  - 转交 owner、跨店报表、审核台
  - 平台管理员绕过当前店
  - 真上传；C3 深链（除非顺手只读，不做）
- 验收:
  - 店管：`GET list` 含同事档案；销售：仍 404 他人详情
  - 店管 summary.overdue/active 等为全店口径
  - `go test` 覆盖店管 vs 销售可见性
  - 人证：店管账号列表能看到他人跟进卡并标出销售名
- 文件白名单:
  - internal/usecase/new_car_follow_usecase.go
  - internal/usecase/new_car_follow_usecase_test.go
  - internal/domain/repository/new_car_follow_repository.go
  - internal/repository/postgres/new_car_follow_repo.go
  - internal/delivery/http/controller/new_car_follow_controller.go
  - docs/new-car-follow-api.md
  - docs/adr/*new-car-follow*（可补一条可见性）
  - docs/acceptance-records/2026-09-23-new-car-follow-c4.md
  - CONTEXT.md
  - plans/epics/new-car-follow.md
  - plans/slices/new-car-follow-c4-store-admin-scope.md
  - （配对仓）my_ai_project/features/home/lib/new_car_follow/**
- 文件黑名单:
  - 改 home_todo 卡语义（除非共享 perm 常量抽取且极小）
  - supabase/**
- 验证命令:
  - go test ./internal/usecase/ -count=1 -run 'Follow|NewCarFollow'
- 证据: docs/acceptance-records/2026-09-23-new-car-follow-c4.md
- Accept 模式: Partial（Go 机跑 + 店管账号人证）

## 架构裁决

| 议题 | 选项 | 推荐 | 影响 |
|------|------|------|------|
| 店管判定 | 职务=经理 vs 权限角色 | **权限** `PermRoleAssignStore`（与 home_todo 一致） | 职务不是权限 |
| 店管可否改他人档案 | 只读 vs 可写 | **可写** PATCH/流水（无转交） | 少一态；转交另 Slice |
| summary 口径 | 双接口 vs 同接口按角色 | **同接口按角色** | Flutter 少分支 |

## Context Card — new-car-follow-c4-store-admin-scope
- 已完成: 店管口径 usecase/repo/测试；DTO owner 字段；Flutter 销售名；ADR 0016
- 未做/Deferred: 转交；跨店；真上传；C3；店管人证
- 变更文件: 见白名单
- harness: post ok? go test 绿
- 下一 Slice 建议: C3 或收工人证
- 已知坑: 勿用门店职务数字当店管
