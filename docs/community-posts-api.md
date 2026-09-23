# 社区动态后端接口设计

> 状态：设计冻结（grill 完成）；本会话不写实现（Q15-C）。  
> ADR：[0007-community-posts-mvp.md](./adr/0007-community-posts-mvp.md)  
> 对齐客户端：`my_ai_project/features/community` 的 `PostModel` / `CommentModel` / `PostRepository`

## 1. 目标与边界

| 做 | 不做（本切片） |
|----|----------------|
| 发帖（正文 + 默认图/视频链接 + 关联话题） | 真实媒体上传 |
| 话题列表 / 搜索 | 真实持仓匹配「盘友」 |
| 动态列表（对齐社区页卡片字段） | 社区公约服务端校验 |
| 点赞、评论、软删 | 转发、@用户实体、关联操作/产品 |
| 「问大家」：标志 + 种子用户 `sys.notify` | 迷你持仓表 |

鉴权：全部走现有 **SessionAuth**（`Authorization` + `X-Session-ID` + `X-Device-ID`），与 transactions / profile 一致。

统一响应壳：`{ code, message, data, timestamp }`（`response.Success`）。列表用 `data: { list, pagination }`。JSON **一律 snake_case**。

---

## 2. 领域约定（已锁定）

| 概念 | 约定 |
|------|------|
| 动态 Post | 一条社区内容；作者 = 会话 `user_id` |
| 媒体 | `none` \| `image` \| `video` **互斥**；不上传，用默认 URL |
| 话题 | 一等公民；一条动态 **至多 1** 个 `topic_id` |
| 正文 | 若关联话题，服务端在 `content` 末尾追加 `\n#话题名`（已含则不重复） |
| 问大家 | `is_ask_everyone=true`；发帖后向种子用户推 `sys.notify` |
| 公约弹窗 | 纯客户端，**每天最多弹一次** |
| 删除 | 软删；仅作者；列表不可见 |

默认媒体常量（实现时可放 config）：

```text
DEFAULT_IMAGE_URLS = [
  "https://picsum.photos/seed/wys_post_1/400/400",
  "https://picsum.photos/seed/wys_post_2/400/400",
  "https://picsum.photos/seed/wys_post_3/400/400",
]
DEFAULT_VIDEO_URL       = "https://flutter.github.io/assets-for-api-docs/assets/videos/bee.mp4"
DEFAULT_VIDEO_COVER_URL = "https://picsum.photos/seed/wys_post_video/640/360"
```

客户端发帖时：

- `media_type=image` → 可不传 URL，服务端填 `DEFAULT_IMAGE_URLS`（或客户端传 1～9 个，服务端校验后入库）
- `media_type=video` → 可不传，服务端填默认 video + cover
- `media_type=none` → 无媒体字段

---

## 3. 表设计（`wys_` 前缀，本地 Postgres）

### 3.1 `wys_topics`

| 列 | 类型 | 说明 |
|----|------|------|
| id | uuid PK | |
| name | text NOT NULL UNIQUE | 不含 `#`，如 `纳指大涨超2%再创新高` |
| heat | bigint NOT NULL DEFAULT 0 | 热度（展示「101.5万」由客户端格式化，或另加 `heat_label`） |
| is_ask_everyone | boolean NOT NULL DEFAULT false | 特殊「问大家」话题 |
| created_at | timestamptz | |

索引：`name` 唯一；`(heat DESC)` 列表排序；`name gin_trgm` 或 `ILIKE` 搜索（MVP 用 `ILIKE %q%` 即可）。

### 3.2 `wys_posts`

| 列 | 类型 | 说明 |
|----|------|------|
| id | uuid PK | |
| user_id | uuid NOT NULL | 作者 |
| content | text NOT NULL | 含拼好的 `#话题` |
| media_type | smallint NOT NULL | `0=none` `1=image` `2=video` |
| image_urls | jsonb NOT NULL DEFAULT '[]' | `string[]`；仅 image |
| video_url | text NULL | 仅 video |
| video_cover_url | text NULL | 仅 video |
| topic_id | uuid NULL FK → wys_topics | 至多一个 |
| is_ask_everyone | boolean NOT NULL DEFAULT false | 可与特殊话题同时 true |
| source | text NOT NULL DEFAULT '' | 如 `来自 iPhone` |
| like_count | int NOT NULL DEFAULT 0 | 冗余计数 |
| comment_count | int NOT NULL DEFAULT 0 | 冗余计数 |
| heat | bigint NOT NULL DEFAULT 0 | 热门排序；`heat = like_count*2 + comment_count`，点赞/评论写路径维护 |
| deleted_at | timestamptz NULL | 软删 |
| created_at | timestamptz | |
| updated_at | timestamptz | |

