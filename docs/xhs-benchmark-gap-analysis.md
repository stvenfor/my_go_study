# my_go_study 后端能力对标整理（基准：小红书级大型实战项目）

> 本文只做**能力盘点与缺口整理**，不含代码改动。所有结论附 `路径:行号`；「不存在」类结论附检索关键词，见附录 B。
> 姊妹文档：[`project-detail.md`](./project-detail.md)（本仓库现状全量测绘，146 路由 / 55 实体 / 68 环境变量）。

## 1. 文档说明

| 项 | 内容 |
|----|------|
| 分析范围 | 本仓库全部 Go 模块（`app` / `common` / `core` / `model`）、`docker/`、`deploy/`、`migrations/`、`scripts/`、`Makefile`、`.githooks/` |
| 基准 | 小红书（UGC 社区 + 搜索推荐 + 私信 + 电商闭环 + 内容风控）级别的后端工程标准，而非「功能列表」级别 |
| 生成时间 | 2026-11-03 |
| 证据规则 | 结论 = 实测；未找到 = 给出 grep 关键词；不引用「应该怎样」的通用教条，除非标为**建议** |

### 1.1 分级口径（下文所有「当前级别」都用这套）

| 级别 | 含义 | 判定标准 |
|------|------|----------|
| **L0 无** | 仓库里不存在 | 关键词检索 0 命中 |
| **L1 雏形** | 单点实现，不成体系 | 只覆盖 1 个域 / 只在单进程内成立 / 有代码但无配置项 |
| **L2 单机可用** | 真实业务能跑通 | 有真逻辑、有持久化、有测试，但只在**单实例 + 低并发**下成立 |
| **L3 生产级** | 大型实战项目标准 | 多实例无状态、可观测、可灰度、可水平扩展、故障可自愈 |

**本仓库整体定位：业务域普遍 L1~L2，横切基建普遍 L0~L1。** 差距不在「业务功能少」，而在「业务功能底下没有生产级地基」——这是下文所有结论的主线。

### 1.2 待确认项汇总（Q1–Q8，见第 9 节）

Q1 是否有真实外部用户流量（决定 P0 优先级）· Q2 是否有浏览器端调用方（决定 CORS/限流口径）· Q3 是否要保留双 Auth 实现 · Q4 短视频/社区是否真要做内容分发 · Q5 支付是否需要走真实回调闭环 · Q6 目标部署形态（继续单机 compose 还是上 K8s）。

---

## 2. 结论速览

### 2.1 业务能力对标（UGC 闭环为主轴）

| # | 能力 | 当前 | 证据 / 缺口 |
|---|------|------|-------------|
| B1 | 文字 UGC 发布 | **L2** | `app/internal/community/`：发帖、删除、话题、计数实时更新（`repo_postgres.go:145-195`） |
| B2 | 图片/视频上传、转码、CDN | **L0** | 全仓 `FormFile\|multipart.` → 0 命中；媒体只存 URL 字符串，缺省填 picsum（`short_video/service.go:32-39`、`community/service.go:30-37`） |
| B3 | 内容审核 / 敏感词 / 风控 / 举报 | **L0（含一处假审核）** | 检索 `敏感词\|moderat\|antispam\|举报\|黑名单` → 0；短视频「审核」= 30 秒自动放行（默认延时 `short_video/service.go:52-58`，判定谓词 `:339-340`，入库即 `reviewing` `:160`） |
| B4 | Feed 分发（时间线/热度/关注） | **L1** | 只有 latest/hot/following 三个 SQL 排序（`community/repo_postgres.go:118-143`），**推模式 feed/inbox 表 0 命中**；关注流是拉模式子查询 |
| B5 | 推荐系统（召回/排序/曝光去重） | **L0** | `recommend\|推荐\|hot_score` → 0 有效命中；热度是静态公式 `like*2+comment`（`model/entity/community.go:50-53`） |
| B6 | 搜索 | **L1** | 社区搜索 = `ILIKE`+pg_trgm（`community/repo_postgres.go:318-331`）；ES 只服务 transactions（`transaction/repo_es.go:20`，索引名硬编码、中文无分词器） |
| B7 | 互动（赞/评/关注） | **L2** | 点赞/评论/关注有真实关系表与计数（`model/entity/community.go:25-84`） |
| B8 | 互动（收藏/转发/分享/@提及/评论点赞） | **L0** | `collect\|favorite\|收藏\|转发` 在 community 域 → 0 命中 |
| B9 | 私信 / IM | **L2** | 真融云 Server API + SHA1 签名（`core/rongcloud/client.go:64-127`），BFF 只管凭证与关系（`/api/v1/im` 15 条） |
| B10 | 站内信 / 通知中心 | **L1** | 无站内信表；仅 `sys.notify` WS 事件 + 整点广播；SMS 队列任务是 stub（`delivery/worker/handler.go:169-177`） |
| B11 | App 推送 | **L2** | 极光真 REST（`core/jpush/client.go:76-148`）；JPush 占位分支仍在（见 `project-detail.md` §10.4） |
| B12 | 支付闭环（预支付→回调→落库→对账） | **L1** | **`/api/v1/payments` 只有 `prepay` 一条路由**（`payment/router.go:37`），`out_trade_no` 生成即丢（`payment/service.go:84`），无 notify、无对账、无订单表 |
| B13 | 商城交易 | **L2** | 库存条件扣减（`mall/repo_postgres.go:322`）、幂等键（`migrations/20260921180000_mall_schema.up.sql:98`）；但选微信/支付宝渠道时不经 payment 域、仅本地标记已付（`mall/service.go:526-543`） |
| B14 | 内容 → 商品转化（笔记带货） | **L0** | `wys_posts` / `wys_short_videos` 无任何 goods/commerce 关联列；社区与商城两域互不引用 |
| B15 | 优惠券 / 营销 | **L0** | 无 coupon 目录与表 |
| B16 | 钱包 / 积分（复式账本） | **L2** | `wallet`/`points` 有复式账本 + 幂等 ref；退款路径吞错 `_, _ = Credit(...)`（`mall/service.go:548-553`） |
| B17 | 会员 / IAP 权益 | **L2** | Apple 验单真实（`membership/iap_apple.go:44-50`）；华为回调纯 stub 返回 `accepted:true`（`membership/handler.go:152-155`） |
| B18 | 实名认证 / KYC | **L0** | 无；手机号 OTP 为 dev 固定码且发送是空实现（`phone_otp_usecase.go:31-42`） |
| B19 | 直播 | **L0** | 无 |
| B20 | 广告 / 商业化 | **L0** | 无 |
| B21 | 数据分析 / BI | **L1** | gRPC AnalyticsService 通，但主体是演示种子（`usecase/analytics_usecase.go:37-49`） |
| B22 | AI 能力 | **L1** | SSE 默认 mock provider（`repository/llm/mock_stream_provider.go`，无 key 回退 mock `main.go:608-616`）；客服 Agent **完全不调 LLM**，模板拼接+伪流式（`cs_agent_usecase.go:98,153`） |

