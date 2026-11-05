# Program Plan — 2026-11-03 对标小红书级后端补齐（xhs-parity）

> 输入：[`docs/xhs-benchmark-gap-analysis.md`](../docs/xhs-benchmark-gap-analysis.md)（差距与分级）
> 现状事实：[`docs/project-detail.md`](../docs/project-detail.md)（146 路由 / 55 实体 / 68 环境变量）
> 编排合同：[`.harness/rules/`](../.harness/rules/)（slice-contract · status-enums · no-silent-accept · dual-truth · module-boundary · auth-token-hygiene）
> **状态：Brief 已批并执行完 M1（P0 全 7 片，验收记录见 `docs/acceptance-records/2026-11-0{3,4}-parity-p0-*.md`）；M2/M3 未开工。**
> M1 残留 7 条（F1 Flutter 渠道支付契约、F2 membership 核销器、F3 对账单向、F4 IAP、F5 unifiedorder 响应验签、F6 真实 T+1 未实弹、F7 核销后不通知）逐条登记在
> [`parity-p0-2` 验收记录](../docs/acceptance-records/2026-11-04-parity-p0-2-payment-notify-ledger.md) §七 与 [`docs/placeholder-inventory.md`](../docs/placeholder-inventory.md)；
> 原第 8 条「membership 吞错补偿」已由 [`membership-money-tx-followup`](../docs/acceptance-records/2026-11-04-membership-money-tx-followup.md) 解除。

## Milestone

| ID | Name | Goal | Out of scope |
|----|------|------|--------------|
| M1 | 正确性与资金安全（P0） | DDL 出启动流程；支付回调闭环；资金同事务；假实现清零登记；MQ 不死循环；密钥治理 | 性能优化、新业务域 |
| M2 | 可观测与排障（P1） | request-id 贯通全链路；指标+告警成体系；CI 门禁；契约机器可读；备份能演练；站内信落地 | OTel 全量接入（M2 只做到「可接」）、K8s |
| M3 | 上量与性能（P2） | 媒体直传；内容审核；计数/Feed 改造；热点缓存；cursor 分页；outbox；多实例语义成立 | 微服务拆分、分库分表、自研排序模型 |
| M4 | 增长与商业化（P3） | 综合搜索体验、推荐召回、AB 实验、作者数据、带货与优惠券 | **本轮不排期**（等 Q1/Q4 回答） |

## Checklist snapshot

| Chapter | Status | Note |
|---------|--------|------|
| CONTEXT 术语 | ✅ 草案 | 见下，需人确认 Avoid 口径 |
| Program + 3 Epic | ☑ 已落盘 | `plans/epics/xhs-parity-p{0,1,2}-*.md`，**等人批** |
| P0 Slice（7 个）已落盘 | ☑ | `plans/slices/parity-p0-{1..7}-*.md`（含 ONLY/不做/验收/白名单/黑名单/验证命令） |
| P1 Slice（8 个） | ☐ | 任务与验收已写在 Epic §各 Slice；Brief 文件执行前由 planner 生成 |
| P2 Slice（10 个） | ☐ | 同上 |
| Q1–Q8 待确认 | ☐ | 见 `docs/xhs-benchmark-gap-analysis.md` §9，**Q1/Q4 决定 M3/M4 是否成立** |

## CONTEXT 术语（草案，含 Avoid）

