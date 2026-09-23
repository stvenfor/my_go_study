# Program Plan — 2026-09-22 新车跟进档案

## Milestone
| ID | Name | Goal | Out of scope |
|----|------|------|--------------|
| M1 | 跟进档案 MVP | 建档/列表/详情/改级别；ABEH→高中低；**UI 对标成交**；**流水**；**店管全店**；假上传 | 转交、审核台、真 OSS、跨店报表、待办深链 |

## Checklist snapshot
| Chapter | Status | Note |
|---------|--------|------|
| CONTEXT 术语 | ☐ | 节可能缺失 → UI/C2 白名单补 |
| Epic + C1 Brief | ✅ | Go Partial Accept |
| Go API + 表 | ✅ | C1；C2/C4 待做 |
| Flutter UI 对标成交 | ☐ | 队首 `ui-parity` |
| 跟进流水 | ☐ | C2 |
| 店管全店 | ☐ | C4 |
| 首页待办深链 | ☐ | C3 / Deferred |

## Epic queue (ordered)
| # | Epic | Plan | Done when |
|---|------|------|-----------|
| 1 | new-car-follow | `plans/epics/new-car-follow.md` | UI+C2+C4 Accept；文档与 CONTEXT 一致 |

## 不做
- 新车成交发票改表 / 审核 API
- 另建客户表（必须复用 `wys_store_customer`）
- PostgREST / supabase 路径注册本模块
- 跟进级别自由文本或 C 级（本项目仅 A/B/E/H）
- 真对象存储上传（假上传/占位 URL 可以）
- 转交 owner、跨店报表、审核台

## Risks Top3
| Risk | Mitigation |
|------|------------|
| 与成交发票抢客户语义 | CONTEXT Avoid；档案独表 + customer_id FK |
| `next_follow_up_at` 双写不一致 | usecase 单写入口事务更新档案+客户 |
| ABEH↔高中低映射漂移 | 映射只在服务端常量；写只收字母 |
| 店管用职务误判 | 与 home_todo 同用 `PermRoleAssignStore` |

## Next knife
1. **人批**队首 `plans/slices/new-car-follow-ui-parity.md`（可顺带批 C2/C4）
2. conductor → executor（配对仓 Flutter）；C2/C4 批后再开 Go

## Changelog
| Date | Change |
|------|--------|
| 2026-09-22 | 立项；纳入跟进级别 ABEH → 购车意向高/中/低 |
| 2026-09-23 | 人批：UI 对标新车成交；纳入 C2 流水 + 店管全店；假上传；切 ui-parity→C2→C4 |
