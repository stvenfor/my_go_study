# 小视频后端接口设计

> 状态：已实现（Go BFF）；Flutter 客户端对接另开。  
> 术语：[CONTEXT.md](../CONTEXT.md)「小视频」节  
> ADR：[0010-short-video-independent-entity.md](./adr/0010-short-video-independent-entity.md)  
> 对齐客户端：`my_ai_project/features/video/lib/short_video` 的 `ShortVideoItemModel` / `ShortVideoProfileModel`

## 1. 目标与边界

| 做 | 不做（本切片） |
|----|----------------|
| 发布小视频（标题 + 默认视频/封面 + 可选话题） | 真实拍摄/上传 |
| 发现流 + 个人列表（同一列表接口，`scope` 区分） | 写入社区动态 feed |
| 个人主页资料 + 统计 | 运营审核台、人工过审 API |
| 点赞 / 取消赞；播放上报（每次 +1） | 评论、转发、收藏、弹幕 |
| 作者按 id 软删 | 硬删；客户端必传真实 CDN URL |
| 话题关联（复用社区话题 API） | 小视频私有话题表 |

鉴权：全部走现有 **SessionAuth**（`Authorization` + `X-Session-ID` + `X-Device-ID`）。

统一响应壳：`{ code, message, data, timestamp }`（`response.Success`）。列表用 `data: { list, pagination }`。JSON **一律 snake_case**。

分页：与社区同构的 **page + size**（底层 offset）；排序一律 `created_at DESC`。

---

## 2. 领域约定（已锁定）

| 概念 | 约定 |
|------|------|
| 小视频 | 独立实体；作者 = 会话 `user_id`；不进社区 `wys_posts` |
| 话题 | 复用 `wys_topics`；一条小视频 **至多 1** 个 `topic_id`；选话题走社区 `GET /api/v1/community/topics*` |
| 标题 | 必填，trim 后 1～50 字；**不**追加 `#话题名` |
| 媒体 | 不上传；`video_url` / `cover_url` / `duration` / `aspect_ratio` 可空 → 服务端默认 |
| 审核态 | 发布 → `reviewing`；**只有人工放行**才变 `normal`（无「等一会儿自动过审」路径） |
| 发现流 | `scope=discovery`：仅 `normal` 且未软删 |
| 个人列表 | `scope=user`：指定 `user_id`（缺省=当前用户）；本人可见 `reviewing`，他人仅 `normal` |
| 放行 | `POST /api/v1/short-videos/:id/review`，body `{"action":"approve"}`；仅平台管理员（能力码 `role.assign_platform`） |
| 播放量 | `POST .../view` **每次 +1**，不做每用户去重 |
| 点赞 | 独立表；可取消；计入视频 `like_count` 与作者统计 |
| 删除 | `DELETE /:id` 软删；仅作者 |

### 配置

```text
short_video.review_mode = reject_until_manual   # env: SHORT_VIDEO_REVIEW_MODE
# reject_until_manual（默认）：待审内容只有人工放行后可见
# auto_allow_local_only     ：仅当 APP_ENV 非 prod 时生效，读路径就地放行（联调/演示用）
```

### 默认媒体（与社区视频同源，便于联调）

```text
DEFAULT_SHORT_VIDEO_URLS = [
  "https://flutter.github.io/assets-for-api-docs/assets/videos/bee.mp4",
  "https://vjs.zencdn.net/v/oceans.mp4",
  "https://www.w3school.com.cn/example/html5/mov_bbb.mp4",
]
DEFAULT_SHORT_VIDEO_COVER_URLS = [
  "https://picsum.photos/seed/sv_play_1/400/640",
  "https://picsum.photos/seed/sv_play_2/400/500",
  "https://picsum.photos/seed/sv_play_3/400/700",
]
DEFAULT_DURATION     = "0:15"
DEFAULT_ASPECT_RATIO = 1.25
```

发布时若 URL 为空：从池中按 `id` 哈希或顺序取一对 video+cover；`duration` / `aspect_ratio` 用默认。