### 2.2 横切基建对标（大型项目的「地基」）

| # | 能力 | 当前 | 证据 / 缺口 |
|---|------|------|-------------|
| I1 | 限流 | **L1** | 仅 SSE 进程内每用户滑动窗口（`completion_usecase.go:311-327`、`main.go:352`）；`middleware/` 11 个文件**无 rate 中间件**（实测 `ls`）；多实例下窗口各自独立，形同无效 |
| I2 | 熔断 / 降级 / 超时治理 | **L0~L1** | `circuit\|breaker\|sentinel\|resilience` → 0；超时零散（`rongcloud/client.go:45`、`jpush/client.go:64` 有，`supabase/auth_admin.go:173` 用**无超时** `http.DefaultClient`） |
| I3 | 幂等 | **L1** | 仅商城下单有唯一索引幂等键（`mall_schema.up.sql:98,117` + `mall/service.go:271-275`）；发帖（`community/service.go:318`）、支付回调**无幂等**；无 `Idempotency-Key` / `X-Request-Id` 中间件 |
| I4 | 分布式锁 / singleflight | **L0** | `redsync\|SetNX-lock\|singleflight\|FOR UPDATE\|pg_advisory` → 0；唯一定时任务去重靠 Redis SetNX（`worker/handler.go:136`）；`mall.go:213` 的 `Version` 字段是**残留无实现** |
| I5 | 链路追踪 / request-id | **L0** | `otel\|jaeger\|trace_id\|request_id` → 0；访问日志无 trace 字段（`middleware/logger.go:22-30`）；`common/logger` 无 `FromContext`（跨请求串不起来） |
| I6 | 指标与告警 | **L1** | 3 个自定义指标（`common/metrics/metrics.go:16,23,31`）+ 3 条告警（`deploy/observability/alerts.yml:4,17,26`）；**无 DB 慢查询 / Redis / 队列积压 / 业务 P99** |
| I7 | API 契约与文档 | **L0** | `swagger\|openapi` → 0；proto 只有 analytics 1 个、无 `buf`；golangci-lint 配置文件 → 0 |
| I8 | 对象存储 / CDN | **L0** | OSS/COS/S3 SDK 调用 → 0（仅 `thunder` 里有配置结构体 `thunder/config/viper.go:481-496`，无调用）；`CONTEXT.md:127` 自证「无对象存储上传」 |
| I9 | 配置中心 / 热加载 / 密钥管理 | **L0** | `nacos\|consul\|vault\|RemoteProvider` → 0；viper 无 `WatchConfig`（fsnotify 仅 indirect）；硬编码兜底密钥 `config.go:335` |
| I10 | 多实例 / 无状态化 | **L1** | session 在 Redis ✓、Pub/Sub 扇出 ✓；但 WS Hub 会话状态、SSE 限流窗口、ticker 都是**进程内**，`docker-compose.yml` 每服务 `replicas=1`，无 LB/反代配置 |
| I11 | 部署编排与发布 | **L1** | `Dockerfile` 质量不错（两阶段 `:4/:45`、非 root `:55`、HEALTHCHECK `:66`）；`k8s\|helm\|systemd` → 0 文件；无灰度/蓝绿；compose 里 `JWT_SECRET` 默认 `change-me-in-production`（`docker-compose.yml:94`） |
| I12 | CI/CD | **L0** | `.github/` 目录不存在；`.gitlab-ci\|Jenkinsfile` → 0；3 个本地 hook 只查密钥与中文首行，**不跑 test/lint**（`.githooks/pre-push:5`） |
| I13 | 测试基建 | **L1** | 66 个 `_test.go` / 34 包，全靠手写内存 fake；`testcontainer\|sqlmock\|miniredis` → 0；`app/cmd/*`、`core/database`、`core/supabase`、`delivery/worker` **0 测试**；无覆盖率、无压测脚本（`vegeta\|k6\|wrk` → 0） |
| I14 | 优雅退出 | **L1** | HTTP 侧完整（`main.go:574-592`，10s 超时 + gRPC GracefulStop）；**Worker 只 `cancel()`，不等在途消息**（`cmd/worker/main.go:132-135`）；`log.Fatal` 会跳过 defer（`project-detail.md` §9.7） |

### 2.3 数据与存储层对标

| # | 能力 | 当前 | 证据 / 缺口 |
|---|------|------|-------------|
| D1 | 迁移治理 | **L1** | **四套并存**：AutoMigrate 白名单（`main.go:637-651`，13 实体）+ 8 处 `EnsureSchema` + 13 个内嵌 SQL + `migrations/`（45 个文件）；建表失败只 `Warn`（`main.go:185`）；无 goose/migrate 工具闸门 |
| D2 | 索引覆盖 | **L1** | 商城/users/transaction 有像样索引（`mall_schema.up.sql:116-218`）；缺 `wys_post_comments.user_id`（`community.go:77`）、本地 transactions 的 `date`（查询 `ORDER BY date DESC` 在 `transaction/repo_postgres.go:55`） |
| D3 | 分页 | **L1** | 全域 offset（`shared/domain_helpers.go:28-39`），每页先 `Count(*)`（`community/repo_postgres.go:114`）；**无 cursor/keyset、无深分页上限** |
| D4 | 查询效率 | **L1** | 亮点：批量 in-query 防 N+1（`community/service.go:599-628`）；问题：全仓 **0 处 Preload/Joins**、几乎无字段投影（`Select(` 无命中，全字段含 text/jsonb）、stats 多次独立 COUNT（`used_car/repo_postgres.go:25-51`） |
| D5 | 缓存层 | **L0** | Redis 只用于 session/WS ticket/presence/SSE 会话/realtime seq（5 类，见 `project-detail.md` §6.6），**无任何 DB 前置缓存、无热点缓存**；`cache\|CacheKey` → 0 |
| D6 | 读写分离 / 分库分表 | **L0** | `ReadReplica\|sharding\|vitess\|tidb` → 0 |
| D7 | 跨存储一致性 | **L0** | PG↔ES 纯 best-effort，失败只打指标不回滚（`transaction/service.go:93-115`，注释自认无 outbox）；`outbox\|对账\|reconcil` → 0；只有 `BackfillPostHeat` 一个回填函数 |
| D8 | 事务边界 | **L1** | `Transaction(` 30+ 处**全在仓储层**，service 层 0 处 → 跨仓储写无事务：`mall/service.go:520` 扣积分、`:535` 扣钱包、再更新订单状态是三个独立事务，**中途失败 = 钱扣了单没付** |
| D9 | 软删除 / 数据治理 | **L1** | `gorm.DeletedAt` 0 处，全靠手写 `DeletedAt *time.Time` + `deleted_at IS NULL`；无归档/冷数据；**手机号/邮箱/收件电话明文**（`mall_schema.up.sql:103`），无 `aes\|encrypt`，仅展示层 `maskPhone`（`im_friend_usecase.go:284`） |
| D10 | 行级安全 RLS | **L1** | 只有 transactions 有策略（`supabase/migrations/003,004`），其余 `wys_*` 全无 POLICY；Supabase 模式下靠应用层过滤兜底；`auth_admin.go:31-71`、`cmd/import-supabase/main.go:65` 是 service_role 绕过路径 |
| D11 | 搜索基建（ES） | **L1** | mapping 内嵌代码（`repo_es.go:39-72`）、索引名硬编码 `transactions`、**无别名/无 reindex 流程**、中文分词缺失（standard tokenizer）、无 `search_after` |
| D12 | 备份 / 慢查询 / 容量 | **L0** | `pg_dump\|backup\|binlog\|log_min_duration` → 0；连接池有配（`core/database/postgres.go:28-30`），Redis 无池参数（`redis.go:15` 仅 Addr）；PG 挂 = 进程直接退出，无降级 |
| D13 | 消息可靠性 | **L1** | 手动 ack + prefetch（`core/queue/consumer.go:50-51,87`），但失败 `Nack(false,true)` **无限重投**（`:100`）、无 DLX / 无重试上限、发布无 publisher confirm（`client.go:64`） |

