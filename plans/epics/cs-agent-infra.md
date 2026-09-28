# Epic — cs-agent-infra

> Program: `plans/2026-09-28-cs-agent-infra-program.md`（**2026-09-28 修订版**）

## 差距表

| 能力 | 现状 | 目标 | Slice |
|------|------|------|-------|
| 任务队列 | Asynq + Redis | **RabbitMQ** 承载任务；Redis Pub/Sub 仍做 WS 扇出 | C2 |
| 交易搜索 | Postgres 过滤 | **ES 全文**（note/category/type…）+ 按 `user_id` 隔离 | C3 |
| 观测 | 无 | Prometheus + Grafana + Alertmanager + **飞书 webhook**，`lan-up` | C1 |
| 客服 | 仅小石头 SSE | Eino 客服 Agent + SSE 帧 | C4 |

## Slice backlog（有序）

| # | ID | ONLY 摘要 | 依赖 |
|---|-----|-----------|------|
| 1 | `cs-agent-c1-observability-lan` | `/metrics` + compose：Prometheus/Grafana/Alertmanager + 飞书 receiver；lan-up 默认拉起 | 无 |
| 2 | `cs-agent-c2-rabbitmq-replace-asynq` | RabbitMQ 进 lan-up；迁移 tasks；去掉 Asynq 运行时依赖（文档标注） | C1 非硬依赖，建议先 C1 |
| 3 | `cs-agent-c3-es-transactions` | ES 进 lan-up；交易写同步索引；搜索 API SessionAuth | C2 非硬依赖 |
| 4 | `cs-agent-c4-eino-cs-agent` | Eino 客服骨架；SSE；Tool 可调交易搜索（有则接 C3） | C3 可选 |

## 文件锁

| Slice | 主写 | 勿抢 |
|-------|------|------|
| C1 | `docker/`、`deploy/observability/`、`pkg/metrics/`、`middleware`、飞书 webhook 配置 | 勿改 queue 语义 |
| C2 | `pkg/queue/`、`cmd/worker`、`cmd/api` 入队点、compose rabbitmq | 勿改 ES |
| C3 | `internal/repository/es`、transaction usecase/controller 搜索、compose ES | 勿删 queue 迁移结果 |
| C4 | 客服 usecase/路由、eino 依赖 | 勿改小石头 completions 契约 |

## Deferred / Removed

| 项 | 状态 |
|----|------|
| 保留 Asynq 并行双写 | Removed（人要求替代） |
| 全表灌 ES | Removed |
| 生产 Grafana SSO | Deferred |

## Next
人批 Brief → executor C1。
