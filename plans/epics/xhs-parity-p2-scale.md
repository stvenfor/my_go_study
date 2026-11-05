# Epic — xhs-parity-p2-scale（上量与性能）

> Program: [`plans/2026-11-03-xhs-parity-program.md`](../2026-11-03-xhs-parity-program.md)
> 差距来源: [`docs/xhs-benchmark-gap-analysis.md`](../docs/xhs-benchmark-gap-analysis.md) §3、§4.5、§5.2、§5.3、§5.4、§7-P2
> **前置**：M1（P0）全 Accept + M2（P1）的 P1-1/P1-2/P1-4 已落地。没有 request-id 与指标，本 Epic 所有「性能改动」都无法评估。
> Brief 状态：**尚未落盘**（执行前由 planner 按本文各节生成）

## 差距表

| 能力 | 现状（证据） | 目标 | Slice |
|------|--------------|------|-------|
| 媒体链路 | 无上传：`FormFile/multipart` 检索 0；缺省 picsum/公开 mp4（`short_video/service.go:32-39`） | 客户端直传（预签名）+ 回调核销 + 多规格 + CDN | P2-1 |
| 内容风控 | 零（P0-4 只去掉了假审核） | 机审 + 人审队列 + 举报 + 频控 + 信用分 | P2-2 |
| 计数与 Feed | 同事务 `like_count+1 / heat+2`（`community/repo_postgres.go:156-160`）；关注流拉模式子查询（`:118-143`） | Redis 增量 + 回写；hybrid 收件箱；曝光去重 | P2-3 |
| 缓存层 | Redis 只有 5 类会话/事件 key，零 DB 前置缓存 | cache-aside + 版本失效广播 | P2-4 |
| 分页与索引 | 全域 offset（`shared/domain_helpers.go:28-39`）+ 每页 `Count(*)`；缺 `wys_post_comments.user_id`、`transactions.date` 索引 | keyset cursor + 索引审计 | P2-5 |
| 跨存储一致性 | PG↔ES best-effort（`transaction/service.go:93-115`）；ES 索引名硬编码无别名（`repo_es.go:20`）、中文无分词 | outbox + 别名/reindex + 中文分词 | P2-6 |
| 多实例 | ticker 无锁重复跑（`core/queue/hourly_ticker.go`）；SSE 限流进程内；WS 无粘滞 | 分布式锁 + Redis 限流 + 粘滞 + 容量数字 | P2-7 |
| 互动补齐 | 收藏/转发/@/楼中楼/多话题 全无（`topic_id` 是单列外键） | 关系表 + 计数 + 频控 + 通知三件套 | P2-8 |
| 事务编排 | `WithTx` 只在资金路径（P0-3 引入） | 推广到全域跨仓储写 | P2-9 |
| 测试与容量 | 66 单测全内存 fake；`app/cmd/*`、`delivery/worker` 0 测试；无压测/覆盖率 | 真库集成测试 + 压测基线 | P2-10 |

## Slice backlog（有序）

| # | ID | ONLY 摘要 | 依赖 | 并行性 |
|---|-----|-----------|------|--------|
| 1 | `parity-p2-1-media-presign-upload` | 预签名直传 + complete 核销 + `wys_media_objects` + 多规格 | P0-1、P1-4 | 可与 2 并行 |
| 2 | `parity-p2-2-moderation-pipeline` | 机审 provider + 人审队列 + 举报 + 状态机 + outbox 驱动 | P2-1（要审媒体）、P0-4 | 与 1 串行更稳 |
| 3 | `parity-p2-5-cursor-pagination-index-audit` | keyset 分页 + 深分页闸 + 索引补齐 | P0-1 | **先做**（改动面小、收益即得） |
| 4 | `parity-p2-3-counter-cache-feed` | Redis 计数 + hybrid inbox + 曝光表 | P2-5 | 与 5 串行（同文件） |
| 5 | `parity-p2-4-hot-cache` | 详情/主页 cache-aside + 失效广播 | P2-3 | 串行 |
| 6 | `parity-p2-6-outbox-es-reindex` | `wys_sync_outbox` + ES 别名/中文分词/重建 | P1-2、P0-6 | 独立 |
| 7 | `parity-p2-7-multi-instance-readiness` | ticker 锁 + SSE 限流 Redis 化 + WS 粘滞 + 容量基线 | P0-1、P1-4、P2-10 | 收尾片 |
| 8 | `parity-p2-8-engagement-gaps` | 收藏/转发/@/楼中楼/多话题 | P2-3、P2-5、P1-8 | 可拆多片 |
| 9 | `parity-p2-9-service-tx-orchestration` | `WithTx` 推广全域 + 计数/审计一致性 | P0-3 | 后置 |
| 10 | `parity-p2-10-integration-load-baseline` | testcontainers 集成测试 + 压测基线 + 覆盖率门禁 | P0-1 | 建议**提前**到 3 之前 |

