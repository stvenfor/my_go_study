# Acceptance — cs-agent-c2-rabbitmq-replace-asynq

- Date: 2026-09-28
- Mode: **Partial**
- harness: local `go build` + `go test ./pkg/queue`（未在本机实跑完整 `lan-up` / `make test-queue-push` 人证）

## 机跑

- `go build -o /tmp/my_go_study_api ./cmd/api` ✅
- `go build -o /tmp/my_go_study_worker ./cmd/worker` ✅
- `go test ./pkg/queue/... -count=1` ✅

## 已交付

- `pkg/queue`：RabbitMQ publish/consume；信封 `{type,task_id,payload}`；任务类型对齐
- 删除 `hibiken/asynq`；依赖 `github.com/rabbitmq/amqp091-go`
- Worker：AMQP 消费 + 整点 ticker（替代 Asynq Scheduler）
- compose：`rabbitmq` 服务；app/worker `RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/`
- 文档：`docs/message-queue.md`、`.env.example`、`.env.lan.example`、`AGENTS.md` 队列表述

## Deferred / 人证

- `make lan-up` 后 `make test-queue-push` / `make test-scheduled-notify`
- 定时广播整点 ticker 实机人证
- 生产级镜像队列 / DLX 重试策略
