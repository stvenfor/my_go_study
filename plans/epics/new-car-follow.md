# Epic — new-car-follow

| Checklist | CONTEXT「新车跟进档案」+ deal-invoice 分层模板 |
| Source | 提案对话 + CONTEXT；2026-09-23 人批扩 scope |
| Target | Go BFF local Postgres + Flutter `my_ai_project` |
| Status | ui-parity + C2 + C4 **代码 Partial**；待人证 |

## DoD
- [x] C1 档案 CRUD + ABEH 级别 + 同步 `next_follow_up_at`（Go）
- [x] Flutter UI 对标「新车成交」（含假上传）— Partial
- [x] C2 跟进流水读写 — Partial
- [x] C4 店管当前店全店可见 — Partial
- [x] API 文档 + CONTEXT 术语
- [x] usecase 测试覆盖级别、流水、店管可见性
- [ ] Flutter / 店管账号人证 Accept

## Gap table
| ID | Gap | Severity | Slice |
|----|-----|----------|-------|
| G1 | Flutter 壳观感差 | High | ui-parity ✅ Partial |
| G2 | 无跟进流水 | High | C2 ✅ Partial |
| G3 | 首页待办项无 `file_id` 深链 | Low | C3 / Deferred |
| G4 | 店管全店可见 | Med | C4 ✅ Partial |

## Slice backlog
| Slice | Status |
|-------|--------|
| `new-car-follow-c1-file-crud` | Go Accept Partial |
| `new-car-follow-ui-parity` | Partial 落地 |
| `new-car-follow-c2-follow-log` | Partial 落地 |
| `new-car-follow-c4-store-admin-scope` | Partial 落地 |
| `new-car-follow-c3-todo-deeplink` | Deferred |

## Context Card
- Done: C1–C4 代码路径 + UI 对标 + 假上传 + 流水 + 店管口径
- Deferred: C3 深链；转交；真 OSS；人证 Full Accept
- Next: **人** 真机人证（列表/流水/店管）→ commit 仅人指令
