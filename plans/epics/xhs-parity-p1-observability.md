# Epic — xhs-parity-p1-observability（可观测与排障）

> Program: [`plans/2026-11-03-xhs-parity-program.md`](../2026-11-03-xhs-parity-program.md)
> 差距来源: [`docs/xhs-benchmark-gap-analysis.md`](../docs/xhs-benchmark-gap-analysis.md) §4.1–§4.7、§5.5（PII）、§5.6、§6（站内信）、§7-P1
> Brief 状态：**尚未落盘**（`plans/slices/parity-p1-*.md` 由 planner 在执行前按本文各节生成）

## 差距表

| 能力 | 现状（证据） | 目标 | Slice |
|------|--------------|------|-------|
| 链路追踪 / request-id | 无；日志字段固定 `status/method/path/query/ip/latency/ua`（`middleware/logger.go:22-30`）；`common/logger` 无 context 传递 | 一个 id 串起 HTTP→usecase→repo→MQ→Worker→外部调用，并回写响应头 | P1-1 |
| 指标与告警 | 3 指标（`common/metrics/metrics.go:16,23,31`）+ 3 告警（`deploy/observability/alerts.yml:4,17,26`） | 队列积压/DB/Redis/外部依赖/业务漏斗 + 6 条新告警 | P1-2 |
| 超时与降级 | 零散超时；`supabase/auth_admin.go:173` 用无超时 `http.DefaultClient` | 统一出站 client + 熔断 + 降级矩阵（以 ES 哨兵为样板） | P1-3 |
| 限流与幂等 | 仅 SSE 进程内窗口（`completion_usecase.go:311-327`）；幂等仅商城 | Redis 全局令牌桶（user/ip/route）+ `Idempotency-Key` 中间件 | P1-4 |
| CI/CD | `.github/` 不存在；hook 不跑测试（`.githooks/pre-push:5`） | 门禁：build all + vet + test + lint + 密钥扫描 | P1-5 |
| API 契约 | 无 swagger/openapi；proto 无 buf；8 份手写 md | 机器可读 openapi + breaking 检查 | P1-6 |
| 备份与容量 | `pg_dump/backup/log_min_duration` 检索 0 命中 | 每日备份 + 恢复演练 + 慢查询日志 + 容量看板 | P1-7 |
| 站内信/通知中心 | 无表；仅 WS `sys.notify` + 整点广播；SMS stub | `wys_notifications` + 未读数 + 已读同步 | P1-8 |

## Slice backlog（有序）

| # | ID | ONLY 摘要 | 依赖 | 文件锁要点 |
|---|-----|-----------|------|-----------|
| 1 | `parity-p1-1-request-id` | 中间件 + logger context + 出站透传 | P0-* 合入后 | `middleware/`、`common/logger/`、`router.go`、`main.go` |
| 2 | `parity-p1-2-metrics-alerts` | gorm/redis/queue/外部依赖指标 + 6 告警 + 3 dashboard | P1-1 | `common/metrics/`、`deploy/observability/`、`core/queue/` |
| 3 | `parity-p1-3-outbound-client` | `common/httpclient` + 全部出站改造 + 熔断/降级矩阵 | P1-2（降级需指标可见） | `common/httpclient/`、`core/{rongcloud,jpush,supabase}`、`app/internal/{payment,membership}` |
| 4 | `parity-p1-4-ratelimit-idempotency` | Redis 令牌桶中间件 + `Idempotency-Key` | P1-1、P1-3 | `middleware/`、`core/database/redis.go`、`configs/` |
| 5 | `parity-p1-5-ci-basic` | GitHub Actions + golangci + pre-push 跑测试 | 可与 1–4 并行 | `.github/`、`.golangci.yml`、`.githooks/`、`Makefile` |
| 6 | `parity-p1-6-openapi-contract` | openapi.yaml + spectral + buf breaking | P1-4（新中间件头字段需入契约） | `docs/api/`、`app/api/proto/`、`buf.yaml` |
| 7 | `parity-p1-7-backup-drill` | pg_dump + WAL 归档 + 恢复演练脚本 + 慢查询 | 独立 | `docker/`、`scripts/backup/`、`deploy/` |
| 8 | `parity-p1-8-notification-inbox` | `wys_notifications` + 列表/已读 + 未读数 | P0-1、P1-1 | 新域 `app/internal/notification/`、`model/entity/`、`migrations/` |

