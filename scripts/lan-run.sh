#!/usr/bin/env bash
# 无 Docker：用本机 Postgres/Redis 跑 lan 模式 API 或 Worker。
# 用法：./scripts/lan-run.sh api
#       ./scripts/lan-run.sh worker
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

TARGET="${1:-api}"

if [[ ! -f "$ROOT/.env.lan" ]]; then
  echo "错误: 缺少 .env.lan。请先: cp .env.lan.example .env.lan 并填写 REALTIME_PUBLIC_WS_HOST"
  exit 1
fi

# shellcheck disable=SC1091
source "$ROOT/scripts/source-env.sh"
set -a
# shellcheck disable=SC1091
source "$ROOT/.env.lan"
set +a

export APP_ENV="${APP_ENV:-lan}"
export AUTH_PROVIDER="${AUTH_PROVIDER:-local}"

if [[ -z "${REALTIME_PUBLIC_WS_HOST:-}" || "${REALTIME_PUBLIC_WS_HOST}" == "YOUR_LAN_IP" ]]; then
  echo "错误: 请在 .env.lan 中设置 REALTIME_PUBLIC_WS_HOST 为 Mac 局域网 IP"
  exit 1
fi

case "$TARGET" in
  api)
    echo "启动 API：APP_ENV=$APP_ENV AUTH_PROVIDER=$AUTH_PROVIDER WS_HOST=$REALTIME_PUBLIC_WS_HOST"
    exec go run ./cmd/api
    ;;
  worker)
    echo "启动 Worker：APP_ENV=$APP_ENV"
    exec go run ./cmd/worker
    ;;
  *)
    echo "用法: $0 api|worker"
    exit 2
    ;;
esac