## 3. UGC 主链路逐环节详解（小红书最核心的那条线）

小红书的骨架是一条链路，本仓库在**第 2、3、5 环直接断掉**：

```text
发布 → 媒体处理 → 审核风控 → 存储/索引 → 分发(Feed+推荐) → 互动 → 搜索 → 私信/评论 → 电商转化
 ✅      ❌          ❌           ⚠️            ⚠️           ✅      ⚠️         ✅            ❌
```

### 3.1 环节 2：媒体上传与处理 —— **当前 L0，优先级最高**

| 项 | 内容 |
|----|------|
| 现状 | 接口只接收 URL 字符串；无 URL 时塞 picsum 占位图与公开 mp4（`short_video/service.go:32-39`）。小红书内容 90% 是图，**这一环不通，整条链路是演示** |
| 缺什么 | ① 直传签名（STS/预签名 PUT）② 上传回调/落库 ③ 图片多规格（缩略图/水印）④ 视频转码与截帧 ⑤ CDN 域名与防盗链 ⑥ 大小/类型/EXIF 与 GPS 脱敏 |
| 落地清单 | 新配置 `storage.enabled` + `storage.provider(oss\|cos\|s3\|minio)` + bucket/AK 走 `.env.local`；新目录 `core/storage/`（对齐 `core/jpush` 的 provider 风格）；新路由 `POST /api/v1/uploads/presign`、`POST /api/v1/uploads/complete`；新表 `wys_media_objects(id, user_id, object_key, mime, size, width, height, status, sha256)`；`wys_posts` 加 `media_ids jsonb`（当前只有单 URL 列） |
| 关键设计 | **客户端直传，BFF 只签发与核销**——不要把字节流过 :8080，否则单机 BFF 立刻成为带宽与 FD 瓶颈 |
| 验收 | 真机选 9 图 + 1 段 60s 视频，发布后 CDN 可访问、列表页拿到的是缩略图 URL |

### 3.2 环节 3：内容审核与风控 —— **当前 L0，且有一处「假审核」比缺失更危险**

| 项 | 内容 |
|----|------|
| 现状 | 短视频把 `status=reviewing` 用 **30 秒延时自动置为通过**（`short_video/service.go:52-58`（默认 30s）、`:339-340`（超时即判通过）、`:160`（入库置 reviewing））；社区发帖零检查（`community/service.go:318,375`）。审核状态机在 `home` 域是**店务审批**语义，不是内容风控 |
| 为什么必须有 | UGC 平台合规红线（国内：先审后发/机器+人工双轨/日志留存 ≥6 个月/举报处置时限）。假审核在真实内容下 = 直接上线违规内容 |
| 缺什么 | ① 机审（文本敏感词、图片鉴黄/涉政、视频抽帧）② 人审工作台队列 ③ 举报与处置 ④ 用户信用分与黑名单 ⑤ 频控（发帖/评论/私信次数）⑥ 设备指纹与批量注册对抗 |
| 落地清单 | 新表 `wys_content_reviews(content_type, content_id, stage(machine\|human), verdict, reason, provider_raw jsonb, reviewer_id, created_at)`、`wys_reports(id, target_type, target_id, reporter_id, reason, status)`、`wys_user_risk(user_id, score, tags, restricted_until)`、`wys_sensitive_words`；`wys_posts` 增加 `review_status` 与 `visibility`（现在的 `status` 语义要与审核解耦）；新目录 `app/internal/moderation/`；配置 `moderation.enabled` + `moderation.provider`；机审走第三方（阿里云绿网/数美）→ 放 `core/moderation/`；发帖链路改成「写 posts → 同事务写 reviews(stage=machine,pending) → 入队」 |
| 关键设计 | **状态机 + outbox**，不要像短视频那样用定时任务改状态。审核结论回调/轮询回来后，同事务更新 `posts.review_status` + 广播给作者（复用 `realtime` 的 `sys.notify`） |
| 立刻要做 | 即便不接机审，也应把短视频的 30s 自动放行**改成默认拒绝或人工放行**——这是假实现，不是简化实现 |

### 3.3 环节 5：Feed 分发 —— **当前 L1，缺「推/拉」架构决策**

| 项 | 内容 |
|----|------|
| 现状 | 三个 Tab 全靠 SQL 排序（`community/repo_postgres.go:118-143`）：latest=时间、hot=`heat DESC`、following=子查询拉关注列表 |
| 问题 | ① 关注流 `user_id IN (SELECT followee_id ...)` 在大 V（百万粉丝）下是**反向查询灾难**；② `heat = like*2+comment`（`model/entity/community.go:50-53`）是实时列 UPDATE，热点行会形成**单行写锁**（`repo_postgres.go:156-160`）；③ 热度无时间衰减，老内容永久霸榜；④ 无曝光记录 → 无法去重、无法算 CTR、无法做冷启动 |
| 缺什么 | ① 收件箱模型（写扩散 `wys_feed_inbox(user_id, post_id, score, created_at)` 给普通用户，大 V 走读扩散 + 混合）② Redis ZSET 存 hot 榜（时间衰减 score）③ 曝光/已读表 ④ 计数走 Redis 增量 + 定期回写 PG（现在同事务 UPDATE 是**点赞 QPS 上限的直接来源**）⑤ 负反馈（不感兴趣） |
| 落地清单 | 新表 `wys_feed_inbox`、`wys_post_impressions(post_id,user_id,slot,created_at)`；新配置 `feed.mode(pull\|hybrid\|push)` + `feed.window_hours`；`community/repo_postgres.go:118-143` 的排序逻辑抽成 `FeedProvider` 接口（先只切出 hybrid，保留 pull 作降级）；热度改为 Redis ZSET `score = 权重/时间^衰减` |
| 验收 | 关注一个 1 万粉账号后刷 Feed 的 P99 不随粉丝数线性变差；同一条内容在同一会话不重复出现 |

