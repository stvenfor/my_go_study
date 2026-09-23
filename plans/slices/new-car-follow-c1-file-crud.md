# Slice — new-car-follow-c1-file-crud

## Slice Brief
- SOURCE_MODULE: 新车跟进档案（绿场）
- TARGET_MODULE: Go BFF `auth.provider=local` +（配对）Flutter 列表/建档/详情壳
- Source entry: CONTEXT「新车跟进档案」；模板 `deal-invoice`
- Target entry: `/api/v1/new-car-follow-files`；Flutter 首页「新车跟进」
- 本轮 ONLY:
  - 表 `wys_new_car_follow_file`（含必填 `follow_level` ∈ {A,B,E,H}）
  - 建档 / 列表（含按级别、按意向档、逾期）/ 详情 / PATCH 改级别·阶段·下次跟进·意向车型等
  - 写下次跟进时事务回写 `wys_store_customer.next_follow_up_at`
  - 读模型带 `follow_level` + 派生 `intent_band`（高/中/低）
  - summary 顶栏（本人×当前店：跟进中/逾期/按意向高中低计数可简化为四格：全部跟进中、逾期、高意向、战败）
  - SessionAuth；`owner_user_id=session`；`store_id=current_store`
  - usecase 单测：非法级别拒绝；H/A→高、B→中、E→低；越权读失败
  - 文档：`docs/new-car-follow-api.md` 短文 + 可选 ADR
  - Flutter：列表 Tab（可按意向高/中/低 + 逾期）、建档选级别 ABEH、详情展示级别与意向档（配对仓，人证）
- 不做:
  - 跟进流水表与 `/logs`（→ C2）
  - 店管全店、转交、权限枚举扩展
  - 成交发票关联 / 审核 / OCR / OSS
  - 另建客户表；改 `wys_deal_invoice` 语义
  - supabase/PostgREST 注册
  - 首页待办响应加 `file_id`（→ C3）
  - Realtime 推送
- 验收:
  - 建档必须带 `follow_level`；缺省或非法字母 → 400
  - 详情/列表 JSON 含 `follow_level` 与 `intent_band`（高|中|低），映射：H→高、A→高、B→中、E→低
  - 列表 `intent_band=高` 仅返回 H∪A；`follow_level=B` 仅 B
  - 改 `next_follow_up_at` 后，同客户出现在 `GET /api/v1/home/follow-up-customers` 逾期条件内（到期时）
  - 仅本人当前店可见；无当前店 → 400
  - `go test` 覆盖 usecase 级别与映射
- 文件白名单:
  - internal/domain/entity/new_car_follow.go
  - internal/domain/repository/new_car_follow_repository.go
  - internal/repository/postgres/new_car_follow_repo.go
  - internal/repository/postgres/sql/new_car_follow_schema.sql
  - internal/usecase/new_car_follow_usecase.go
  - internal/usecase/new_car_follow_usecase_test.go
  - internal/delivery/http/controller/new_car_follow_controller.go
  - internal/delivery/http/router/new_car_follow_routes.go
  - internal/delivery/http/router/router.go
  - cmd/api/main.go
  - migrations/*new_car_follow*
  - docs/new-car-follow-api.md
  - docs/adr/*new-car-follow*
  - CONTEXT.md
  - plans/epics/new-car-follow.md
  - plans/slices/new-car-follow-c1-file-crud.md
  - （配对仓，非本仓机检）my_ai_project/features/**/new_car_follow/**
  - （配对仓）my_ai_project/commons/wys_router/**/route_path.dart
- 文件黑名单:
  - internal/usecase/deal_invoice_usecase.go（除必要注释外勿改行为）
  - internal/repository/supabase/**
  - transactions / mall / community 无关模块
- 验证命令:
  - go test ./internal/usecase/ -count=1 -run Follow
  - make agent-pre（若已落地 check-brief）
- 证据: docs/acceptance-records/2026-09-22-new-car-follow-c1.md
- Accept 模式: Partial（Go 机跑 + Flutter 人证联调）

## 跟进级别契约（本 Slice 冻结）

| `follow_level` | `intent_band` | 业务含义（建议文案） |
|----------------|---------------|----------------------|
| H | 高 | 高意向（约 7 日内强可能） |
| A | 高 | 高意向（约 15 日内） |
| B | 中 | 中意向 |
| E | 低 | 低意向 |

- 库：`follow_level char(1) NOT NULL` + CHECK
- API 写：只接受 `follow_level`
- API 读：同时返回 `follow_level` 与 `intent_band`
- UI：建档/编辑用 A/B/E/H 四选一；列表可用「高/中/低」筛选（服务端按上表展开）

## 建议默认跟进周期（展示用，C1 可不强制校验）

| 级别 | 建议下次跟进间隔 |
|------|------------------|
| H | 1–2 天 |
| A | 3 天 |
| B | 7 天 |
| E | 15 天 |

## Context Card — new-car-follow-c1-file-crud
- 已完成: Go CRUD+ABEH+测试；docs/ADR；Flutter 列表/建档/详情壳与首页入口
- 未做/Deferred: 流水 C2；待办深链 C3；店管全店；Flutter 精致 UI
- 关键文件: 见白名单 + my_ai_project new_car_follow/*
- harness: post ok? usecase Follow 测试绿（无 agent-post make 目标）
- 下一 Slice 建议: `new-car-follow-c2-follow-log`
- 已知坑: 客户与档案双写 `next_follow_up_at` 必须同一 usecase 事务（已实现）