> 建议实际执行序：P2-10 → P2-5 → P2-1 → P2-2 → P2-3 → P2-4 → P2-6 → P2-8 → P2-7 → P2-9。理由：没有压测基线（P2-10）就无法证明后续任何性能改动的收益。

## 各 Slice 任务拆解与验收标准

### P2-1 `parity-p2-1-media-presign-upload`
**任务**：新 `core/storage/`（provider: `oss|cos|s3|minio`，照 `core/jpush` 的 provider 风格）；配置 `storage.enabled` + bucket/region/CDN domain + AK/SK **仅 `.env.local`**；`POST /api/v1/uploads/presign`（校验 mime/大小/用户配额，返回 PUT URL + objectKey）→ `POST /api/v1/uploads/complete`（HEAD 核销存在性、算 sha256、落 `wys_media_objects`）；派生规格（缩略图/水印，优先用云侧处理能力而非自建）；`wys_posts`/`wys_short_videos` 增加 media 引用（多对多或 jsonb，替换单 URL 列）；读取侧统一走 CDN 域名 + 签名/防盗链；EXIF 与 GPS 剥离。
**验收**
- [ ] 真机（Flutter `my_ai_project`）选 9 图 + 1 段 60s 视频 → 发布成功 → Feed 与详情页可访问；**列表页拿到的是缩略图 URL**，不是原图。
- [ ] 字节流**不经过 BFF**：上传期间 :8080 无大 body 流量（抓包或 nginx/compose 日志佐证），这是设计成立与否的判据。
- [ ] 越权用例：A 拿 B 的 objectKey 调 complete → 拒绝；未 complete 的 objectKey 不可被引用。
- [ ] 校验用例：超 size 上限、伪装 mime（.jpg 实为 .exe）→ 拒绝。
- [ ] `storage.enabled=false` 时 posts 接口行为不变（不阻断现有客户端）。
- [ ] 代码内不再有 picsum/公开 mp4 兜底（`git grep picsum` 证据；历史数据在 record 里说明处置方式）。
- [ ] 无密钥泄漏：`make check-secrets` 通过，AK/SK 不出现在任何可提交文件与日志。

### P2-2 `parity-p2-2-moderation-pipeline`
**任务**：`app/internal/moderation/` 垂直切片；表 `wys_content_reviews(content_type,content_id,stage,verdict,reason,provider_raw jsonb,reviewer_id,created_at)`、`wys_reports`、`wys_user_risk(user_id,score,tags,restricted_until)`、`wys_sensitive_words`；`core/moderation/` provider（优先第三方机审：文本/图片/视频，配置 `moderation.enabled|provider`）；链路：发布 → 同事务写 `reviews(stage=machine,status=pending)` + outbox → Worker 调机审 → 结论回写 → 通过则可见 + 通知作者，可疑进人审队列；`wys_posts` 的 `review_status` 与业务 `status` **语义分离**；人审最小工作台接口（list/verdict，角色校验）；举报接口与处置；发帖/评论/私信频控规则落配置。
**验收**
- [ ] 端到端一次：发含违规文本的笔记 → 机审拒绝 → 作者收到通知 + 内容对他人不可见 + 库里留痕（`provider_raw` 有第三方原始响应）。
- [ ] **绕过测试**：直接调发布接口带违规内容，不绕过审核（审核在发布事务内，非事后补）；未过审内容在任何列表/搜索/详情路径均不可见（逐接口验证并列表记录）。
- [ ] 机审 provider 不可用时：**默认拒绝或默认待审**，不得默认放行（与 P0-4 同一红线）。
- [ ] 举报 → 处置 → 状态收敛可追溯（谁、何时、依据）。
- [ ] 日志留存满足合规口径（保留时长与访问控制写进文档；具体月数按 Q1 结论确定）。
- [ ] 审核结论回写走 outbox/P2-6 通道，机审超时/重复回调幂等。

### P2-3 `parity-p2-3-counter-cache-feed`
**任务**：互动计数改 Redis `INCR` + 定时回写 PG（替换 `community/repo_postgres.go:156-160` 的同事务 UPDATE）；热度榜改 Redis ZSET，`score = 权重/时间^衰减`（替换静态 `heat=like*2+comment`，`model/entity/community.go:50-53`）；`wys_feed_inbox(user_id,post_id,score,created_at)` 混合模式（普通作者写扩散，大 V 读扩散合并）；`wys_post_impressions` 曝光记录 + 同会话去重；配置 `feed.mode(pull|hybrid|push)` + `feed.window_hours`；`FeedProvider` 接口抽象（保留 pull 作降级）。
**验收**
- [ ] 计数不丢不飘：并发 200 次点赞 → 回写后 PG 值 = 200；Redis 宕机恢复后从 PG 重建，误差为 0（有实测数字）。
- [ ] 写锁热点消失：同一热门 post 的点赞 QPS 相比改造前提升 ≥5 倍（**压测数字**，依赖 P2-10 基线）。
- [ ] 关注一个 1 万粉账号后刷 Feed，P99 不随粉丝数线性恶化（前后对比数字）。
- [ ] 同一会话内不重复出现同一内容（曝光去重生效）；「不感兴趣」负反馈能降低后续出现概率。
- [ ] `feed.mode=pull` 降级开关可用，切回后行为与现状一致。
- [ ] 时间衰减生效：24h 前的高热帖不再永久霸榜。