### 3.4 环节 6：互动补齐 —— **L0 的收藏/转发**

| 项 | 内容 |
|----|------|
| 现状 | 有 `wys_post_likes`、`wys_post_comments`、`wys_user_follows`（`model/entity/community.go:25-84`） |
| 缺什么 | 收藏（小红书用户主行为，比点赞权重高）、转发带评、@提及、评论点赞、评论盖楼（现在只有平铺 comment）、作者主页聚合、访客记录 |
| 落地清单 | 新表 `wys_post_collects`、`wys_comment_likes`、`wys_post_shares`、`wys_mentions`；评论加 `parent_id`+`root_id`（多级楼中楼）；话题加多对多 `wys_post_topics`（**当前 `topic_id` 是单列外键，一条内容只能挂一个话题**，与小红书多话题标签不符） |
| 注意 | 每加一个互动，就要同步加：计数缓存策略（见 3.3 ④）、防刷频控（见 3.2 ⑤）、消息通知（见 B10）三件事。缺任一件就是半成品 |

### 3.5 环节 12~14：搜索 / 推荐 / 商业化转化

| 能力 | 现状 | 缺口与最小落地 |
|------|------|----------------|
| 综合搜索 | L1：社区走 pg_trgm，ES 只索引 transactions | 把 posts/videos 纳入 ES：新建索引 + **别名 `posts_alias` + reindex 流程**（当前索引名硬编码 `repo_es.go:20`，无法平滑重建）；中文接 `ik`/`smartcn` 分词（standard 分词下中文检索基本失效）；搜索类型聚合（综合/用户/话题/笔记）+ `search_after` 深分页 |
| 推荐 | L0 | 分三步，**不要一上来就自研排序模型**：① 热度+时间衰减+关注加权（可用 Redis ZSET 实现）② 标签/协同过滤召回（离线任务算 `wys_similar_posts`）③ 特征与实时模型。前提是先有**曝光表**（3.3 ④），否则无法评估任何改动 |
| 电商转化 | L0：社区与商城互不引用 | `wys_posts` 加 `goods_ids jsonb`、新表 `wys_post_goods`；笔记详情页返回商品卡；订单归因字段 `wys_mall_order_item.source_type/source_id`（内容带货的计量起点）。这是小红书 GMV 链路的核心，不是「附加功能」 |

## 4. 横切基建补齐（决定「能不能上量」）

小红书级后端的差别不在接口数量，而在下面这 8 件地基。按「不补就会出事故」排序。

### 4.1 I5 链路追踪（request-id）—— **所有排障的前提，建议第一个做**

| 项 | 内容 |
|----|------|
| 现状 | 访问日志字段固定为 `status/method/path/query/ip/latency/ua`（`middleware/logger.go:22-30`），**无 request-id、无 trace**；`common/logger` 无 context 传递（`FromContext` 0 命中）；`otel` 0 命中 |
| 后果 | 一次 Flutter 报错，无法把「客户端这一次请求」对应到服务端 41 步启动后的任意一行日志；多实例后彻底无法排障 |
| 最小落地 | ① 新中间件 `middleware/request_id.go`：入口取 `X-Request-Id` 否则生成 ULID，`c.Set` + **回写响应头** + 注入 `context`；② `common/logger` 加 `logger.WithContext(ctx)`（zap 的 `context` field）；③ 全局链在 `router.go` 最前注册（现有链见 `project-detail.md` §9.1，9 个中间件、28 处 `Use`）；④ 出站 HTTP 与 MQ 载荷透传该 id（`core/rongcloud`、`core/jpush`、`core/supabase` 三个 client 各一处）；⑤ 之后再上 OTel（`otelhttp` + `otelgrpc` + gorm 插件），因为**已经有 id 了，接 OTel 只是把 id 换成 trace_id** |
| 验收 | 给一个 `X-Request-Id`，`grep` 能串起 HTTP → usecase → repo → MQ → Worker → 推送 全链路 |

### 4.2 I1+I3 限流与幂等 —— **写接口的两道闸门**

| 子项 | 现状 | 落地 |
|------|------|------|
| 入口限流 | 只有 SSE 每用户内存滑动窗口（`completion_usecase.go:311-327`），**多实例失效** | 全局中间件 + Redis 令牌桶（按 `user_id` / `ip` / `route` 三维），配置 `ratelimit.enabled` + `per_user_rps` + `per_ip_rps` + 白名单；返回 429 + `Retry-After`；用 `golang.org/x/time/rate` 做本地兜底 + Redis 做全局（当前 `x/time` 只是 indirect，**要加依赖必须显式 pin 版本再 tidy**，见 AGENTS.md 全局约束 9） |
| 业务频控 | 无发帖/评论/私信次数限制 | 内容类接口独立配额（发帖 10/日、评论 30/h、私信首条限制），与入口限流分层——**配额是产品/风控规则，限流是系统保护** |
| 幂等 | 只有商城（`mall_schema.up.sql:98`） | 中间件 `Idempotency-Key`：Redis `SETNX idem:{user}:{key}` + DB 唯一索引兜底；**强制覆盖**：`POST /payments/prepay`、`POST /community/posts`、`POST /mall/orders`、评论、关注、点赞（点赞已有唯一键则只需文档化） |
| 幂等关键细节 | — | 键由**客户端生成**（现有 mall 就是这么做的，`mall/handler.go:243`），服务端只核销；重试要返回**首次结果**而非 409，否则客户端无法安全重试 |

### 4.3 I2 熔断降级与超时收敛