## 各 Slice 任务拆解与验收标准

### P1-1 `parity-p1-1-request-id`
**任务**：入口中间件取/生成 `X-Request-Id`（ULID）→ `c.Set` + 注入 `context` + 回写响应头；`common/logger` 加 `WithContext`/`WithRequestID`，zap 以字段携带；访问日志与业务日志同字段；全局链置于最前（现有 9 个中间件、28 处 `Use`，见 `project-detail.md` §9.1）；`core/{rongcloud,jpush,supabase}`、`payment` 出站请求带该头；RabbitMQ 载荷 header 写入该 id，Worker 消费时恢复；gRPC interceptor 同步（metadata `x-request-id`，注意 AGENTS.md 约束：gRPC 鉴权走 metadata，键名小写）。
**验收**
- [ ] 带自定义 `X-Request-Id: smoke-123` 调 `POST /api/v1/user/login` → 响应头回显同值；`docker logs` grep `smoke-123` 命中该请求产生的**全部**日志行。
- [ ] 触发一次异步链路（`make test-queue-push`，需 API+Worker+RabbitMQ）→ 同一 id 同时出现在 API 侧与 Worker 侧日志。
- [ ] 不带该头时服务端自动生成，且客户端可从响应头取到（Flutter 侧可据此排查）。
- [ ] 无 request-id 的日志行仅剩启动期日志（`grep -c` 统计写入 record）。
- [ ] `common/logger` 不引入对 `app` 的依赖（模块边界：`model`/`common` 不得 import `app`/`core`）。
- [ ] gRPC 路径 `AnalyticsService` 同样携带（metadata）。
- [ ] 无性能回归：`/metrics` 的 `http_request_duration_seconds` P99 与改造前同量级（附数字）。

### P1-2 `parity-p1-2-metrics-alerts`
**任务**：gorm 回调 histogram（慢查询阈值、错误率）+ `DBStats()` collector（打开/空闲连接）；Redis 命令 histogram + 池指标（配合 D12 Redis 无池参数的问题，`core/database/redis.go:15`）；RabbitMQ 积压抓管理口 `/api/queues` 暴露 gauge；外部依赖成功率/时延 counter+histogram（rongcloud/jpush/wechat/alipay/supabase/es）；业务 counter（发帖成功率、支付成功率、推送到达）；`deploy/observability/alerts.yml` 补 6 条规则；Grafana 新增 3 个 dashboard（依赖健康 / 队列 / 业务漏斗）。
**验收**
- [ ] `curl -s http://127.0.0.1:8080/metrics` 可看到新增指标族（列出名称，AGENTS.md 指标检查项）。
- [ ] 每条新告警规则都有**一次真实触发**证据（人为制造：停 ES 看搜索降级、停 RabbitMQ 看积压归零、注入慢查询）。
- [ ] `amtool check-config deploy/observability/alerts.yml` 通过；Alertmanager 路由到飞书 webhook 成功收到一条测试告警。
- [ ] Grafana dashboard 导入后无「No data」空面板。
- [ ] 指标基数受控：新增 label 不含 user_id/post_id 这类高基数字段（人工核对并写进 record）。