### P2-4 `parity-p2-4-hot-cache`
**任务**：笔记详情/作者主页 cache-aside（`post:{id}:v{n}` 60s + 版本号）；写路径 bump 版本 + 通过现有 Redis Pub/Sub 广播失效（复用 `core/queue/pubsub_broadcaster.go` 通道风格，勿新造总线）；缓存击穿防护（singleflight——当前全仓 0 命中）；穿透防护（空值短 TTL）；`hot:` 榜单缓存；命中率/时延指标接入 P1-2。
**验收**
- [ ] 热点场景压测（同一 post 高并发读）：DB QPS 相比改造前下降 ≥80%，命中率 ≥90%（实测数字）。
- [ ] 编辑/删除后**其它实例**在 1s 内读到新值（多实例失效广播真的生效，单实例通过不算通过）。
- [ ] 缓存未命中且后端不可用时不雪崩：singleflight 生效，并发请求只回源 1 次（日志或指标证明）。
- [ ] 无脏读窗口导致的用户可见错误（抽查作者主页与详情一致性）。

### P2-5 `parity-p2-5-cursor-pagination-index-audit`
**任务**：列表类接口（Feed/评论/消息/通知/流水）改 keyset cursor（`(created_at,id) < $1` 风格），保留 page 参数兼容旧客户端；每页不再先 `Count(*)`（估算或仅首页返回）；深分页硬闸（`page*size <= 10000`）；索引审计与补齐：`wys_post_comments.user_id`、本地 `transactions.date`、`wys_topics.created_at` + 每个 `WHERE+ORDER BY` 组合对应一个索引（产出清单表）；ES 侧补 `search_after`。
**验收**
- [ ] 10 万行表翻到第 1000 页：P99 < 50ms（改造前对比数字，附 `EXPLAIN (ANALYZE, BUFFERS)` 计划）。
- [ ] 关键查询走索引：审计清单每条给出改造前/后的执行计划差异（不允许只看耗时）。
- [ ] 旧客户端（page 参数）不受影响；新 cursor 客户端翻页无重复/漏项（元素集合一致性校验）。
- [ ] 深分页攻击（`page=1000000`）被拒且返回可读错误，不打爆 DB。
- [ ] 全域无 `Count(*)` 在热路径（`git grep` + 慢查询日志双证）。

### P2-6 `parity-p2-6-outbox-es-reindex`
**任务**：`wys_sync_outbox(source_type,source_id,op,payload,status,retries,next_at)`，与业务写**同事务**入队；Worker 消费投 ES，指数退避、失败留痕、超阈进 DLQ（复用 P0-6）；ES 改别名（`posts_alias`/`transactions_alias`）替换硬编码索引名（`transaction/repo_es.go:20`）；mapping 与分词配置外置到文件（现内嵌 `:39-72`），中文接 `ik`/`smartcn`；提供全量 reindex + 对账脚本（DB 计数 vs ES 计数）；posts/videos 纳入索引范围（替代 community 的 ILIKE+trgm，`repo_postgres.go:318-331`）。
**验收**
- [ ] PG↔ES 最终一致：断掉 ES 后写数据 → 恢复后 outbox 重放补齐，对账脚本差异为 0（**这是与现状 best-effort 的根本区别**）。
- [ ] 中文搜索：搜「二手车」能命中（standard 分词下现在命中不了）。
- [ ] 零停机重建：改 mapping 后通过别名切换完成 reindex，期间 `/api/v1/transactions/search` 不 5xx。
- [ ] outbox 不积压：正常流量下 `max(age) < 5s`，并有指标（接 P1-2）。
- [ ] 重复消费不产生重复文档（幂等 id 策略）。
- [ ] 回退路径明确（ES 不可用 → 降级为 DB trgm 或 503，二选一并写进契约 P1-6）。