| 项 | 内容 |
|----|------|
| 现状 | 零散超时：`rongcloud/client.go:45`、`jpush/client.go:64`、`payment/service.go:69`、`membership/iap_apple.go:27` 有；**`supabase/auth_admin.go:173` 用无超时 `http.DefaultClient`**——Supabase 抖动会挂死登录链路 |
| 落地 | ① 统一 `common/httpclient`：默认 3s Dial / 10s Total / 连接池上限，所有出站 client 走它；② 每个外部依赖包一层 `circuit`（连续 N 次失败开熔断，半开探测），熔断打开时返回**明确的业务码**而非 500；③ 降级矩阵：ES 已有哨兵 `ErrSearchUnavailable` → 503（`transaction/handler.go:309`）是**全仓最好的一个降级样板**，把它推广到图片/推荐/搜索；④ `context` 超时贯通（`project-detail.md` §9.7 记录了 context 缺口） |

### 4.4 I6 指标与告警补齐

| 现有什么 | 缺什么 | 落地 |
|----------|--------|------|
| 3 个指标（`common/metrics/metrics.go:16,23,31`）+ 3 条告警（`deploy/observability/alerts.yml`） | ① 队列积压与处理时延 ② DB 慢查询与连接池使用率 ③ Redis 命中率 ④ 外部依赖成功率（融云/极光/微信/支付宝）⑤ 业务漏斗（发帖成功率、支付成功率、推送到达）⑥ WS 在线数与广播扇出规模 | GORM 侧现在只有 Info 日志（`core/database/postgres.go:17`）→ 加 `prometheus.NewHistogram` 的 gorm 回调 + `DBStats()` collector；Worker 侧在 `delivery/worker/handler.go` 每个任务包一层 histogram + 失败 counter；RabbitMQ 积压直接抓管理口 `/api/queues`（`core/queue/client.go` 已连上，缺 export）；**告警规则补 6 条**（积压 > 1k、外部依赖错误率 > 5%、支付成功率下跌、P99、进程重启、磁盘） |

### 4.5 I10 多实例化 —— **当前架构的真实天花板**

| 组件 | 现状 | 是否阻碍水平扩展 |
|------|------|------------------|
| Auth session | Redis ✓（`auth:session:*`） | 可扩 |
| 实时广播扇出 | Redis Pub/Sub ✓（`core/queue/pubsub_broadcaster.go`） | 可扩 |
| **WS Hub 连接** | 进程内 map + `ws:ticket` 校验 | **可扩**（ticket 在 Redis，连接粘滞即可），但需要**网关层会话粘滞**配置，仓库里没有 |
| SSE 限流窗口 | 进程内滑动窗口 | **不可扩**（每实例独立配额，实际放行 = 配额 × 实例数） |
| 整点 ticker | 每进程各跑一份，无分布式锁（`core/queue/hourly_ticker.go`；文档自认 `go-deep-dive/11:310`） | **会重复推送**，必须加锁或独占 leader |
| AutoMigrate/EnsureSchema | **每个 API 实例启动都跑建表**（`main.go:107`、`main.go:637-651`） | 多实例并发 DDL = 锁冲突/竞态，**这是最硬的多实例阻塞点** |
| compose | 每服务 `replicas=1` | — |

落地顺序：① DDL 从启动流程剥离（`migrate up` 作为发布前置闸门，API 启动只校验 schema 版本）② ticker 加 Redis 锁 + lease ③ SSE 限流挪 Redis ④ 网关层配 WS 粘滞 ⑤ 压测确定单实例容量，再谈 replicas。

### 4.6 I12+I11 CI/CD 与发布

| 项 | 现状 | 最小落地 |
|----|------|----------|
| CI | `.github/` 不存在；hook 不跑测试（`.githooks/pre-push:5` 只查密钥） | 加一条 workflow：`go build all` → `go vet all` → `go test ./...`（多模块用 `go -C <mod>` 或 `go test all`）→ golangci-lint → 镜像构建。**注意 workspace 根无 `go.mod`，`./...` 已失效**（AGENTS.md 约束 8） |
| Lint | 无 `.golangci.yml` | 引入，规则对齐 `.cursor/rules`；`gofmt` + `vet` 先行，别一次上 40 个 linter |
| 密钥扫描 | `check-secrets.sh:13` 只匹配 `sb_secret_*`，**抓不到默认弱密钥**（`project-detail.md` H11） | 加通用熵检测 + 明确禁止 `change-me-in-production`（compose `:97` 现在是默认值） |
| pre-push | 不跑测试 | 加定向测试（改哪个模块跑哪个），全量放 CI |
| 部署 | 单机 compose，无 LB、无灰度 | 先补 **compose 的 healthcheck 全覆盖**（Worker 现在关了健康检查 `docker-compose.yml:155`）+ 一条 `docker compose pull && up -d` 的回滚脚本；K8s 在没有多实例需求前不必上 |

### 4.7 I7 API 契约

| 项 | 内容 |
|----|------|
| 现状 | 无 swagger/openapi（0 命中）；契约散落在 8 个手写 md（如 `docs/community-posts-api.md`）+ `project-detail.md` §7 的 146 条表 |
| 后果 | Flutter 侧字段对不上只能靠联调；无契约测试，改 DTO 无破坏性检测 |
| 落地 | ① 优先做**机器可读**：`swag` 注解生成 openapi.json，或直接手写 `openapi.yaml`（146 条路由已有清单，成本可控）；② CI 跑 `spectral` lint + 与 Flutter 端 codegen diff；③ proto 侧引入 `buf`（现在手写 protoc，`Makefile:17-24`），加 breaking change 检测 |

### 4.8 I14 可靠性收尾（低成本高收益）

| 项 | 证据 | 动作 |
|----|------|------|
| MQ 无限重投 | `core/queue/consumer.go:100` `_ = d.Nack(false, true)`（返回值也被丢弃） | 加重试计数头 + 上限 + **DLX 死信队列**（当前 `QueueDeclare` args=nil，`client.go:32`） |
| 无 publisher confirm | `client.go:64` | 入队失败当前无痕；开启 confirm 或对关键任务落 outbox |
| Worker 不等在途消息 | `cmd/worker/main.go:132-135` | `cancel()` 后等 consumer close + `WaitGroup` |
| `log.Fatal` 跳过 defer | `project-detail.md` §9.7 | 启动失败路径改成返回 error 再统一退出 |
| Redis 无池参数 | `core/database/redis.go:15` | 补 `PoolSize`/`MinIdleConns`/`DialTimeout` |

## 5. 数据与存储演进（决定「数据不会错、不会丢」）

### 5.1 D1 迁移治理：**四套建表机制必须先收敛**

这是本仓库**唯一一个「不做就会在上线当天出事」**的项：现在每个 API 实例启动都跑 AutoMigrate/EnsureSchema（`main.go:107`、`:637-651`；8 处 `EnsureSchema`；13 个内嵌 SQL；`migrations/` 45 个文件），失败只 `Warn`（`main.go:185`）。

