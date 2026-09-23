# Program Plan — 2026-09-22 新车跟进档案

## Milestone
| ID | Name | Goal | Out of scope |
|----|------|------|--------------|
| M1 | 跟进档案 MVP | 建档/列表/详情/改级别与下次跟进；ABEH→高中低；流水 | 店管全店、转交、审核台、真上传、跨店报表 |

## Checklist snapshot
| Chapter | Status | Note |
|---------|--------|------|
| CONTEXT 术语 | ✅ | 「新车跟进档案」节 |
| Epic + C1 Brief | ✅ | 待人批 |
| Go API + 表 | ☐ | executor |
| Flutter 列表/详情 | ☐ | 配对仓 `my_ai_project` |
| 首页待办深链 | ☐ | C3 / Deferred |

## Epic queue (ordered)
| # | Epic | Plan | Done when |
|---|------|------|-----------|
| 1 | new-car-follow | `plans/epics/new-car-follow.md` | C1+C2 Accept；文档与 CONTEXT 一致 |

## 不做
- 新车成交发票改表 / 审核 API
- 另建客户表（必须复用 `wys_store_customer`）
- PostgREST / supabase 路径注册本模块
- 跟进级别自由文本或 C 级（本项目仅 A/B/E/H）

## Risks Top3
| Risk | Mitigation |
|------|------------|
| 与成交发票抢客户语义 | CONTEXT Avoid；档案独表 + customer_id FK |
| `next_follow_up_at` 双写不一致 | usecase 单写入口事务更新档案+客户 |
| ABEH↔高中低映射漂移 | 映射只在服务端常量；写只收字母 |

## Next knife
1. 人批 `plans/slices/new-car-follow-c1-file-crud.md`
2. conductor → executor（Go）；Flutter 可并行若白名单不交

## Changelog
| Date | Change |
|------|--------|
| 2026-09-22 | 立项；纳入跟进级别 ABEH → 购车意向高/中/低 |