| 术语 | 定义 | Avoid |
|------|------|-------|
| 假实现 | 代码路径存在但语义是占位/自动成功（返回 200 却没做事） | 不要把「简化实现」当假实现；`ponytail:` 注释标明天花板的是**已知降级**，需登记为 `Degraded` 而非 `Placeholder` |
| 迁移闸门 | schema 变更由 `migrate up` 在发布流程执行，API 启动只**校验版本** | 不要「保留 AutoMigrate 但加个开关」——每个实例启动跑 DDL 的竞态就是问题本身 |
| 支付闭环 | prepay → 渠道 → **notify 验签 → 流水落库 → 幂等核销 → 主动查单兜底 → T+1 对账** | 不要只补 notify 路由就称闭环；无 `wys_payment_records` 等于无可对账对象 |
| 资金同事务 | 扣积分/扣钱包/改订单状态在**一个**事务内提交或回滚 | 不要靠「失败后 Credit 回滚补偿」冒充事务；补偿路径本身正在吞错（`mall/service.go:548-553`） |
| request-id | 每请求一个 id，入口生成/透传、日志携带、出站透传、响应回写 | 不要以为等于 tracing；本 Program 先做 id，OTel 是后续替换动作 |
| 全局限流 | 跨实例共享配额（Redis 令牌桶），维度 = user/ip/route | 不要把现有 SSE 进程内窗口当限流（多实例 = 配额 × 实例数） |
| 内容审核 | 机审 + 人审双轨 + 举报处置 + 留痕 | 不要把 `home` 域的店务审批状态机复用为内容风控（语义不同） |
| 计数缓存 | 互动计数走 Redis 增量 + 定期回写 PG | 不要「加一层读缓存」而仍保留同事务 `like_count+1` UPDATE——写锁热点没解决 |
| 多实例语义成立 | 起 ≥2 个 API 实例后：无重复 DDL、无重复推送、限流不放大、WS 不丢线 | 不要等于「compose 改 replicas=2」 |
| 垂直切片 | 新域落在 `app/internal/<domain>/`（router+handler+service+repo 同目录） | 不要往 `internal/usecase` + `internal/repository` 老横切结构回退（阶段 3 已改切片，见 recent commits） |

## Epic queue（ordered）

| # | Epic | Plan | Done when |
|---|------|------|-----------|
| 1 | P0 正确性与资金安全 | [`plans/epics/xhs-parity-p0-correctness.md`](./epics/xhs-parity-p0-correctness.md) | 7 个 Slice 全 Accept；假实现清单**归零或全部登记 status+note**；生产启动不执行 DDL；一笔真单可对账 <br>→ **已达成前 3 条**；「一笔真单可对账」为 **Partial**：对账链路在真 PG + 真 RabbitMQ 上跑通，但渠道用测试密钥/桩网关，**没有真实商户的一笔单**（需线上凭据，非代码问题） |
| 2 | P1 可观测与排障 | [`plans/epics/xhs-parity-p1-observability.md`](./epics/xhs-parity-p1-observability.md) | 给一个 request-id 能 grep 出全链路；CI 门禁阻断 build/test/lint/密钥失败；有一次真实恢复演练记录 |
| 3 | P2 上量与性能 | [`plans/epics/xhs-parity-p2-scale.md`](./epics/xhs-parity-p2-scale.md) | 双实例并行运行满足「多实例语义成立」；真机 9 图 1 视频发布成功且经审核后分发；10 万行表 cursor 翻页 P99 < 50ms |
| 4 | P3 增长与商业化 | 未建 | **本轮不排期**（Q1/Q4 未答） |

## 执行顺序与并行建议

```text
P0-1 migration-gate ──┐
P0-7 secret-hardening ┤→ 先做（无前置依赖，且 P0-1 是 P2-7 的硬前置）
P0-5 fake-impl-inventory ┘
P0-3 money-tx-atomic → P0-2 payment-notify-ledger（P0-2 复用 P0-3 的事务入口）
P0-4 remove-fake-review → P0-5（假审核登记进清单）
P0-6 mq-dlx-retry（独立）
P1-1 request-id → P1-2 metrics-alerts（日志/指标共用关联字段）→ P1-4 ratelimit（限流需按 request-id 记录拒绝原因）
P1-5 ci-basic 可在 P0-1 之后立刻插队（CI 能挡住后续所有回归）
P2-5 cursor-pagination → P2-3 counter-cache-feed → P2-4 hot-cache（三者都动 community/repo，串行避免打架）
P2-9 service-tx-orchestration 依赖 P0-3 引入的 WithTx 入口
```

## 文件锁（Program 级热点）

| 热点文件 | 谁会改 | 冲突处置 |
|----------|--------|----------|
| `app/cmd/api/main.go` | P0-1（去 DDL）、P0-7（密钥校验）、P1-1（中间件）、P1-4（限流装配）、P2-4（缓存客户端）、P2-7（ticker 锁） | **必须串行**；每片改完即合，不留长分支 |
| `common/config/config.go` | 几乎每片（新配置项） | 同一 Epic 内串行；新配置一律 `yaml + BindEnv + 默认值` 三件套，照现有 `:647-716` 写法 |
| `app/internal/delivery/http/router/router.go` | P1-1、P1-4、P0-2（notify 需跳过 SessionAuth） | notify 路由注册放独立 `app/internal/payment/router.go`，别堆 router.go |
| `app/internal/mall/*` | P0-2、P0-3、P2-9 | 同 Epic 串行 |
| `app/internal/community/repo_postgres.go` | P2-3、P2-4、P2-5、P2-8 | 按 backlog 顺序，勿并行 |
| `core/queue/*` | P0-6、P2-6 | 串行 |
| `docker/docker-compose.yml` | P0-1、P0-7、P2-1、P2-6 | 每片只动自己那段，注意 `JWT_SECRET:94` |
| `Makefile` | P0-1（migrate 闸门）、P1-5（CI 入口）、P2-10（压测目标） | `.PHONY` 行（`:15`）易撞，改完立即合 |