### P1-3 `parity-p1-3-outbound-client`
**任务**：新建 `common/httpclient`（默认 3s Dial / 10s Total / 连接池上限 / keepalive），全部出站调用改用它——重点 `core/supabase/auth_admin.go:173`（现为无超时 `http.DefaultClient`）；每个外部依赖包一层最小熔断（连续 N 次失败开、半开探测），熔断打开返回**明确业务码**而非 500；降级矩阵照 `transaction/handler.go:309` 的 `ErrSearchUnavailable → 503` 样板推广到图片/推荐/外部 IM；`context` 超时贯通（现缺口见 `project-detail.md` §9.7）。
**验收**
- [ ] `git grep -n "http.DefaultClient"` 在业务代码 0 命中。
- [ ] 单一超时配置点（`common/httpclient`），`configs` 可覆盖；无 per-client 手填 timeout 的散落写法。
- [ ] 故障演练：把 Supabase/融云域名指向黑洞 → 请求在 10s 内失败并返回可读业务码，**不挂死**、不占满 goroutine（`/debug/pprof/goroutine` 数量稳定，附数字）。
- [ ] 熔断半开恢复后自动闭合（人证或测试）。
- [ ] 登录链路（本地 Auth）无回归：`POST /api/v1/user/login` 正常。

### P1-4 `parity-p1-4-ratelimit-idempotency`
**任务**：Redis 令牌桶中间件，维度 = user / ip / route，配置 `ratelimit.enabled` + `per_user_rps` + `per_ip_rps` + `route_overrides` + 白名单；429 + `Retry-After`；`enforce = log_only | reject` 两档（R3 风险：客户端未改造前不能直接拒）；业务频控独立（发帖/评论/私信配额，属产品规则）；`Idempotency-Key` 中间件（Redis SETNX + DB 唯一索引兜底，重试**返回首次结果**而非 409），覆盖 `POST /payments/prepay`、`/community/posts`、`/mall/orders`、评论、关注。
**验收**
- [ ] 多实例验证：起 2 个 API 实例，同一用户配额**不翻倍**（这是替换 SSE 进程内窗口的核心目的）；SSE 限流改造后同样结论成立。
- [ ] 压 100 QPS 单用户 → 超出部分 429 且带 `Retry-After`；未超部分正常；`log_only` 档下只记录不拒绝（先在 dev 跑一天）。
- [ ] 幂等：同一 `Idempotency-Key` 并发 5 次下单 → 只创建 1 单，5 次响应一致。
- [ ] 未带 key 的旧客户端**仍可正常工作**（向后兼容，Flutter 未改造不阻断）。
- [ ] CORS 白名单加 `Idempotency-Key` 与 `X-Request-Id`（现白名单 `Origin, Content-Type, Authorization, Accept` 不含 session 头，`project-detail.md` M1 记录该冲突；顺带处理并说明 Q2 结论）。
- [ ] 限流/幂等命中情况在 `/metrics` 可见（与 P1-2 指标打通）。

### P1-5 `parity-p1-5-ci-basic`
**任务**：`.github/workflows/ci.yml`：`go build all` → `go -C <mod> vet` → `go test`（多模块，注意 workspace 根无 `go.mod`，`./...` 失效）→ golangci-lint → `make check-secrets` → 镜像构建（不推送）；新增 `.golangci.yml`（先 gofmt/govet/errcheck/ineffassign/staticcheck，别一次上 40 个）；`.githooks/pre-push` 加定向测试（现只跑密钥检查 `:5`）；主干保护 + 必须状态检查。
**验收**
- [ ] 一次真实 PR 上 CI 跑通全绿；一次故意 break（编译错误 / 测试失败 / 塞 `change-me` 密钥）被 CI **拦住**。
- [ ] CI 明确验证「不跑逐模块 `go mod tidy`」（防止拉最新版，AGENTS.md 约束 9）——例如 `go mod verify` + 检查 `go.sum` diff。
- [ ] `golangci-lint` 初次引入的告警数为**基线记录**，后续 PR 不得新增（不要求一次清零）。
- [ ] CI 时长 < 5 分钟（超了则拆并行 job）。
- [ ] 本地 `git push` 被 hook 拦下的场景有一次截图/记录。