| 步骤 | 动作 |
|------|------|
| 1 | 以 `migrations/` 为**唯一真相**，`migrate` CLI 已在 Makefile（`:68-72`），把 `up` 变成发布前置闸门（CI/compose `depends_on: condition: service_completed_successfully`） |
| 2 | AutoMigrate 与 EnsureSchema 加 `APP_ENV=local` 守卫，**生产启动禁止 DDL** |
| 3 | 补 `20260924100000_rongcloud_im` 缺失的 down 文件 |
| 4 | 启动改成「schema 版本校验不通过 → 拒绝启动」，而不是 Warn 继续 |

### 5.2 D3+D2 分页与索引

| 问题 | 证据 | 动作 |
|------|------|------|
| 全域 offset 分页 | `shared/domain_helpers.go:28-39`；每页先 `Count(*)`（`community/repo_postgres.go:114`） | 列表类（Feed/评论/消息）改 **keyset cursor**（`(created_at,id) < $1`）；`Count` 换估算（`relcard`/`pg_class.reltuples`）或首页才返回总数 |
| 深分页无上限 | 无 `page` 深度校验 | 加 `page*size <= 10000` 硬闸；对外只允许 cursor |
| 缺索引 | `wys_post_comments.user_id`（`community.go:77`）、本地 transactions 的 `date`（`ORDER BY date DESC` 在 `transaction/repo_postgres.go:55`）、`wys_topics.created_at` | 迁移补索引；建立「每个 `WHERE`+`ORDER BY` 组合都要有对应索引」的审计清单 |

### 5.3 D5 缓存层 —— **当前完全没有**

Redis 只用于 5 类会话/事件 key（`project-detail.md` §6.6），零 DB 前置缓存。小红书级读放大下这是第一性能瓶颈。

| 缓存对象 | 策略 | 失效 |
|----------|------|------|
| 笔记详情 / 作者主页 | cache-aside，`post:{id}` 60s + 版本号 | 编辑/删除时广播失效（复用现有 `realtime:*` Pub/Sub 通道） |
| Feed 列表 | Redis ZSET 分桶（时间/热度），按用户或按话题 | TTL + 新内容入队更新 |
| 计数（赞/评/收藏/浏览） | **Redis INCR + 定时回写 PG**，替换当前「同事务 UPDATE 计数列」（`community/repo_postgres.go:156-160`） | 回写任务幂等；读时合并 |
| 话题/配置/热搜 | 长 TTL + 手动刷新 | — |

> 为什么把计数从「同事务 UPDATE」挪走是**性能项而非优化项**：热门笔记的所有点赞都撞同一行的写锁，单条内容的点赞 QPS 有硬上限，且这个 UPDATE 就在发帖域的主事务里。

### 5.4 D7+D8 一致性与事务边界 —— 资金链路的洞

| 风险 | 证据 | 动作 |
|------|------|------|
| **扣钱与改单不在同一事务** | `mall/service.go:520`（积分 Debit）、`:535`（钱包 Debit）与订单状态更新是三个独立仓储事务 | service 层引入 `WithTx(ctx, fn)`（把 `*gorm.DB` 放 context 传递），跨仓储写共享同一事务 |
| **退款吞错** | `mall/service.go:548-553`、`:579`、`:583` `_, _ = Credit(...)` | 失败必须落库 + 告警 + 重试任务；资金操作的错误**不允许无痕** |
| PG↔ES 无补偿 | `transaction/service.go:93-115`（注释自认无 outbox） | 新表 `wys_sync_outbox(source_type, source_id, op, payload, status, retries)`，同事务写入；Worker 消费并指数退避；补一个对账/reindex 任务 |
| 支付无回调闭环 | `payment/router.go:37` 只有 `prepay`；`out_trade_no` 生成即丢（`service.go:84`） | 见 5.5 |

### 5.5 支付与账号安全（对标小红书的「钱与证」）

| 缺口 | 现状证据 | 必须补 |
|------|----------|--------|
| 无支付回调 | `/api/v1/payments` 仅 `prepay`，**无 notify 路由**；`membership/ConfirmBuyout` 不校验支付直接 `MarkOrderPaid`（`membership/service.go:279`） | ① `POST /api/v1/payments/notify/{channel}`（微信/支付宝验签，**跳过 SessionAuth**）② 支付流水表 `wys_payment_records`（channel、out_trade_no、amount、status、raw jsonb）③ 幂等核销 ④ 主动查单兜底 + T+1 对账任务 ⑤ 金额用整数分，不用 float |
| 渠道与商城脱节 | `mall/service.go:526-543`：选微信/支付宝时不经 payment 域，直接 `PayOrderLocal` 标记已付 | 商城支付必须委托 payment 域，否则无流水可对账 |
| 华为 IAP 回调是 stub | `membership/handler.go:152-155` 直接返回 `accepted:true` | 真验签 + 落库，否则丢单 |
| 手机号 OTP 是空实现 | `phone_otp_usecase.go:31-42`（发送 no-op）+ dev 固定码 | 生产路径接真短信通道；固定码必须有 `APP_ENV=local` 守卫（现在 dev 配置里就开着） |
| PII 明文 | `mall_schema.up.sql:103` `receiver_phone` 明文；`encrypt\|aes` 0 命中；仅展示层 `maskPhone`（`im_friend_usecase.go:284`） | 存储加密（AES-GCM，密钥走 KMS/env）+ 审计日志 + 手机号唯一索引改用 HMAC 盲索引 |
| 无实名认证 | B18 | UGC + 电商在国内是硬要求（发帖/开播/提现均需） |
| D10 RLS 只覆盖一表 | 仅 `supabase/migrations/003,004`；其余 `wys_*` 无 POLICY | Supabase 模式下其余表靠应用层 `.Eq("user_id", userID)` 兜底 —— **要么补齐策略，要么明确放弃 Supabase 模式**（见 Q3） |

### 5.6 D12 备份与容量（当前 0）

`pg_dump|backup|binlog|log_min_duration` 全仓 0 命中。最小集合：每日全量 + WAL 归档 + **恢复演练脚本**（没演练过的备份等于没有）；Postgres 开 `log_min_duration_statement=200ms`；慢查询与容量（表大小、索引膨胀）进 Grafana。

## 6. 其余业务域补齐（按「小红书有、本仓库没有」列，但分级取舍）

