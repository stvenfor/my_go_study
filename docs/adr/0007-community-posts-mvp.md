# 社区动态 MVP：读对齐 Flutter PostModel，问大家不做真实持仓

社区动态从 Flutter Mock 迁到 Go BFF。读模型用 snake_case 对齐现有 `PostModel`（图文与视频互斥）；写模型用干净字段（`media_type` + `topic_id`）。「问大家」只落 `is_ask_everyone` 并向种子用户发 `sys.notify`，不新建持仓域——两仓库都没有持仓数据，真实匹配留到以后。公约弹窗纯客户端、每天最多一次。列表 Tab：`tab=latest|hot|following`；热门靠 `wys_posts.heat`（`like*2+comment`），关注靠 `wys_user_follows`。本 ADR 定契约与边界。