索引：`(deleted_at, created_at DESC)` 最新；`(deleted_at, heat DESC, created_at DESC)` 热门；`(user_id)`；`(topic_id)`。

### 3.3 `wys_post_likes`

| 列 | 类型 |
|----|------|
| post_id | uuid |
| user_id | uuid |
| created_at | timestamptz |
| PRIMARY KEY (post_id, user_id) |

### 3.4 `wys_post_comments`

| 列 | 类型 | 说明 |
|----|------|------|
| id | uuid PK | |
| post_id | uuid NOT NULL | |
| user_id | uuid NOT NULL | |
| content | text NOT NULL | |
| reply_to_nickname | text NULL | 对齐 Mock，非树形 |
| deleted_at | timestamptz NULL | 可选；MVP 可硬删评论 |
| created_at | timestamptz | |

### 3.5 `wys_user_follows`（关注关系）

| 列 | 类型 | 说明 |
|----|------|------|
| follower_id | text NOT NULL | 关注者（当前用户） |
| followee_id | text NOT NULL | 被关注作者 |
| created_at | timestamptz | |
| PRIMARY KEY (follower_id, followee_id) | | |
| INDEX (followee_id) | | |

用于社区页「关注」Tab：只返回作者在我关注列表中的动态。

### 3.6 作者展示（不新表）

列表 JOIN / 二次查询本地用户或 profile：

- `nickname` ← `display_name` / username  
- `avatar` ← `avatar_url`（空则客户端占位）

---

## 4. 枚举

```text
media_type: 0 = none | 1 = image | 2 = video
post_tab: latest | hot | following   # 对齐 UI「最新 / 热门 / 关注」
```

写接口也可接受字符串 `"none"|"image"|"video"`，服务端归一成 smallint。

---

## 5. API 一览

前缀：`/api/v1/community`  
中间件：SessionAuth

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/topics` | 话题列表（热度降序） |
| GET | `/topics/search?q=` | 搜索话题（兼容发帖选话题） |
| GET | `/search?q=&type=` | **统一搜索**：`type=all\|post\|topic\|user` |
| POST | `/posts` | 发帖 |
| GET | `/posts` | 动态流 |
| DELETE | `/posts/:id` | 软删（作者） |
| POST | `/posts/:id/like` | 点赞 |
| DELETE | `/posts/:id/like` | 取消赞 |
| GET | `/posts/:id/comments` | 评论列表 |
| POST | `/posts/:id/comments` | 发评论 |

> 点赞也可用单接口 `PUT /posts/:id/like { liked: true|false }`；上表拆分更 REST。Flutter `toggleLike(postId, liked)` 任选其一包装即可。

---

## 6. 请求 / 响应契约

### 6.1 `GET /topics`

Query：`page`（默认 1）、`size`（默认 20）

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [
      {
        "id": "uuid",
        "name": "纳指大涨超2%再创新高",
        "heat": 1015000,
        "is_ask_everyone": false
      },
      {
        "id": "uuid-ask",
        "name": "问大家",
        "heat": 0,
        "is_ask_everyone": true
      }
    ],
    "pagination": { "page": 1, "size": 20, "total": 12, "totalPages": 1 }
  },
  "timestamp": 0
}
```

客户端展示：`#` + `name`；热度自行格式化为「101.5万」。

「问大家」卡片：列表里 `is_ask_everyone=true` 的项置顶（服务端：`ORDER BY is_ask_everyone DESC, heat DESC`）。

### 6.2 `GET /topics/search?q=`

同列表结构；`q` 空则等价列表首页。`name ILIKE '%'||q||'%'`（已 escape `%` `_`）。

### 6.2b `GET /search` — 社区统一搜索

Query：

| 参数 | 说明 |
|------|------|
| `q` | 关键词；trim；最长 64 字；空时 post/user 返回空列表 |
| `type` | `all`（默认）\| `post` \| `topic` \| `user` |
| `page` / `size` | 同其它列表；`all` 时各分区共用 |

匹配：话题 `name`、动态 `content`、用户 `user_name`，均为 `ILIKE %q% ESCAPE '\'`。可选 `pg_trgm` GIN 索引（migration `20260923170000`）。

`type=all` 响应 `data`：