> **仅联调环境**（parity-p0-5）：填充开关是 `config.IsLabAppEnv(APP_ENV)`。`APP_ENV=prod|production` 时
> 缺 `video_url` 或 `cover_url` → `400`（`code 10001`）并提示「上传通道未接入」，不再伪造媒体；
> 即便传了真实 URL，`duration` / `aspect_ratio` 仍是默认值（未做真实探测）。
> 登记见 [placeholder-inventory.md](./placeholder-inventory.md) 条目 5。

### 审核状态机（parity-p0-4 后）

```text
Create            → status = reviewing
POST /:id/review  → status = normal, approved_at = now()   （需 role.assign_platform）
```

没有任何按时间自动置 `normal` 的读路径；`GET /:id`、点赞、播放上报对未过审内容一律走「作者可见、他人 404」。

演示模式例外：`review_mode = auto_allow_local_only` 且 `APP_ENV` 非 prod 时，
`List` / `Get` 读到 `reviewing` 行会就地 `UPDATE status = normal, approved_at = now()`
（保持联调时客户端仍能看到内容）。正式环境不会命中这一支。

发现流 / 他人视角个人列表 SQL 过滤：

```text
status = normal
```

---

## 3. 表设计（`wys_` 前缀，本地 Postgres）

### 3.1 `wys_short_videos`

| 列 | 类型 | 说明 |
|----|------|------|
| id | uuid PK | |
| user_id | uuid NOT NULL | 作者 |
| title | text NOT NULL | 1～50 |
| video_url | text NOT NULL | 默认池填充后必有 |
| cover_url | text NOT NULL | 封面 |
| duration | text NOT NULL | 展示用，如 `"1:10"` |
| aspect_ratio | double precision NOT NULL DEFAULT 1.25 | 网格高度 |
| topic_id | uuid NULL FK → wys_topics | 至多一个 |
| status | smallint NOT NULL DEFAULT 0 | `0=reviewing` `1=normal` |
| view_count | bigint NOT NULL DEFAULT 0 | |
| like_count | int NOT NULL DEFAULT 0 | 冗余 |
| approved_at | timestamptz NULL | 过审时间 |
| deleted_at | timestamptz NULL | 软删 |
| created_at | timestamptz NOT NULL | |
| updated_at | timestamptz NOT NULL | |

索引：

- `(deleted_at, status, created_at DESC)` — 发现流  
- `(user_id, deleted_at, created_at DESC)` — 个人列表  
- `(topic_id)` 可选

> 服务端 **永不** 持久化 Flutter 的 `uploading`；该态仅客户端本地。

### 3.2 `wys_short_video_likes`

| 列 | 类型 |
|----|------|
| short_video_id | uuid NOT NULL |
| user_id | uuid NOT NULL |
| created_at | timestamptz NOT NULL |
| PRIMARY KEY (short_video_id, user_id) |

与 `wys_post_likes` 分离，互不共用。

### 3.3 作者展示（不新表）

Profile / 列表项 JOIN 本地用户或 profile：

- `nickname` / `display_name` ← 用户展示名  
- `avatar` / `avatar_url` ← 头像（空则客户端占位）  
- `role_badge` / `store_name` ← 本期固定返回 `""`（Flutter 现硬编码；门店职务接入后另开切片）

统计（`scope` 无关，按目标用户聚合，**排除软删**）：

| 字段 | 规则 |
|------|------|
| `video_count` | 本人：含 reviewing；他人：仅 normal |
| `view_count` | 同上可见集合上 `SUM(view_count)` |
| `like_count` | 同上可见集合上 `SUM(like_count)` |

JSON 里统计用 **number**；客户端可自行格式化成字符串（现 `ShortVideoStatsModel` 是 String）。

---

## 4. 枚举

```text
status: 0 = reviewing | 1 = normal
scope:  discovery | user
```

写/读 JSON 对外用字符串 `"reviewing"|"normal"`；库内 smallint。

---

## 5. API 一览

前缀：`/api/v1/short-videos`  
中间件：SessionAuth  

话题：**不**在本前缀下重复实现 → 使用 `/api/v1/community/topics`、`/topics/search`。