| 能力 | 小红书里的角色 | 本仓库 | 建议 |
|------|----------------|--------|------|
| 站内信 / 通知中心 | 赞评收藏@的聚合入口，是留存主驱动 | **L1**：无站内信表，只有 WS `sys.notify` + 整点广播 | **P1 必做**：新表 `wys_notifications(user_id,type,target,payload,read_at)` + 未读数计数（Redis）+ 列表/已读接口；推送改为「站内信落库 → 再决定是否 Push」，取代现在直接推的路径 |
| 举报与处置 | 合规必需 | L0 | 随审核一起做（3.2） |
| 用户标签/兴趣画像 | 推荐召回输入 | L0 | 随推荐 P2 做，先只做「关注话题 + 历史互动」的粗标签 |
| 收藏夹/专辑 | 用户资产，收藏行为的主要载体 | L0 | P2 |
| 关注/粉丝列表分页与互相关 | 社交基础 | L1（有 `wys_user_follows`，无粉丝列表分页与互关标记的完整实现） | P1 |
| 作者数据中心 | 创作者留存 | L0（analytics 是演示种子 `analytics_usecase.go:37-49`） | P2，先把曝光/互动真实埋点存下来（依赖 3.3 ④） |
| 优惠券 / 满减 / 秒杀 | GMV 主要杠杆 | L0 | **P3**；秒杀另需库存预热与限流（依赖 4.2） |
| 直播带货 / 广告投放 | 商业化 | L0 | **不做**（本仓库定位不是它） |
| 直播 | 内容与电商放大器 | L0 | **P3/不做**，依赖媒体链路 |
| 多语言/国际化、无障碍 | 出海必需 | L0 | 若不出海则不做 |
| 灰度/AB 实验平台 | 推荐与产品迭代地基 | L0 | P2 起，先做一个极简 `wys_experiments` + 分桶中间件，不自研平台 |

---

## 7. 分阶段路线图

**排序原则**：先正确性（钱/数据不会错）→ 再可运维（出事能看见能查）→ 再上量（多实例/缓存/异步）→ 最后才是增长功能。

### P0 正确性与资金安全（阻塞上线）

| # | 事项 | 涉及位置 | 完成判据 |
|---|------|----------|----------|
| P0-1 | DDL 从启动剥离，迁移成为发布闸门 | `main.go:107`、`:637-651`、8 处 EnsureSchema、`migrations/` | 生产启动不执行任何 DDL；schema 版本不符则拒绝启动 |
| P0-2 | 支付回调闭环 + 流水表 + 对账 | `payment/router.go`、`payment/service.go:84`、`mall/service.go:526-543`、`membership/service.go:279` | 一笔真单从 prepay → notify → 落库 → 订单状态推进，重复回调不重复发货 |
| P0-3 | 资金操作同事务 + 退款不吞错 | `mall/service.go:520,535,548-553,579,583` | 扣积分/扣钱包/改单一个事务；失败可重试且留痕 |
| P0-4 | 干掉假审核（默认拒绝或人审） | `short_video/service.go:48-58,338-383` | 无「超时自动通过」路径 |
| P0-5 | 干掉假实现清单（华为回调、SMS stub、OTP 固定码、mock 图/视频 URL） | `membership/handler.go:152-155`、`worker/handler.go:169-177`、`phone_otp_usecase.go:31-42`、`short_video/service.go:32-39` | 每一项要么真接、要么显式 `501 not implemented`，不返回成功 |
| P0-6 | MQ 重试上限 + DLX + publisher confirm | `core/queue/consumer.go:100`、`client.go:32,64` | 毒丸消息进死信而非无限循环 |
| P0-7 | 弱密钥治理 | `docker-compose.yml:94`、`config.go:335`、`check-secrets.sh:13` | 启动时校验密钥强度；CI 能抓 `change-me` |

### P1 可观测与排障（出事能看见）

| # | 事项 | 完成判据 |
|---|------|----------|
| P1-1 | request-id 中间件 + logger context 贯通（4.1） | 一个 id 串起 HTTP→DB→MQ→Worker→推送 |
| P1-2 | 指标补齐 + 6 条新告警（4.4） | Grafana 有队列积压、外部依赖成功率、业务漏斗四盘 |
| P1-3 | 统一出站 httpclient + 超时收敛（4.3） | 无 `http.DefaultClient` 裸调用 |
| P1-4 | 全局限流 + `Idempotency-Key`（4.2） | Redis 桶，多实例下配额不放大 |
| P1-5 | CI（build/vet/test/lint/密钥）+ pre-push 跑测试（4.6） | 主干每次提交自动跑 |
| P1-6 | openapi 契约 + buf | Flutter 端 DTO 与后端 diff 可自动检查 |
| P1-7 | 备份与恢复演练、慢查询日志（5.6） | 演练过一次真实恢复 |
| P1-8 | 站内信/通知中心（第 6 节） | 未读数准确、多端已读同步 |

### P2 上量与性能（能扛真实流量）

| # | 事项 | 完成判据 |
|---|------|----------|
| P2-1 | 媒体直传 + CDN + 多规格（3.1） | 真机 9 图 1 视频发布成功，列表拿缩略图 |
| P2-2 | 内容审核（机审+人审+举报）（3.2） | 违规内容不出现在 Feed，有处置日志 |
| P2-3 | 计数与 Feed 改造：Redis 计数 + hybrid 收件箱（3.3、5.3） | 点赞不再撞同一行写锁；关注大 V 后 Feed P99 稳定 |
| P2-4 | 热点缓存 cache-aside + 失效广播（5.3） | 详情页 DB QPS 下降一个数量级 |
| P2-5 | cursor 分页 + 索引审计（5.2） | 10 万行表翻页 P99 < 50ms |
| P2-6 | outbox + ES 重建/别名 + 中文分词（5.4、D11） | PG↔ES 最终一致，可无损重建索引 |
| P2-7 | 多实例：ticker 锁、SSE 限流挪 Redis、WS 粘滞（4.5） | 起 2 个实例无重复推送、无掉线 |
| P2-8 | 互动补齐（收藏/转发/@/楼中楼/多话题）（3.4） | 与小红书主行为对齐 |
| P2-9 | service 层事务编排能力（`WithTx`） | 跨仓储写有统一事务入口 |
| P2-10 | 集成测试（testcontainers）+ 压测基线 | 有一条真实 PG/Redis 的回归与容量数字 |

### P3 增长与商业化

搜索体验升级（综合搜索/联想/热搜榜）、推荐召回与排序、AB 实验、作者数据中心、优惠券与带货链路（3.5）。

## 8. 明确「现在不该做」的（避免把对标读成照抄）

对标小红书**不等于**把它的架构抄一遍。以下几项在本仓库当前阶段做，是纯负债：