```json
{
  "q": "Flutter",
  "posts":  { "list": [ /* PostDTO */ ], "pagination": { "page": 1, "size": 5, "total": 2, "totalPages": 1 } },
  "topics": { "list": [ /* TopicDTO */ ], "pagination": { ... } },
  "users":  { "list": [ { "user_id", "nickname", "avatar", "is_followed" } ], "pagination": { ... } }
}
```

`type=post|topic|user`：标准 `SuccessList`（`data.list` + `data.pagination`）。

Flutter：社区页搜索条 → `CommunitySearchPage`；发帖选话题仍走 `/topics/search`。

### 6.3 `POST /posts` — 发帖（写模型）

```json
{
  "content": "记录一下吧",
  "media_type": "image",
  "image_urls": [],
  "video_url": null,
  "video_cover_url": null,
  "topic_id": "uuid-or-null",
  "is_ask_everyone": false,
  "source": "来自 iPhone"
}
```

校验：

| 条件 | 结果 |
|------|------|
| `content` trim 后空且无媒体 | 10001 |
| `media_type=image` 且 `image_urls` 空 | 填默认 3 张图 |
| `media_type=image` 且 urls > 9 | 10001 |
| `media_type=video` 且 url 空 | 填默认 video + cover |
| `media_type=none` 却带了 urls/video | 10001 或忽略媒体 |
| `topic_id` 不存在 | 10004 |
| 关联「问大家」话题 | 强制 `is_ask_everyone=true` |
| 已关联话题 | `content` 追加 `\n#name`（若未包含） |

成功 `data`：完整 **读模型**（同列表项，见 6.4），便于客户端插到列表顶部。

副作用（`is_ask_everyone`）：

1. 查种子被邀请用户 ID 列表（config / 种子表常量，排除作者）  
2. `PushToUser` → topic `sys.notify`，payload 示例：

```json
{
  "type": "community.ask_everyone",
  "title": "有人邀请你回答",
  "body": "来自盘友圈·问大家",
  "post_id": "uuid",
  "deep_link": "/community?post_id=uuid"
}
```

Flutter 侧后续接 Banner → 定位该帖；本设计只定 payload。

### 6.4 `GET /posts` — 动态流（读模型 ↔ PostModel）

Query：`page`、`size`、`tab=latest|hot|following`（默认 `latest`）。

单条 `list[]` 字段（与 Flutter 对齐）：

| JSON 字段 | PostModel | 说明 |
|-----------|-----------|------|
| `id` | id | string uuid |
| `user_id` | userId | |
| `nickname` | nickname | |
| `avatar` | avatar | |
| `content` | content | |
| `publish_time` | publishTime | RFC3339 |
| `source` | source | |
| `images` | images | `image_urls` 投影；非 image 为 `[]` |
| `video_url` | videoUrl | 非 video 为 null |
| `video_cover_url` | videoCoverUrl | |
| `like_count` | likeCount | |
| `comment_count` | commentCount | |
| `is_liked` | isLiked | 当前用户是否赞 |
| `is_mine` | isMine | `user_id == 会话用户` |
| `preview_comments` | previewComments | 最新 2 条 |
| `topic` | （扩展） | 可选 `{ id, name, is_ask_everyone }` |
| `is_ask_everyone` | （扩展） | bool |
| `media_type` | （扩展） | `"none"\|"image"\|"video"` |

`preview_comments[]`：

| 字段 | CommentModel |
|------|----------------|
| `id` | id |
| `post_id` | postId |
| `nickname` | nickname |
| `avatar` | avatar |
| `content` | content |
| `create_time` | createTime |
| `reply_to_nickname` | replyToNickname |

