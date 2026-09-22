# 07 — 问大家：种子用户邀请通知

**What to build:** 以「问大家」模式发帖后，配置的种子用户收到 Realtime `sys.notify` 邀请（类型 `community.ask_everyone`，带 `post_id` 深链）；无真实持仓匹配。评论即回答入口由深链进社区定位该帖（客户端可最小实现）。

**Blocked by:** 03 — 关联话题（列表 / 搜索 / 发帖绑定）

**Status:** done

- [x] 发帖 `is_ask_everyone=true`（或绑定问大家话题）后触发邀请副作用
- [x] 种子用户列表来自配置；排除作者；经现有推送通道投递
- [x] 通知 payload 含 `type`、`post_id`、`deep_link`（约定见契约文档）
- [x] 联调可证明种子账号收到通知（Realtime 已启前提）
- [x] 不引入持仓 / 盘友数据表