| 看起来该做 | 为什么现在不做 | 什么时候做 |
|-----------|----------------|-----------|
| 微服务拆分 | 现在是 `app/cmd/{api,worker}` 双进程单体；模块边界（`app/core/model/common`）已经是垂直切片，拆进程只会引入 RPC、注册发现、分布式事务三座山，而当前流量瓶颈全在**单点 DDL、无缓存、offset 分页** | P2 完成后，若某域（推荐/审核/媒体）容量特征明显不同，再按域拆 |
| K8s / Helm | 无 replicas 需求时，compose + 迁移闸门 + 备份就够；K8s 换来的是 HPA/滚动发布，而滚动发布当前被 P0-1（启动跑 DDL）**直接阻塞**——不解决 DDL 上 K8s 更危险 | P0-1 + P2-7 之后 |
| 自研推荐模型 / 向量库 | 没有曝光与点击数据（3.3 ④ 缺）之前，任何排序改动都无法评估 | 先埋曝光，用热度+时间衰减 + 关注加权吃到 80% 收益 |
| 分库分表 / TiDB | 单表最大的是流水与计数类，靠分区表（Postgres native partition）+ 归档即可；分片键会污染全部查询代码 | 出现单表 > 5 亿行或写 QPS 单机饱和 |
| 服务网格 / 全链路 Sentinel | 熔断降级目前只有 4 个外部依赖（融云、极光、微信/支付宝、Supabase），包一层 `circuit` 接口足够 | 外部依赖 > 10 个 |
| 配置中心 Nacos/Vault | 68 个环境变量 + 8 份 yaml 目前是「改配置要重启」，但热加载本身不是瓶颈；先解决密钥治理（P0-7） | 出现需要灰度开关/多环境秒级下发 |
| 补齐 RLS 到全部 `wys_*` | 若决定走 local Auth（现状默认），RLS 是死代码 | 先回答 Q3，再决定补哪一半 |

另一条判断线：**本仓库有 6 个 4S 店务域**（used_car / new_car_follow / after_sales / deal_invoice / purchase_calculator / home）与小红书无关。它们的路由、表、测试在盘点里都算「已完成」，但**不应**占用上面的优先级。

---

## 9. 待确认项（这些答案会改变上面的排序）

| # | 问题 | 影响 |
|---|------|------|
| Q1 | 是否有**真实外部用户**在用（还是个人/面试演示项目）？ | 若为演示：P0 只保留 P0-1/P0-4/P0-5（假实现清单），其余降级；若真上线：P0 全部为阻塞项 |
| Q2 | 是否有**浏览器端**调用方？ | 决定 CORS（`project-detail.md` M1 记录的 `X-Session-ID` 未授权问题）与限流是否按 IP 维度 |
| Q3 | Auth 双实现（local vs Supabase）保留哪个？ | 决定是否补 RLS（5.5）、是否清 `core/supabase` 与 PostgREST 一大片代码 |
| Q4 | 社区/短视频是否**真要做内容分发**？ | 决定 3.1~3.5 整块是否进路线图（否则本文一半作废） |
| Q5 | 支付是否要真实资金流？ | 若只做演示，P0-2 降为「返回明确未实现」；若真收钱，P0-2/P0-3 是最高优先级 |
| Q6 | 目标部署形态？ | 单机 compose / 云上多实例 / K8s，决定 P2-7 与是否引入 LB 与会话粘滞 |
| Q7 | 媒体与审核是否接第三方（OSS+绿网）还是自建？ | 第三方能把 3.1/3.2 从「周级」降到「天级」 |
| Q8 | `my_ai_project`（Flutter）是否同步改造？ | 上传直传、cursor 分页、幂等键、站内信都要求客户端配合改契约 |

---

## 附录 A：本文的维护约定

1. 现状类结论**不长期保鲜**——本仓库路由/表数量增长快（见 `project-detail.md` §7、§6）。复核方式：重跑附录 B 的检索 + `make test-all` 确认无测试项仍为 0。
2. 新增能力时**同步更新第 2 节的分级表**，不要只写代码；分级从 L0→L1 的判据是「有配置开关 + 至少一个真实调用方」，L1→L2 是「有测试且有失败路径处理」，L2→L3 是「多实例下语义仍成立」。
3. P0 项完成后在本节打勾并附 commit：`P0-1 ☐ / P0-2 ☐ / P0-3 ☐ / P0-4 ☐ / P0-5 ☐ / P0-6 ☐ / P0-7 ☐`。
4. 本文与 `project-detail.md` 的分工：**detail = 现状事实**（改代码时查），**本文 = 差距与取舍**（排期时查）。不要在本文重复现状细节，引用它。

## 附录 B：「不存在」类结论的检索方式（可复现）

> 统一命令：`grep -rn --include="*.go" --include="*.yml" --include="*.yaml" --include="*.sql" -iE '<关键词>' app common core model docker deploy migrations scripts`
> ⚠️ zsh 下 `--include=*.go` **必须加引号**，否则报 `no matches found`（本次实测踩过）。

| 结论 | 检索关键词 | 命中 |
|------|-----------|------|
| 无对象存储/上传 | `FormFile`、`multipart.`、`SaveFile`、`oss.`、`cos.`、`s3`、`minio` | 仅业务列名 `UploaderUserID` |
| 无内容审核/风控 | `敏感词`、`sensitive`、`moderation`、`antispam`、`绿网`、`risk_control`、`举报`、`黑名单` | 0 |
| 无推荐 | `recommend`、`推荐`、`hot_score` | 仅 gRPC 生成注释 |
| 无收藏/转发 | `collect`、`favorite`、`收藏`、`转发`（community 域内） | 0 |
| 无 Feed 表 | `feed`、`timeline`、`inbox` | 0（只有 SQL 排序 Tab） |
| 无分布式锁 | `redsync`、`redlock`、`DistributedLock`、`singleflight`、`FOR UPDATE`、`pg_advisory` | 0 |
| 无链路追踪 | `opentelemetry`、`otel`、`jaeger`、`zipkin`、`TraceID` | 0 |
| 无 API 契约 | `swagger`、`openapi`、`buf.yaml`、`.golangci` | 0 |
| 无 CI | `.github/` 目录、`.gitlab-ci`、`Jenkinsfile` | 目录不存在 |
| 无 K8s | `k8s`、`helm`、`Chart`、`systemd`、`nginx`、`traefik` | 0 文件 |
| 无缓存层 | `ReadReplica`、`sharding`、`CacheKey`、`cache` | 0 |
| 无 outbox/对账 | `outbox`、`compensat`、`reconcil`、`对账`、`rebuild` | 仅 `BackfillPostHeat` |
| 无集成测试/压测 | `testcontainer`、`sqlmock`、`miniredis`、`vegeta`、`k6`、`wrk`、`coverprofile` | 0 |
| 无备份 | `pg_dump`、`backup`、`binlog`、`log_min_duration` | 0 |
| 无字段加密 | `aes`、`encrypt`、`kms` | 0（仅展示层 `maskPhone`） |
| 无支付回调 | `payment/router.go` 全量 + `notify`、`callback` | 仅 `prepay` 一条 |





