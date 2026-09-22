# 02 — 默认图片 / 视频媒体

**What to build:** 发布时可选择图片或视频（二者互斥、不真实上传）；服务端写入默认外链与媒体类型；社区卡片按现有图文格 / 视频卡正确展示。

**Blocked by:** 01 — 纯文字动态：发布 → 社区列表可见

**Status:** done

- [x] 发帖支持 `media_type` 为 image / video / none；图与视频互斥
- [x] 未传 URL 时写入约定默认图片列表或默认视频+封面
- [x] `GET /posts` 投影 `images` / `video_url` / `video_cover_url` / `media_type`
- [x] Flutter 发布页可选图或视频；社区页展示与 Mock 时代一致（互斥）
- [x] 验收：各发一条图帖、一条视频帖，列表展示正确