## 不做（Program 级）

- **不拆微服务**（当前瓶颈是单点 DDL / 无缓存 / offset 分页，不是进程边界）
- **不上 K8s**（P0-1 未完成前上 K8s 只会放大 DDL 竞态）
- **不分库分表 / 不换 TiDB**（大表先用 Postgres 原生分区 + 归档）
- **不自研推荐模型、不引入向量库**（无曝光数据前任何排序改动无法评估；先用热度+时间衰减+关注加权）
- **不补齐全部 `wys_*` 的 RLS**（先回答 Q3：local Auth 还是 Supabase，再只补保留的那一半）
- **不做直播 / 广告**（与本仓库定位无关）
- **不动 6 个 4S 店务域**（used_car / new_car_follow / after_sales / deal_invoice / purchase_calculator / home）——盘点里算已完成，不占本 Program 优先级
- 不新增第三方消息中间件（**勿再引入 Asynq**，见 AGENTS.md 修改规范 9）

## 全局约束（每片 Slice 都必须遵守，源自 AGENTS.md）

1. 所有 `go` / `make` 在本仓库根目录执行
2. **多模块**：workspace 根无 `go.mod` → 用 `go build all` 或 `go -C app <cmd>`；`go build ./...` 已失效
3. **勿盲跑 `go mod tidy`**：缺依赖时它会拉最新版（实测 viper 1.19→1.21、zap 1.27→1.28、gorm 1.25.12→1.31.2）。加依赖必须在对应模块 `go.mod` 显式写 pinned 版本再 tidy；升级依赖单独一次改动 + 全量 `make test`
4. 模块边界：`model` / `common` **不得** import `app` / `core`
5. 新建业务表一律 `wys_` 前缀
6. `SUPABASE_SERVICE_ROLE_KEY` 仅 `.env.local`；每片合入前 `make check-secrets`
7. 用户数据过滤一律取会话 `userID`，勿信任客户端传的 `user_id`；PostgREST 必须 `WithUserToken`
8. HTTP `:8080` 与 gRPC `:9090` 勿混端口
9. **commit/push 只在人明确指令时**

## Risks Top 5

| # | 风险 | 影响 | 缓解 |
|---|------|------|------|
| R1 | P0-1 去 DDL 后，既有本地库 schema 可能与 `migrations/` 不一致（历史上四套机制各建了一部分） | 生产启动被版本校验挡住 | P0-1 内含「schema 对齐审计 + 幂等补建迁移」；本地提供 `make migrate-up` 逃生门（`Makefile:68-72` 已有） |
| R2 | P0-3/P0-2 触及资金路径，且 `mall`/`payment` 现有测试是内存 fake，真实 PG 行为未验证（`project-detail.md` §11.2：零真库测试） | 改完看似通过、线上算错钱 | P0-3 必须先补一条真库事务回滚集成测试（不等 P2-10），否则不得 Accept |
| R3 | 新增强制限流/幂等会**破坏 Flutter 现有调用**（客户端未传 `Idempotency-Key`） | 线上写接口大面积 4xx | 一律 `enabled` + `enforce(log_only\|reject)` 两档上线，先观察再收紧；跨仓改动需在 `my_ai_project` 立对应任务（Q8） |
| R4 | `.harness` 的机检目标（`make agent-pre` / `agent-post` / `agent-check-boundary`）**尚未在 Makefile 落地**（实测 grep 0 命中） | 「不得静默 Accept」缺机械证据，全靠人核对 | 明确写入验收：以 Brief 的「验证命令」人工执行为准；如需自动化，另立 Slice，不在本 Program 偷偷加 |
| R5 | 单 Program 跨度大（P0→P2 约 25 片），易变成「永远在做 P0」 | 排期失真 | 每片必须能独立合入独立可回滚；M1 Accept 后即对人汇报一次，再决定是否继续 M2/M3 |
