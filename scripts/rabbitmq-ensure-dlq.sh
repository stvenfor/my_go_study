#!/usr/bin/env bash
# RabbitMQ 死信拓扑迁移助手（parity-p0-6）。
#
# 背景：队列参数不可变。给 configs/config.yaml 的 queue.rabbitmq 加了
# x-dead-letter-* 之后，对**已存在**的同名队列再声明会 406 PRECONDITION_FAILED，
# 应用侧会降级为「沿用旧队列」并打 Warn（DLQ 不生效）。本脚本用于把旧队列迁走。
#
# 用法：
#   scripts/rabbitmq-ensure-dlq.sh status                 # 看当前队列 args 与 DLQ 是否存在
#   scripts/rabbitmq-ensure-dlq.sh recreate --yes          # 删旧队列重建（会丢在途消息，仅限测试环境）
#   scripts/rabbitmq-ensure-dlq.sh rename                  # 打印生产双写迁移步骤（不改任何东西）
#
# 依赖：docker exec + rabbitmqctl（容器名见 docker/docker-compose.yml）。
set -euo pipefail

CONTAINER="${RABBITMQ_CONTAINER:-my_go_study_rabbitmq}"
QUEUE="${RABBITMQ_QUEUE:-my_go_study.tasks}"
DLQ="${RABBITMQ_DLQ:-${QUEUE}.dlq}"
DLX="${RABBITMQ_DLX:-${QUEUE}.dlx}"

ctl() {
  if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER"; then
    echo "❌ 未找到运行中的 RabbitMQ 容器：${CONTAINER}（先 make lan-up，或设 RABBITMQ_CONTAINER）" >&2
    exit 1
  fi
  docker exec "$CONTAINER" rabbitmqctl "$@"
}

cmd="${1:-status}"

case "$cmd" in
  status)
    echo "# vhost=/ queue=$QUEUE dlq=$DLQ dlx=$DLX"
    echo "--- queues（messages / unacked / consumers）---"
    ctl list_queues name messages messages_unacknowledged consumers
    echo "--- exchanges ---"
    ctl list_exchanges name type
    # rabbitmqctl 不导出队列 arguments，要看 x-dead-letter-* 用 management API：
    #   curl -su guest:guest http://127.0.0.1:15672/api/queues | \
    #     python3 -c "import json,sys;[print(q['name'],q['arguments']) for q in json.load(sys.stdin)]"
    ;;

  recreate)
    # 删队列是不可逆操作：必须显式 --yes，且提示在途消息会丢。
    if [ "${2:-}" != "--yes" ]; then
      echo "❌ 需要显式确认：$0 recreate --yes（会丢弃 $QUEUE 中未消费的消息）" >&2
      exit 1
    fi
    if [ -n "${APP_ENV:-}" ] && [ "$APP_ENV" != "dev" ] && [ "$APP_ENV" != "lan" ] && [ "$APP_ENV" != "test" ]; then
      echo "❌ APP_ENV=$APP_ENV 不是联调环境，禁止删队列；生产请走 $0 rename" >&2
      exit 1
    fi
    echo "# 删旧队列与 DLQ（DLQ 里已有的死信也会没，先备份：rabbitmqctl export_definitions）"
    ctl delete_queue "$QUEUE" || true
    ctl delete_queue "$DLQ" || true
    echo "✅ 已删除；下次 API/Worker 启动会自动声明带 x-dead-letter-* 的新队列 + $DLX/$DLQ"
    ;;

  rename)
    # 零丢消息路径：队列名是配置项（queue.rabbitmq.queue / RABBITMQ_QUEUE），
    # 所以「改名」比「原地改 args」更安全，代价是需要一个消费端切换窗口。
    cat <<EOF
生产迁移步骤（不丢消息）：
  1) 部署新队列名：给 API 与 Worker 设 RABBITMQ_QUEUE=${QUEUE}.v2
     （DLX/DLQ 会派生为 ${QUEUE}.v2.dlx / ${QUEUE}.v2.dlq）
  2) make lan-up（或滚动重启）→ 新入队落到 ${QUEUE}.v2，Worker 同时消费两者需起两份：
       RABBITMQ_QUEUE=${QUEUE}.v2 ./scripts/... worker
  3) 等旧队列排空：ctl list_queues 中 $QUEUE 的 messages=0
  4) 删旧队列与绑定：rabbitmqctl delete_queue $QUEUE
  5) 告警侧确认 rabbitmq_dlq_messages{queue="${QUEUE}.v2"} 正常出数
EOF
    ;;

  *)
    echo "用法: $0 [status|recreate --yes|rename]" >&2
    exit 2
    ;;
esac
