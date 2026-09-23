# Slice — community-search-c1-unified-api

## Slice Brief
- SOURCE_MODULE: community (topics search only)
- TARGET_MODULE: community unified search
- Source entry: CommunityPage search bar + GET /topics/search
- Target entry: GET /api/v1/community/search + Flutter CommunitySearchPage
- 本轮 ONLY:
  - Go: SearchPosts / SearchUsers / Search usecase / controller / route
  - Go: pg_trgm 索引 migration（可选加速）
  - Go: escape ILIKE + q 长度校验
  - docs/community-posts-api.md 补 search 节
  - Flutter（并行仓）: CommunitySearchPage + 接线 + debounce（已批「全做」）
- 不做:
  - ES / 拼音 / 搜索热词上报
  - 首页 SearchPage 合并
  - 真实媒体上传
- 验收:
  - `go test ./internal/usecase/ -run CommunitySearch` 绿
  - curl `GET /api/v1/community/search?q=Flutter&type=all` 返回 posts/topics/users
  - Flutter 社区页搜索条可进搜索页（人证）
- 文件白名单:
  - internal/domain/repository/community_repository.go
  - internal/repository/postgres/community_repo.go
  - internal/usecase/community_usecase.go
  - internal/usecase/community_usecase_test.go
  - internal/delivery/http/controller/community_controller.go
  - internal/delivery/http/router/community_routes.go
  - migrations/20260923170000_community_search_trgm.*
  - docs/community-posts-api.md
  - plans/slices/community-search-c1-unified-api.md
  - docs/acceptance-records/
  - .harness/changes/current.md
- 文件黑名单:
  - membership / after-sales / new-car-follow
- 验证命令:
  - go test ./internal/usecase/ -run 'CommunitySearch|NormalizeMedia|NormalizePostTab' -count=1
- 证据: docs/acceptance-records/2026-09-23-community-search.md

## Context Card — community-search-c1
- 已完成: Go Search API + trgm；Flutter 搜索页并联；机跑绿
- 未做/Deferred: ES/拼音；首页 SearchPage 合并；真机人证
- 关键文件: community_controller/usecase/repo；Flutter community_search_*；migration 20260923170000
- harness: post ok? usecase + EscapeILIKE 绿
- 下一 Slice 建议: 无（本 Epic MVP 完）；人证后 archive
- 已知坑: trgm 需 Postgres 扩展权限；失败时仍走 ILIKE

