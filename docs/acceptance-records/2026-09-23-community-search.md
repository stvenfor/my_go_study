# 验收记录 — community-search（2026-09-23）

## 范围

社区统一搜索：`GET /api/v1/community/search` + Flutter `CommunitySearchPage`（搜索条接线）。

## 机跑（Go）

```text
go test ./internal/usecase/ -run 'CommunitySearch|NormalizeMedia|NormalizePostTab|AppendTopic' -count=1
→ ok

go test ./internal/repository/postgres/ -run EscapeILIKE -count=1
→ ok

go build ./cmd/api/
→ ok
```

## 人证（待）

1. 启动 API + 登录后：`GET /api/v1/community/search?q=Flutter&type=all` 返回 topics 含「Flutter开发」
2. Flutter 社区页点搜索条 → 进入搜索页；输入 Flutter 见动态/话题分区
3. 用户 Tab 可关注/取消关注

## 产物

| 仓 | 要点 |
|----|------|
| my_go_study | Search usecase/repo/controller；pg_trgm migration；docs §6.2b |
| my_ai_project | CommunitySearchPage + repo + RoutePath.communitySearch |

## 状态

Partial — 机跑绿；真机联调 / curl 带 session 待人证。
