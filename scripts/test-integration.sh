#!/usr/bin/env bash
# test-integration.sh — harden-h3：跑 -tags=integration 用例，绝不连业务库 my_go_study
#
# 两种模式：
#   * 本机（默认）：docker run 一次性 postgres:16-alpine，POSTGRES_DB=h3_it 在容器内
#     自建独立库，随机宿主端口；另建 p06b_it / p06b_am / p08_it，跑完 trap 删除容器。
#     业务库 my_go_study 完全不碰。
#   * 外部库 / CI：H3_IT_EXTERNAL_DB=1 时直接用 DATABASE_* 指向已有 PG service
#     （GitHub Actions 的 integration job 用此模式；service 自建 h3_it，ci.yml 的
#      幂等建库步补齐另外三个，本脚本再兜底一次，缺库即显式报错退出）。
#     用 pg_isready -h/-p 探活，未就绪 exit 1，不允许静默跳过。
#
# 用例读的 env（见 app/internal/*/*_integration_test.go 的 openXxxDB）：
#   DATABASE_HOST / DATABASE_PORT / DATABASE_USER / DATABASE_PASSWORD
#   P02_PG_DBNAME / P03_PG_DBNAME / P06B_PG_DBNAME / P06B_AM_DBNAME / P08_PG_DBNAME
# P02/P03 指同一 scratch 库 h3_it（用例自己 AutoMigrate）；p06b_it 必须跑 migrations/
# 全链，金额分用例断言 amount_fen 是 bigint NOT NULL + CHECK；p06b_am / p08_it 是
# 独立空库（用例自己 AutoMigrate），绝不与跑过迁移的库共用，否则形状断言失效。
# 配合 -p 1 串行跑各包测试，避免并行包在同一库上 DDL 竞态（各用例 t.Cleanup 会 DropTable）。
#
# 门禁防「空绿 / 跳过失败」：-v 输出统计 --- PASS: / --- FAIL: / --- SKIP:，
# go test 退出码（PIPESTATUS[0]）非 0、任一 FAIL/SKIP、或 PASS 数低于 integration
# 测试函数总数，都 exit 1；杜绝「用例被 t.Skipf 跳过仍报绿」。
# -run 只跑 *_integration_test.go 里的 21 个用例：`./...` 还会编译/执行若干
# 「需要活 WS 服务」的非集成 live 用例（如 ws/handler_test.go、ws/integration_test.go，
# 无 //go:build integration），它们在无 :8080 服务时必然 t.Skip；这些文件不在本轮
# 白名单，故用 -run 把门禁收口到集成用例集合本身，而不是放宽 SKIP 判定。
#
# 工具链由调用方钉死：CI = setup-go 1.26.x + GOTOOLCHAIN=local；
# 本机 = PATH=/usr/local/go/bin:$PATH GOTOOLCHAIN=local make test-integration。
set -euo pipefail
cd "$(git rev-parse --show-toplevel)"

DB_NAME="${H3_IT_DBNAME:-h3_it}"
export P02_PG_DBNAME="${P02_PG_DBNAME:-$DB_NAME}"
export P03_PG_DBNAME="${P03_PG_DBNAME:-$DB_NAME}"
export P06B_PG_DBNAME="${P06B_PG_DBNAME:-p06b_it}"
export P06B_AM_DBNAME="${P06B_AM_DBNAME:-p06b_am}"
export P08_PG_DBNAME="${P08_PG_DBNAME:-p08_it}"

# 需要 migrations/ 全链的库（金额分形状断言）；其余只建空库。
MIGRATED_DB="$P06B_PG_DBNAME"
PLAIN_DBS=("$P06B_AM_DBNAME" "$P08_PG_DBNAME")

CONTAINER=""
IT_LOG=""
cleanup() {
  if [ -n "$CONTAINER" ]; then
    docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
  fi
  if [ -n "$IT_LOG" ]; then
    rm -f "$IT_LOG" || true
  fi
}
trap cleanup EXIT

# psql 统一入口：本机走 docker exec，外部库走宿主 psql；ON_ERROR_STOP 让失败显式。
psql_run() {
  if [ -n "$CONTAINER" ]; then
    docker exec -i "$CONTAINER" psql -v ON_ERROR_STOP=1 -U postgres "$@"
  else
    PGPASSWORD="$DATABASE_PASSWORD" psql -v ON_ERROR_STOP=1 \
      -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" "$@"
  fi
}

# 幂等建库；连不上/建不了都打印明确错误并 exit 1，不静默跳过。
create_db_if_missing() {
  local db="$1" exists
  exists="$(psql_run -d postgres -Atqc "SELECT 1 FROM pg_database WHERE datname = '$db'")" || {
    echo "[test-integration] 无法查询库 $db 是否存在（psql 连不上）" >&2; exit 1
  }
  if [ "$exists" = "1" ]; then
    echo "[test-integration] 库已存在：$db"
    return 0
  fi
  echo "[test-integration] 建库：$db"
  psql_run -d postgres -c "CREATE DATABASE $db" >/dev/null || {
    echo "[test-integration] 建库失败：$db（不允许静默跳过）" >&2; exit 1
  }
}

