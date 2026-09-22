# 05 — 评论与预览

**What to build:** 用户可对动态发评论、拉评论列表；动态卡展示最多两条预览评论；支持按昵称回复（非楼中楼树）。

**Blocked by:** 01 — 纯文字动态：发布 → 社区列表可见

**Status:** done

- [x] `GET/POST .../posts/:id/comments` 可用；`comment_count` 递增
- [x] 列表项 `preview_comments` 最多 2 条，字段对齐 `CommentModel`
- [x] 支持可选 `reply_to_nickname`
- [x] Flutter 评论面板走真 API
- [x] 验收：发评后卡片预览与评论列表一致