### P2-7 `parity-p2-7-multi-instance-readiness`
**任务**：整点 ticker 加 Redis 锁 + lease 续租（现每进程各跑一份，`core/queue/hourly_ticker.go`；文档自认无锁 `go-deep-dive/11:310`）；SSE 限流窗口挪 Redis（现进程内 `completion_usecase.go:311-327`）；WS 会话粘滞（网关配置 + `ws:ticket` 校验复用）；启动流程确认无 DDL（P0-1 已把关，本片做回归）；compose/文档给出 ≥2 实例运行方式与单实例容量数字（依赖 P2-10）。
**验收**
- [ ] 起 2 API + 2 Worker：整点通知**只推一份**（用户侧不重复，DB/日志计数为证）。
- [ ] 多实例下限流配额不放大（同 P1-4 结论在多实例下二次验证）。
- [ ] WS：客户端重连后消息不丢、不重复（seq 连续性校验，`realtime:seq` 现有机制）。
- [ ] 滚动重启（逐实例替换）期间无 5xx 尖峰、无掉线累积（P1-2 指标曲线为证）。
- [ ] gRPC `:9090` 与 HTTP `:8080` 双实例均正常，互不串（AGENTS.md 约束 6：端口勿混）。
- [ ] 输出「单实例容量基线」一页：QPS/P99/瓶颈点，写入 `docs/`。

### P2-8 `parity-p2-8-engagement-gaps`
**任务**：`wys_post_collects`、`wys_comment_likes`、`wys_post_shares`、`wys_mentions`；评论加 `parent_id`+`root_id`（楼中楼）；`wys_post_topics` 多对多（替换单列 `topic_id`，历史数据迁移）；每个新互动同步落三件套：计数（走 P2-3 机制）、频控（走 P2-2 频控）、通知（走 P1-8 站内信）；作者主页聚合接口。
**验收**
- [ ] 收藏/评论点赞/转发各有一次完整链路（写→计数→通知→多端已读）。
- [ ] 一条内容可挂**多话题**，历史单话题数据迁移后不丢（迁移脚本幂等，迁移前后计数一致）。
- [ ] 楼中楼分页正确（root + 子层各自游标）。
- [ ] @提及触发站内信且被 @人有限制（防骚扰）；重复 @ 不产生重复通知。
- [ ] 所有新计数字段与关系表在「并发 200 次」下无脏值（同 P2-3 断言方式）。

### P2-9 `parity-p2-9-service-tx-orchestration`
**任务**：把 P0-3 的最小 `WithTx` 推广为通用能力（含 context key、仓储「有事务则用」规范）；梳理全部跨仓储写点（`git grep` 列表）并逐个判定是否需要同事务；统一事务内禁止外部 IO 的规则与 lint/评审检查项；`project-detail.md` D8 结论回写。
**验收**
- [ ] 跨仓储写清单逐条有结论（需同事务 / 需 outbox / 可最终一致），写在 `docs/` 表格中。
- [ ] 抽查 3 条「中途失败」场景，数据库无中间态残留（测试断言）。
- [ ] 无事务内 HTTP/ES/MQ 调用（人工核对 + 评审检查项）。
- [ ] `model`/`common` 未因事务传递反向 import `app`/`core`（AGENTS.md 约束 10）。

### P2-10 `parity-p2-10-integration-load-baseline`
**任务**：真库集成测试（testcontainers 或本机 `make deps-up`，用 build tag 隔离）；补齐 0 测试包：`app/cmd/*`、`core/database`、`core/supabase`、`app/internal/delivery/worker`；压测脚本（`hey`/`k6`，放 `scripts/load/`）与基线场景：登录、发帖、Feed、搜索、下单-支付；覆盖率输出与门禁（只要求新增代码覆盖）；CI 跑（依赖 P1-5）。
**验收**
- [ ] 一条真实 PG/Redis/RabbitMQ/ES 的集成测试在 CI 通过（非内存 fake）。
- [ ] 上述 4 个 0 测试包**至少各有一条**有意义的测试（不是空壳 `TestXxx(t){}`）。
- [ ] 5 个基线场景有**数字**（QPS/P99/错误率）并存档为后续改动的对照组；这份基线是 P2-3/P2-4/P2-5 收益证明的前提。
- [ ] 覆盖率报告可输出，新增代码覆盖率有下限门禁。
- [ ] 集成测试默认不在 `make test` 中（需 tag 显式开启），避免本地误连生产库。

---

## P3（Deferred，不排期）

| 能力 | 前置 | 阻塞问题 |
|------|------|----------|
| 综合搜索（联想/热搜/聚合） | P2-6 | Q4 是否真做内容分发 |
| 推荐召回与排序 | P2-3（曝光数据） | 无曝光数据前改动无法评估 |
| AB 实验 | P1-2 | Q1 是否有真实流量（无流量则无意义） |
| 作者数据中心 | P2-3 | 同上 |
| 带货链路 / 优惠券 / 秒杀 | P0-2、P1-4 | Q5 是否真收钱 |