| 方法 | 路径 | 用途 |
|------|------|------|
| POST | `/` | 发布 |
| GET | `/` | 列表（发现 / 个人） |
| GET | `/profile` | 个人主页资料 + 统计 |
| GET | `/:id` | 详情（播放页可选） |
| DELETE | `/:id` | 软删（作者） |
| POST | `/:id/like` | 点赞 |
| DELETE | `/:id/like` | 取消赞 |
| POST | `/:id/view` | 播放上报（每次 +1） |
| POST | `/:id/review` | 人工放行（`{"action":"approve"}`，仅平台管理员） |

---

## 6. 请求 / 响应契约

### 6.1 `POST /api/v1/short-videos` — 发布

```json
{
  "title": "周末试驾片段",
  "video_url": null,
  "cover_url": null,
  "duration": null,
  "aspect_ratio": null,
  "topic_id": "uuid-or-null"
}
```

校验：

| 条件 | 结果 |
|------|------|
| `title` trim 后空或 > 50 | 10001 |
| `topic_id` 非空但不存在 | 10004 |
| URL 字段空 | 填默认池 |
| `aspect_ratio` 空或 ≤ 0 | `1.25` |
| `duration` 空 | `"0:15"` |

入库：`status=reviewing`，`view_count=0`，`like_count=0`。

成功 `data`：完整 **读模型**（同列表项，见 6.2），便于客户端插入网格。

### 6.2 `GET /api/v1/short-videos` — 列表

Query：

| 参数 | 说明 |
|------|------|
| `scope` | 必填：`discovery` \| `user` |
| `user_id` | `scope=user` 时可选；缺省 = 当前会话用户 |
| `page` | 默认 1 |
| `size` | 默认 20，上限 50 |

行为：

| scope | 可见性 |
|-------|--------|
| `discovery` | 未软删，且（已 `normal` 或已到惰性过审时点） |
| `user` | 目标用户未软删；若查看者是本人 → 含 reviewing；否则仅 normal（含惰性） |

排序：`created_at DESC`。

单条 `list[]`（对齐 `ShortVideoItemModel` + 扩展）：

| JSON 字段 | Flutter | 说明 |
|-----------|---------|------|
| `id` | id | uuid string |
| `title` | title | |
| `cover_url` | coverUrl | |
| `video_url` | videoUrl | |
| `view_count` | viewCount | int |
| `duration` | duration | 如 `"1:10"` |
| `aspect_ratio` | aspectRatio | double |
| `status` | status | `"normal"` \| `"reviewing"`（永不下发 `uploading`） |
| `like_count` | — | 扩展 |
| `is_liked` | — | 当前用户是否赞 |
| `is_mine` | — | 作者 == 会话用户 |
| `user_id` | — | 作者 |
| `nickname` | — | 作者昵称 |
| `avatar` | — | 作者头像 |
| `topic` | — | 可选 `{ id, name }`；无则 `null` |
| `publish_time` | — | RFC3339 = `created_at` |
| `approved_at` | — | 未过审为 `null` |

> 网格里的「发布」瓷砖（`type=publish`）纯客户端本地插入，**不下发**。

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
        "avatar": "https://…",
        "title": "周末试驾片段",
        "cover_url": "https://picsum.photos/seed/sv_play_1/400/640",
        "video_url": "https://flutter.github.io/assets-for-api-docs/assets/videos/bee.mp4",
        "view_count": 3,
        "like_count": 1,
        "duration": "0:15",
        "aspect_ratio": 1.25,
        "status": "reviewing",
        "is_liked": false,
        "is_mine": true,
        "topic": { "id": "…", "name": "纳指大涨超2%再创新高" },
        "publish_time": "2026-09-22T09:00:00Z",
        "approved_at": null
      }
    ],
    "pagination": { "page": 1, "size": 20, "total": 1, "totalPages": 1 }
  },
  "timestamp": 0
}
```

### 6.3 `GET /api/v1/short-videos/profile`

Query：`user_id` 可选，缺省 = 当前用户。

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "user_id": "…",
    "display_name": "开发者",
    "avatar_url": "https://…",
    "role_badge": "",
    "store_name": "",
    "is_me": true,
    "stats": {
      "video_count": 2,
      "view_count": 15,
      "like_count": 4
    }
  },
  "timestamp": 0
}
```