示例：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "list": [
      {
        "id": "…",
        "user_id": "…",
        "nickname": "开发者",
        "avatar": "https://i.pravatar.cc/200?img=1",
        "content": "周末打卡\n#问大家",
        "publish_time": "2026-09-22T10:00:00Z",
        "source": "来自 iPhone",
        "media_type": "image",
        "images": [
          "https://picsum.photos/seed/wys_post_1/400/400"
        ],
        "video_url": null,
        "video_cover_url": null,
        "like_count": 0,
        "comment_count": 2,
        "is_liked": false,
        "is_mine": true,
        "is_ask_everyone": true,
        "topic": { "id": "…", "name": "问大家", "is_ask_everyone": true },
        "preview_comments": [
          {
            "id": "…",
            "post_id": "…",
            "nickname": "张三",
            "avatar": "…",
            "content": "说得对！",
            "create_time": "2026-09-22T10:01:00Z",
            "reply_to_nickname": null
          }
        ]
      }
    ],
    "pagination": { "page": 1, "size": 10, "total": 35, "totalPages": 4 }
  },
  "timestamp": 0
}
```

过滤 / 排序（`?tab=`，默认 `latest`）：

| tab | 过滤 | 排序 |
|-----|------|------|
| `latest` | `deleted_at IS NULL` | `created_at DESC` |
| `hot` | 同上 | `heat DESC, created_at DESC` |
| `following` | 作者 ∈ `wys_user_follows`（当前用户作 follower） | `created_at DESC` |

未关注任何人时 `following` 返回空列表。

### 6.5 `DELETE /posts/:id`

作者校验；设 `deleted_at=now()`。`data: {}`。非作者 → 10003；不存在/已删 → 10004。

### 6.6 点赞

`POST /posts/:id/like`：幂等插入 like，`like_count++`、`heat+=2`（未赞过时）。  
`DELETE /posts/:id/like`：删除 like，`like_count--`、`heat-=2`（有赞时）。

`data`：返回更新后的读模型片段即可：

```json
{ "id": "…", "like_count": 12, "is_liked": true }
```

### 6.7 评论

`GET /posts/:id/comments?page=&size=` → `list` + `pagination`（字段同 preview）。

`POST /posts/:id/comments`：

```json
{
  "content": "同感 +1",
  "reply_to_nickname": "张三"
}
```

成功返回完整评论对象；`posts.comment_count++`、`heat+=1`。

### 6.8 关注

`POST /users/:id/follow`：幂等关注；不可关注自己。  
`DELETE /users/:id/follow`：取消关注。

`data`：

```json
{ "followee_id": "…", "is_followed": true }
```

---

## 7. 错误码（复用现有）

| code | 场景 |
|------|------|
| 0 | 成功 |
| 10001 | 参数非法（空正文且无媒体、媒体冲突等） |
| 10002 / 10021 / 10022 | 未登录 / session |
| 10003 | 非作者删帖等 |
| 10004 | 动态/话题不存在 |
| 50000 | 内部错误 |

---

## 8. 种子数据

1. **话题**：含 `问大家`（`is_ask_everyone=true`）+ 若干热度话题（可用示例图名称）。  
2. **动态**：若干 image / video / none 演示帖，挂不同话题，带 0～2 条预览评论。  
3. **问大家种子被邀请人**：config 数组 `community.ask_everyone_invite_user_ids`（或 SQL 常量），联调账号 UUID。

---

## 9. 分层落点（实现时，非本会话）

```text
entity:        Post / Topic / Comment
repository:    postgres（非 PostgREST；与 mall/analytics 同类本地表）
usecase:       CommunityUsecase（发帖拼话题、默认媒体、问大家 notify）
controller:    CommunityController
router:        community_routes.go → SessionAuth
```

Flutter（另会话）：

1. `HttpPostRepository` 实现 `PostRepository` + `createPost`  
2. `PublishPage` 按示例图：正文、选 image/video（只设 type）、关联话题页  
3. 公约弹窗：SharedPreferences 记「当天已弹」  
4. 通知：`community.ask_everyone` → `/community?post_id=`

---

## 10. 验收清单（实现会话用）

- [ ] `POST /posts` media_type=image，库中有默认图 URL，`media_type=1`  
- [ ] `POST /posts` media_type=video，有默认视频与封面  
- [ ] 关联话题后 `content` 含 `#name`，列表 `topic` 非空  
- [ ] `GET /posts` 字段可直接映射 `PostModel`，社区页卡片图/视频互斥展示正常  
- [ ] 点赞/评论计数与 `is_liked` / `preview_comments` 正确  
- [ ] 软删后列表不可见  
- [ ] `is_ask_everyone` 发帖后种子用户收到 `sys.notify`（需 Realtime 已启）  
- [ ] 公约弹窗不出现在任何后端日志/表中  

---

## 11. 决策摘要

| 项 | 值 |
|----|-----|
| Q1 | Go + Flutter 真接口（实现另会话） |
| Q2 | Topic 实体 + 正文拼 `#` |
| Q3 | 图/视频互斥 + 默认外链 |
| Q4 | 发帖/列表/话题/赞/评全做 |
| Q5 | 写干净 / 读对齐 PostModel |
| Q6 | 至多 1 话题 |
| Q7→Q12 | 问大家：标志 + 种子 notify，无持仓表 |
| Q8 | 公约客户端每天一弹 |
| Q9 | 作者软删 |
| Q10 | reply_to_nickname |
| Q11 | 种子话题 + 演示动态 |
| Q13 | deep_link 带 post_id |
| Q14 | snake_case |
| Q15 | **本会话仅设计** |