# 对 MIGRATED_DB 应用 migrations/ 全链（按文件名序，等价 migrate up 的 up 序列）。
# money_fen 用例断言列来自迁移而非 AutoMigrate，空库直接跑会 Fail。
apply_migrations() {
  local db="$1" f shape
  shape="$(psql_run -d "$db" -Atqc \
    "SELECT count(*) FROM information_schema.columns WHERE table_name='transactions' AND column_name='amount_fen' AND is_nullable='NO'")" || {
    echo "[test-integration] 无法读取 $db 的迁移形状" >&2; exit 1
  }
  if [ "$shape" = "1" ]; then
    echo "[test-integration] $db 已是迁移形状，跳过 migrations/"
    return 0
  fi
  echo "[test-integration] 对 $db 应用 migrations/*.up.sql（按文件名序）"
  for f in $(ls migrations/*.up.sql | sort); do
    psql_run -d "$db" -q -f - < "$f" >/dev/null || {
      echo "[test-integration] 迁移失败：$f（目标库 $db）" >&2; exit 1
    }
  done
}

if [ "${H3_IT_EXTERNAL_DB:-}" = "1" ]; then
  echo "[test-integration] 外部库模式（CI service）：${DATABASE_HOST}:${DATABASE_PORT} db=${DB_NAME}"
  command -v psql >/dev/null 2>&1 || { echo "[test-integration] 外部库模式需要 psql" >&2; exit 1; }
  command -v pg_isready >/dev/null 2>&1 || { echo "[test-integration] 外部库模式需要 pg_isready" >&2; exit 1; }
  for v in DATABASE_HOST DATABASE_PORT DATABASE_USER DATABASE_PASSWORD; do
    if [ -z "${!v:-}" ]; then
      echo "[test-integration] 外部库模式缺少环境变量 $v" >&2; exit 1
    fi
  done
  ready=0
  for _ in $(seq 1 60); do
    if PGPASSWORD="$DATABASE_PASSWORD" pg_isready \
        -h "$DATABASE_HOST" -p "$DATABASE_PORT" -U "$DATABASE_USER" -d "$DB_NAME" >/dev/null 2>&1; then
      ready=1; break
    fi
    sleep 1
  done
  if [ "$ready" -ne 1 ]; then
    echo "[test-integration] 外部库 PG 未就绪（${DATABASE_HOST}:${DATABASE_PORT} db=${DB_NAME}），不允许跳过" >&2
    exit 1
  fi
  echo "[test-integration] 外部库 PG 就绪于 ${DATABASE_HOST}:${DATABASE_PORT}"
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

create_db_if_missing "$DB_NAME"
create_db_if_missing "$MIGRATED_DB"
for db in "${PLAIN_DBS[@]}"; do create_db_if_missing "$db"; done
apply_migrations "$MIGRATED_DB"

# 期望 PASS 数 = 所有 *_integration_test.go 用例里的顶层 Test 函数个数（当前 21）。
IT_TEST_NAMES="$(grep -rh '^func Test' app/internal --include='*_integration_test.go' \
  | sed -E 's/^func (Test[A-Za-z0-9_]+).*/\1/' | sort -u)"
EXPECTED_PASS="$(printf '%s\n' "$IT_TEST_NAMES" | grep -c .)"
RUN_RE="$(printf '%s\n' "$IT_TEST_NAMES" | paste -sd '|' -)"
IT_LOG="$(mktemp)"
echo "[test-integration] go -C app test -tags=integration -p 1 ./... -count=1 -v -run <${EXPECTED_PASS} 个集成用例>（PASS 期望 ≥ ${EXPECTED_PASS}；FAIL/SKIP 必须为 0）"
set -o pipefail
set +e
go -C app test -tags=integration -p 1 ./... -count=1 -v -run "^(${RUN_RE})$" 2>&1 | tee "$IT_LOG"
go_status=${PIPESTATUS[0]}
set -e

n_pass="$(grep -c -- '--- PASS:' "$IT_LOG" || true)"
n_fail="$(grep -c -- '--- FAIL:' "$IT_LOG" || true)"
n_skip="$(grep -c -- '--- SKIP:' "$IT_LOG" || true)"
echo "[test-integration] 结果：go_exit=${go_status} PASS=${n_pass}（期望 ≥ ${EXPECTED_PASS}） FAIL=${n_fail} SKIP=${n_skip}"

if [ "$go_status" -ne 0 ]; then
  echo "FAIL: go test 退出码 ${go_status}（不允许跳过失败）" >&2
  exit 1
fi
if [ "$n_fail" -gt 0 ]; then
  echo "FAIL: 出现 ${n_fail} 个 --- FAIL:" >&2
  grep -n -- '--- FAIL:' "$IT_LOG" >&2 || true
  exit 1
fi
if [ "$n_skip" -gt 0 ]; then
  echo "FAIL: 出现 ${n_skip} 个 --- SKIP:（不允许跳过失败 / 空跑）" >&2
  grep -n -- '--- SKIP:' "$IT_LOG" >&2 || true
  exit 1
fi
if [ "$n_pass" -lt "$EXPECTED_PASS" ]; then
  echo "FAIL: PASS 数 ${n_pass} < 期望 ${EXPECTED_PASS}（可能有用例未执行 / 空跑）" >&2
  exit 1
fi
echo "OK: integration 门禁通过（PASS=${n_pass} ≥ ${EXPECTED_PASS}，FAIL=0，SKIP=0）"