对齐 `ShortVideoProfileModel` / `ShortVideoStatsModel`（统计为 number，客户端 `toString()` 即可）。

### 6.4 `GET /api/v1/short-videos/:id`

单条读模型（同列表项）。不可见（软删 / 他人看 reviewing 且未到过审时点）→ 10004。读时执行惰性过审。

### 6.5 `DELETE /api/v1/short-videos/:id`

仅作者；设 `deleted_at=now()`。非作者 → 10003；不存在 → 10004。成功 `data: null`。

### 6.6 `POST /:id/like` · `DELETE /:id/like`

- 目标须对当前用户可见（规则同详情）。  
- 点赞：插入 likes，`like_count++`（已赞则幂等成功）。  
- 取消：删除 likes，`like_count--`（未赞则幂等成功）。  
- 成功返回更新后的读模型（或 `{ like_count, is_liked }` 最小集；推荐完整读模型，与社区一致）。

### 6.7 `POST /:id/view`

每次成功调用：`view_count + 1`（**不去重**）。目标须可见。成功返回 `{ "view_count": N }`。

### 6.8 `POST /:id/review`

body `{"action":"approve"}`（其它值 → 400）。权限：平台能力码 `role.assign_platform`
（现网只有 `platform_admin` 拥有），无权限 → 10003/403。放行后返回完整读模型
（`status=normal`、`approved_at` 非空）。重复放行为幂等 no-op，不刷新 `approved_at`。

机审 provider 与「驳回态」不在本片（见 `plans/epics/xhs-parity-p2-scale.md` P2-2）。

---

## 7. 错误码（与社区同系）

| code | 场景 |
|------|------|
| 10001 | 参数校验失败（标题空/过长等） |
| 10003 | 无权限（删别人的视频） |
| 10004 | 不存在 / 不可见 / topic 不存在 |
| 401/403 | SessionAuth 失败 |

---

## 8. 与 Flutter 对接要点

1. 用 Repository 替换 `short_video_mock_data.dart`；「发布」瓷砖仍本地画，点按进发布页。  
2. 发布页：标题 + 话题选择（复用社区 `TopicSelect` / topics API）→ `POST /short-videos`；不必选本地文件。  
3. 列表：`scope=user` 拉网格；若产品有发现页则 `scope=discovery`。  
4. `status=reviewing` 显示角标；播放前可调详情或直接用列表里的 `video_url`。  
5. 播放开始/结束任选时机调 `POST .../view`（每次进播放页 +1 即可）。  
6. 待审内容对外不可见，需人工放行（`POST /:id/review`）后才会出现在发现流；演示环境请配 `review_mode=auto_allow_local_only`。  
7. Profile 的 `role_badge` / `store_name` 空串时，客户端可继续用本地占位文案。

---

## 9. 实现切片建议（非本会话）

1. migration：`wys_short_videos` + `wys_short_video_likes`  
2. entity → repo → usecase（惰性过审抽一函数）→ controller → router  
3. `main.go` 注入 + config `review_mode`  
4. usecase 单测：发布默认 URL、惰性过审边界、scope 可见性、软删、like、view 累加  
5. Flutter：Repository + 发布页最小字段 + 去掉 mock  

---

## 10. 决策摘要（grill）

| # | 决策 |
|---|------|
| Q1 | 独立实体，不复用社区帖 |
| Q2 | 发现流 + 个人列表都做 |
| Q3 | 复用社区话题，至多一个 |
| Q4 | 有审核态；发布为 reviewing |
| Q5 | ~~配置时延后自动过审（默认 30s）~~ **已被 parity-p0-4 推翻**：无时间自动过审，默认拒绝直到人工放行；演示模式 `auto_allow_local_only` 例外 |
| Q6 | 单一列表接口 + `scope` |
| Q7 | 播放计数 + 点赞都做 |
| Q8 | 按 id 软删；发布 URL 可空用默认 |
| Q10 | 播放每次 +1，不去重 |
| Q11 | page/size 偏移分页，`created_at DESC` |
| Q12 | 独立 `GET /profile` |
| Q13 | 读时惰性过审 |
| Q14 | 话题复用社区 API；标题 1～50；不拼 `#` |
