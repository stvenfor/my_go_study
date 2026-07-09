#!/usr/bin/env bash
# 供其它脚本 source：从项目根加载 .env + .env.local
_ENV_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
set -a
if [[ -f "$_ENV_ROOT/.env" ]]; then
  # shellcheck disable=SC1091
  source "$_ENV_ROOT/.env"
fi
if [[ -f "$_ENV_ROOT/.env.local" ]]; then
  # shellcheck disable=SC1091
  source "$_ENV_ROOT/.env.local"
fi
set +a