### P1-6 `parity-p1-6-openapi-contract`
**任务**：产出 `docs/api/openapi.yaml`（146 路由，起点用 `project-detail.md` §7 的清单）；两套响应信封与 10 个业务码（含 10021/10022/10023/10024）显式建模；`spectral` lint 进 CI；proto 侧引入 `buf`（替换手写 `Makefile:17-24`）+ `buf breaking`  against main；WS 6 种入站帧与 64KB/32-buffer 限制、SSE 帧格式补进契约文档。
**验收**
- [ ] openapi 校验通过；与真实路由**逐条 diff**：`docs/project-detail.md` §7.1 的 146 条中不允许有「文档有代码无」或「代码有文档无」。
- [ ] 一次故意改坏响应字段名 → CI breaking 检查报错（证明门禁有效）。
- [ ] `make proto` 与 `buf generate` 产物一致（无漂移）；Flutter 侧 `commons/network` stub 同步方式在文档写明（跨仓，遵守 Q8）。
- [ ] `/api/v1/transactions/search` 的「路由无条件注册、无 ES 时 503」契约写明（避免读成 404，`project-detail.md` C3）。

### P1-7 `parity-p1-7-backup-drill`
**任务**：`scripts/backup/`：每日 `pg_dump -Fc` + WAL 归档 + 保留策略；**恢复演练脚本**（恢复到临时实例并跑一致性校验）；Postgres 开 `log_min_duration_statement=200ms`；表大小/索引膨胀/连接数进 Grafana；Redis 快照策略确认（session 可丢，`auth:session` 丢失=全员重登，需明确 RPO）；备份文件加密与访问控制；恢复责任人/步骤写进 `docs/`。
**验收**
- [ ] **有一次真实演练记录**：从备份恢复到新实例，`/health` 200、登录可用、抽查 3 张表行数与备份前一致（没演练过的备份等于没有）。
- [ ] 演练文档含耗时（RTO）与数据丢失窗口（RPO）实测数字。
- [ ] 备份产物中不含明文密钥（`make check-secrets` 对备份目录同样跑一次）。
- [ ] 慢查询日志生效且至少捕获到一条（配合 P2-5 索引审计取证）。
- [ ] 备份任务失败会告警（接 P1-2 通道）。

### P1-8 `parity-p1-8-notification-inbox`
**任务**：新垂直切片 `app/internal/notification/`（勿回退到 `internal/usecase` 老横切结构）；`wys_notifications(id,user_id,type,target_type,target_id,payload jsonb,read_at,created_at)` + 索引 `(user_id, created_at desc)`、`(user_id, read_at)`；互动事件产生站内信（先接 community 点赞/评论/关注）；未读数走 Redis 计数 + 兜底 SQL；接口 `GET /api/v1/notifications`（cursor 分页，复用 P2-5 前先在域内实现 keyset）、`POST /notifications/{id}/read`、`POST /notifications/read-all`；推送策略改为「先落站内信 → 再决定是否 Push」，替代现在直接推；WS `sys.notify` 作为在线即时通道。
**验收**
- [ ] A 给 B 点赞 → B 站内信 +1、未读数 +1；B 已读后归零；**多端一致**（WS 在线时实时收到，离线时下次拉取补齐）。
- [ ] 重复互动不产生重复未读（同一用户对同一内容的点赞→取消→再点，计数不飘）。
- [ ] 新表遵守 `wys_` 前缀 + 迁移进 `migrations/`（不由 AutoMigrate 建，P0-1 已把关）。
- [ ] 用户隔离：A 读不到 B 的通知（按会话 `userID` 过滤，勿信任客户端传的 user_id）。
- [ ] 未读数在 1 万条通知下 P99 < 30ms（有实测数字，不是估算）。
- [ ] Flutter 侧对标页/接口需求已记录（跨仓任务，Program R3/Q8）。
