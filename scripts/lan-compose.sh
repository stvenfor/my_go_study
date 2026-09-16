#!/usr/bin/env bash
# LAN Backend Host：加载 supabase/.env/.env.local/.env.lan 后执行 docker compose（基础 + lan overlay）。
# 用法：./scripts/lan-compose.sh up -d --build
#       ./scripts/lan-compose.sh down
# 改 Go 代码后须带 --build 重建；仅改 env 也建议 down 后再 up。
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# shellcheck disable=SC1091
source "$ROOT/scripts/source-env.sh"

if [[ ! -f "$ROOT/.env.lan" ]]; then
  echo "错误: 缺少 .env.lan。请先: cp .env.lan.example .env.lan 并填写 REALTIME_PUBLIC_WS_HOST"
  exit 1
fi

set -a
# shellcheck disable=SC1091
source "$ROOT/.env.lan"
set +a

if [[ -z "${REALTIME_PUBLIC_WS_HOST:-}" || "${REALTIME_PUBLIC_WS_HOST}" == "YOUR_LAN_IP" ]]; then
  echo "错误: 请在 .env.lan 中把 REALTIME_PUBLIC_WS_HOST 设为 Mac 的固定局域网 IP"
  exit 1
fi

export APP_ENV="${APP_ENV:-lan}"

exec docker compose \
  -f "$ROOT/docker/docker-compose.yml" \
  -f "$ROOT/docker/docker-compose.lan.yml" \
  "$@"
