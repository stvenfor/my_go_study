# 06 — 作者软删动态

**What to build:** 作者可删除自己的动态；删除后社区列表不再出现该条；他人不可删。

**Blocked by:** 01 — 纯文字动态：发布 → 社区列表可见

**Status:** done

- [x] `DELETE /api/v1/community/posts/:id` 仅作者成功；软删不可见
- [x] 非作者删除返回禁止
- [x] Flutter 删除菜单走真 API，列表移除该卡
- [x] 验收：删后刷新列表无该 id
