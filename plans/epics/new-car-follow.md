# Epic — new-car-follow

| Checklist | CONTEXT「新车跟进档案」+ deal-invoice 分层模板 |
| Source | 提案对话 + CONTEXT |
| Target | Go BFF local Postgres + Flutter `my_ai_project` |
| Status | Audit → Brief 待人批 |

## DoD
- [ ] C1 档案 CRUD + ABEH 级别 + 同步 `next_follow_up_at`
- [ ] C2 跟进流水读写
- [ ] API 文档 + CONTEXT 术语一致
- [ ] usecase 测试覆盖级别校验与逾期过滤
- [ ] harness post green on last Go slice
- [ ] Flutter 列表/建档/详情可联调（人证或配对仓 Accept）

## Gap table
| ID | Gap | Severity | Slice |
|----|-----|----------|-------|
| G1 | 无 `wys_new_car_follow_file` / API / UI | High | C1 |
| G2 | 无跟进流水表与写接口 | High | C2 |
| G3 | 首页待办项无 `file_id` 深链 | Low | C3 / Deferred |
| G4 | 店管全店可见 | Med | Deferred（本 Epic 不做） |

## Slice backlog
| Slice | Status |
|-------|--------|
| `new-car-follow-c1-file-crud` | Brief 待批 |
| `new-car-follow-c2-follow-log` | 草案（C1 批后细化） |
| `new-car-follow-c3-todo-deeplink` | Deferred |

## 架构裁决

| 议题 | 选项 | 推荐 | 影响 |
|------|------|------|------|
| 级别存储 | smallint 0–3 vs `char(1)` A/B/E/H | **`follow_level char(1)`** CHECK IN ('A','B','E','H') | 与业务话术一致；响应另带 `intent_band` |
| 意向档 | 库列 vs 派生 | **派生**（H/A→高，B→中，E→低） | 防双源；写禁传意向字符串 |
| 客户表 | 新表 vs 复用 | **复用 `wys_store_customer`** | 对齐成交/待办 |
| 可见范围 C1 | 本人 vs 全店 | **仅本人**（对齐成交发票） | 店管延后 |
| 模块路径 | supabase 双注册 | **仅 local** | 与 deal-invoice 一致 |

## Context Card
- Done: CONTEXT 术语；Program/Epic/C1 Brief
- Deferred: 店管全店、待办深链、关联成交发票
- Next: 人批 C1 → executor
