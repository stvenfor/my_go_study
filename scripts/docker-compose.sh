#!/usr/bin/env bash
# 开发全栈：加载 supabase/.env/.env.local 后执行 docker compose（仅基础文件）。
# 用法：./scripts/docker-compose.sh up -d --build
#       ./scripts/docker-compose.sh down
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# shellcheck disable=SC1091
source "$ROOT/scripts/source-env.sh"

exec docker compose -f "$ROOT/docker/docker-compose.yml" "$@"
