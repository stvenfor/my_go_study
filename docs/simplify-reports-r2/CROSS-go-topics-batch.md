# CROSS — Topics N+1 → batch

| 字段 | 值 |
|------|-----|
| ID | CROSS |
| Program | Round-2 deep simplify |
| 状态 | Done |
| 硬约束 | 业务/页面语义不变 |

## 1. Before
community mapPosts / short_video 列表循环 GetTopic。
## 2. Goals
G1 TopicsByIDs 批量；缺 topic 仍跳过（同旧「找不到就忽略」）。
## 3. After
iface+postgres+usecase；短视频 mem mock 补齐。go test usecase ok。

