# Program Plan — 2026-09-28 客服 Agent + 基建（修订）

## Milestone
| ID | Name | Goal | Out of scope |
|----|------|------|--------------|
| M1 | 观测全栈 | Prometheus + Grafana + Alertmanager + 飞书 webhook；进 `make lan-up`；BFF `/metrics` | 生产多集群联邦 |
| M2 | MQ 切换 | **RabbitMQ 替代 Asynq**；现有任务类型等价迁移；Redis 保留（session / Realtime / PubSub） | 多租户 vhost 治理 |
| M3 | 交易全文 | ES 索引 `transactions`（note/category/type 等）+ 搜索 API；写路径同步 | 全库表灌 ES |
| M4 | 客服 Agent | Eino 客服 + 复用 SSE 帧；Tool 可调交易搜索 | 人机工单中台 |

## Checklist snapshot
| Chapter | Status | Note |
|---------|--------|------|
| CONTEXT 术语 | ☐ | 见下 |
| Epic + Briefs | ✅ 修订草稿 | **等人批** |
| C1 观测 lan-up | ☐ | |
| C2 RabbitMQ | ☐ | 替换 Asynq |
| C3 ES 交易全文 | ☐ | |
| C4 Eino 客服 | ☐ | |

## CONTEXT 术语（草案）
| 术语 | 定义 | Avoid |
|------|------|-------|
| 交易全文 | 对 `transactions` 的 note/category/type 等字段的 ES 检索 | 不要当成「全站搜索」或仅 FAQ |
| 任务队列 | RabbitMQ 承载原 Asynq 任务 | 不要用 RabbitMQ 替换 Redis Pub/Sub（WS 扇出仍 Redis） |
| 观测栈 | Prometheus 抓取 + Grafana 看图 + Alertmanager 路由 + 飞书 webhook | 不要只交 `/metrics` 算 M1 完成 |
| 客服 Agent | Eino 编排的客服对话 | 不要与「AI 小石头」闲聊混为同一 usecase |

## Epic queue (ordered)
| # | Epic | Plan | Done when |
|---|------|------|-----------|
| 1 | cs-agent-infra | `plans/epics/cs-agent-infra.md` | C1–C4 Accept |

## 不做（Program 级）
- 用 Eino/RabbitMQ 替换 Gin
- 删除 Redis（session / ticket / PubSub 仍依赖）
- ES 同步全部业务表（仅 transactions；客服 FAQ 可后置）
- 未批 Brief 直接大改 `pkg/queue` / compose

## Risks Top3
| Risk | Mitigation |
|------|------------|
| Asynq→RabbitMQ 漏迁任务 / 定时广播无 Cron | C2 清单对照 `pkg/queue/tasks.go`；小时广播用 worker ticker 或 delayed exchange，写进 Brief |
| ES 与 RLS/user_id 泄漏 | 搜索强制会话 `user_id` 过滤；禁止 Admin 全库搜 |
| lan-up 镜像过重拖垮本机 | 单节点 ES；Prometheus retention 短；文档写内存下限 |

## Architecture 裁决（已按人确认修订）
| 议题 | 决定 |
|------|------|
| ES | **交易全文**（CRUD 同步索引 + 搜索 API） |
| MQ | **RabbitMQ 替代 Asynq**；Redis Pub/Sub 保留 |
| 观测 | **Prometheus + Grafana + Alertmanager + 飞书 webhook**，进 **`make lan-up`** |
| Eino | 客服 Agent（C4）；可 Tool 调交易搜索 |
| 环境 | 本地 Docker `lan-up` 为默认联调真相源 |

## Next knife
1. **人批**队首起 Brief（建议一次批 C1–C4，或至少 C1）
2. conductor → executor 自 C1 起

## Changelog
| Date | Change |
|------|--------|
| 2026-09-28 | 初稿：Eino 客服；ES=FAQ；RabbitMQ Deferred；仅 Prometheus |
| 2026-09-28 | **修订**：ES=交易全文；RabbitMQ 替 Asynq；Grafana/AM/飞书；lan-up |
