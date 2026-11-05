#!/usr/bin/env bash
# migrate-entrypoint.sh — compose 一次性 migrate 服务的入口（parity-p0-1）。
#
# 为什么需要它：API/Worker 启动期不再建表（database.auto_migrate=false），
# schema 由这里在 app 起来之前推平；app 服务用 depends_on:
# service_completed_successfully 等它退出码 0。
#
# 三种库状态：
#   1) 有 schema_migrations      → 直接 up（正常增量）
#   2) 无版本记录 + 有 public.users → 历史 AutoMigrate 建的库，基线化到最新再 up
#      （判据与 app/internal/schemaboot 一致：users 是本地 Auth 的第一张表）
#   3) 无版本记录 + 空库          → 直接 up（全量建表）
# 状态 2 想拒绝自动基线：MIGRATE_BASELINE=off（此时脚本非 0 退出，app 不会启动）。
set -euo pipefail

MIGRATE_BIN="${MIGRATE_BIN:-/app/migrate}"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-/app/migrations}"
DB_HOST="${DATABASE_HOST:-postgres}"
DB_PORT="${DATABASE_PORT:-5432}"
DB_USER="${DATABASE_USER:-postgres}"
DB_NAME="${DATABASE_DBNAME:-my_go_study}"
DB_SSLMODE="${DATABASE_SSLMODE:-disable}"
export PGPASSWORD="${DATABASE_PASSWORD:?DATABASE_PASSWORD 未设置}"

# golang-migrate 只吃 URL；口令含 : / @ ? 时需先 percent-encode（本机/内网口令无此问题）。
DSN="postgres://${DB_USER}:${PGPASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}"

latest="$(ls "$MIGRATIONS_DIR"/*.up.sql | sed -E 's#.*/([0-9]+)_.*#\1#' | sort -n | tail -1)"
echo "[migrate] 迁移目录=$MIGRATIONS_DIR 最新=$latest"

state="$("$MIGRATE_BIN" -path "$MIGRATIONS_DIR" -database "$DSN" version 2>&1 || true)"

if [[ "$state" == *"no migration"* ]]; then
  has_users="$(psql "host=$DB_HOST port=$DB_PORT user=$DB_USER dbname=$DB_NAME sslmode=$DB_SSLMODE" \
    -Atc "select to_regclass('public.users') is not null")"
  if [[ "$has_users" == "t" ]]; then
    if [[ "${MIGRATE_BASELINE:-auto}" == "off" ]]; then
      echo "[migrate] 库由历史 AutoMigrate 建立且无版本记录；已按要求拒绝自动基线。"
      echo "[migrate] 确认该库 schema 已是最新后：make migrate-baseline（或去掉 MIGRATE_BASELINE=off）"
      exit 1
    fi
    echo "[migrate] 检测到历史 AutoMigrate 库：基线化到 ${latest}（不重放 DDL）"
    "$MIGRATE_BIN" -path "$MIGRATIONS_DIR" -database "$DSN" force "$latest"
  fi
elif [[ "$state" == *"(dirty)"* ]]; then
  echo "[migrate] 库处于 dirty（${state}）：上一次迁移中断，需人工修好后 make migrate-force VERSION=<n>"
  exit 1
fi

"$MIGRATE_BIN" -path "$MIGRATIONS_DIR" -database "$DSN" up
echo "[migrate] 完成，当前版本：$("$MIGRATE_BIN" -path "$MIGRATIONS_DIR" -database "$DSN" version)"
