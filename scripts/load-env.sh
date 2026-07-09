#!/usr/bin/env bash
# 加载 .env + .env.local 后执行命令（用于 make run 等）。
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
# shellcheck disable=SC1091
source "$ROOT/scripts/source-env.sh"
exec "$@"
