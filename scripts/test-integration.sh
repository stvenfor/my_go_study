#!/usr/bin/env bash
# test-integration.sh — harden-h3：跑 -tags=integration 用例，绝不连业务库 my_go_study
#
# 两种模式：
#   * 本机（默认）：docker run 一次性 postgres:16-alpine，POSTGRES_DB=h3_it 在容器内
#     自建独立库，随机宿主端口；跑完 trap 删除容器。业务库 my_go_study 完全不碰。
#   * 外部库 / CI：H3_IT_EXTERNAL_DB=1 时直接用 DATABASE_* 指向已有 PG service
#     （GitHub Actions 的 integration job 用此模式，service 里同样自建独立库 h3_it）。
#
# 用例读的 env（见 app/internal/*/*_integration_test.go 的 openXxxDB）：
#   DATABASE_HOST / DATABASE_PORT / DATABASE_USER / DATABASE_PASSWORD
#   P02_PG_DBNAME / P03_PG_DBNAME
# 这里把两个库名都指向同一个 scratch 库 h3_it；配合 -p 1 串行跑各包测试，
# 避免并行包在同一库上 DDL 竞态（各用例 t.Cleanup 会 DropTable）。
#
# 工具链由调用方钉死：CI = setup-go 1.26.x + GOTOOLCHAIN=local；
# 本机 = PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local make test-integration。
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

DB_NAME="${H3_IT_DBNAME:-h3_it}"
export P02_PG_DBNAME="${P02_PG_DBNAME:-$DB_NAME}"
export P03_PG_DBNAME="${P03_PG_DBNAME:-$DB_NAME}"

CONTAINER=""
cleanup() {
  if [ -n "$CONTAINER" ]; then
    docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

if [ "${H3_IT_EXTERNAL_DB:-}" = "1" ]; then
  echo "[test-integration] 外部库模式（CI service）：${DATABASE_HOST}:${DATABASE_PORT} db=${DB_NAME}"
else
  command -v docker >/dev/null 2>&1 || { echo "需要 docker（本机一次性 PG16）" >&2; exit 1; }
  CONTAINER="h3-it-pg-$$"
  echo "[test-integration] 起一次性 PG16 容器 ${CONTAINER}（POSTGRES_DB=${DB_NAME}）"
  docker run -d --rm --name "$CONTAINER" \
    -e POSTGRES_DB="$DB_NAME" -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=h3it \
    -p 127.0.0.1::5432 postgres:16-alpine >/dev/null
  if [ "$(docker inspect -f '{{.State.Running}}' "$CONTAINER" 2>/dev/null || echo false)" != "true" ]; then
    echo "容器未在运行，docker logs：" >&2; docker logs "$CONTAINER" >&2 2>&1 || true; exit 1
  fi
  port_map="$(docker port "$CONTAINER" 5432/tcp | head -1)"   # 形如 127.0.0.1:49154
  export DATABASE_HOST=127.0.0.1
  export DATABASE_PORT="${port_map##*:}"
  export DATABASE_USER=postgres
  export DATABASE_PASSWORD=h3it
  ready=0
  for _ in $(seq 1 60); do
    if docker exec "$CONTAINER" pg_isready -U postgres -d "$DB_NAME" >/dev/null 2>&1; then ready=1; break; fi
    if [ "$(docker inspect -f '{{.State.Running}}' "$CONTAINER" 2>/dev/null || echo false)" != "true" ]; then break; fi
    sleep 1
  done
  if [ "$ready" -ne 1 ]; then
    echo "PG 未就绪，docker logs：" >&2; docker logs "$CONTAINER" >&2 2>&1 || true; exit 1
  fi
  echo "[test-integration] PG 就绪于 ${DATABASE_HOST}:${DATABASE_PORT}"
fi

echo "[test-integration] go -C app test -tags=integration -p 1 ./... -count=1（P02=${P02_PG_DBNAME} P03=${P03_PG_DBNAME}）"
go -C app test -tags=integration -p 1 ./... -count=1
