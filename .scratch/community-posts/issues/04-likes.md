# 04 — 点赞 / 取消赞

**What to build:** 用户在社区动态上点赞或取消后，计数与「我是否已赞」持久化，下拉刷新仍正确。

**Blocked by:** 01 — 纯文字动态：发布 → 社区列表可见

**Status:** done

- [x] 点赞 / 取消赞 API 幂等；`like_count` 与 `is_liked` 正确
- [x] 列表读模型带上当前用户的 `is_liked`
- [x] Flutter 点赞栏走真 API，局部刷新与 Mock 行为一致
- [x] 验收：赞 → 刷新仍赞；再取消 → 计数回落
