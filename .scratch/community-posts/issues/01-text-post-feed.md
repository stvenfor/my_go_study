# 01 — 纯文字动态：发布 → 社区列表可见

**What to build:** 已登录用户发布一条纯文字动态后，社区页从真实后端拉到该条（含作者昵称与头像）；打开社区可见种子演示帖。本票不做图片/视频、话题关联、点赞、评论。

**Blocked by:** None — can start immediately

**Status:** done

- [x] `POST /api/v1/community/posts` 可创建纯文字动态（SessionAuth）
- [x] `GET /api/v1/community/posts` 分页返回读模型字段，可映射 Flutter `PostModel`（媒体可为空）
- [x] 库中有种子演示动态；新发帖出现在列表最前
- [x] Flutter 社区列表走真 API；发布页可发纯文字并回到列表看见新帖
- [x] curl 或自动化测试可验收发帖 + 列表
